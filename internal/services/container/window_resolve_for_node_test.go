package container

import (
	"testing"

	"yotta/internal/automation/target"

	_ "yotta/internal/nodes/all"
)

func TestWin32WindowTargetForNode(t *testing.T) {
	// 构造 Start → WT_A(Title=A) → n1(ClickAt) → WT_B(Title=B) → n2(ClickAt)
	// 用 exec 边 "<id>.<outpin>"→"<id>.In" 串联.
	c := &Container{Graph: Graph{
		Nodes: []GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "wta", Kind: "Win32WindowTarget", Config: map[string]any{"Title": "A"}},
			{ID: "n1", Kind: "ClickAt"},
			{ID: "wtb", Kind: "Win32WindowTarget", Config: map[string]any{"Title": "B"}},
			{ID: "n2", Kind: "ClickAt"},
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
			{ID: "winClick", Kind: "ClickAt"},
			{ID: "at", Kind: "AndroidTarget"},
			{ID: "androidClick", Kind: "ClickAt"},
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
