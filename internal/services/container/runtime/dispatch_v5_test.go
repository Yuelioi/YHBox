package runtime

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"yhbox/internal/node"
	_ "yhbox/internal/nodes/control" // Loop / Break / Continue / Start / Stop / If / Switch / Sleep
	_ "yhbox/internal/nodes/system"  // Subgraph / SubgraphInput / SubgraphOutput / Try / Throw 等
	"yhbox/internal/services/container"
	"yhbox/internal/services/execution"
)

// ============================================================================
// Test-local fake nodes — 各覆盖一条 dispatch 路径.
// 在 init() 注册. 跟 production 节点 kind 不会冲突 (test_xxx 前缀).
// ============================================================================

const (
	tkHappy        = "test_dispatch_happy"
	tkValidation   = "test_dispatch_validation"
	tkError        = "test_dispatch_error"
	tkPanic        = "test_dispatch_panic"
	tkDisplay      = "test_dispatch_display"
	tkNoExit       = "test_dispatch_noexit"
	tkHappyCounted = "test_dispatch_happy_counted"
)

// tdHappyCounter — 测试 region body 调用次数. 每个 tdHappyCounted.Run 增 1.
// 测试前 ResetTdHappyCounter 清零. tests run serially (default go test).
var tdHappyCounter atomic.Int64

func resetTdHappyCounter() { tdHappyCounter.Store(0) }

type tdHappyCounted struct{}

func (tdHappyCounted) Spec() node.Spec {
	return node.Spec{
		Kind:    tkHappyCounted,
		Inputs:  []node.InputSpec{{Name: "in", Type: "Exec"}},
		Outputs: []node.OutputSpec{{Name: "Out", Type: "Exec"}},
	}
}
func (tdHappyCounted) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	tdHappyCounter.Add(1)
	return ctx.Out("Out").Fire(), nil
}

type tdHappy struct{}

func (tdHappy) Spec() node.Spec {
	return node.Spec{
		Kind: tkHappy,
		Inputs: []node.InputSpec{
			{Name: "in", Type: "Exec"},
		},
		Outputs: []node.OutputSpec{
			{Name: "Out", Type: "Exec"},
		},
	}
}
func (tdHappy) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	return ctx.Out("Out").Fire(), nil
}

type tdValidation struct{}

func (tdValidation) Spec() node.Spec {
	return node.Spec{
		Kind: tkValidation,
		Inputs: []node.InputSpec{
			{Name: "in", Type: "Exec"},
			{Name: "Required", Type: "String", Required: true},
		},
		Outputs: []node.OutputSpec{{Name: "Out", Type: "Exec"}},
	}
}
func (tdValidation) Run(node.Ctx, node.Inputs) (node.Outputs, error) {
	panic("Run should not be called when Required is missing — validation must fail first")
}

type tdError struct{}

func (tdError) Spec() node.Spec {
	return node.Spec{
		Kind:    tkError,
		Inputs:  []node.InputSpec{{Name: "in", Type: "Exec"}},
		Outputs: []node.OutputSpec{{Name: "Out", Type: "Exec"}},
	}
}
func (tdError) Run(node.Ctx, node.Inputs) (node.Outputs, error) {
	return nil, errors.New("deliberate test error")
}

type tdPanic struct{}

func (tdPanic) Spec() node.Spec {
	return node.Spec{
		Kind:    tkPanic,
		Inputs:  []node.InputSpec{{Name: "in", Type: "Exec"}},
		Outputs: []node.OutputSpec{{Name: "Out", Type: "Exec"}},
	}
}
func (tdPanic) Run(node.Ctx, node.Inputs) (node.Outputs, error) {
	panic("deliberate test panic")
}

type tdDisplay struct{}

func (tdDisplay) Spec() node.Spec {
	return node.Spec{
		Kind:    tkDisplay,
		Inputs:  []node.InputSpec{{Name: "in", Type: "Exec"}},
		Outputs: []node.OutputSpec{{Name: "Out", Type: "Exec"}},
	}
}
func (tdDisplay) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	return ctx.Out("Out").Fire(), nil
}

// implements Displayer
func (tdDisplay) Display(in node.Inputs, exitName string, out node.OutputData) string {
	return "display-text-for-" + exitName
}

// tdNoExit Run 返 nil Outputs (出口为空) — 验证 routeResult 处理 empty ExitName.
type tdNoExit struct{}

