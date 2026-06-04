package runtime

import (
	"context"
	"testing"
	"time"

	"yotta/internal/services/container"
	"yotta/internal/services/execution"

	_ "yotta/internal/nodes/input"    // OnEvent
	_ "yotta/internal/nodes/variable" // SetVar
)

// buildSpawnListener 造一个只含 [OnEvent 节点 → SetVar(global)] 的容器, 返回 listener + runner 供断言。
func buildSpawnListener(t *testing.T, setVarName string, value any) (*EventListener, *ContainerRunner) {
	t.Helper()
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "test-seed",
		Name:          "test-seed",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "ev", Kind: "OnEvent"},
				{ID: "sv", Kind: "SetVar", Config: map[string]any{
					"VarName": setVarName,
					"Scope":   "global",
					"literal": map[string]any{"Value": value},
				}},
			},
			Edges: []container.GraphEdge{
				{From: "ev.Out", To: "sv.In"}, // 真实边: 大写 .Out (跟前端一致)
			},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, nil, nil, nil, 0)
	r := NewContainerRunner(rt)
	var evNode *container.GraphNode
	for i := range r.rt.Container.Graph.Nodes {
		if r.rt.Container.Graph.Nodes[i].ID == "ev" {
			evNode = &r.rt.Container.Graph.Nodes[i]
		}
	}
	l := newEventListener(r, evNode)
	return l, r
}

func TestEventListener_SeedReachesDownstream_OnEvent(t *testing.T) {
	l, r := buildSpawnListener(t, "seeded", float64(1))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l.spawn(ctx)
	waitActiveSubsZero(t, l)

	if v, ok := r.bundle.Vars.GetScoped("seeded", "global"); !ok || v != float64(1) {
		t.Fatalf("global seeded = %v,%v, want 1,true — listener 没 seed 到下游 (.out/.Out 大小写 bug)", v, ok)
	}
}

// waitActiveSubsZero 轮询等 spawn 的 goroutine 跑完 (activeSubs 归零), 最多 2s。
func waitActiveSubsZero(t *testing.T, l *EventListener) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if l.activeSubs.Load() == 0 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("spawn goroutine 没在 2s 内完成 (activeSubs 未归零)")
}
