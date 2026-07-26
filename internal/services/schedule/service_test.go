package schedule

import (
	"errors"
	"testing"
)

func TestScheduleService_CreateDefault(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	svc := NewService(s)
	sc, err := svc.Create("套餐 1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if sc.Trigger.Kind != TriggerManual {
		t.Errorf("default trigger should be manual, got %q", sc.Trigger.Kind)
	}
	if sc.OnError != OnErrorStop {
		t.Errorf("default onError should be stop")
	}
	if sc.Enabled {
		t.Error("new schedules must remain disabled until explicit schedule consent")
	}
	// Create 不持久化（spec），只返默认 Schedule
	if len(s.List()) != 0 {
		t.Error("Create should not persist; user calls Save")
	}
}

func TestScheduleService_SaveAndList(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	svc := NewService(s)
	sc, _ := svc.Create("x")
	sc.Targets = []TargetRef{{Kind: TargetWorkflowInstallation, ID: "c1"}}
	if err := svc.Save(sc); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if len(svc.List()) != 1 {
		t.Errorf("expected 1, got %d", len(svc.List()))
	}
}

func TestScheduleServicePropagatesReloadFailure(t *testing.T) {
	store, _ := NewStore(t.TempDir())
	want := errors.New("reload failed")
	svc := NewService(store, WithChangeListener(func() error { return want }))
	schedule, _ := svc.Create("x")
	schedule.Targets = []TargetRef{{Kind: TargetWorkflowInstallation, ID: "c1"}}
	if err := svc.Save(schedule); !errors.Is(err, want) {
		t.Fatalf("Save error = %v, want reload failure", err)
	} else {
		var committed *PostCommitError
		if !errors.As(err, &committed) || !committed.Committed() || committed.Operation != "save" {
			t.Fatalf("Save error does not express partial commit: %T %v", err, err)
		}
	}
}

func TestScheduleServiceRejectsArmingUnreadyInstallationBeforeCommit(t *testing.T) {
	store, _ := NewStore(t.TempDir())
	want := errors.New("schedule consent required")
	svc := NewService(store, WithTargetReadiness(func(installationID string) error {
		if installationID != "installation-1" {
			t.Fatalf("readiness installation = %q", installationID)
		}
		return want
	}))
	schedule, _ := svc.Create("blocked")
	schedule.Enabled = true
	schedule.Targets = []TargetRef{{Kind: TargetWorkflowInstallation, ID: "installation-1"}}
	if err := svc.Save(schedule); !errors.Is(err, want) {
		t.Fatalf("Save error = %v", err)
	}
	if len(store.List()) != 0 {
		t.Fatal("unready enabled schedule was persisted")
	}
	schedule.Enabled = false
	if err := svc.Save(schedule); err != nil {
		t.Fatalf("Save disabled schedule = %v", err)
	}
}

func TestScheduleServiceRejectsArmingWithoutReadinessAuthority(t *testing.T) {
	store, _ := NewStore(t.TempDir())
	svc := NewService(store)
	schedule, _ := svc.Create("blocked")
	schedule.Enabled = true
	schedule.Targets = []TargetRef{{Kind: TargetWorkflowInstallation, ID: "installation-1"}}
	if err := svc.Save(schedule); err == nil {
		t.Fatal("Save armed a schedule without a Workflow Installation readiness authority")
	}
	if len(store.List()) != 0 {
		t.Fatal("schedule without readiness authority was persisted")
	}
}

func TestScheduleServicePausesOnlyEnabledSchedulesForInstallation(t *testing.T) {
	store, _ := NewStore(t.TempDir())
	reloads := 0
	svc := NewService(
		store,
		WithTargetReadiness(func(string) error { return nil }),
		WithChangeListener(func() error { reloads++; return nil }),
	)
	save := func(id, installationID string, enabled bool) {
		t.Helper()
		schedule := Schedule{
			SchemaVersion: CurrentSchemaVersion, ID: id, Name: id, Enabled: enabled,
			Targets: []TargetRef{{Kind: TargetWorkflowInstallation, ID: installationID}},
			Trigger: Trigger{Kind: TriggerManual}, OnError: OnErrorStop,
		}
		if err := svc.Save(schedule); err != nil {
			t.Fatal(err)
		}
	}
	save("affected-b", "installation-a", true)
	save("unrelated", "installation-b", true)
	save("already-paused", "installation-a", false)
	reloads = 0

	paused, err := NewInstallationPauser(svc).PauseInstallation("installation-a")
	if err != nil || len(paused) != 1 || paused[0] != "affected-b" || reloads != 1 {
		t.Fatalf("PauseInstallation() = %#v, reloads=%d err=%v", paused, reloads, err)
	}
	affected, _ := store.Get("affected-b")
	unrelated, _ := store.Get("unrelated")
	if affected.Enabled || !unrelated.Enabled {
		t.Fatalf("schedule states = affected:%v unrelated:%v", affected.Enabled, unrelated.Enabled)
	}
}

func TestScheduleService_Update_PathTraversalProtected(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	svc := NewService(s)
	sc, _ := svc.Create("x")
	sc.Targets = []TargetRef{{Kind: TargetWorkflowInstallation, ID: "c1"}}
	_ = svc.Save(sc)
	originalID := sc.ID

	patch := `{"id":"../../etc/evil","name":"hacked"}`
	if err := svc.Update(originalID, patch); err != nil {
		t.Fatalf("Update should succeed: %v", err)
	}
	got, ok := s.Get(originalID)
	if !ok {
		t.Fatal("original missing")
	}
	if got.Name != "hacked" {
		t.Errorf("name patch should apply")
	}
	if _, ok := s.Get("../../etc/evil"); ok {
		t.Error("malicious ID should not exist")
	}
}

func TestScheduleService_Delete(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	svc := NewService(s)
	sc, _ := svc.Create("x")
	sc.Targets = []TargetRef{{Kind: TargetWorkflowInstallation, ID: "c1"}}
	_ = svc.Save(sc)
	if err := svc.Delete(sc.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if len(svc.List()) != 0 {
		t.Error("not deleted")
	}
}

func TestScheduleServiceUpdateRejectsUnknownFields(t *testing.T) {
	store, _ := NewStore(t.TempDir())
	svc := NewService(store)
	schedule, _ := svc.Create("x")
	schedule.Targets = []TargetRef{{Kind: TargetWorkflowInstallation, ID: "installation-1"}}
	if err := svc.Save(schedule); err != nil {
		t.Fatal(err)
	}
	if err := svc.Update(schedule.ID, `{"unknown":true}`); err == nil {
		t.Fatal("Update accepted an unknown field")
	}
}
