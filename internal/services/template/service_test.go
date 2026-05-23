package template

import (
	"strings"
	"testing"
)

const testContainerID = "test_container"

func newTestService(t *testing.T) *Service {
	t.Helper()
	return NewService(t.TempDir(), nil)
}

func TestService_List_Empty(t *testing.T) {
	svc := newTestService(t)
	out, err := svc.List(testContainerID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("expected empty, got %d", len(out))
	}
}

func TestService_Save_BadDataURL(t *testing.T) {
	svc := newTestService(t)
	err := svc.Save(testContainerID, "ui.btn", "not-a-dataurl", "x", "", [2]int{}, [4]float32{})
	if err == nil || !strings.Contains(err.Error(), "dataURL") {
		t.Errorf("expected dataURL error, got %v", err)
	}
}

func TestService_SaveDataURL(t *testing.T) {
	svc := newTestService(t)

	// 1x1 black PNG base64
	dataURL := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="

	if err := svc.Save(testContainerID, "ui.btn", dataURL, "btn", "", [2]int{1920, 1080}, [4]float32{0, 0, 0.1, 0.1}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	out, err := svc.List(testContainerID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	meta, ok := out["ui.btn"]
	if !ok {
		t.Fatal("key ui.btn missing after Save")
	}
	if meta.Name != "btn" {
		t.Errorf("name: got %q, want %q", meta.Name, "btn")
	}
}

func TestService_Delete(t *testing.T) {
	svc := newTestService(t)

	dataURL := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	if err := svc.Save(testContainerID, "ui.btn", dataURL, "btn", "", [2]int{1920, 1080}, [4]float32{0, 0, 0.1, 0.1}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := svc.Delete(testContainerID, "ui.btn"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	out, err := svc.List(testContainerID)
	if err != nil {
		t.Fatalf("List after Delete: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("expected empty after delete, got %d", len(out))
	}
}

func TestService_ReadPngDataURL(t *testing.T) {
	svc := newTestService(t)

	dataURL := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	if err := svc.Save(testContainerID, "ui.btn", dataURL, "btn", "", [2]int{1920, 1080}, [4]float32{0, 0, 0.1, 0.1}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := svc.ReadPngDataURL(testContainerID, "ui.btn")
	if err != nil {
		t.Fatalf("ReadPngDataURL: %v", err)
	}
	if !strings.HasPrefix(got, "data:image/png;base64,") {
		t.Errorf("unexpected prefix: %q", got[:min(40, len(got))])
	}
}

func TestService_PerContainerIsolation(t *testing.T) {
	svc := newTestService(t)

	dataURL := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	if err := svc.Save("container_a", "ui.btn", dataURL, "btn", "", [2]int{1920, 1080}, [4]float32{0, 0, 0.1, 0.1}); err != nil {
		t.Fatalf("Save container_a: %v", err)
	}

	outB, err := svc.List("container_b")
	if err != nil {
		t.Fatalf("List container_b: %v", err)
	}
	if len(outB) != 0 {
		t.Errorf("container_b should be isolated but got %d entries", len(outB))
	}

	outA, err := svc.List("container_a")
	if err != nil {
		t.Fatalf("List container_a: %v", err)
	}
	if len(outA) != 1 {
		t.Errorf("container_a should have 1 entry, got %d", len(outA))
	}
}

func TestService_ListVariants_MultipleSizes(t *testing.T) {
	svc := newTestService(t)
	dataURL := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	if err := svc.Save(testContainerID, "ui.btn", dataURL, "btn", "", [2]int{1920, 1080}, [4]float32{0, 0, 0.1, 0.1}); err != nil {
		t.Fatalf("Save 1080p: %v", err)
	}
	if err := svc.Save(testContainerID, "ui.btn", dataURL, "btn", "", [2]int{1280, 720}, [4]float32{0, 0, 0.1, 0.1}); err != nil {
		t.Fatalf("Save 720p: %v", err)
	}
	vs, err := svc.ListVariants(testContainerID, "ui.btn")
	if err != nil {
		t.Fatalf("ListVariants: %v", err)
	}
	if len(vs) != 2 {
		t.Errorf("len = %d, want 2", len(vs))
	}
	// area DESC: 1080p first
	if vs[0].Resolution != [2]int{1920, 1080} {
		t.Errorf("first = %v, want 1920x1080", vs[0].Resolution)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
