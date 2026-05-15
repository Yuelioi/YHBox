package schedule

import (
	"testing"
)

func TestScheduleStore_SaveLoad(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	in := &Schedule{
		SchemaVersion: 1, ID: "id-1", Name: "n",
		Targets: []TargetRef{{Kind: "container", ID: "c"}},
		Trigger: Trigger{Kind: "manual"},
		OnError: "stop",
	}
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
		sc := &Schedule{
			SchemaVersion: 1, ID: id, Name: "n",
			Targets: []TargetRef{{Kind: "container", ID: "c"}},
			Trigger: Trigger{Kind: "manual"},
			OnError: "stop",
		}
		if err := s.Save(sc); err == nil {
			t.Errorf("id %q should be rejected", id)
		}
	}
}

func TestScheduleStore_Delete(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	_ = s.Save(&Schedule{
		SchemaVersion: 1, ID: "id-1", Name: "n",
		Targets: []TargetRef{{Kind: "container", ID: "c"}},
		Trigger: Trigger{Kind: "manual"}, OnError: "stop",
	})
	if err := s.Delete("id-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, ok := s.Get("id-1"); ok {
		t.Error("expected gone")
	}
}
