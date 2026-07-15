package services

import (
	"testing"

	"github.com/yottaapp/yotta/internal/ai"
)

func modelSettingsForTest(slot, label string) AIModelSettings {
	return AIModelSettings{
		Slot: slot, Label: label, Provider: ai.ProviderOpenAIResponses, Model: "gpt-test", MaxOutputTokens: 4096,
		Capabilities: ai.ProfileCapabilities{StructuredOutput: true}, Evaluation: ai.EvaluationUnverified,
	}
}

func TestAISettingsRequireUniqueSlotsAndLabels(t *testing.T) {
	settings := defaultSettings()
	settings.AI.Profiles = []AIModelSettings{modelSettingsForTest("primary", "Model"), modelSettingsForTest("primary", "Other")}
	if err := settings.Validate(); err == nil {
		t.Fatal("accepted duplicate AI profile slots")
	}
	settings.AI.Profiles = []AIModelSettings{modelSettingsForTest("primary", "Model"), modelSettingsForTest("secondary", "Model")}
	if err := settings.Validate(); err == nil {
		t.Fatal("accepted duplicate AI profile labels")
	}
}

func TestAISettingsPinProviderNativeProfileAndConsent(t *testing.T) {
	settings := defaultSettings()
	configured := modelSettingsForTest("primary", "Model")
	settings.AI.Profiles = []AIModelSettings{configured}
	if err := settings.Validate(); err != nil {
		t.Fatal(err)
	}
	profile, err := ai.SealModelProfile(configured.profileDraft())
	if err != nil {
		t.Fatal(err)
	}
	configured.WorkflowConsent, err = ai.WorkflowConsentDigest(configured.Slot, profile)
	if err != nil {
		t.Fatal(err)
	}
	settings.AI.Profiles[0] = configured
	if err := settings.Validate(); err != nil {
		t.Fatal(err)
	}
	settings.AI.Profiles[0].Model = "changed"
	if err := settings.Validate(); err == nil {
		t.Fatal("accepted workflow consent for a different model profile")
	}
	settings.AI.Profiles[0] = modelSettingsForTest("primary", "Model")
	settings.AI.Profiles[0].Provider = "openai-compatible"
	if err := settings.Validate(); err == nil {
		t.Fatal("accepted an emulated provider protocol")
	}
}
