package ai

import (
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/securestore"
)

type installationCredentials struct{}

func (installationCredentials) Get(string) (string, error) { return "", securestore.ErrNotFound }

func TestInstallationsShareProviderByProfileAndBindExactSlots(t *testing.T) {
	profile, _ := evaluatedProfileForTest(t, EvaluationApproved)
	profile.Provider = ProviderOpenAIResponses
	profile.Model = "gpt-test"
	sealed, err := SealModelProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	subject, _ := EvaluationSubjectDigest(sealed)
	suite, _ := BuiltinEvalSuite()
	candidate, _ := NewEvalCandidate(subject, []artifact.Digest{suite.Machine().Baseline})
	evaluation, err := GradeEvalSuite(suite, candidate, passingEvalObservations(suite))
	if err != nil {
		t.Fatal(err)
	}
	profile.EvaluationReport = evaluation.Digest
	installations, err := Install([]InstallationDraft{
		{Slot: "secondary", Profile: profile, Evaluation: evaluation}, {Slot: "primary", Profile: profile, Evaluation: evaluation},
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
	filtered, err := installations.ForEvaluationArtifacts([]artifact.Digest{suite.Machine().Baseline})
	if err != nil || len(filtered.Entries()) != 2 {
		t.Fatalf("exact evaluation artifacts = %#v, %v", filtered.Entries(), err)
	}
	changed, _ := artifact.Sum("yotta/test/ai-eval/v1", []byte("changed"))
	filtered, err = installations.ForEvaluationArtifacts([]artifact.Digest{changed})
	if err != nil || len(filtered.Entries()) != 0 {
		t.Fatalf("stale evaluation artifacts remained installed = %#v, %v", filtered.Entries(), err)
	}
	installations.CloseIdleConnections()
}

func TestInstallationsRejectDuplicateSlotsAndRequireCredentials(t *testing.T) {
	profileDraft := ModelProfileDraft{
		Provider: ProviderAnthropicMessages, Model: "claude-test", MaxOutputTokens: 4096,
		Capabilities: ProfileCapabilities{StructuredOutput: true}, Evaluation: EvaluationUnverified,
	}
	if _, err := Install([]InstallationDraft{{Slot: "same", Profile: profileDraft}, {Slot: "same", Profile: profileDraft}}, installationCredentials{}); err == nil {
		t.Fatal("accepted duplicate AI installation slots")
	}
	if _, err := Install([]InstallationDraft{{Slot: "same", Profile: profileDraft}}, nil); err == nil {
		t.Fatal("accepted a model installation without credential storage")
	}
	installed, err := Install([]InstallationDraft{{Slot: "unverified", Profile: profileDraft}}, installationCredentials{})
	if err != nil || len(installed.Entries()) != 1 {
		t.Fatalf("unverified generation profile was not installed: %#v, %v", installed.Entries(), err)
	}
	filtered, err := installed.ForEvaluationArtifacts([]artifact.Digest{testArtifactDigest(t, "current")})
	if err != nil || len(filtered.Entries()) != 1 {
		t.Fatalf("unverified generation profile was removed by the authoring artifact filter: %#v, %v", filtered.Entries(), err)
	}
	empty, err := Install(nil, nil)
	if err != nil || !empty.Valid() || len(empty.Entries()) != 0 {
		t.Fatalf("explicit empty installation set = %#v, %v", empty, err)
	}
}

func TestInstallationsDoNotRequireOptionalCodexCLIAtStartup(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	profile := ModelProfileDraft{
		Provider: ProviderCodexSubscription, Endpoint: "codex://subscription", Model: "gpt-5.6-sol",
		MaxOutputTokens: 4096,
		Capabilities:    ProfileCapabilities{StructuredOutput: true, ToolCalling: true, ParallelTools: true},
		Evaluation:      EvaluationUnverified,
	}
	installed, err := Install([]InstallationDraft{{Slot: "codex", Profile: profile}}, installationCredentials{})
	if err != nil || len(installed.Entries()) != 1 {
		t.Fatalf("optional Codex profile blocked application startup: %#v, %v", installed.Entries(), err)
	}
}

func testArtifactDigest(t *testing.T, value string) artifact.Digest {
	t.Helper()
	digest, err := artifact.Sum("yotta/test/ai-installation/v1", []byte(value))
	if err != nil {
		t.Fatal(err)
	}
	return digest
}
