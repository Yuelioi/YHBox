package container

import (
	"testing"

	"github.com/yottaapp/yotta/internal/automation/target"
	"github.com/yottaapp/yotta/internal/node"

	_ "github.com/yottaapp/yotta/internal/nodes/all"
)

type targetPathNode struct{ kind string }

func (n targetPathNode) Spec() node.Spec {
	return node.Spec{Kind: n.kind, Inputs: []node.InputSpec{{Name: "In", Type: node.TypeExec}}, Outputs: []node.OutputSpec{{Name: "Done", Type: node.TypeExec}}}
}
func (targetPathNode) Run(node.Ctx, node.Inputs) (node.Outputs, error) { return nil, nil }

func TestWin32WindowTargetForNode(t *testing.T) {
	// 构造 Start → WT_A → n1(Sleep) → WT_B → n2(Sleep)
	// 用 exec 边 "<id>.<outpin>"→"<id>.In" 串联.
	c := &Container{Graph: Graph{
		Nodes: []GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "wta", Kind: "Win32WindowTarget", Config: map[string]any{"Title": "A"}},
			{ID: "n1", Kind: "Sleep"},
			{ID: "wtb", Kind: "Win32WindowTarget", Config: map[string]any{"Title": "B"}},
			{ID: "n2", Kind: "Sleep"},
		},
		Edges: []GraphEdge{
			{From: "start.Done", To: "wta.In"},
			{From: "wta.Done", To: "n1.In"},
			{From: "n1.Done", To: "wtb.In"},
			{From: "wtb.Done", To: "n2.In"},
		},
	}}
	if wt := win32WindowTargetForNode(c, "n2"); wt == nil || PinString(wt, "Title") != "B" {
		t.Fatalf("n2 应回溯到 WT_B, got %v", wt)
	}
	if wt := win32WindowTargetForNode(c, "n1"); wt == nil || PinString(wt, "Title") != "A" {
		t.Fatalf("n1 应回溯到 WT_A, got %v", wt)
	}
	if wt := win32WindowTargetForNode(c, ""); wt == nil || PinString(wt, "Title") != "A" {
		t.Fatalf("空 nodeID 应回落主窗口 WT_A, got %v", wt)
	}
}

func TestEditorTargetKindForNode(t *testing.T) {
	c := &Container{Graph: Graph{
		Nodes: []GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "wt", Kind: "Win32WindowTarget"},
			{ID: "winClick", Kind: "Sleep"},
			{ID: "at", Kind: "AndroidTarget"},
			{ID: "androidClick", Kind: "Sleep"},
		},
		Edges: []GraphEdge{
			{From: "start.Done", To: "wt.In"},
			{From: "wt.Done", To: "winClick.In"},
			{From: "winClick.Done", To: "at.In"},
			{From: "at.Done", To: "androidClick.In"},
		},
	}}
	if got, ok := editorTargetKindForNode(c, "winClick"); !ok || got != target.KindWin32Window {
		t.Fatalf("winClick target = %q,%v want %q,true", got, ok, target.KindWin32Window)
	}
	if got, ok := editorTargetKindForNode(c, "androidClick"); !ok || got != target.KindAndroidADB {
		t.Fatalf("androidClick target = %q,%v want %q,true", got, ok, target.KindAndroidADB)
	}
	if got, ok := editorTargetKindForNode(c, ""); !ok || got != target.KindWin32Window {
		t.Fatalf("empty node target = %q,%v want first target %q,true", got, ok, target.KindWin32Window)
	}
}

func TestEditorTargetKindForNode_DefaultsToWin32WhenNoTarget(t *testing.T) {
	c := &Container{Graph: Graph{Nodes: []GraphNode{{ID: "start", Kind: "Start"}, {ID: "sleep", Kind: "Sleep"}}}}
	if got, ok := editorTargetKindForNode(c, "sleep"); !ok || got != target.KindWin32Window {
		t.Fatalf("target = %q,%v want %q,true", got, ok, target.KindWin32Window)
	}
}

func TestEditorTargetForNodeUsesExplicitRegistryAcrossCustomNode(t *testing.T) {
	registry := node.NewRegistry()
	for _, kind := range []string{"Start", "AndroidTarget", "CustomBridge", "Sleep"} {
		registry.Register(targetPathNode{kind: kind})
	}
	c := &Container{Graph: Graph{
		Nodes: []GraphNode{{ID: "start", Kind: "Start"}, {ID: "target", Kind: "AndroidTarget"}, {ID: "bridge", Kind: "CustomBridge"}, {ID: "click", Kind: "Sleep"}},
		Edges: []GraphEdge{{From: "start.Done", To: "target.In"}, {From: "target.Done", To: "bridge.In"}, {From: "bridge.Done", To: "click.In"}},
	}}
	got, ok := editorTargetForNodeWithRegistry(registry.Snapshot(), c, "click")
	if !ok || got.Kind != target.KindAndroidADB {
		t.Fatalf("target through custom node = %+v,%v", got, ok)
	}
}

func TestEditorTargetForNode_AndroidTargetConfig(t *testing.T) {
	c := &Container{Graph: Graph{
		Nodes: []GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "at", Kind: "AndroidTarget", Config: map[string]any{"literal": map[string]any{
				"Serial": "127.0.0.1:7555",
				"Name":   "MuMu",
				"Width":  1280,
				"Height": 720,
			}}},
			{ID: "click", Kind: "Sleep"},
		},
		Edges: []GraphEdge{
			{From: "start.Done", To: "at.In"},
			{From: "at.Done", To: "click.In"},
		},
	}}
	got, ok := editorTargetForNode(c, "click")
	if !ok {
		t.Fatal("expected editor target")
	}
	if got.Kind != target.KindAndroidADB || got.Ref.ADBSerial != "127.0.0.1:7555" {
		t.Fatalf("target = %+v", got)
	}
	if got.DisplayName != "MuMu" || got.Resolution.W != 1280 || got.Resolution.H != 720 {
		t.Fatalf("target metadata = %+v", got)
	}
}