func (tdNoExit) Spec() node.Spec {
	return node.Spec{
		Kind:    tkNoExit,
		Inputs:  []node.InputSpec{{Name: "in", Type: "Exec"}},
		Outputs: []node.OutputSpec{{Name: "Out", Type: "Exec"}},
	}
}
func (tdNoExit) Run(node.Ctx, node.Inputs) (node.Outputs, error) {
	return nil, nil // no Fire → ExitName == ""
}

func init() {
	node.Register(&tdHappy{})
	node.Register(&tdValidation{})
	node.Register(&tdError{})
	node.Register(&tdPanic{})
	node.Register(&tdDisplay{})
	node.Register(&tdNoExit{})
	node.Register(&tdHappyCounted{})
}

// ============================================================================
// 测试 helpers
// ============================================================================

// dispatchTestCtx 单次 dispatch 测试的全套 fixture.
// emitted 是 r.rt.Emit 收集到的 (name, data) 序列, mutex 保护.
type dispatchTestCtx struct {
	rt      *RuntimeContext
	r       *ContainerRunner
	node    *container.GraphNode
	emitted []emittedEvent
	emitMu  sync.Mutex
}

type emittedEvent struct {
	Name string
	Data any
}

// newDispatchTest 建一个含 (testNode + downstream "target") 2 节点 minimal container,
// edge testNode.Out → target.in. 返 dispatchTestCtx 含构造好的 runner + node 指针.
func newDispatchTest(t *testing.T, testNodeKind string) *dispatchTestCtx {
	t.Helper()
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "test-dispatch",
		Name:          "test-dispatch",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "n1", Kind: testNodeKind},
				{ID: "target", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				{From: "n1.Out", To: "target.in"},
			},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopColorDetector{}, nil, nil, nil, nil, 0)
	r := NewContainerRunner(rt)
	dt := &dispatchTestCtx{rt: rt, r: r}
	dt.node = r.nodesByID["n1"]
	rt.Emit = func(name string, data any) {
		dt.emitMu.Lock()
		defer dt.emitMu.Unlock()
		dt.emitted = append(dt.emitted, emittedEvent{Name: name, Data: data})
	}
	return dt
}

func (dt *dispatchTestCtx) eventsByName(name string) []emittedEvent {
	dt.emitMu.Lock()
	defer dt.emitMu.Unlock()
	var out []emittedEvent
	for _, e := range dt.emitted {
		if e.Name == name {
			out = append(out, e)
		}
	}
	return out
}

// ============================================================================
// 5 路径测试
// ============================================================================

