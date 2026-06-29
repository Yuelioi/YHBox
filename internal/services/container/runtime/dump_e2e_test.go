package runtime

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"yotta/internal/services/container"
	"yotta/internal/services/execution"
)

// capturedEvent 单条 Emit 记录.
type capturedEvent struct {
	Name string
	Data any
}

// TestDumpE2E_NodeDumpEmittedOnWiredGraph 端到端验证:
// Start → Log[LogEnabled=true] → Stop 这条 WIRED 图跑完整 dispatch,
// 当 LogEnabled 节点真被 reach 并执行后, 必有 container:node-dump emit.
// (用户报的"无 dump"实为 Log 节点不可达, 非 dispatch bug; 此测试锁定可达即 emit.)
func TestDumpE2E_NodeDumpEmittedOnWiredGraph(t *testing.T) {
	var mu sync.Mutex
	var events []capturedEvent
	emit := func(name string, data any) {
		mu.Lock()
		events = append(events, capturedEvent{Name: name, Data: data})
		mu.Unlock()
	}

	c := &container.Container{
		SchemaVersion: 1,
		ID:            "dump_e2e",
		Name:          "dump_e2e",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{
					ID:         "log",
					Kind:       "Log",
					LogEnabled: true,
					Config: map[string]any{
						"literal": map[string]any{
							"Message": "hello",
							"Level":   "info",
						},
					},
				},
				{ID: "stop", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				{From: "start.Done", To: "log.In"},
				{From: "log.Done", To: "stop.In"},
			},
		},
	}

	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, nil /* game */, emit, nil, 0)
	stubRuntimeWindowAndInput(rt)
	r := NewContainerRunner(rt)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := r.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	var dump map[string]any
	for _, e := range events {
		if e.Name != "container:node-dump" {
			continue
		}
		p, ok := e.Data.(map[string]any)
		if !ok {
			t.Fatalf("node-dump payload type = %T, want map[string]any", e.Data)
		}
		if p["nodeId"] == "log" {
			dump = p
			break
		}
	}
	if dump == nil {
		t.Fatalf("expected a container:node-dump for the Log node; got events: %v", events)
	}

	if dump["nodeKind"] != "Log" {
		t.Errorf("node-dump nodeKind = %v, want Log", dump["nodeKind"])
	}
	if dump["isError"] != false {
		t.Errorf("node-dump isError = %v, want false on success", dump["isError"])
	}
	line, _ := dump["line"].(string)
	if line == "" {
		t.Fatal("node-dump line must be non-empty")
	}
	if !strings.Contains(line, "Message=hello") {
		t.Errorf("node-dump line %q should contain Message=hello", line)
	}
}
