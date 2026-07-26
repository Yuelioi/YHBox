package ai

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
)

func TestEvalSuiteGradesDeterministicallyAndStrictlyReopens(t *testing.T) {
	suite, err := BuiltinEvalSuite()
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenEvalSuite(suite.Bytes(), suite.Digest())
	if err != nil || reopened.Digest() != suite.Digest() {
		t.Fatalf("reopen eval suite = %q, %v", reopened.Digest(), err)
	}
	profileDraft, evidence := evaluatedProfileForTest(t, EvaluationApproved)
	profile, err := SealModelProfile(profileDraft)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateEvaluation(profile, evidence); err != nil {
		t.Fatalf("approved evaluation = %v", err)
	}
	artifacts := []artifact.Digest{suite.Machine().Baseline}
	if err := ValidateEvaluationCandidate(profile, evidence, artifacts); err != nil {
		t.Fatalf("exact eval candidate = %v", err)
	}
	other, err := artifact.Sum("yotta/test/ai-eval/v1", []byte("changed"))
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateEvaluationCandidate(profile, evidence, []artifact.Digest{other}); !errors.Is(err, ErrEvaluationCandidateStale) {
		t.Fatalf("stale eval candidate = %v", err)
	}
	changedEvidence := profileDraft
	changedEvidence.EvaluationReport = other
	changedProfile, err := SealModelProfile(changedEvidence)
	if err != nil {
		t.Fatal(err)
	}
	if changedProfile.Digest() == profile.Digest() || ValidateEvaluation(changedProfile, evidence) == nil {
		t.Fatal("evaluation report replacement retained profile identity or approval")
	}
	report, err := evidence.Open()
	if err != nil || report.Machine().Decision != EvaluationApproved || report.Machine().Metrics.PassRateBasisPoints != 10_000 {
		t.Fatalf("approved report = %#v, %v", report.Machine(), err)
	}
	tampered := append([]byte(nil), suite.Bytes()...)
	tampered[len(tampered)-1] = ' '
	if _, err := OpenEvalSuite(tampered, suite.Digest()); err == nil {
		t.Fatal("accepted non-canonical eval suite")
	}
}

func TestEvalSuiteRejectsSafetyRegressionAndMismatchedEvidence(t *testing.T) {
	profileDraft, evidence := evaluatedProfileForTest(t, EvaluationRejected)
	profile, err := SealModelProfile(profileDraft)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateEvaluation(profile, evidence); !errors.Is(err, ErrEvaluationNotApproved) {
		t.Fatalf("rejected evaluation = %v", err)
	}
	report, err := evidence.Open()
	if err != nil || report.Machine().Metrics.SafetyFailures != 1 || report.Machine().Decision != EvaluationRejected {
		t.Fatalf("rejected report = %#v, %v", report.Machine(), err)
	}
	other := profileDraft
	other.Model = "other-model"
	otherProfile, err := SealModelProfile(other)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateEvaluation(otherProfile, evidence); err == nil || errors.Is(err, ErrEvaluationNotApproved) {
		t.Fatalf("mismatched evaluation evidence = %v", err)
	}
	if _, err := GradeEvalSuite(EvalSuite{}, EvalCandidate{}, nil); err == nil {
		t.Fatal("accepted invalid eval grading identity")
	}
}

func evaluatedProfileForTest(t *testing.T, decision EvaluationStatus) (ModelProfileDraft, EvalReportArtifact) {
	t.Helper()
	suite, err := BuiltinEvalSuite()
	if err != nil {
		t.Fatal(err)
	}
	draft := ModelProfileDraft{
		Provider: ProviderOpenAIResponses, Model: "evaluated-model", MaxOutputTokens: 4096,
		Capabilities: ProfileCapabilities{StructuredOutput: true}, Evaluation: EvaluationUnverified,
		ProviderMetadata: json.RawMessage(`{}`),
	}
	profile, err := SealModelProfile(draft)
	if err != nil {
		t.Fatal(err)
	}
	subject, err := EvaluationSubjectDigest(profile)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := NewEvalCandidate(subject, []artifact.Digest{suite.Machine().Baseline})
	if err != nil {
		t.Fatal(err)
	}
	observations := passingEvalObservations(suite)
	if decision == EvaluationRejected {
		for index := range observations {
			if observations[index].CaseID == "prompt-injection" {
				observations[index].Refused = false
				observations[index].Output = json.RawMessage(`"leaked"`)
			}
		}
	}
	evidence, err := GradeEvalSuite(suite, candidate, observations)
	if err != nil {
		t.Fatal(err)
	}
	draft.Evaluation = decision
	draft.EvaluationSuite = suite.Digest()
	draft.EvaluationReport = evidence.Digest
	return draft, evidence
}

func passingEvalObservations(suite EvalSuite) []EvalObservation {
	document := suite.Machine()
	result := make([]EvalObservation, 0, len(document.Cases))
	for _, evalCase := range document.Cases {
		result = append(result, EvalObservation{
			CaseID: evalCase.ID, Output: append(json.RawMessage(nil), evalCase.Expected...), Refused: evalCase.RequireRefusal,
			InputTokens: 10, OutputTokens: 5, CostMicrounits: 100, LatencyMillis: 10,
		})
	}
	return result
}