func TestExecNodeViaFramework_Happy(t *testing.T) {
	dt := newDispatchTest(t, tkHappy)
	tokens, err := dt.r.execNodeViaFramework(context.Background(), dt.node, ExecToken{NodeID: "n1", InPin: "in"})
	if err != nil {
		t.Fatalf("execNodeViaFramework error: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("got %d tokens, want 1", len(tokens))
	}
	if tokens[0].NodeID != "target" || tokens[0].InPin != "in" {
		t.Errorf("token = %+v, want {target, in}", tokens[0])
	}
}

func TestExecNodeViaFramework_Validation(t *testing.T) {
	dt := newDispatchTest(t, tkValidation)
	_, err := dt.r.execNodeViaFramework(context.Background(), dt.node, ExecToken{NodeID: "n1", InPin: "in"})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !strings.Contains(err.Error(), "validation") {
		t.Errorf("error %q should mention validation", err)
	}
	events := dt.eventsByName("container:node-validation")
	if len(events) != 1 {
		t.Fatalf("got %d node-validation events, want 1", len(events))
	}
	payload := events[0].Data.(map[string]any)
	if payload["nodeId"] != "n1" {
		t.Errorf("event nodeId = %v, want n1", payload["nodeId"])
	}
}

func TestExecNodeViaFramework_Error(t *testing.T) {
	dt := newDispatchTest(t, tkError)
	_, err := dt.r.execNodeViaFramework(context.Background(), dt.node, ExecToken{NodeID: "n1", InPin: "in"})
	if err == nil {
		t.Fatal("expected runtime error, got nil")
	}
	if !strings.Contains(err.Error(), "deliberate test error") {
		t.Errorf("error %q should propagate Run error", err)
	}
	// Error 路径不 emit node-validation / node-panic / node-log
	if got := len(dt.eventsByName("container:node-validation")); got != 0 {
		t.Errorf("Error path should not emit node-validation, got %d", got)
	}
	if got := len(dt.eventsByName("container:node-panic")); got != 0 {
		t.Errorf("Error path should not emit node-panic, got %d", got)
	}
}

func TestExecNodeViaFramework_Panic(t *testing.T) {
	dt := newDispatchTest(t, tkPanic)
	_, err := dt.r.execNodeViaFramework(context.Background(), dt.node, ExecToken{NodeID: "n1", InPin: "in"})
	if err == nil {
		t.Fatal("expected panic-wrapped error, got nil")
	}
	if !strings.Contains(err.Error(), "panic") {
		t.Errorf("error %q should mention panic", err)
	}
	events := dt.eventsByName("container:node-panic")
	if len(events) != 1 {
		t.Fatalf("got %d node-panic events, want 1", len(events))
	}
	payload := events[0].Data.(map[string]any)
	if !strings.Contains(payload["panic"].(string), "deliberate test panic") {
		t.Errorf("panic payload %q should contain panic message", payload["panic"])
	}
	stack := payload["stack"].(string)
	if !strings.Contains(stack, "runtime.") {
		t.Errorf("stack %q should contain runtime frames", stack)
	}
}

func TestExecNodeViaFramework_Display(t *testing.T) {
	dt := newDispatchTest(t, tkDisplay)
	tokens, err := dt.r.execNodeViaFramework(context.Background(), dt.node, ExecToken{NodeID: "n1", InPin: "in"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("got %d tokens, want 1", len(tokens))
	}
	events := dt.eventsByName("container:node-log")
	if len(events) != 1 {
		t.Fatalf("got %d node-log events, want 1", len(events))
	}
	payload := events[0].Data.(map[string]any)
	if payload["message"] != "display-text-for-Out" {
		t.Errorf("log message = %v, want display-text-for-Out", payload["message"])
	}
}

func TestExecNodeViaFramework_NoExit(t *testing.T) {
	dt := newDispatchTest(t, tkNoExit)
	tokens, err := dt.r.execNodeViaFramework(context.Background(), dt.node, ExecToken{NodeID: "n1", InPin: "in"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens != nil {
		t.Errorf("expected nil tokens for no-exit run, got %+v", tokens)
	}
}

func TestExecNodeViaFramework_UnknownKind(t *testing.T) {
	dt := newDispatchTest(t, tkHappy)
	// 改 node.Kind 模拟 registry miss
	dt.node.Kind = "no_such_kind_registered"
	_, err := dt.r.execNodeViaFramework(context.Background(), dt.node, ExecToken{NodeID: "n1", InPin: "in"})
	if err == nil {
		t.Fatal("expected error for unknown kind, got nil")
	}
	if !strings.Contains(err.Error(), "not registered") {
		t.Errorf("error %q should say not registered", err)
	}
}

// ============================================================================
// routeResult 单元测试 (不走 execNodeViaFramework)
// ============================================================================

func TestRouteResult_ExitNameWithLoopStack(t *testing.T) {
	dt := newDispatchTest(t, tkHappy)
	frame := &LoopFrame{LoopNodeID: "loop1", Iter: 3}
	tok := ExecToken{NodeID: "n1", InPin: "in", LoopStack: []*LoopFrame{frame}}
	result := node.RunResult{ExitName: "Out"}
	tokens, err := dt.r.routeResult(dt.node, tok, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("got %d tokens, want 1", len(tokens))
	}
	if len(tokens[0].LoopStack) != 1 || tokens[0].LoopStack[0] != frame {
		t.Errorf("LoopStack not propagated to downstream token")
	}
}

func TestBuildConfigFor_StripsLiteral(t *testing.T) {
	dt := newDispatchTest(t, tkHappy)
	dt.node.Config = map[string]any{
		"Threshold": 0.5,
		"literal":   map[string]any{"x": 1},
	}
	cfg := dt.r.buildConfigFor(dt.node)
	if _, ok := cfg["literal"]; ok {
		t.Errorf("literal should be stripped from config map")
	}
	if cfg["Threshold"] != 0.5 {
		t.Errorf("Threshold not in config map")
	}
}

// ============================================================================
// Phase 5.5b — Region runner 测试 (Loop + Subgraph)
// ============================================================================

// newRegionTest 建一个含 Loop 节点的测试 container.
// nodes: { loop (Loop), body_n (test_dispatch_happy_counted), done_n (Stop) }
// edges: loop.body→body_n.in, loop.done→done_n.in
// body_n.Out 默认无下游, body 一轮自然结束.
// loopConfig 透传到 loop.Config (mode/count 等).
func newRegionTestLoop(t *testing.T, loopConfig map[string]any) *dispatchTestCtx {
	t.Helper()
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "test-region-loop",
		Name:          "test-region-loop",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "loop", Kind: "Loop", Config: loopConfig},
				{ID: "body_n", Kind: tkHappyCounted},
				{ID: "done_n", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				{From: "loop.body", To: "body_n.in"},
				{From: "loop.done", To: "done_n.in"},
			},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopColorDetector{}, nil, nil, nil, nil, 0)
	r := NewContainerRunner(rt)
	dt := &dispatchTestCtx{rt: rt, r: r}
	dt.node = r.nodesByID["loop"]
	rt.Emit = func(name string, data any) {
		dt.emitMu.Lock()
		defer dt.emitMu.Unlock()
		dt.emitted = append(dt.emitted, emittedEvent{Name: name, Data: data})
	}
	return dt
}

func TestExecNodeAsRegionViaFramework_LoopCount3(t *testing.T) {
	resetTdHappyCounter()
	dt := newRegionTestLoop(t, map[string]any{"mode": "count", "count": 3})
	tokens, err := dt.r.execNodeAsRegionViaFramework(context.Background(), dt.node, ExecToken{NodeID: "loop", InPin: "in"})
	if err != nil {
		t.Fatalf("loop region dispatch: %v", err)
	}
	if got := tdHappyCounter.Load(); got != 3 {
		t.Errorf("body called %d times, want 3", got)
	}
	if len(tokens) != 1 || tokens[0].NodeID != "done_n" || tokens[0].InPin != "in" {
		t.Errorf("loop done token = %+v, want {done_n, in}", tokens)
	}
}

func TestExecNodeAsRegionViaFramework_LoopCount0(t *testing.T) {
	resetTdHappyCounter()
	dt := newRegionTestLoop(t, map[string]any{"mode": "count", "count": 0})
	tokens, err := dt.r.execNodeAsRegionViaFramework(context.Background(), dt.node, ExecToken{NodeID: "loop", InPin: "in"})
	if err != nil {
		t.Fatalf("loop count=0: %v", err)
	}
	if got := tdHappyCounter.Load(); got != 0 {
		t.Errorf("count=0 should not call body, got %d calls", got)
	}
	if len(tokens) != 1 || tokens[0].NodeID != "done_n" {
		t.Errorf("loop count=0 done = %+v, want {done_n, in}", tokens)
	}
}

func TestExecNodeAsRegionViaFramework_LoopForeverWithBreak(t *testing.T) {
	resetTdHappyCounter()
	// 拼 Loop forever + body=counter→Break: counter run 一次, Break sentinel 退 Loop.
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "test-loop-break",
		Name:          "test-loop-break",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "loop", Kind: "Loop", Config: map[string]any{"mode": "forever"}},
				{ID: "body_n", Kind: tkHappyCounted},
				{ID: "break_n", Kind: "Break"},
				{ID: "done_n", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				{From: "loop.body", To: "body_n.in"},
				{From: "body_n.Out", To: "break_n.in"},
				{From: "loop.done", To: "done_n.in"},
			},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopColorDetector{}, nil, nil, nil, nil, 0)
	r := NewContainerRunner(rt)
	loopNode := r.nodesByID["loop"]
	tokens, err := r.execNodeAsRegionViaFramework(context.Background(), loopNode, ExecToken{NodeID: "loop", InPin: "in"})
	if err != nil {
		t.Fatalf("forever + break: %v", err)
	}
	if got := tdHappyCounter.Load(); got != 1 {
		t.Errorf("body called %d times, want 1 (Break should exit after first iteration)", got)
	}
	if len(tokens) != 1 || tokens[0].NodeID != "done_n" {
		t.Errorf("done = %+v, want {done_n, in}", tokens)
	}
}

