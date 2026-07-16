package script

import "testing"

func TestDependenciesExtractsLiteralSubgraphCalls(t *testing.T) {
	code := `
let first = Subgraph({SubgraphID: "press_esc"});
let second = Subgraph({ SubgraphID: 'try_hook_F', msg: "hi" });
Subgraph({SubgraphID: "press_esc"});
`
	got := Dependencies(code)
	if len(got) != 2 {
		t.Fatalf("dependency count = %d, want 2: %+v", len(got), got)
	}
	if got[0].Kind != "subgraph" || got[0].Key != "press_esc" ||
		got[1].Kind != "subgraph" || got[1].Key != "try_hook_F" {
		t.Fatalf("dependencies = %+v", got)
	}
}

func TestDependenciesDoesNotInferAssetsFromUUIDs(t *testing.T) {
	code := `const image = "3680b3d2-d31d-461c-b697-0d9c3e6a87ed";`
	if got := Dependencies(code); len(got) != 0 {
		t.Fatalf("arbitrary UUID became a dependency: %+v", got)
	}
}

func TestDependenciesEmptyOrDynamicSubgraphID(t *testing.T) {
	for _, code := range []string{"", "Subgraph({SubgraphID: id})", "log.info('none')"} {
		if got := Dependencies(code); len(got) != 0 {
			t.Fatalf("Dependencies(%q) = %+v, want none", code, got)
		}
	}
}
