package inputclip

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreSaveLoad(t *testing.T) {
	tmp := t.TempDir()
	s := NewStore(tmp)
	clip := &InputClip{
		ID:    "clip-test",
		Label: "Test",
		Meta:  ClipMeta{MouseMode: "relative", BaseResolution: [2]int{1920, 1080}},
		Events: []Event{
			{TUs: 0, Seq: 0, Type: EventTypeKeyDown, A: 0x57},
			{TUs: 50000, Seq: 1, Type: EventTypeKeyUp, A: 0x57},
		},
	}
	clip.UpdateDuration()

	if err := s.Save(clip); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "clip-test.iclp")); err != nil {
		t.Fatalf("文件未生成: %v", err)
	}

	got, err := s.Get("clip-test")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Label != "Test" || len(got.Events) != 2 {
		t.Errorf("round-trip 错: %+v", got)
	}

	list := s.List()
	if len(list) != 1 || list[0].ID != "clip-test" {
		t.Errorf("List 错: %+v", list)
	}

	if err := s.Delete("clip-test"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "clip-test.iclp")); !os.IsNotExist(err) {
		t.Error("文件未删")
	}
}

func TestStoreGetNotFound(t *testing.T) {
	s := NewStore(t.TempDir())
	_, err := s.Get("nonexistent")
	if err == nil {
		t.Error("应返 error")
	}
}
