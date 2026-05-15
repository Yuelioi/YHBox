package schedule

import "testing"

func TestScheduleService_CreateDefault(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	svc := NewService(s)
	sc, err := svc.Create("套餐 1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if sc.Trigger.Kind != "manual" {
		t.Errorf("default trigger should be manual, got %q", sc.Trigger.Kind)
	}
	if sc.OnError != "stop" {
		t.Errorf("default onError should be stop")
	}
	if !sc.Enabled {
		t.Error("default enabled true")
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
	sc.Targets = []TargetRef{{Kind: "container", ID: "c1"}}
	if err := svc.Save(sc); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if len(svc.List()) != 1 {
		t.Errorf("expected 1, got %d", len(svc.List()))
	}
}

func TestScheduleService_Update_PathTraversalProtected(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	svc := NewService(s)
	sc, _ := svc.Create("x")
	sc.Targets = []TargetRef{{Kind: "container", ID: "c1"}}
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
	sc.Targets = []TargetRef{{Kind: "container", ID: "c1"}}
	_ = svc.Save(sc)
	if err := svc.Delete(sc.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if len(svc.List()) != 0 {
		t.Error("not deleted")
	}
}
