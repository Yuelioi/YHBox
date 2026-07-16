package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/runid"
)

type aiNativeFactory func(ai.ModelProfile) (ai.Provider, error)

// AIService exposes transient credential, verification, and explicit
// workflow-consent operations without returning stored secret values.
type AIService struct {
	app       *App
	secrets   *AISecrets
	newNative aiNativeFactory
}

func NewAIService(app *App, secrets *AISecrets) *AIService {
	return newAIService(app, secrets, func(profile ai.ModelProfile) (ai.Provider, error) {
		return ai.NewNativeProvider(profile, ai.HTTPOptions{})
	})
}

func newAIService(app *App, secrets *AISecrets, factory aiNativeFactory) *AIService {
	return &AIService{app: app, secrets: secrets, newNative: factory}
}

type TestProfileRequest struct {
	Profile AIModelSettings `json:"profile"`
	APIKey  string          `json:"apiKey"`
}

type TestProfileResult struct {
	Ok             bool            `json:"ok"`
	Provider       ai.ProviderKind `json:"provider"`
	RequestedModel string          `json:"requestedModel"`
	ResolvedModel  string          `json:"resolvedModel"`
	Finish         ai.FinishKind   `json:"finish"`
	FailureClass   ai.FailureClass `json:"failureClass,omitempty"`
	Error          string          `json:"error,omitempty"`
}

// TestProfile performs one explicit provider-native generation against the
// exact configured model. It never probes compatibility endpoints or falls
// back to a Chat emulation path.
func (s *AIService) TestProfile(request TestProfileRequest) TestProfileResult {
	profile, err := ai.SealModelProfile(request.Profile.profileDraft())
	if err != nil {
		return aiTestFailure(err)
	}
	credential := request.APIKey
	if credential == "" && s.secrets != nil {
		credential, err = s.secrets.Get(ai.CredentialBindingID(request.Profile.Slot))
		if err != nil {
			return aiTestFailure(errors.New("AI credential is unavailable"))
		}
	}
	provider, err := s.newNative(profile)
	if err != nil {
		return aiTestFailure(err)
	}
	attemptID, err := runid.New()
	if err != nil {
		return aiTestFailure(err)
	}
	maximum := min(profile.Machine().MaxOutputTokens, int64(8))
	manifest, err := ai.SealPromptManifest(ai.PromptManifestDraft{
		ID: "yotta.ai.connection-test", Version: "1.0.0", Owner: "settings",
		Instructions: "Return the requested short connection-test response without adding unrelated content.",
	})
	if err != nil {
		return aiTestFailure(err)
	}
	prompt, err := ai.RenderPrompt(manifest, []ai.PromptBlock{{Kind: ai.PromptBlockUser, Content: "Reply with OK."}})
	if err != nil {
		return aiTestFailure(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	outcome, err := provider.Generate(ctx, credential, ai.GenerateRequest{
		AttemptID: attemptID, Prompt: prompt, Retention: ai.RetentionNoApplicationState,
		Limits: ai.GenerationLimits{MaxOutputTokens: &maximum},
	})
	if err != nil {
		return aiTestFailure(err)
	}
	return TestProfileResult{
		Ok: true, Provider: outcome.Provider, RequestedModel: outcome.RequestedModel,
		ResolvedModel: outcome.ResolvedModel, Finish: outcome.Finish.Kind,
	}
}

func aiTestFailure(err error) TestProfileResult {
	result := TestProfileResult{Error: err.Error()}
	var failure *ai.ProviderFailure
	if errors.As(err, &failure) {
		result.FailureClass = failure.Class
	}
	return result
}

func (s *AIService) SecretStatus(slots []string) map[string]bool {
	status := make(map[string]bool, len(slots))
	for _, slot := range slots {
		has, err := s.secrets.HasSlot(slot)
		status[slot] = err == nil && has
	}
	return status
}

func (s *AIService) SetAPIKey(slot, apiKey string) error {
	return s.secrets.SetSlot(slot, apiKey)
}

func (s *AIService) DeleteAPIKey(slot string) error {
	return s.secrets.DeleteSlot(slot)
}

func (s *AIService) GrantWorkflowUse(slot string) (string, error) {
	var consent string
	_, current, err := s.app.MutateSettings(func(settings *Settings) error {
		profile := findAIProfile(settings, slot)
		if profile == nil {
			return fmt.Errorf("AI profile slot %q is not configured", slot)
		}
		digest := expectedAIConsent(*profile)
		if !digest.Valid() {
			return errors.New("AI profile cannot produce workflow consent")
		}
		profile.WorkflowConsent = digest
		consent = digest.String()
		return nil
	})
	if err != nil && current == nil {
		return "", err
	}
	s.app.Emit("settings:changed", map[string]any{})
	return consent, err
}

func (s *AIService) RevokeWorkflowUse(slot string) error {
	_, current, err := s.app.MutateSettings(func(settings *Settings) error {
		profile := findAIProfile(settings, slot)
		if profile == nil {
			return fmt.Errorf("AI profile slot %q is not configured", slot)
		}
		profile.WorkflowConsent = ""
		return nil
	})
	if err != nil && current == nil {
		return err
	}
	s.app.Emit("settings:changed", map[string]any{})
	return err
}

func findAIProfile(settings *Settings, slot string) *AIModelSettings {
	for index := range settings.AI.Profiles {
		if settings.AI.Profiles[index].Slot == slot {
			return &settings.AI.Profiles[index]
		}
	}
	return nil
}
