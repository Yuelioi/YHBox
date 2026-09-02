package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/aiauthoring"
	"github.com/yottaapp/yotta/internal/apperr"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/runid"
)

type aiNativeFactory func(ai.ModelProfile) (ai.Provider, error)

const aiConnectionTestTimeout = 20 * time.Second

// AIService exposes transient credential, verification, and authoring
// operations without returning stored secret values.
type AIService struct {
	app       *App
	secrets   *AISecrets
	newNative aiNativeFactory
	authoring *aiauthoring.Manager
}

func NewAIService(app *App, secrets *AISecrets, authoring ...*aiauthoring.Manager) *AIService {
	service := newAIService(app, secrets, func(profile ai.ModelProfile) (ai.Provider, error) {
		return ai.NewNativeProvider(profile, ai.HTTPOptions{})
	})
	if len(authoring) != 0 {
		service.authoring = authoring[0]
	}
	return service
}

func (s *AIService) ProposeWorkflow(slot, workflowID string, baseRevision int64, instruction, runID string) (aiauthoring.Review, error) {
	runtime, err := s.authoringRuntime(slot)
	if err != nil {
		return aiauthoring.Review{}, err
	}
	review, err := s.authoring.Propose(context.Background(), runtime, aiauthoring.ProposeRequest{
		WorkflowID: workflowID, RunID: runID, BaseRevision: baseRevision, Instruction: instruction, TrustClass: "user-authored",
	})
	return review, projectAIAuthoringError(err)
}

func (s *AIService) ListWorkflowAIConversations(workflowID string) ([]aiauthoring.ConversationSummary, error) {
	if s.authoring == nil {
		return nil, apperr.New("ai.authoring.unavailable", nil)
	}
	items, err := s.authoring.ListConversations(workflowID)
	return items, projectAIAuthoringError(err)
}

func (s *AIService) CreateWorkflowAIConversation(workflowID string) (aiauthoring.Conversation, error) {
	if s.authoring == nil {
		return aiauthoring.Conversation{}, apperr.New("ai.authoring.unavailable", nil)
	}
	conversation, err := s.authoring.CreateConversation(workflowID)
	return conversation, projectAIAuthoringError(err)
}

func (s *AIService) GetWorkflowAIConversation(workflowID, conversationID string) (aiauthoring.Conversation, error) {
	if s.authoring == nil {
		return aiauthoring.Conversation{}, apperr.New("ai.authoring.unavailable", nil)
	}
	conversation, err := s.authoring.GetConversation(workflowID, conversationID)
	return conversation, projectAIAuthoringError(err)
}

func (s *AIService) DeleteWorkflowAIConversation(workflowID, conversationID string) error {
	if s.authoring == nil {
		return apperr.New("ai.authoring.unavailable", nil)
	}
	return projectAIAuthoringError(s.authoring.DeleteConversation(workflowID, conversationID))
}

func (s *AIService) SendWorkflowAIMessage(slot, workflowID, conversationID string, baseRevision int64, instruction, runID string) (aiauthoring.Conversation, error) {
	runtime, err := s.authoringRuntime(slot)
	if err != nil {
		if s.authoring != nil {
			envelope := apperr.From(err)
			_, _ = s.authoring.RecordFailedTurn(workflowID, conversationID, instruction, envelope.ID, envelope.OperationID)
		}
		return aiauthoring.Conversation{}, err
	}
	conversation, _, err := s.authoring.ProposeTurn(context.Background(), runtime, aiauthoring.ConversationTurnRequest{
		ConversationID: conversationID, WorkflowID: workflowID, RunID: runID, BaseRevision: baseRevision,
		Instruction: instruction, TrustClass: "user-authored",
		Progress: func(progress aiauthoring.ConversationProgress) { s.app.Emit("ai:conversation-progress", progress) },
	})
	projected := projectAIAuthoringError(err)
	if projected != nil {
		envelope := apperr.From(projected)
		conversation, _ = s.authoring.RecordFailure(workflowID, conversationID, envelope.ID, envelope.OperationID)
	}
	return conversation, projected
}

