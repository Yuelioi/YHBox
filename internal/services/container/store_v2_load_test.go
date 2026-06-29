package container

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestStore_LoadGracefulVersionMismatch(t *testing.T) {
	root := t.TempDir()
	cid := "future-container"
	if err := os.MkdirAll(filepath.Join(root, cid), 0o755); err != nil {
		t.Fatal(err)
	}
	raw := map[string]any{
		"schemaVersion": 1,
		"id":            cid,
		"name":          "from-the-future",
		"graph": map[string]any{
			"id":      "g",
			"version": 999,
			"nodes":   []any{},
			"edges":   []any{},
		},
		"createdAt": "2026-05-16T00:00:00Z",
		"updatedAt": "2026-05-16T00:00:00Z",
	}
	b, _ := json.Marshal(raw)
	if err := os.WriteFile(filepath.Join(root, cid, "container.json"), b, 0o644); err != nil {
		t.Fatal(err)
	}

	st, err := NewStore(root)
	if err != nil {
		t.Fatalf("store should not error on version mismatch, got %v", err)
	}
	list := st.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 container loaded (incompatible), got %d", len(list))
	}
	if list[0].Status != StatusIncompatible {
		t.Errorf("expected Status=Incompatible, got %q", list[0].Status)
	}
	if list[0].IncompatibleReason == "" {
		t.Errorf("expected IncompatibleReason populated")
	}
}

func TestStore_LoadOKMatchingVersion(t *testing.T) {
	root := t.TempDir()
	cid := "normal"
	os.MkdirAll(filepath.Join(root, cid), 0o755)
	raw := map[string]any{
		"schemaVersion": 1,
		"id":            cid,
		"name":          "ok",
		"graph": map[string]any{
			"id":      "g",
			"version": GraphSchemaVersion,
			"nodes":   []any{},
			"edges":   []any{},
		},
		"createdAt": "2026-05-16T00:00:00Z",
		"updatedAt": "2026-05-16T00:00:00Z",
	}
	b, _ := json.Marshal(raw)
	os.WriteFile(filepath.Join(root, cid, "container.json"), b, 0o644)

	st, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	list := st.List()
	if len(list) != 1 || list[0].Status == StatusIncompatible {
		t.Errorf("expected ok container, got %+v", list)
	}
}
