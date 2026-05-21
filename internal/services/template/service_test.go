package template

import (
	"strings"
	"testing"
)

func TestService_List_Empty(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	svc := NewService(s, nil)
	out := svc.List()
	if len(out) != 0 {
		t.Errorf("expected empty, got %d", len(out))
	}
}

func TestService_SaveRaw(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	svc := NewService(s, nil)

	pngData := []byte{0x89, 0x50, 0x4E, 0x47}

	if err := svc.SaveRaw("ui.btn", pngData, TemplateMeta{Name: "btn", Width: 1, Height: 1}); err != nil {
		t.Fatalf("SaveRaw: %v", err)
	}

	got := svc.List()
	if len(got) != 1 {
		t.Errorf("expected 1, got %d", len(got))
	}
	if _, ok := got["ui.btn"]; !ok {
		t.Errorf("key ui.btn missing")
	}
}

func TestService_SaveDataURL(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	svc := NewService(s, nil)

	// 1x1 black PNG base64
	dataURL := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="

	if err := svc.Save("ui.btn", dataURL, "btn", "", [2]int{1920, 1080}, [4]float32{0, 0, 0, 0}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	meta, ok := s.Get("ui.btn")
	if !ok {
		t.Fatal("not saved")
	}
	if meta.Name != "btn" {
		t.Errorf("name: got %q", meta.Name)
	}
}

func TestService_Save_BadDataURL(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	svc := NewService(s, nil)
	err := svc.Save("ui.btn", "not-a-dataurl", "x", "", [2]int{}, [4]float32{})
	if err == nil || !strings.Contains(err.Error(), "dataURL") {
		t.Errorf("expected dataURL error, got %v", err)
	}
}
