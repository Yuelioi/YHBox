package container

import (
	"errors"
	"testing"
)

func TestExecOutPinsForNode_Static_Sleep(t *testing.T) {
	n := &GraphNode{Kind: "Sleep"}
	pins := execOutPinsForNode(n)
	if _, ok := pins["Done"]; !ok {
		t.Errorf("Sleep 应有 Done pin, got %v", pins)
	}
	if len(pins) != 1 {
		t.Errorf("Sleep 应只 1 exec-out pin, got %d", len(pins))
	}
}

func TestExecOutPinsForNode_Switch_NamedByValue(t *testing.T) {
	// named-by-value: 出口 = config.cases 里每个值 + 'default'.
	n := &GraphNode{Kind: "Switch", Config: map[string]any{"cases": []any{"IDLE", "FISHING"}}}
	pins := execOutPinsForNode(n)
	for _, want := range []string{"IDLE", "FISHING", "default"} {
		if _, ok := pins[want]; !ok {
			t.Errorf("Switch 应有 %s exec-out pin, got %v", want, pins)
		}
	}
	if len(pins) != 3 {
		t.Errorf("应只 3 pins (2 case + default), got %d: %v", len(pins), pins)
	}
}

func TestNodeHasExecOutPin(t *testing.T) {
	n := &GraphNode{Kind: "Switch", Config: map[string]any{"cases": []any{"IDLE"}}}
	if !nodeHasExecOutPin(n, "IDLE") {
		t.Error("应识别 case pin IDLE")
	}
	if !nodeHasExecOutPin(n, "default") {
		t.Error("应识别 default pin")
	}
	if nodeHasExecOutPin(n, "BogusPin") {
		t.Error("不应识别未知 pin")
	}
}

// A.6: Validate() 返 *ValidationFailure 聚合多个 error, caller errors.As 取结构化列表.
func TestValidateReturnsValidationFailure(t *testing.T) {
	// Container with multiple errors: no Start node + missing Win32WindowTarget.
	c := &Container{
		SchemaVersion: 4,
		Graph: Graph{
			ID: "g", SchemaVersion: 1,
			Nodes: []GraphNode{{ID: "n1", Kind: "WindowState", Config: map[string]any{}}},
		},
	}
	err := c.Validate(nil)
	if err == nil {
		t.Fatal("expected validation error")
	}
	var vf *ValidationFailure
	if !errors.As(err, &vf) {
		t.Fatalf("expected *ValidationFailure, got %T", err)
	}
	if len(vf.Errors) < 2 {
		t.Errorf("expected ≥2 errors (NO_START + MISSING_WIN32_WINDOW_TARGET), got %d: %+v", len(vf.Errors), vf.Errors)
	}
	// Sanity: every entry is SeverityError
	for _, e := range vf.Errors {
		if e.Severity != SeverityError {
			t.Errorf("ValidationFailure contains non-error: %+v", e)
		}
	}
}

func TestValidatePassesWhenNoErrors(t *testing.T) {
	// Empty container should pass (existing convention — no nodes = no errors).
	c := &Container{SchemaVersion: 4, Graph: Graph{ID: "g", SchemaVersion: 1}}
	if err := c.Validate(nil); err != nil {
		t.Errorf("empty container should validate, got: %v", err)
	}
}