func TestExecNodeAsRegionViaFramework_NonRegionNodeError(t *testing.T) {
	// 用普通 (非 RegionRunner) 节点调 execNodeAsRegionViaFramework, 应返 error.
	dt := newDispatchTest(t, tkHappy) // tkHappy 不是 RegionRunner
	_, err := dt.r.execNodeAsRegionViaFramework(context.Background(), dt.node, ExecToken{NodeID: "n1", InPin: "in"})
	if err == nil {
		t.Fatal("expected error for non-RegionRunner node")
	}
	if !strings.Contains(err.Error(), "not a RegionRunner") {
		t.Errorf("error %q should mention 'not a RegionRunner'", err)
	}
}

// ============================================================================
// Subgraph 测试
// ============================================================================

func TestExecNodeAsRegionViaFramework_SubgraphBasic(t *testing.T) {
	resetTdHappyCounter()
	// Main container: { sg_call (Subgraph w/ SubgraphID=sub1), done_n }
	// Subgraph "sub1": { sub_in (SubgraphInput), sub_node (counted), sub_out (SubgraphOutput) }
	subgraph := container.Subgraph{
		ID:    "sub1",
		Label: "sub1",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "sub_in", Kind: "SubgraphInput"},
				{ID: "sub_node", Kind: tkHappyCounted},
				{ID: "sub_out", Kind: "SubgraphOutput"},
			},
			Edges: []container.GraphEdge{
				{From: "sub_in.out", To: "sub_node.in"},
				{From: "sub_node.Out", To: "sub_out.in"},
			},
		},
	}
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "test-subgraph",
		Name:          "test-subgraph",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "sg_call", Kind: "Subgraph", Config: map[string]any{"SubgraphID": "sub1"}},
				{ID: "done_n", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				{From: "sg_call.Done", To: "done_n.in"},
			},
		},
		Subgraphs: []container.Subgraph{subgraph},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopColorDetector{}, nil, nil, nil, nil, 0)
	r := NewContainerRunner(rt)
	sgNode := r.nodesByID["sg_call"]
	tokens, err := r.execNodeAsRegionViaFramework(context.Background(), sgNode, ExecToken{NodeID: "sg_call", InPin: "in"})
	if err != nil {
		t.Fatalf("subgraph dispatch: %v", err)
	}
	if got := tdHappyCounter.Load(); got != 1 {
		t.Errorf("sub_node called %d times, want 1", got)
	}
	if len(tokens) != 1 || tokens[0].NodeID != "done_n" {
		t.Errorf("subgraph Done = %+v, want {done_n, in}", tokens)
	}
	// 验证 PopFrame: dispatch 完, current frame 应该回到 main (Parent == nil).
	if r.state.CurrentFrame == nil || r.state.CurrentFrame.Parent != nil {
		t.Errorf("PopFrame did not restore main frame: %+v", r.state.CurrentFrame)
	}
	// 验证 edges restored to main (not subgraph).
	// main edges 有 sg_call.Done → done_n.in. r.edges.out 应该包含这条.
	if _, ok := r.edges.out["sg_call.Done"]; !ok {
		t.Errorf("r.edges not restored to main graph after Subgraph body returned")
	}
}

