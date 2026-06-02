package catalog

import (
	"testing"

	_ "yotta/internal/nodes/control"
	_ "yotta/internal/nodes/detect"
	_ "yotta/internal/nodes/input"
	_ "yotta/internal/nodes/io"
	_ "yotta/internal/nodes/purefunc"
	_ "yotta/internal/nodes/stopwatch"
	_ "yotta/internal/nodes/system"
	_ "yotta/internal/nodes/variable"
)

func TestBuild_HasNodesAndSorted(t *testing.T) {
	cat := Build()
	if len(cat) == 0 {
		t.Fatal("catalog empty — node packages not registered?")
	}
	for i := 1; i < len(cat); i++ {
		a, b := cat[i-1], cat[i]
		if a.Category > b.Category || (a.Category == b.Category && a.Kind > b.Kind) {
			t.Fatalf("not sorted at %d: %s/%s before %s/%s", i, a.Category, a.Kind, b.Category, b.Kind)
		}
	}
}

func TestBuild_KeyPressShape(t *testing.T) {
	for _, n := range Build() {
		if n.Kind != "KeyPress" {
			continue
		}
		if !n.NeedsWindow {
			t.Error("KeyPress should be needsWindow")
		}
		var vk *Pin
		for i := range n.Inputs {
			if n.Inputs[i].Name == "VK" {
				vk = &n.Inputs[i]
			}
		}
		if vk == nil || !vk.Required {
			t.Fatalf("KeyPress.VK should exist and be required, got %+v", n.Inputs)
		}
		return
	}
	t.Fatal("KeyPress not found in catalog")
}
