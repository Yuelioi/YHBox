package schedule

import (
	"errors"
	"testing"

	"github.com/yottaapp/yotta/internal/apperr"
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
		t.Error("new schedules must remain disabled until the user enables them")
	}
	// Create 不持久化（spec），只返默认 Schedule
	if len(s.List()) != 0 {
		t.Error("Create should not persist; user calls Save")
	}
}

func TestScheduleServiceProjectsStableProblems(t *testing.T) {
	store, _ := NewStore(t.TempDir())
	service := NewService(store)

	_, err := service.Get("missing")
	assertScheduleProblem(t, err, "schedule.not_found", apperr.CategoryDomain, false)

	_, err = service.FireNow("missing")
	assertScheduleProblem(t, err, "schedule.runner_unavailable", apperr.CategoryInfrastructure, true)

	schedule, _ := service.Create("x")
	schedule.Targets = []TargetRef{{Kind: TargetWorkflow, ID: "workflow-1"}}
	if err := service.Save(schedule); err != nil {
		t.Fatal(err)
	}
	assertScheduleProblem(t, service.Update(schedule.ID, `{"unknown":true}`), "schedule.update.invalid_patch", apperr.CategoryValidation, false)
}

func assertScheduleProblem(t *testing.T, err error, id, category string, retryable bool) {
	t.Helper()
	got := apperr.From(err)
	if got.ID != id || got.Category != category || got.Retryable != retryable {
		t.Fatalf("problem = %#v, want id=%q category=%q retryable=%v", got, id, category, retryable)
	}
}

func TestSchedulePostCommitProblemProjection(t *testing.T) {
	err := &PostCommitError{Operation: "save", Err: errors.New("private reload failure")}
	problem := apperr.From(err)
	if problem.ID != "schedule.committed_reload_failed" || problem.Category != apperr.CategoryInfrastructure || problem.Retryable {
		t.Fatalf("problem = %#v", problem)
	}
	params, ok := problem.Params.(map[string]any)
	if !ok || params["operation"] != "save" || !err.Committed() || !errors.Is(err, err.Err) {
		t.Fatalf("post-commit semantics = %#v", problem)
	}
	if err.Error() == "" {
		t.Fatal("post-commit error has no diagnostic string")
	}
	wrapped := scheduleProblem("schedule.test", apperr.CategoryDomain, nil, false, errors.New("private"))
	if wrapped.Error() == "" {
		t.Fatal("schedule problem has no diagnostic string")
	}
}

func TestScheduleService_SaveAndList(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	svc := NewService(s)
	sc, _ := svc.Create("x")
	sc.Targets = []TargetRef{{Kind: TargetWorkflow, ID: "c1"}}
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
	schedule.Targets = []TargetRef{{Kind: TargetWorkflow, ID: "c1"}}
	if err := svc.Save(schedule); !errors.Is(err, want) {
		t.Fatalf("Save error = %v, want reload failure", err)
	} else {
		var committed *PostCommitError
		if !errors.As(err, &committed) || !committed.Committed() || committed.Operation != "save" {
			t.Fatalf("Save error does not express partial commit: %T %v", err, err)
		}
	}
}

func TestScheduleServiceFireNowUsesInjectedDaemon(t *testing.T) {
	store, _ := NewStore(t.TempDir())
	want := FireResult{
		Status: FireStatusFailed,
		Readiness: &RunReadiness{
			State: "credential-required", Slot: "account",
		},
	}
	service := NewService(store, WithManualFire(func(id string) (FireResult, error) {
		if id != "schedule-1" {
			t.Fatalf("FireNow id = %q", id)
		}
		return want, nil
	}))
	got, err := service.FireNow("schedule-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != want.Status || got.Readiness == nil || got.Readiness.Slot != "account" {
		t.Fatalf("FireNow = %#v", got)
	}
}

func TestScheduleServiceFireNowRequiresRunner(t *testing.T) {
	store, _ := NewStore(t.TempDir())
	if _, err := NewService(store).FireNow("schedule-1"); err == nil {
		t.Fatal("FireNow accepted a missing daemon")
	}
}

func TestScheduleService_Update_PathTraversalProtected(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	svc := NewService(s)
	sc, _ := svc.Create("x")
	sc.Targets = []TargetRef{{Kind: TargetWorkflow, ID: "c1"}}
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
	sc.Targets = []TargetRef{{Kind: TargetWorkflow, ID: "c1"}}
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
	schedule.Targets = []TargetRef{{Kind: TargetWorkflow, ID: "workflow-1"}}
	if err := svc.Save(schedule); err != nil {
		t.Fatal(err)
	}
	if err := svc.Update(schedule.ID, `{"unknown":true}`); err == nil {
		t.Fatal("Update accepted an unknown field")
	}
}
