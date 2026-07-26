package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunWritesAndChecksCanonicalReport(t *testing.T) {
	observations := filepath.Join("..", "..", "testdata", "ai-eval", "mandatory-observations.json")
	report := filepath.Join(t.TempDir(), "report.json")
	if err := run(observations, report, true); err != nil {
		t.Fatal(err)
	}
	if err := run(observations, report, false); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(report, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run(observations, report, false); err == nil {
		t.Fatal("accepted a drifted AI eval report")
	}
}
