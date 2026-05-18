package container

import "testing"

func TestExecOutPinsForNode_Static(t *testing.T) {
	n := &GraphNode{Kind: "Sleep"}
	pins := execOutPinsForNode(n)
	if _, ok := pins["out"]; !ok {
		t.Errorf("Sleep 应有 out pin, got %v", pins)
	}
	if len(pins) != 1 {
		t.Errorf("Sleep 应只 1 pin, got %d", len(pins))
	}
}

func TestExecOutPinsForNode_Parallel_Dynamic(t *testing.T) {
	n := &GraphNode{Kind: "Parallel", Config: map[string]any{"literal": map[string]any{"n": float64(3)}}}
	pins := execOutPinsForNode(n)
	for _, want := range []string{"branch0", "branch1", "branch2", "complete"} {
		if _, ok := pins[want]; !ok {
			t.Errorf("Parallel n=3 应有 %s pin, got %v", want, pins)
		}
	}
	if _, has := pins["branch3"]; has {
		t.Error("Parallel n=3 不应有 branch3")
	}
}

func TestExecOutPinsForNode_Switch_Dynamic(t *testing.T) {
	n := &GraphNode{Kind: "Switch", Config: map[string]any{
		"value": "$vars.x",
		"cases": []any{"A", "B", "C"},
	}}
	pins := execOutPinsForNode(n)
	for _, want := range []string{"A", "B", "C", "default"} {
		if _, ok := pins[want]; !ok {
			t.Errorf("Switch 应有 %s pin, got %v", want, pins)
		}
	}
	if len(pins) != 4 {
		t.Errorf("Switch 应 4 pin (3 case + default), got %d", len(pins))
	}
}

func TestExecOutPinsForNode_Switch_DedupesDefaultCase(t *testing.T) {
	n := &GraphNode{Kind: "Switch", Config: map[string]any{
		"cases": []any{"default", "A"},
	}}
	pins := execOutPinsForNode(n)
	if len(pins) != 2 {
		t.Errorf("dedupe 后应 2 pin (default + A), got %d: %v", len(pins), pins)
	}
}

func TestExecOutPinsForNode_Switch_FiltersEmptyCase(t *testing.T) {
	n := &GraphNode{Kind: "Switch", Config: map[string]any{
		"cases": []any{"", "A"},
	}}
	pins := execOutPinsForNode(n)
	if _, has := pins[""]; has {
		t.Error("空 case 应被过滤")
	}
	if _, has := pins["A"]; !has {
		t.Error("A pin 应存在")
	}
}

func TestNodeHasExecOutPin(t *testing.T) {
	n := &GraphNode{Kind: "Switch", Config: map[string]any{"cases": []any{"X"}}}
	if !nodeHasExecOutPin(n, "X") {
		t.Error("应识别 case pin")
	}
	if !nodeHasExecOutPin(n, "default") {
		t.Error("应识别 default pin")
	}
	if nodeHasExecOutPin(n, "Y") {
		t.Error("不应识别未知 pin")
	}
}
