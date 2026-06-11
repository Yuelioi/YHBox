package codesnippet

import (
	"path/filepath"
	"testing"
)

func TestListMissingFileReturnsEmpty(t *testing.T) {
	svc := NewService(filepath.Join(t.TempDir(), "snippets.json"))
	got, err := svc.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("want empty, got %d", len(got))
	}
}

func TestSaveAllRoundTrip(t *testing.T) {
	svc := NewService(filepath.Join(t.TempDir(), "snippets.json"))
	in := []Snippet{
		{ID: "a", Lang: "script", Name: "循环", Body: "for (let i = 0; i < 10; i++) {\n}"},
		{ID: "b", Lang: "expr", Name: "求和", Body: "Add({A: $x, B: 1})"},
	}
	if err := svc.SaveAll(in); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}
	got, err := svc.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 || got[0] != in[0] || got[1] != in[1] {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}

func TestSaveAllNilClearsFile(t *testing.T) {
	svc := NewService(filepath.Join(t.TempDir(), "snippets.json"))
	if err := svc.SaveAll([]Snippet{{ID: "a", Lang: "script", Name: "x", Body: "y"}}); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}
	if err := svc.SaveAll(nil); err != nil {
		t.Fatalf("SaveAll(nil): %v", err)
	}
	got, err := svc.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("want empty after nil save, got %d", len(got))
	}
}
