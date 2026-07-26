package durablefs_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yottaapp/yotta/internal/durablefs"
)

func TestWriteReplaceAndRemove(t *testing.T) {
	path := filepath.Join(t.TempDir(), "record.json")
	if err := durablefs.WriteFile(path, []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := durablefs.WriteFile(path, []byte("two"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "two" {
		t.Fatalf("read = %q, %v", got, err)
	}
	if err := durablefs.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("removed file still exists: %v", err)
	}
}

func TestPublishNewNeverClobbersAnExistingFile(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "target.png")
	staged := filepath.Join(directory, "staged.png")
	if err := os.WriteFile(target, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(staged, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := durablefs.PublishNew(staged, target); err == nil {
		t.Fatal("PublishNew replaced an existing file")
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != "old" {
		t.Fatalf("existing target = %q, %v", got, err)
	}
	if got, err := os.ReadFile(staged); err != nil || string(got) != "new" {
		t.Fatalf("rejected staged file = %q, %v", got, err)
	}
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	if err := durablefs.PublishNew(staged, target); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != "new" {
		t.Fatalf("published target = %q, %v", got, err)
	}
}
