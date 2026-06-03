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

func TestBuildWithI18n_KeyPressLabeled(t *testing.T) {
	for _, n := range BuildWithI18n() {
		if n.Kind != "KeyPress" {
			continue
		}
		if n.Label != "按键" {
			t.Errorf("KeyPress.Label = %q, want 按键", n.Label)
		}
		if n.Description == "" {
			t.Error("KeyPress.Description should be non-empty")
		}
		for _, p := range n.Inputs {
			if p.Name == "VK" && p.Label == "" {
				t.Error("KeyPress.VK should have a label")
			}
		}
		return
	}
	t.Fatal("KeyPress not found in catalog")
}

// drift guard: zh.ts 加节点没同步 node-i18n.json (忘跑 pnpm gen:node-i18n) 即 fail。
func TestBuildWithI18n_AllKindsLabeled(t *testing.T) {
	var missing []string
	for _, n := range BuildWithI18n() {
		if n.Label == "" || n.Description == "" {
			missing = append(missing, n.Kind)
		}
	}
	if len(missing) > 0 {
		t.Errorf("缺 label/description 的节点 (zh.ts 改了没跑 `pnpm gen:node-i18n`?): %v", missing)
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
