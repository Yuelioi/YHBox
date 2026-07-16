package dependency

import "testing"

func TestDependencyString(t *testing.T) {
	dependency := Dependency{Kind: KindSubgraph, Key: "press_esc"}
	if got := dependency.String(); got != "subgraph:press_esc" {
		t.Fatalf("String() = %q", got)
	}
}
