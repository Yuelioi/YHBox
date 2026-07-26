package schedule

import (
	"errors"
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
		Targets:       []TargetRef{{Kind: TargetWorkflowInstallation, ID: "installation-1"}},
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

func TestScheduleStoreDisarmsLegacyWorkflowTargetsForExplicitRepair(t *testing.T) {
	dir := t.TempDir()
	legacy := `{"schemaVersion":"1","id":"legacy","name":"legacy","enabled":true,"targets":[{"kind":"workflow","id":"workflow-1"}],"trigger":{"kind":"manual"},"timeoutMinutes":0,"onError":"stop","createdAt":"2026-07-26T00:00:00Z","updatedAt":"2026-07-26T00:00:00Z"}`
	if err := os.WriteFile(filepath.Join(dir, "legacy.json"), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	migrated, found := store.Get("legacy")
	if !found || migrated.Enabled || migrated.SchemaVersion != CurrentSchemaVersion ||
		len(migrated.Targets) != 1 || migrated.Targets[0].Kind != TargetWorkflowInstallation ||
		migrated.Targets[0].ID != "workflow-1" {
		t.Fatalf("migrated legacy schedule = %#v, found=%v", migrated, found)
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

func TestScheduleStorePauseInstallationFailsBeforeCommitWithoutPartialWrites(t *testing.T) {
	dir := t.TempDir()
	store, err := newStore(dir, storeFaults{
		beforePauseCommit: func() error { return errors.New("injected precommit failure") },
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"a", "b"} {
		schedule := validSchedule(id)
		schedule.Enabled = true
		if err := store.Save(schedule); err != nil {
			t.Fatal(err)
		}
	}
	paused, err := store.PauseInstallation("installation-1")
	if err == nil || len(paused) != 0 {
		t.Fatalf("PauseInstallation() = %#v, %v", paused, err)
	}
	for _, id := range []string{"a", "b"} {
		current, found := store.Get(id)
		if !found || !current.Enabled {
			t.Fatalf("schedule %q changed before commit: %#v", id, current)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, pauseJournalFilename)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("pause journal exists before commit: %v", err)
	}
}

func TestScheduleStorePauseInstallationRecoversCommittedPartialMaterialization(t *testing.T) {
	dir := t.TempDir()
	store, err := newStore(dir, storeFaults{
		afterPauseWrite: func(completed int) error {
			if completed == 1 {
				return errors.New("injected postcommit interruption")
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"a", "b"} {
		schedule := validSchedule(id)
		schedule.Enabled = true
		if err := store.Save(schedule); err != nil {
			t.Fatal(err)
		}
	}
	paused, err := store.PauseInstallation("installation-1")
	var committed interface{ Committed() bool }
	if len(paused) != 2 || !errors.As(err, &committed) || !committed.Committed() {
		t.Fatalf("PauseInstallation() = %#v, %v", paused, err)
	}
	for _, id := range []string{"a", "b"} {
		current, found := store.Get(id)
		if !found || current.Enabled {
			t.Fatalf("logical schedule %q was not paused: %#v", id, current)
		}
	}
	changed, _ := store.Get("b")
	changed.Name = "changed after committed pause"
	if err := store.Save(&changed); err != nil {
		t.Fatalf("Save after committed pause = %v", err)
	}
	reopened, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"a", "b"} {
		current, found := reopened.Get(id)
		if !found || current.Enabled {
			t.Fatalf("recovered schedule %q = %#v, found=%v", id, current, found)
		}
		if id == "b" && current.Name != "changed after committed pause" {
			t.Fatalf("recovery overwrote a later save: %#v", current)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, pauseJournalFilename)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("pause journal survived recovery: %v", err)
	}
}
