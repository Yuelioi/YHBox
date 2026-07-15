package ai

import (
	"testing"

	"github.com/yottaapp/yotta/internal/securestore"
)

type installationCredentials struct{}

func (installationCredentials) Get(string) (string, error) { return "", securestore.ErrNotFound }

func TestInstallationsShareProviderByProfileAndBindExactSlots(t *testing.T) {
	profile := ModelProfileDraft{
		Provider: ProviderOpenAIResponses, Model: "gpt-test", MaxOutputTokens: 4096,
		Capabilities: ProfileCapabilities{StructuredOutput: true}, Evaluation: EvaluationUnverified,
	}
	installations, err := Install([]InstallationDraft{
		{Slot: "secondary", Profile: profile}, {Slot: "primary", Profile: profile},
	}, installationCredentials{})
	if err != nil {
		t.Fatal(err)
	}
	entries := installations.Entries()
	if len(entries) != 2 || entries[0].Slot != "primary" || entries[1].Slot != "secondary" {
		t.Fatalf("installation order = %#v", entries)
	}
	if entries[0].ProviderID != entries[1].ProviderID || entries[0].Provider != entries[1].Provider {
		t.Fatal("identical model profiles did not share one provider implementation")
	}
	if entries[0].TargetID != "ai-model/primary" || entries[0].CredentialBindingID != "ai-credential/primary" {
		t.Fatalf("slot bindings = %#v", entries[0])
	}
	installations.CloseIdleConnections()
}

func TestInstallationsRejectDuplicateSlotsAndStaleConsent(t *testing.T) {
	profileDraft := ModelProfileDraft{
		Provider: ProviderAnthropicMessages, Model: "claude-test", MaxOutputTokens: 4096,
		Capabilities: ProfileCapabilities{StructuredOutput: true}, Evaluation: EvaluationUnverified,
	}
	if _, err := Install([]InstallationDraft{{Slot: "same", Profile: profileDraft}, {Slot: "same", Profile: profileDraft}}, installationCredentials{}); err == nil {
		t.Fatal("accepted duplicate AI installation slots")
	}
	profile, err := SealModelProfile(profileDraft)
	if err != nil {
		t.Fatal(err)
	}
	consent, err := WorkflowConsentDigest("other", profile)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Install([]InstallationDraft{{Slot: "same", Profile: profileDraft, Consent: consent}}, installationCredentials{}); err == nil {
		t.Fatal("accepted stale workflow consent")
	}
	if _, err := Install([]InstallationDraft{{Slot: "same", Profile: profileDraft}}, nil); err == nil {
		t.Fatal("accepted a model installation without credential storage")
	}
	empty, err := Install(nil, nil)
	if err != nil || !empty.Valid() || len(empty.Entries()) != 0 {
		t.Fatalf("explicit empty installation set = %#v, %v", empty, err)
	}
}
