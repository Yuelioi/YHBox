package container

import (
	"testing"

	_ "yotta/internal/nodes/input"  // ClickAt (NeedsWindow)
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
