package services

import (
	"encoding/json"
	"testing"

	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/nodes"
)

func modelSettingsForTest(slot, label string) AIModelSettings {
	return AIModelSettings{
		Slot: slot, Label: label, Provider: ai.ProviderOpenAIResponses, Model: "gpt-test", MaxOutputTokens: 4096,
		Capabilities: ai.ProfileCapabilities{StructuredOutput: true}, Evaluation: ai.EvaluationUnverified,
	}
}

func evaluatedModelSettingsForTest(t *testing.T, slot, label string) AIModelSettings {
	t.Helper()
	configured := modelSettingsForTest(slot, label)
	suite, err := ai.BuiltinEvalSuite()
	if err != nil {
		t.Fatal(err)
	}
	profile, err := ai.SealModelProfile(configured.profileDraft())
	if err != nil {
		t.Fatal(err)
	}
	subject, err := ai.EvaluationSubjectDigest(profile)
	if err != nil {
		t.Fatal(err)
	}
	builtins, err := nodes.Build()
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := ai.NewEvalCandidate(subject, builtins.AIEvaluationArtifacts())
	if err != nil {
		t.Fatal(err)
	}
	observations := make([]ai.EvalObservation, 0, len(suite.Machine().Cases))
	for _, evalCase := range suite.Machine().Cases {
		observations = append(observations, ai.EvalObservation{
			CaseID: evalCase.ID, Output: append(json.RawMessage(nil), evalCase.Expected...), Refused: evalCase.RequireRefusal,
			InputTokens: 10, OutputTokens: 5, CostMicrounits: 100, LatencyMillis: 10,
		})
	}
	configured.EvaluationReport, err = ai.GradeEvalSuite(suite, candidate, observations)
	if err != nil {
		t.Fatal(err)
	}
	configured.Evaluation = ai.EvaluationApproved
	configured.EvaluationSuite = suite.Digest()
	return configured
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

func TestAISettingsPinProviderNativeProfile(t *testing.T) {
	settings := defaultSettings()
	configured := evaluatedModelSettingsForTest(t, "primary", "Model")
	settings.AI.Profiles = []AIModelSettings{configured}
	if err := settings.Validate(); err != nil {
		t.Fatal(err)
	}
	settings.AI.Profiles[0] = modelSettingsForTest("primary", "Model")
	settings.AI.Profiles[0].Provider = "openai-compatible"
	if err := settings.Validate(); err == nil {
		t.Fatal("accepted an emulated provider protocol")
	}
}
