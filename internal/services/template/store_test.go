package template

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStore_SaveGet(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a} // PNG magic
	meta, err := s.Save("fishing.hook_icon", png, TemplateMeta{
		Name:               "Hook Icon",
		RecordedResolution: [2]int{1920, 1080},
		Width:              45, Height: 54,
	})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if meta.SHA256 == "" {
		t.Fatal("SHA256 not filled")
	}
	if _, err := os.Stat(filepath.Join(dir, "fishing.hook_icon.png")); err != nil {
		t.Fatalf("png file missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "fishing.hook_icon.json")); err != nil {
		t.Fatalf("json file missing: %v", err)
	}

	got, ok := s.Get("fishing.hook_icon")
	if !ok {
		t.Fatal("Get not found")
	}
	if got.Name != "Hook Icon" {
		t.Errorf("Name = %q, want Hook Icon", got.Name)
	}
}

func TestStore_RejectsInvalidKey(t *testing.T) {
	s, _ := NewStore(t.TempDir())
	_, err := s.Save("no_dot_key", []byte{1}, TemplateMeta{})
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
}

func TestStore_List(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	s.Save("a.x", []byte{1}, TemplateMeta{Name: "X"})
	s.Save("a.y", []byte{2}, TemplateMeta{Name: "Y"})
	got := s.List()
	if len(got) != 2 {
		t.Errorf("List() = %d, want 2", len(got))
	}
}

func TestStore_Delete(t *testing.T) {
	s, _ := NewStore(t.TempDir())
	s.Save("a.x", []byte{1}, TemplateMeta{Name: "X"})
	if err := s.Delete("a.x"); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Get("a.x"); ok {
		t.Fatal("still exists after Delete")
	}
}
