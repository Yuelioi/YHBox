package runtime

import (
	"context"
	"sync"
	"testing"
	"time"

	"yotta/internal/node"
	"yotta/internal/services/container"
	"yotta/internal/services/execution"

	_ "yotta/internal/nodes/event"    // EventTick
	_ "yotta/internal/nodes/input"    // OnEvent
	_ "yotta/internal/nodes/variable" // SetVar
)

// buildSpawnListener 造一个只含 [eventKind 节点 → SetVar(global)] 的容器, 返回 listener + runner 供断言。
func buildSpawnListener(t *testing.T, eventKind string, setVarName string, value any) (*EventListener, *ContainerRunner) {
	t.Helper()
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "test-seed",
		Name:          "test-seed",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "ev", Kind: eventKind},
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
	l, r := buildSpawnListener(t, "OnEvent", "seeded", float64(1))
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

func TestRunner_SpawnsListenerForEventTick(t *testing.T) {
	c := &container.Container{
		SchemaVersion: 1, ID: "test-ev-run", Name: "test-ev-run",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "ev", Kind: "EventTick", Config: map[string]any{"literal": map[string]any{"IntervalMs": float64(5)}}},
				{ID: "sv", Kind: "SetVar", Config: map[string]any{
					"VarName": "ticked", "Scope": "global",
					"literal": map[string]any{"Value": float64(1)},
				}},
			},
			Edges: []container.GraphEdge{{From: "ev.Out", To: "sv.In"}},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, nil, nil, nil, 0)
	r := NewContainerRunner(rt)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	_ = r.Run(ctx) // 主流程 Start 无下游立即返回; listener 后台 tick; ctx 到点 cancel 退出
	if v, ok := r.bundle.Vars.GetScoped("ticked", "global"); !ok || v != float64(1) {
		t.Fatalf("global ticked = %v,%v, want 1,true — EventTick 没被 spawn listener / tick 没触发", v, ok)
	}
}

// tickProbe 测试专用探针节点: 把 DeltaMs 落到 global 变量 probed_delta。
type tickProbe struct{}

func (tickProbe) Spec() node.Spec {
	return node.Spec{
		Kind:     "TickProbe",
		Category: "Test",
		Inputs: []node.InputSpec{
			{Name: "In", Type: "Exec"},
			{Name: "DeltaMs", Type: "Number"},
		},
		Outputs: []node.OutputSpec{{Name: "Out", Type: "Exec"}},
	}
}

func (tickProbe) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	ctx.Vars().SetScoped("probed_delta", "global", in.Float64("DeltaMs"))
	return ctx.Out("Out").Fire(), nil
}

var registerTickProbeOnce sync.Once

func TestEventListener_TickInjectsDeltaMs(t *testing.T) {
	registerTickProbeOnce.Do(func() { node.Register(&tickProbe{}) })
	c := &container.Container{
		SchemaVersion: 1, ID: "test-delta", Name: "test-delta",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "ev", Kind: "EventTick", Config: map[string]any{"literal": map[string]any{"IntervalMs": float64(10)}}},
				{ID: "probe", Kind: "TickProbe"},
			},
			Edges: []container.GraphEdge{{From: "ev.Out", To: "probe.In"}},
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l.spawn(ctx)
	waitActiveSubsZero(t, l)
	time.Sleep(50 * time.Millisecond)
	l.spawn(ctx)
	waitActiveSubsZero(t, l)
	v, ok := r.bundle.Vars.GetScoped("probed_delta", "global")
	if !ok {
		t.Fatal("probed_delta 未写入 — DeltaMs 没经 ExecData 到下游")
	}
	d, _ := v.(float64)
	if d < 40 || d > 200 {
		t.Errorf("第二次 DeltaMs = %v, want ≈50ms (容差 40-200)", d)
	}
}

func TestEventListener_TickStopsOnCtxCancel(t *testing.T) {
	l, _ := buildSpawnListener(t, "EventTick", "ticked", float64(1))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { l.run(ctx); close(done) }()
	time.Sleep(30 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("listener.run 没在 ctx cancel 后退出 (goroutine 泄漏)")
	}
}