func (s *AIService) authoringRuntime(slot string) (aiauthoring.Runtime, error) {
	if s.authoring == nil || s.app == nil || s.secrets == nil {
		return aiauthoring.Runtime{}, apperr.New("ai.authoring.unavailable", nil)
	}
	configured := findAIProfile(s.app.Settings(), slot)
	if configured == nil {
		return aiauthoring.Runtime{}, apperr.New("ai.authoring.profile_not_found", map[string]any{"slot": slot})
	}
	profile, err := ai.SealModelProfile(configured.profileDraft())
	if err != nil {
		return aiauthoring.Runtime{}, fmt.Errorf("%w: %v", apperr.New("ai.authoring.profile_invalid", map[string]any{"slot": slot}), err)
	}
	if !profile.Machine().Capabilities.ToolCalling {
		return aiauthoring.Runtime{}, apperr.New("ai.authoring.tool_calling_required", map[string]any{"slot": slot})
	}
	credential := "codex-subscription"
	if profile.Machine().Provider != ai.ProviderCodexSubscription {
		credential, err = s.secrets.Get(ai.CredentialBindingID(slot))
		if err != nil || credential == "" {
			return aiauthoring.Runtime{}, apperr.New("ai.authoring.credential_unavailable", map[string]any{"slot": slot})
		}
	}
	provider, err := s.newNative(profile)
	if err != nil {
		return aiauthoring.Runtime{}, fmt.Errorf("%w: %v", apperr.New("ai.authoring.provider_unavailable", map[string]any{"slot": slot, "provider": profile.Machine().Provider}), err)
	}
	agent, ok := provider.(ai.AgentProvider)
	if !ok {
		return aiauthoring.Runtime{}, apperr.New("ai.authoring.agent_unsupported", map[string]any{"slot": slot})
	}
	return aiauthoring.Runtime{Profile: profile, Provider: agent, Credential: credential}, nil
}

func projectAIAuthoringError(err error) error {
	if err == nil {
		return nil
	}
	var failure *ai.ProviderFailure
	if errors.As(err, &failure) {
		return fmt.Errorf("%w: %v", apperr.New("ai.authoring.provider_failed", map[string]any{"stage": failure.Stage, "class": failure.Class, "retry": failure.Retry}), err)
	}
	var toolInput *aiauthoring.ToolInputError
	if errors.As(err, &toolInput) {
		return fmt.Errorf("%w: %v", apperr.New("ai.authoring.tool_input_invalid", map[string]any{"tool": toolInput.Tool}), err)
	}
	var envelopeProvider apperr.EnvelopeProvider
	if errors.As(err, &envelopeProvider) {
		return err
	}
	if errors.Is(err, aiauthoring.ErrConversationNotFound) {
		return fmt.Errorf("%w: %v", apperr.New("ai.authoring.conversation_not_found", nil), err)
	}
	if errors.Is(err, aiauthoring.ErrConversationCapacity) {
		return fmt.Errorf("%w: %v", apperr.New("ai.authoring.conversation_capacity", nil), err)
	}
	if errors.Is(err, aiauthoring.ErrDiagnosticRunUnavailable) {
		return fmt.Errorf("%w: %v", apperr.New("ai.authoring.run_unavailable", nil), err)
	}
	if errors.Is(err, aiauthoring.ErrDiagnosticRunWorkflowMismatch) {
		return fmt.Errorf("%w: %v", apperr.New("ai.authoring.run_workflow_mismatch", nil), err)
	}
	return fmt.Errorf("%w: %v", apperr.New("ai.authoring.failed", nil), err)
}

func (s *AIService) AcceptWorkflowProposal(reviewID string) (aiauthoring.Review, error) {
	if s.authoring == nil {
		return aiauthoring.Review{}, errors.New("AI workflow authoring is unavailable")
	}
	return s.authoring.Accept(context.Background(), reviewID)
}

func (s *AIService) RejectWorkflowProposal(reviewID string) (aiauthoring.Review, error) {
	if s.authoring == nil {
		return aiauthoring.Review{}, errors.New("AI workflow authoring is unavailable")
	}
	return s.authoring.Reject(reviewID)
}

func (s *AIService) GetWorkflowProposal(reviewID string) (aiauthoring.Review, error) {
	if s.authoring == nil {
		return aiauthoring.Review{}, errors.New("AI workflow authoring is unavailable")
	}
	return s.authoring.Get(reviewID)
}

