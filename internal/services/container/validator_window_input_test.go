package container

import (
	"testing"

	_ "yotta/internal/nodes/input"  // ClickAt (NeedsTarget)
	_ "yotta/internal/nodes/system" // AndroidTarget / Win32WindowTarget
	_ "yotta/internal/nodes/window" // GetWindow (产出 Window, 接 ClickAt.Window)
)

// 连了 Window 输入的 NeedsWindow 节点 + 无 Win32WindowTarget → 不应报 MISSING_WIN32_WINDOW_TARGET。
func TestValidate_WiredWindowInput_NoMissingWin32WindowTarget(t *testing.T) {
	c := minContainer()
	c.Graph.Nodes = append(c.Graph.Nodes,
		GraphNode{ID: "gw", Kind: "GetWindow"},
		GraphNode{ID: "ca", Kind: "ClickAt"},
	)
	c.Graph.Edges = append(c.Graph.Edges,
		GraphEdge{From: "start.Done", To: "gw.In"},
		GraphEdge{From: "gw.Done", To: "ca.In"},
		GraphEdge{From: "gw.Window", To: "ca.Window"}, // data 边: 连了 ClickAt 的 Window 输入
	)
	errs := ValidateContainer(c, nil)
	if hasCode(errs, CodeMissingWin32WindowTarget) {
		t.Errorf("ClickAt 连了 Window 输入, 无 Win32WindowTarget 不该报缺: %+v", errs)
	}
}

// Window 没连的 NeedsWindow 节点 + 无 Win32WindowTarget → 应报 MISSING_WIN32_WINDOW_TARGET。
func TestValidate_UnwiredNeedsWindow_ReportsMissingWin32WindowTarget(t *testing.T) {
	c := minContainer()
	c.Graph.Nodes = append(c.Graph.Nodes, GraphNode{ID: "ca", Kind: "ClickAt"})
	c.Graph.Edges = append(c.Graph.Edges, GraphEdge{From: "start.Done", To: "ca.In"})
	errs := ValidateContainer(c, nil)
	if !hasCode(errs, CodeMissingWin32WindowTarget) {
		t.Errorf("ClickAt 没连 Window, 无 Win32WindowTarget 应报缺: %+v", errs)
	}
}

// AndroidTarget 已显式选择非 Win32 自动化目标；target-aware 输入节点不应再要求 Win32WindowTarget。
func TestValidate_AndroidTargetWithInput_NoMissingWin32WindowTarget(t *testing.T) {
	c := minContainer()
	c.Graph.Nodes = append(c.Graph.Nodes,
		GraphNode{ID: "at", Kind: "AndroidTarget", Config: map[string]any{
			"literal": map[string]any{"Serial": "emulator-5554", "Width": 1080, "Height": 1920},
		}},
		GraphNode{ID: "ca", Kind: "ClickAt"},
	)
	c.Graph.Edges = append(c.Graph.Edges,
		GraphEdge{From: "start.Done", To: "at.In"},
		GraphEdge{From: "at.Done", To: "ca.In"},
	)
	errs := ValidateContainer(c, nil)
	if hasCode(errs, CodeMissingWin32WindowTarget) {
		t.Errorf("AndroidTarget + ClickAt 不应要求 Win32WindowTarget: %+v", errs)
	}
	if hasCode(errs, CodeUnsupportedTargetCapability) {
		t.Errorf("AndroidTarget + ClickAt 的基础点击能力应可用: %+v", errs)
	}
}

func TestValidate_AndroidTargetWithMouseMoveRel_ReportsUnsupportedTargetCapability(t *testing.T) {
	c := minContainer()
	c.Graph.Nodes = append(c.Graph.Nodes,
		GraphNode{ID: "at", Kind: "AndroidTarget", Config: map[string]any{
			"literal": map[string]any{"Serial": "emulator-5554", "Width": 1080, "Height": 1920},
		}},
		GraphNode{ID: "move", Kind: "MouseMoveRel"},
	)
	c.Graph.Edges = append(c.Graph.Edges,
		GraphEdge{From: "start.Done", To: "at.In"},
		GraphEdge{From: "at.Done", To: "move.In"},
	)

	errs := ValidateContainer(c, nil)
	if !hasCode(errs, CodeUnsupportedTargetCapability) {
		t.Fatalf("AndroidTarget + MouseMoveRel 应报 target capability 不支持: %+v", errs)
	}
	if hasCode(errs, CodeMissingWin32WindowTarget) {
		t.Fatalf("AndroidTarget 已显式选目标, 不应同时要求 Win32WindowTarget: %+v", errs)
	}
}

func TestValidate_AndroidTargetWithKeyPress_ReportsUnsupportedTargetCapability(t *testing.T) {
	c := minContainer()
	c.Graph.Nodes = append(c.Graph.Nodes,
		GraphNode{ID: "at", Kind: "AndroidTarget", Config: map[string]any{
			"literal": map[string]any{"Serial": "emulator-5554", "Width": 1080, "Height": 1920},
		}},
		GraphNode{ID: "key", Kind: "KeyPress"},
	)
	c.Graph.Edges = append(c.Graph.Edges,
		GraphEdge{From: "start.Done", To: "at.In"},
		GraphEdge{From: "at.Done", To: "key.In"},
	)

	errs := ValidateContainer(c, nil)
	if !hasCode(errs, CodeUnsupportedTargetCapability) {
		t.Fatalf("AndroidTarget + KeyPress 应报 key-state 不支持: %+v", errs)
	}
}
