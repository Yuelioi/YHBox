package template

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStore_SaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

	meta, err := s.Save("battle/skill1-cd", pngData, TemplateMeta{
		Name:               "技能1 冷却完成",
		Description:        "右下技能1冰红光",
		RecordedResolution: [2]int{1920, 1080},
		Width:              64, Height: 32,
		Region: [4]float32{0.85, 0.92, 0.05, 0.04},
	})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if meta.SHA256 == "" {
		t.Error("expected SHA256 to be populated")
	}
	if meta.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}

	if _, err := os.Stat(filepath.Join(dir, "battle", "skill1-cd.png")); err != nil {
		t.Errorf("png not on disk: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "_index.json")); err != nil {
		t.Errorf("_index.json not on disk: %v", err)
	}

	s2, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore reload: %v", err)
	}
	got, ok := s2.Get("battle/skill1-cd")
	if !ok {
		t.Fatal("Get returned !ok after reload")
	}
	if got.Name != "技能1 冷却完成" {
		t.Errorf("Name lost: got %q", got.Name)
	}
}

func TestStore_Delete(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	_, _ = s.Save("foo/bar", []byte{0x89, 0x50}, TemplateMeta{Name: "bar", Width: 1, Height: 1})

	if err := s.Delete("foo/bar"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, ok := s.Get("foo/bar"); ok {
		t.Error("expected !ok after delete")
	}
	if _, err := os.Stat(filepath.Join(dir, "foo", "bar.png")); !os.IsNotExist(err) {
		t.Error("png should be gone")
	}
}

func TestStore_List(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	_, _ = s.Save("a/x", []byte{1}, TemplateMeta{Name: "x", Width: 1, Height: 1})
	_, _ = s.Save("b/y", []byte{2}, TemplateMeta{Name: "y", Width: 1, Height: 1})

	got := s.List()
	if len(got) != 2 {
		t.Errorf("expected 2 entries, got %d", len(got))
	}
}

func TestStore_KeyValidation(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	cases := []struct {
		key   string
		valid bool
	}{
		{"foo/bar", true},
		{"foo", true},
		{"foo/bar/baz", true},
		{"/foo", false},
		{"foo/", false},
		{"foo//bar", false},
		{"../foo", false},
		{"foo/../bar", false},
		{"foo.png", false},
		{"", false},
	}
	for _, tc := range cases {
		_, err := s.Save(tc.key, []byte{1}, TemplateMeta{Name: "x", Width: 1, Height: 1})
		got := err == nil
		if got != tc.valid {
			t.Errorf("key %q: want valid=%v, got valid=%v (err=%v)", tc.key, tc.valid, got, err)
		}
	}
}