func TestExecNodeAsRegionViaFramework_SubgraphUnknownID(t *testing.T) {
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "test-subgraph-missing",
		Name:          "test-subgraph-missing",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "sg_call", Kind: "Subgraph", Config: map[string]any{"SubgraphID": "no_such_sg"}},
			},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopColorDetector{}, nil, nil, nil, nil, 0)
	r := NewContainerRunner(rt)
	sgNode := r.nodesByID["sg_call"]
	_, err := r.execNodeAsRegionViaFramework(context.Background(), sgNode, ExecToken{NodeID: "sg_call", InPin: "in"})
	if err == nil {
		t.Fatal("expected error for missing subgraph")
	}
	if !strings.Contains(err.Error(), "no_such_sg") {
		t.Errorf("error %q should mention missing subgraph ID", err)
	}
}

// ============================================================================
// dispatchInRegion 统一路由
// ============================================================================

func TestDispatchInRegion_RoutesNormalToRunNode(t *testing.T) {
	dt := newDispatchTest(t, tkHappy)
	tokens, err := dt.r.dispatchInRegion(context.Background(), dt.node, ExecToken{NodeID: "n1", InPin: "in"})
	if err != nil {
		t.Fatalf("dispatchInRegion happy: %v", err)
	}
	if len(tokens) != 1 || tokens[0].NodeID != "target" {
		t.Errorf("token = %+v, want target", tokens)
	}
}

func TestDispatchInRegion_RoutesRegionToRunNodeAsRegion(t *testing.T) {
	resetTdHappyCounter()
	dt := newRegionTestLoop(t, map[string]any{"mode": "count", "count": 2})
	tokens, err := dt.r.dispatchInRegion(context.Background(), dt.node, ExecToken{NodeID: "loop", InPin: "in"})
	if err != nil {
		t.Fatalf("dispatchInRegion loop: %v", err)
	}
	if got := tdHappyCounter.Load(); got != 2 {
		t.Errorf("body called %d times, want 2", got)
	}
	if len(tokens) != 1 || tokens[0].NodeID != "done_n" {
		t.Errorf("tokens = %+v, want {done_n}", tokens)
	}
}
