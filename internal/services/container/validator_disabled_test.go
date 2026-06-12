package container

import "testing"

func TestValidateDisabledNodes_TerminalError(t *testing.T) {
	c := &Container{
		Graph: Graph{Nodes: []GraphNode{
			{ID: "s", Kind: "Start", Disabled: true},
		}},
	}
	errs := validateDisabledNodes(c, nil)
	if len(errs) != 1 || errs[0].Code != CodeInvalidDisabledTerminal || errs[0].Severity != SeverityError {
		t.Fatalf("disabled Start should error, got: %+v", errs)
	}
}

func TestValidateDisabledNodes_EventTickTerminalError(t *testing.T) {
	// EventTick 是 listener-driven container-level 终端: 禁用 = listener 永不启动 → error。
	c := &Container{
		Graph: Graph{Nodes: []GraphNode{
			{ID: "ev", Kind: "EventTick", Disabled: true},
		}},
	}
	errs := validateDisabledNodes(c, nil)
	if len(errs) != 1 || errs[0].Code != CodeInvalidDisabledTerminal || errs[0].Severity != SeverityError {
		t.Fatalf("disabled EventTick should error, got: %+v", errs)
	}
}

func TestValidateDisabledNodes_BranchWarn(t *testing.T) {
	c := &Container{
		Graph: Graph{Nodes: []GraphNode{
			{ID: "loop", Kind: "Loop", Disabled: true},
		}},
	}
	errs := validateDisabledNodes(c, nil)
	if len(errs) != 1 || errs[0].Code != CodeDisabledBranchNodeWarn || errs[0].Severity != SeverityWarning {
		t.Fatalf("disabled Loop should warn, got: %+v", errs)
	}
}

func TestValidateDisabledNodes_LinearOK(t *testing.T) {
	c := &Container{
		Graph: Graph{Nodes: []GraphNode{
			{ID: "sleep", Kind: "Sleep", Disabled: true},
		}},
	}
	errs := validateDisabledNodes(c, nil)
	if len(errs) != 0 {
		t.Fatalf("disabled Sleep should not warn/error, got: %+v", errs)
	}
}

func TestValidateDisabledNodes_NotDisabled(t *testing.T) {
	c := &Container{
		Graph: Graph{Nodes: []GraphNode{
			{ID: "loop", Kind: "Loop"}, // not disabled
		}},
	}
	errs := validateDisabledNodes(c, nil)
	if len(errs) != 0 {
		t.Fatalf("non-disabled Loop should not warn/error, got: %+v", errs)
	}
}
