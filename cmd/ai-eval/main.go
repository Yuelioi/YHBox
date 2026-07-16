package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/nodes31"
)

type runInput struct {
	Subject      artifact.Digest       `json:"subject"`
	Profile      *ai.ModelProfileDraft `json:"profile,omitempty"`
	Observations []ai.EvalObservation  `json:"observations"`
}

func main() {
	observationsPath := flag.String("observations", "testdata/ai-eval/mandatory-observations.json", "recorded offline observations")
	reportPath := flag.String("report", "testdata/ai-eval/mandatory-report.json", "canonical report artifact")
	write := flag.Bool("write", false, "update the canonical report artifact")
	flag.Parse()
	if err := run(*observationsPath, *reportPath, *write); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(observationsPath, reportPath string, write bool) error {
	raw, err := os.ReadFile(observationsPath)
	if err != nil {
		return err
	}
	var input runInput
	if err := decodeExact(raw, &input); err != nil {
		return fmt.Errorf("decode eval observations: %w", err)
	}
	suite, err := ai.BuiltinEvalSuite()
	if err != nil {
		return err
	}
	subject := input.Subject
	if input.Profile != nil {
		if subject != "" {
			return errors.New("eval observations must choose either subject or profile")
		}
		draft := *input.Profile
		draft.Evaluation = ai.EvaluationUnverified
		draft.EvaluationSuite = ""
		draft.EvaluationReport = ""
		profile, sealErr := ai.SealModelProfile(draft)
		if sealErr != nil {
			return sealErr
		}
		subject, err = ai.EvaluationSubjectDigest(profile)
		if err != nil {
			return err
		}
	}
	builtins, err := nodes31.Build()
	if err != nil {
		return err
	}
	candidate, err := ai.NewEvalCandidate(subject, builtins.AIEvaluationArtifacts())
	if err != nil {
		return err
	}
	evidence, err := ai.GradeEvalSuite(suite, candidate, input.Observations)
	if err != nil {
		return err
	}
	output, err := artifact.Marshal(evidence)
	if err != nil {
		return err
	}
	if write {
		if err := os.WriteFile(reportPath, append(output, '\n'), 0o644); err != nil {
			return err
		}
	} else {
		expected, err := os.ReadFile(reportPath)
		if err != nil {
			return err
		}
		if !bytes.Equal(bytes.TrimSpace(expected), output) {
			return errors.New("AI eval report drifted; run task ai:eval:update and review the report")
		}
	}
	report, err := evidence.Open()
	if err != nil {
		return err
	}
	document := report.Machine()
	fmt.Printf("AI eval %s: %s, pass=%d/%d, safety=%d, report=%s\n", suite.Digest(), document.Decision, document.Metrics.Passed, document.Metrics.Cases, document.Metrics.SafetyFailures, evidence.Digest)
	return nil
}

func decodeExact(raw []byte, target any) error {
	if len(raw) == 0 || len(raw) > ai.MaxEvalArtifactBytes {
		return errors.New("eval input exceeds byte budget")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("eval input contains trailing values")
	}
	return nil
}
