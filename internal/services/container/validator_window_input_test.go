package container

import (
	"testing"

	_ "github.com/yottaapp/yotta/internal/nodes/detect" // ClickTemplate (NeedsTarget + config-derived key-state)
	_ "github.com/yottaapp/yotta/internal/nodes/input"  // ClickAt (NeedsTarget)
	_ "github.com/yottaapp/yotta/internal/nodes/system" // AndroidTarget / Win32WindowTarget
	_ "github.com/yottaapp/yotta/internal/nodes/window" // GetWindow (产出 Window, 接 ClickAt.Window)
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

func TestValidate_AndroidTargetWithClickAtModifierKeys_ReportsUnsupportedTargetCapability(t *testing.T) {
	c := minContainer()
	c.Graph.Nodes = append(c.Graph.Nodes,
		GraphNode{ID: "at", Kind: "AndroidTarget", Config: map[string]any{
			"literal": map[string]any{"Serial": "emulator-5554", "Width": 1080, "Height": 1920},
		}},
		GraphNode{ID: "click", Kind: "ClickAt", Config: map[string]any{
			"literal": map[string]any{"Keys": "ctrl"},
		}},
	)
	c.Graph.Edges = append(c.Graph.Edges,
		GraphEdge{From: "start.Done", To: "at.In"},
		GraphEdge{From: "at.Done", To: "click.In"},
	)

	errs := ValidateContainer(c, nil)
	if !hasCode(errs, CodeUnsupportedTargetCapability) {
		t.Fatalf("AndroidTarget + ClickAt.Keys 应报 key-state 不支持: %+v", errs)
	}
}

func TestValidate_AndroidTargetWithClickTemplateModifierKeys_ReportsUnsupportedTargetCapability(t *testing.T) {
	c := minContainer()
	c.Graph.Nodes = append(c.Graph.Nodes,
		GraphNode{ID: "at", Kind: "AndroidTarget", Config: map[string]any{
			"literal": map[string]any{"Serial": "emulator-5554", "Width": 1080, "Height": 1920},
		}},
		GraphNode{ID: "clickTemplate", Kind: "ClickTemplate", Config: map[string]any{
			"literal": map[string]any{"Templates": []any{"template-1"}, "Keys": "ctrl"},
		}},
	)
	c.Graph.Edges = append(c.Graph.Edges,
		GraphEdge{From: "start.Done", To: "at.In"},
		GraphEdge{From: "at.Done", To: "clickTemplate.In"},
	)

	errs := ValidateContainer(c, nil)
	if !hasCode(errs, CodeUnsupportedTargetCapability) {
		t.Fatalf("AndroidTarget + ClickTemplate.Keys 应报 key-state 不支持: %+v", errs)
	}
}

func TestValidate_AndroidTargetWithClickAtRightButton_ReportsUnsupportedTargetCapability(t *testing.T) {
	c := minContainer()
	c.Graph.Nodes = append(c.Graph.Nodes,
		GraphNode{ID: "at", Kind: "AndroidTarget", Config: map[string]any{
			"literal": map[string]any{"Serial": "emulator-5554", "Width": 1080, "Height": 1920},
		}},
		GraphNode{ID: "click", Kind: "ClickAt", Config: map[string]any{
			"literal": map[string]any{"Button": "right"},
		}},
	)
	c.Graph.Edges = append(c.Graph.Edges,
		GraphEdge{From: "start.Done", To: "at.In"},
		GraphEdge{From: "at.Done", To: "click.In"},
	)

	errs := ValidateContainer(c, nil)
	if !hasCode(errs, CodeUnsupportedTargetCapability) {
		t.Fatalf("AndroidTarget + ClickAt.Button=right 应报 mouse-button 不支持: %+v", errs)
	}
}

func TestValidate_AndroidTargetWithClickTemplateMiddleButton_ReportsUnsupportedTargetCapability(t *testing.T) {
	c := minContainer()
	c.Graph.Nodes = append(c.Graph.Nodes,
		GraphNode{ID: "at", Kind: "AndroidTarget", Config: map[string]any{
			"literal": map[string]any{"Serial": "emulator-5554", "Width": 1080, "Height": 1920},
		}},
		GraphNode{ID: "clickTemplate", Kind: "ClickTemplate", Config: map[string]any{
			"literal": map[string]any{"Templates": []any{"template-1"}, "Button": "middle"},
		}},
	)
	c.Graph.Edges = append(c.Graph.Edges,
		GraphEdge{From: "start.Done", To: "at.In"},
		GraphEdge{From: "at.Done", To: "clickTemplate.In"},
	)

	errs := ValidateContainer(c, nil)
	if !hasCode(errs, CodeUnsupportedTargetCapability) {
		t.Fatalf("AndroidTarget + ClickTemplate.Button=middle 应报 mouse-button 不支持: %+v", errs)
	}
}

func TestValidate_SubgraphInheritsAndroidTargetCapabilities(t *testing.T) {
	c := minContainer()
	c.Graph.Nodes = append(c.Graph.Nodes,
		GraphNode{ID: "at", Kind: "AndroidTarget", Config: map[string]any{
			"literal": map[string]any{"Serial": "emulator-5554", "Width": 1080, "Height": 1920},
		}},
		GraphNode{ID: "call", Kind: "Subgraph", Config: map[string]any{
			"literal": map[string]any{"SubgraphID": "sg-move"},
		}},
	)
	c.Graph.Edges = append(c.Graph.Edges,
		GraphEdge{From: "start.Done", To: "at.In"},
		GraphEdge{From: "at.Done", To: "call.In"},
	)
	sgs := []Subgraph{{
		ID:    "sg-move",
		Label: "move",
		Graph: Graph{
			Nodes: []GraphNode{{ID: "move", Kind: "MouseMoveRel"}},
		},
		OutputPins: []SubgraphOutputDecl{{ID: "done", Name: "Done"}},
	}}

	errs := ValidateContainer(c, sgs)
	if !hasCode(errs, CodeUnsupportedTargetCapability) {
		t.Fatalf("AndroidTarget -> Subgraph(MouseMoveRel) 应继承 Android target 并报 move-relative 不支持: %+v", errs)
	}
}

func TestValidate_SubgraphLocalWin32TargetOverridesInheritedAndroidTarget(t *testing.T) {
	c := minContainer()
	c.Graph.Nodes = append(c.Graph.Nodes,
		GraphNode{ID: "at", Kind: "AndroidTarget", Config: map[string]any{
			"literal": map[string]any{"Serial": "emulator-5554", "Width": 1080, "Height": 1920},
		}},
		GraphNode{ID: "call", Kind: "Subgraph", Config: map[string]any{
			"literal": map[string]any{"SubgraphID": "sg-win32-move"},
		}},
	)
	c.Graph.Edges = append(c.Graph.Edges,
		GraphEdge{From: "start.Done", To: "at.In"},
		GraphEdge{From: "at.Done", To: "call.In"},
	)
	sgs := []Subgraph{{
		ID:    "sg-win32-move",
		Label: "win32 move",
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "wt", Kind: "Win32WindowTarget", Config: map[string]any{
					"literal": map[string]any{"Title": "After Effects", "TitleMatch": "contains"},
				}},
				{ID: "move", Kind: "MouseMoveRel"},
			},
			Edges: []GraphEdge{{From: "wt.Done", To: "move.In"}},
		},
		OutputPins: []SubgraphOutputDecl{{ID: "done", Name: "Done"}},
	}}

	errs := ValidateContainer(c, sgs)
	if hasCode(errs, CodeUnsupportedTargetCapability) {
		t.Fatalf("子图本地 Win32WindowTarget 应覆盖父图 Android target: %+v", errs)
	}
}

