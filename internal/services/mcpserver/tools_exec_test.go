package mcpserver

import (
	"context"
	"testing"

	// 注册节点
	_ "yotta/internal/nodes/collection"
	_ "yotta/internal/nodes/control"
	_ "yotta/internal/nodes/detect"
	_ "yotta/internal/nodes/event"
	_ "yotta/internal/nodes/image"
	_ "yotta/internal/nodes/input"
	_ "yotta/internal/nodes/io"
	_ "yotta/internal/nodes/purefunc"
	_ "yotta/internal/nodes/random"
	_ "yotta/internal/nodes/stopwatch"
	_ "yotta/internal/nodes/system"
	_ "yotta/internal/nodes/variable"
)

func TestRunNode_NotArmed_Rejects(t *testing.T) {
	s := NewServer(Deps{Armed: func() bool { return false }, Busy: func() bool { return false }})
	res, _ := s.runNode(context.Background(), "ClickAt", map[string]any{"X": 1, "Y": 1}, 0)
	if res.Ok || res.Error == nil || res.Error.Code != "NOT_ARMED" {
		t.Fatalf("未武装应返 NOT_ARMED, got %+v", res)
	}
}

func TestRunNode_Busy_Rejects(t *testing.T) {
	s := NewServer(Deps{Armed: func() bool { return true }, Busy: func() bool { return true }})
	res, _ := s.runNode(context.Background(), "ClickAt", map[string]any{"X": 1, "Y": 1}, 0)
	if res.Error == nil || res.Error.Code != "BUSY" {
		t.Fatalf("GUI 在跑应返 BUSY, got %+v", res)
	}
}

func TestRunNode_UnrunnableKind_Rejects(t *testing.T) {
	s := NewServer(Deps{Armed: func() bool { return true }, Busy: func() bool { return false }})
	res, _ := s.runNode(context.Background(), "Loop", nil, 0)
	if res.Error == nil || res.Error.Code != "UNRUNNABLE_KIND" {
		t.Fatalf("Loop 应返 UNRUNNABLE_KIND, got %+v", res)
	}
}
