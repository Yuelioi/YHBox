package template

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStore_SaveMeta_GetMeta(t *testing.T) {
	s, _ := NewStore(t.TempDir())
	want := KeyMeta{
		Name:        "Hook Icon",
		Description: "钓鱼点鱼钩",
		Tags:        []string{"fishing"},
		Origin:      TemplateOrigin{Kind: "user"},
	}
	if err := s.SaveMeta("fishing.hook_icon", want); err != nil {
		t.Fatalf("SaveMeta: %v", err)
	}
	got, ok := s.GetMeta("fishing.hook_icon")
	if !ok {
		t.Fatal("GetMeta not found")
	}
	if got.Name != want.Name || got.Description != want.Description {
		t.Errorf("got %+v, want %+v", got, want)
	}
	if _, err := os.Stat(filepath.Join(s.root, "fishing.hook_icon", "_meta.json")); err != nil {
		t.Fatalf("_meta.json missing: %v", err)
	}
}

func TestStore_SaveMeta_RejectsInvalidKey(t *testing.T) {
	s, _ := NewStore(t.TempDir())
	if err := s.SaveMeta("no_dot_key", KeyMeta{Name: "X"}); err == nil {
		t.Fatal("expected error for invalid key")
	}
}

func TestStore_SaveVariant_GetVariant(t *testing.T) {
	s, _ := NewStore(t.TempDir())
	s.SaveMeta("fishing.hook_icon", KeyMeta{Name: "Hook"})

	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	v, err := s.SaveVariant("fishing.hook_icon", png, VariantMeta{
		Resolution: [2]int{1920, 1080},
		BBox:       [4]int{1752, 944, 1797, 998},
	})
	if err != nil {
		t.Fatalf("SaveVariant: %v", err)
	}
	if v.SHA256 == "" {
		t.Fatal("SHA256 not filled")
	}
	if v.Width != 45 || v.Height != 54 {
		t.Errorf("Width=%d Height=%d, want 45/54", v.Width, v.Height)
	}
	got, ok := s.GetVariant("fishing.hook_icon", [2]int{1920, 1080})
	if !ok {
		t.Fatal("GetVariant not found")
	}
	if got.SHA256 != v.SHA256 {
		t.Errorf("sha mismatch")
	}
}

func TestStore_PickBest_ExactMatch(t *testing.T) {
	s, _ := NewStore(t.TempDir())
	s.SaveMeta("fishing.hook_icon", KeyMeta{Name: "Hook"})
	png := []byte{0x89, 0x50}
	s.SaveVariant("fishing.hook_icon", png, VariantMeta{Resolution: [2]int{1920, 1080}})
	s.SaveVariant("fishing.hook_icon", png, VariantMeta{Resolution: [2]int{1280, 720}})

	v, ok := s.PickBest("fishing.hook_icon", 1280, 720)
	if !ok || v.Resolution != [2]int{1280, 720} {
		t.Errorf("PickBest(1280,720) = %v, want 1280x720", v.Resolution)
	}
	v, ok = s.PickBest("fishing.hook_icon", 1920, 1080)
	if !ok || v.Resolution != [2]int{1920, 1080} {
		t.Errorf("PickBest(1920,1080) = %v, want 1920x1080", v.Resolution)
	}
	_, ok = s.PickBest("fishing.hook_icon", 1366, 768)
	if ok {
		t.Error("PickBest(1366,768) should miss (no fallback)")
	}
}

func TestStore_ListVariants_AreaDescSort(t *testing.T) {
	s, _ := NewStore(t.TempDir())
	s.SaveMeta("a.x", KeyMeta{Name: "X"})
	png := []byte{0x89, 0x50}
	s.SaveVariant("a.x", png, VariantMeta{Resolution: [2]int{1280, 720}})
	s.SaveVariant("a.x", png, VariantMeta{Resolution: [2]int{1920, 1080}})
	s.SaveVariant("a.x", png, VariantMeta{Resolution: [2]int{1366, 768}})

	vs := s.ListVariants("a.x")
	if len(vs) != 3 {
		t.Fatalf("len = %d, want 3", len(vs))
	}
	want := [][2]int{{1920, 1080}, {1366, 768}, {1280, 720}}
	for i, w := range want {
		if vs[i].Resolution != w {
			t.Errorf("[%d] %v, want %v", i, vs[i].Resolution, w)
		}
	}
}

func TestStore_DeleteVariant(t *testing.T) {
	s, _ := NewStore(t.TempDir())
	s.SaveMeta("a.x", KeyMeta{Name: "X"})
	png := []byte{0x89, 0x50}
	s.SaveVariant("a.x", png, VariantMeta{Resolution: [2]int{1920, 1080}})
	s.SaveVariant("a.x", png, VariantMeta{Resolution: [2]int{1280, 720}})

	if err := s.DeleteVariant("a.x", [2]int{1280, 720}); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.GetVariant("a.x", [2]int{1280, 720}); ok {
		t.Fatal("variant still exists after DeleteVariant")
	}
	if _, ok := s.GetVariant("a.x", [2]int{1920, 1080}); !ok {
		t.Fatal("1080p variant was wrongly deleted")
	}
	if _, ok := s.GetMeta("a.x"); !ok {
		t.Fatal("_meta should still exist after partial delete")
	}
}

func TestStore_Delete_FullKey(t *testing.T) {
	s, _ := NewStore(t.TempDir())
	s.SaveMeta("a.x", KeyMeta{Name: "X"})
	s.SaveVariant("a.x", []byte{0x89, 0x50}, VariantMeta{Resolution: [2]int{1920, 1080}})

	if err := s.Delete("a.x"); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.GetMeta("a.x"); ok {
		t.Fatal("_meta still exists after Delete")
	}
	if vs := s.ListVariants("a.x"); len(vs) != 0 {
		t.Fatal("variants still exist after Delete")
	}
	if _, err := os.Stat(filepath.Join(s.root, "a.x")); !os.IsNotExist(err) {
		t.Fatal("key dir should be removed")
	}
}

func TestStore_List_Snapshot(t *testing.T) {
	s, _ := NewStore(t.TempDir())
	s.SaveMeta("a.x", KeyMeta{Name: "X"})
	s.SaveMeta("a.y", KeyMeta{Name: "Y"})
	got := s.List()
	if len(got) != 2 {
		t.Errorf("List len = %d, want 2", len(got))
	}
}

func TestStore_Preload_FromDisk(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	s.SaveMeta("a.x", KeyMeta{Name: "X"})
	s.SaveVariant("a.x", []byte{0x89, 0x50}, VariantMeta{Resolution: [2]int{1920, 1080}})

	s2, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if km, ok := s2.GetMeta("a.x"); !ok || km.Name != "X" {
		t.Errorf("preload meta failed: %v", km)
	}
	if _, ok := s2.GetVariant("a.x", [2]int{1920, 1080}); !ok {
		t.Error("preload variant failed")
	}
}

func TestStore_AtomicWrite_NoTmpLeak(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	s.SaveMeta("a.x", KeyMeta{Name: "X"})
	keyDir := filepath.Join(dir, "a.x")
	entries, _ := os.ReadDir(keyDir)
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Errorf("tmp file leaked: %s", e.Name())
		}
	}
}