func newAIService(app *App, secrets *AISecrets, factory aiNativeFactory) *AIService {
	return &AIService{app: app, secrets: secrets, newNative: factory}
}

type TestProfileRequest struct {
	Profile AIModelSettings `json:"profile"`
}

type TestProfileResult struct {
	Ok             bool             `json:"ok"`
	Provider       ai.ProviderKind  `json:"provider"`
	RequestedModel string           `json:"requestedModel"`
	ResolvedModel  string           `json:"resolvedModel"`
	Finish         ai.FinishKind    `json:"finish"`
	FailureClass   ai.FailureClass  `json:"failureClass,omitempty"`
	HTTPStatus     int              `json:"httpStatus,omitempty"`
	Problem        *apperr.Envelope `json:"problem,omitempty"`
}

// TestProfile performs one explicit provider-native generation against the
// exact configured model. It never probes compatibility endpoints or falls
// back to a Chat emulation path.
func (s *AIService) TestProfile(request TestProfileRequest) TestProfileResult {
	profile, err := ai.SealModelProfile(request.Profile.profileDraft())
	if err != nil {
		return aiTestFailure(err)
	}
	if s.secrets == nil && profile.Machine().Provider != ai.ProviderCodexSubscription {
		return aiTestFailure(errors.New("AI credential is unavailable"))
	}
	credential := "codex-subscription"
	if profile.Machine().Provider != ai.ProviderCodexSubscription {
		credential, err = s.secrets.Get(ai.CredentialBindingID(request.Profile.Slot))
		if err != nil || credential == "" {
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
	maximum := int64(8)
	if configuredMaximum := profile.Machine().MaxOutputTokens; configuredMaximum > 0 {
		maximum = min(configuredMaximum, maximum)
	}
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
	ctx, cancel := context.WithTimeout(context.Background(), aiConnectionTestTimeout)
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
	envelope := apperr.From(err)
	result := TestProfileResult{Problem: &envelope}
	var failure *ai.ProviderFailure
	if errors.As(err, &failure) {
		result.FailureClass = failure.Class
		result.Problem.ID = "ai.provider." + string(failure.Class)
		result.Problem.Category = apperr.CategoryAdapter
		if failure.HTTPStatus != nil {
			result.HTTPStatus = *failure.HTTPStatus
		}
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

func (s *AIService) ApplyEvaluation(slot string, evidence ai.EvalReportArtifact) error {
	if s.app == nil {
		return errors.New("settings are unavailable")
	}
	report, err := evidence.Open()
	if err != nil {
		return err
	}
	builtins, err := nodes.Build()
	if err != nil {
		return err
	}
	document := report.Machine()
	_, current, err := s.app.MutateSettings(func(settings *Settings) error {
		configured := findAIProfile(settings, slot)
		if configured == nil {
			return fmt.Errorf("AI profile slot %q is not configured", slot)
		}
		configured.Evaluation = document.Decision
		configured.EvaluationSuite = document.Suite
		configured.EvaluationReport = evidence
		profile, sealErr := ai.SealModelProfile(configured.profileDraft())
		if sealErr != nil {
			return sealErr
		}
		validationErr := ai.ValidateEvaluationCandidate(profile, evidence, builtins.AIEvaluationArtifacts())
		if validationErr != nil && !(document.Decision == ai.EvaluationRejected && errors.Is(validationErr, ai.ErrEvaluationNotApproved)) {
			return validationErr
		}
		return nil
	})
	if err != nil && current == nil {
		return err
	}
	s.app.Emit("settings:changed", map[string]any{})
	return err
}

func (s *AIService) RevokeEvaluation(slot string) error {
	if s.app == nil {
		return errors.New("settings are unavailable")
	}
	_, current, err := s.app.MutateSettings(func(settings *Settings) error {
		configured := findAIProfile(settings, slot)
		if configured == nil {
			return fmt.Errorf("AI profile slot %q is not configured", slot)
		}
		configured.Evaluation = ai.EvaluationUnverified
		configured.EvaluationSuite = ""
		configured.EvaluationReport = ai.EvalReportArtifact{}
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
