package library

import (
	"testing"

	"yhbox/internal/services/container"
)

func TestComputeMissingGlobals_AllMissing(t *testing.T) {
	pkg := &SubgraphPackage{
		Root: container.Subgraph{
			ID: "root",
			RequiredGlobals: []container.SubgraphRequiredGlobal{
				{Name: "state", Type: "string"},
				{Name: "hookStreak", Type: "number"},
			},
		},
		Embedded: map[string]container.Subgraph{
			"callee": {
				ID: "callee",
				RequiredGlobals: []container.SubgraphRequiredGlobal{
					{Name: "controlDir", Type: "string"},
					{Name: "state"}, // dedup with root
				},
			},
		},
	}
	got := computeMissingGlobals(pkg, nil) // container 没声明任何 var
	if len(got) != 3 {
		t.Fatalf("expected 3 unique missing globals (state/hookStreak/controlDir), got %d: %+v", len(got), got)
	}
	names := map[string]bool{}
	for _, g := range got {
		names[g.Name] = true
	}
	for _, want := range []string{"state", "hookStreak", "controlDir"} {
		if !names[want] {
			t.Errorf("missing expected name %q in result", want)
		}
	}
}

func TestComputeMissingGlobals_PartialDeclared(t *testing.T) {
	pkg := &SubgraphPackage{
		Root: container.Subgraph{
			RequiredGlobals: []container.SubgraphRequiredGlobal{
				{Name: "state"},
				{Name: "hookStreak"},
			},
		},
	}
	got := computeMissingGlobals(pkg, []container.VarDecl{
		{Name: "state", Type: "string"}, // 已声明 → 不在 missing
	})
	if len(got) != 1 || got[0].Name != "hookStreak" {
		t.Fatalf("expected only hookStreak missing, got %+v", got)
	}
}

func TestComputeMissingGlobals_None(t *testing.T) {
	pkg := &SubgraphPackage{Root: container.Subgraph{}}
	got := computeMissingGlobals(pkg, nil)
	if got != nil {
		t.Errorf("expected nil missing globals for sg without RequiredGlobals, got %+v", got)
	}
}
