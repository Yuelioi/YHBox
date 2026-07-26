package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestInspectIsReadOnlyRedactedAndReportsRecoveryState(t *testing.T) {
	root := filepath.Join(t.TempDir(), "profile")
	profile, err := Open(context.Background(), OpenOptions{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	defer profile.Close()
	recovery := filepath.Join(root, "config", "recovery")
	if err := os.MkdirAll(recovery, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(recovery, "settings-broken.json"), []byte("broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config", ".settings.json.staging-test"), []byte("stage"), 0o600); err != nil {
		t.Fatal(err)
	}

	report, err := Inspect(context.Background(), InspectOptions{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if report.Root != RedactedRoot || !report.Supported || report.LayoutVersion != LayoutVersion {
		t.Fatalf("unexpected report identity: %#v", report)
	}
	if report.RecoveryFiles != 1 || report.StagingFiles != 1 {
		t.Fatalf("unexpected recovery report: %#v", report)
	}
	report, err = Inspect(context.Background(), InspectOptions{Root: root, ShowPhysicalPath: true})
	if err != nil {
		t.Fatal(err)
	}
	if report.Root != root {
		t.Fatalf("physical root = %q, want %q", report.Root, root)
	}
}

func TestInspectDoesNotCreateAMissingRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "missing")
	if _, err := Inspect(context.Background(), InspectOptions{Root: root}); err == nil {
		t.Fatal("Inspect unexpectedly accepted a missing profile")
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("Inspect changed missing root: %v", err)
	}
}