func TestValidate_CollapsedNodeInheritsAndroidTargetCapabilities(t *testing.T) {
	c := minContainer()
	c.Graph.Nodes = append(c.Graph.Nodes,
		GraphNode{ID: "at", Kind: "AndroidTarget", Config: map[string]any{
			"literal": map[string]any{"Serial": "emulator-5554", "Width": 1080, "Height": 1920},
		}},
		GraphNode{ID: "collapsed", Kind: "CollapsedNode", Config: map[string]any{
			"literal": map[string]any{"SubgraphID": "sg-collapsed-move"},
		}},
	)
	c.Graph.Edges = append(c.Graph.Edges,
		GraphEdge{From: "start.Done", To: "at.In"},
		GraphEdge{From: "at.Done", To: "collapsed.In"},
	)
	sgs := []Subgraph{{
		ID:          "sg-collapsed-move",
		Label:       "collapsed move",
		IsAnonymous: true,
		Graph: Graph{
			Nodes: []GraphNode{{ID: "move", Kind: "MouseMoveRel"}},
		},
		OutputPins: []SubgraphOutputDecl{{ID: "done", Name: "Done"}},
	}}

	errs := ValidateContainer(c, sgs)
	if !hasCode(errs, CodeUnsupportedTargetCapability) {
		t.Fatalf("AndroidTarget -> CollapsedNode(MouseMoveRel) 应继承 Android target 并报 move-relative 不支持: %+v", errs)
	}
}
