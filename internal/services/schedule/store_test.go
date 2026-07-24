package schedule

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validSchedule(id string) *Schedule {
	return &Schedule{
		SchemaVersion: CurrentSchemaVersion,
		ID:            id,
		Name:          "n",
		Targets:       []TargetRef{{Kind: TargetWorkflow, ID: "workflow-1"}},
		Trigger:       Trigger{Kind: TriggerManual},
		OnError:       OnErrorStop,
	}
}

func TestScheduleStore_SaveLoad(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	in := validSchedule("id-1")
	if err := s.Save(in); err != nil {
		t.Fatalf("Save: %v", err)
	}
	s2, _ := NewStore(dir)
	got, ok := s2.Get("id-1")
	if !ok || got.Name != "n" {
		t.Errorf("reload lost: %+v", got)
	}
}

func TestScheduleStore_Save_InvalidID(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	bad := []string{"", "../foo", "a/b", "a\\b", ".", "..", "foo:bar"}
	for _, id := range bad {
		sc := validSchedule(id)
		if err := s.Save(sc); err == nil {
			t.Errorf("id %q should be rejected", id)
		}
	}
}

func TestScheduleStore_Delete(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	_ = s.Save(validSchedule("id-1"))
	if err := s.Delete("id-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, ok := s.Get("id-1"); ok {
		t.Error("expected gone")
	}
}

func TestScheduleStoreRejectsLegacySchema(t *testing.T) {
	dir := t.TempDir()
	legacy := `{"schemaVersion":1,"id":"legacy","name":"legacy","targets":[{"kind":"container","id":"c"}],"trigger":{"kind":"manual"},"timeoutMinutes":0,"onError":"stop","createdAt":"0001-01-01T00:00:00Z","updatedAt":"0001-01-01T00:00:00Z"}`
	if err := os.WriteFile(filepath.Join(dir, "legacy.json"), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := NewStore(dir); err == nil || !strings.Contains(err.Error(), "schemaVersion") {
		t.Fatalf("NewStore error = %v, want strict legacy schema rejection", err)
	}
}

func TestScheduleStoreRejectsUnknownFields(t *testing.T) {
	dir := t.TempDir()
	doc := `{"schemaVersion":"1","id":"unknown","name":"unknown","targets":[{"kind":"workflow","id":"w"}],"trigger":{"kind":"manual"},"timeoutMinutes":0,"onError":"stop","surprise":true,"createdAt":"0001-01-01T00:00:00Z","updatedAt":"0001-01-01T00:00:00Z"}`
	if err := os.WriteFile(filepath.Join(dir, "unknown.json"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := NewStore(dir); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("NewStore error = %v, want unknown field rejection", err)
	}
}

func TestScheduleStoreRejectsFilenameIdentityMismatch(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(validSchedule("actual")); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(dir, "actual.json"), filepath.Join(dir, "alias.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := NewStore(dir); err == nil || !strings.Contains(err.Error(), "does not match filename") {
		t.Fatalf("NewStore error = %v, want filename identity rejection", err)
	}
}
