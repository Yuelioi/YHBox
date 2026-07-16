package ai

import (
	"errors"
	"fmt"
	"regexp"
	"sort"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/resource"
)

const workflowConsentDomain = "yotta/ai-workflow-consent/v1"

var installationSlotPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$`)

type InstallationDraft struct {
	Slot       string
	Profile    ModelProfileDraft
	Evaluation EvalReportArtifact
	Consent    artifact.Digest
}

type Installation struct {
	Slot                string
	Profile             ModelProfile
	ProviderID          string
	ProviderArtifact    artifact.Digest
	TargetID            string
	CredentialBindingID string
	Consent             artifact.Digest
	Evaluation          EvalReportArtifact
	Provider            resource.Provider
}

type installationState struct {
	entries []Installation
	native  []Provider
}

// Installations is an immutable startup snapshot. It creates one native
// provider per distinct model profile while preserving one target and
// credential binding per logical workflow slot.
type Installations struct{ state *installationState }

func Install(drafts []InstallationDraft, credentials CredentialStore) (Installations, error) {
	ordered := append([]InstallationDraft(nil), drafts...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Slot < ordered[j].Slot })
	if len(ordered) != 0 && credentials == nil {
		return Installations{}, errors.New("AI installations require a credential store")
	}
	entries := make([]Installation, 0, len(ordered))
	byProfile := make(map[artifact.Digest]Installation)
	nativeProviders := make([]Provider, 0, len(ordered))
	previousSlot := ""
	for _, draft := range ordered {
		if !installationSlotPattern.MatchString(draft.Slot) || draft.Slot <= previousSlot {
			return Installations{}, errors.New("invalid or duplicate AI installation slot")
		}
		previousSlot = draft.Slot
		profile, err := SealModelProfile(draft.Profile)
		if err != nil {
			return Installations{}, fmt.Errorf("seal AI profile for slot %q: %w", draft.Slot, err)
		}
		consent, err := WorkflowConsentDigest(draft.Slot, profile)
		if err != nil {
			return Installations{}, err
		}
		if draft.Consent != "" && draft.Consent != consent {
			return Installations{}, fmt.Errorf("AI installation slot %q has stale workflow consent", draft.Slot)
		}
		if err := ValidateEvaluation(profile, draft.Evaluation); err != nil {
			if errors.Is(err, ErrEvaluationNotApproved) {
				continue
			}
			return Installations{}, fmt.Errorf("validate AI evaluation for slot %q: %w", draft.Slot, err)
		}
		shared, exists := byProfile[profile.Digest()]
		if !exists {
			providerID, err := InstallationID("ai-provider", profile)
			if err != nil {
				return Installations{}, err
			}
			native, err := NewNativeProvider(profile, HTTPOptions{})
			if err != nil {
				return Installations{}, err
			}
			provider, err := NewResourceProvider(profile, native, credentials)
			if err != nil {
				return Installations{}, err
			}
			providerArtifact, err := ProviderArtifactDigest(profile)
			if err != nil {
				return Installations{}, err
			}
			shared = Installation{
				Profile: profile, ProviderID: providerID, ProviderArtifact: providerArtifact, Provider: provider,
			}
			byProfile[profile.Digest()] = shared
			nativeProviders = append(nativeProviders, native)
		}
		shared.Slot = draft.Slot
		shared.TargetID = TargetID(draft.Slot)
		shared.CredentialBindingID = CredentialBindingID(draft.Slot)
		shared.Consent = draft.Consent
		shared.Evaluation = draft.Evaluation
		entries = append(entries, shared)
	}
	return Installations{state: &installationState{entries: entries, native: nativeProviders}}, nil
}

func (i Installations) Valid() bool { return i.state != nil }

func (i Installations) Entries() []Installation {
	if !i.Valid() {
		return nil
	}
	return append([]Installation(nil), i.state.entries...)
}

func (i Installations) ForEvaluationArtifacts(artifacts []artifact.Digest) (Installations, error) {
	if !i.Valid() {
		return Installations{}, errors.New("AI installations are unavailable")
	}
	entries := make([]Installation, 0, len(i.state.entries))
	for _, installed := range i.state.entries {
		if err := ValidateEvaluationCandidate(installed.Profile, installed.Evaluation, artifacts); err != nil {
			if errors.Is(err, ErrEvaluationCandidateStale) {
				continue
			}
			return Installations{}, fmt.Errorf("validate AI evaluation for slot %q: %w", installed.Slot, err)
		}
		entries = append(entries, installed)
	}
	return Installations{state: &installationState{entries: entries, native: append([]Provider(nil), i.state.native...)}}, nil
}

func (i Installations) CloseIdleConnections() {
	if !i.Valid() {
		return
	}
	for _, provider := range i.state.native {
		if closer, ok := provider.(interface{ CloseIdleConnections() }); ok {
			closer.CloseIdleConnections()
		}
	}
}

func TargetID(slot string) string { return "ai-model/" + slot }

func CredentialBindingID(slot string) string { return "ai-credential/" + slot }

func ValidateInstallationSlot(slot string) error {
	if !installationSlotPattern.MatchString(slot) {
		return errors.New("invalid AI installation slot")
	}
	return nil
}

func WorkflowConsentDigest(slot string, profile ModelProfile) (artifact.Digest, error) {
	if !installationSlotPattern.MatchString(slot) || !profile.Valid() {
		return "", errors.New("AI workflow consent identity is invalid")
	}
	raw, err := artifact.Marshal(map[string]any{
		"slot": slot, "profileDigest": profile.Digest(), "providerAbi": ProviderABI,
		"operations": []string{OperationGenerate, OperationGenerateStructured, OperationAgentStart, OperationAgentContinue},
	})
	if err != nil {
		return "", err
	}
	return artifact.Sum(workflowConsentDomain, raw)
}
