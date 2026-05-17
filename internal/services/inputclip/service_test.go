package inputclip

import "testing"

func TestServiceListEmpty(t *testing.T) {
	svc := NewService(NewStore(t.TempDir()))
	if got := svc.List(); len(got) != 0 {
		t.Errorf("empty store List() = %d, want 0", len(got))
	}
}

func TestServiceCreateGetDelete(t *testing.T) {
	svc := NewService(NewStore(t.TempDir()))
	clip := &InputClip{
		ID: "svc-test", Label: "svc",
		Meta: ClipMeta{MouseMode: "relative", BaseResolution: [2]int{1920, 1080}},
	}
	clip.UpdateDuration()
	if err := svc.Save(clip); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := svc.Get("svc-test")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Label != "svc" {
		t.Errorf("Get 错: %+v", got)
	}
	if err := svc.Delete("svc-test"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}
