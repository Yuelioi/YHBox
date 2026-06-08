package catalog

import (
	"strings"
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

// Part A 守卫: exec 出口携带的 Data 字段被序列化进 catalog (此前丢失, 调研要 grep 源码才知道)。
func TestBuild_OutputDataSerialized(t *testing.T) {
	cat := Build()
	outData := func(kind, exit string) []PinData {
		for _, n := range cat {
			if n.Kind != kind {
				continue
			}
			for _, o := range n.Outputs {
				if o.Name == exit {
					return o.Data
				}
			}
			t.Fatalf("%s: 出口 %q 不存在", kind, exit)
		}
		t.Fatalf("catalog 里没有 %s", kind)
		return nil
	}
	hasField := func(data []PinData, name, typ string) bool {
		for _, d := range data {
			if d.Name == name && d.Type == typ {
				return true
			}
		}
		return false
	}

	if d := outData("DetectColor", "Found"); !hasField(d, "Center", "Point") {
		t.Errorf("DetectColor.Found 应携带 Center(Point), 实得 %+v", d)
	}
	if d := outData("CheckTemplate", "Found"); !hasField(d, "Point", "Point") {
		t.Errorf("CheckTemplate.Found 应携带 Point(Point), 实得 %+v", d)
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

// TestNoPinNameSplit — 守卫: 全节点 pin 名不准出现命名分裂 (形近撞名 / 同名不同具体类型)。
// 新加节点若另起同义名 (如又写个 `Roi`), 这条 FAIL。详情/对齐看 node-spec-style §9 + `task nodes:pins`。
func TestNoPinNameSplit(t *testing.T) {
	if splits := DetectNameSplits(); len(splits) > 0 {
		t.Errorf("检测到 pin 命名分裂 %d 处 — 同概念复用既有名 (见 node-spec-style §9 / 跑 `task nodes:pins`):\n  %s",
			len(splits), strings.Join(splits, "\n  "))
	}
}
