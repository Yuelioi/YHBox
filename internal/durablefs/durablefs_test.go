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
