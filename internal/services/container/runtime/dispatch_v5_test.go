package runtime

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"yhbox/internal/node"
	_ "yhbox/internal/nodes/control"   // Loop / Break / Continue / Start / Stop / If / Switch / Sleep
	_ "yhbox/internal/nodes/detect"    // CheckTemplate / WaitTemplate / ClickTemplate / DetectColor* / ColorBarTrack / Screenshot
	_ "yhbox/internal/nodes/input"     // KeyPress / ClickAt / MouseMove / Scroll / KeyHold* / MouseHold* / BringGameForeground / OnEvent
	_ "yhbox/internal/nodes/io"        // Log / Toast / PlayClip
	_ "yhbox/internal/nodes/purefunc"  // Add / Sub / .../Select / Expr
	_ "yhbox/internal/nodes/stopwatch" // StopwatchStart / Stop / Read
	_ "yhbox/internal/nodes/system"    // Subgraph / SubgraphInput / SubgraphOutput / Try / Throw / WindowTarget / MouseCalibration / CommentBox / CollapsedNode
	_ "yhbox/internal/nodes/variable"  // SetVar / IncVar / GetVar / GetParam / GetSys
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

// tdSource — Run 时通过 OutputData.Set("Path", "/foo").Set("Count", 42) 推 exec-data.
// 用于验 OutputData carry plumb 到下游 ExecToken.
const tkSource = "test_dispatch_source"

type tdSource struct{}

func (tdSource) Spec() node.Spec {
	return node.Spec{
		Kind:   tkSource,
		Inputs: []node.InputSpec{{Name: "in", Type: "Exec"}},
		Outputs: []node.OutputSpec{
			{Name: "Out", Type: "Exec", Data: []node.DataField{
				{Name: "Path", Type: "String"},
				{Name: "Count", Type: "Number"},
			}},
		},
	}
}
func (tdSource) Run(ctx node.Ctx, _ node.Inputs) (node.Outputs, error) {
	return ctx.Out("Out").Set("Path", "/foo").Set("Count", 42).Fire(), nil
}

// tdSink — 把 Inputs 字段照 Raw 暴露到 var 供测试断言. Spec.Inputs 只有 "in" Exec,
// 但 inputsImpl.merged 含 exec-data 携带的 "Path" / "Count", 通过 in.Raw 可见.
const tkSink = "test_dispatch_sink"

var tdSinkRecorded struct {
	Path  string
	Count int
	Saw   bool
}

func resetTdSinkRecorded() {
	tdSinkRecorded.Path = ""
	tdSinkRecorded.Count = 0
	tdSinkRecorded.Saw = false
}

type tdSink struct{}

func (tdSink) Spec() node.Spec {
	return node.Spec{
		Kind:    tkSink,
		Inputs:  []node.InputSpec{{Name: "in", Type: "Exec"}},
		Outputs: []node.OutputSpec{{Name: "Out", Type: "Exec"}},
	}
}
func (tdSink) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	tdSinkRecorded.Path = in.String("Path")
	tdSinkRecorded.Count = in.Int("Count")
	tdSinkRecorded.Saw = true
	return ctx.Out("Out").Fire(), nil
}

func init() {
	node.Register(&tdHappy{})
	node.Register(&tdValidation{})
	node.Register(&tdError{})
	node.Register(&tdPanic{})
	node.Register(&tdDisplay{})
	node.Register(&tdNoExit{})
	node.Register(&tdHappyCounted{})
	node.Register(&tdSource{})
	node.Register(&tdSink{})
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
				{From: "n1.Out", To: "target.In"},
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
	if tokens[0].NodeID != "target" || tokens[0].InPin != "In" {
		t.Errorf("token = %+v, want {target, In}", tokens[0])
	}
}

// TestExecNodeViaFramework_OutputDataCarry:
// 源节点 ctx.Out("Out").Set("Path","/foo").Set("Count", 42).Fire() →
// routeResult 通过 edges.nextWithData 把 OutputData 挂到下游 ExecToken.ExecData →
// 下游节点 in.String("Path") / in.Int("Count") 拿到值.
func TestExecNodeViaFramework_OutputDataCarry(t *testing.T) {
	resetTdSinkRecorded()
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "test-dispatch-carry",
		Name:          "test-dispatch-carry",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "src", Kind: tkSource},
				{ID: "snk", Kind: tkSink},
			},
			Edges: []container.GraphEdge{
				{From: "src.Out", To: "snk.In"},
			},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopColorDetector{}, nil, nil, nil, nil, 0)
	r := NewContainerRunner(rt)
	srcNode := r.nodesByID["src"]
	tokens, err := r.execNodeViaFramework(context.Background(), srcNode, ExecToken{NodeID: "src", InPin: "in"})
	if err != nil {
		t.Fatalf("source dispatch error: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("got %d tokens, want 1", len(tokens))
	}
	tok := tokens[0]
	if tok.ExecData == nil {
		t.Fatal("downstream token has nil ExecData — OutputData carry not plumbed")
	}
	if tok.ExecData["Path"] != "/foo" {
		t.Errorf("ExecData[Path] = %v, want /foo", tok.ExecData["Path"])
	}
	// Set 接受 int 直传, 走到 framework merged map. inputsImpl.Int 处理 int → 42.
	if tok.ExecData["Count"] != 42 {
		t.Errorf("ExecData[Count] = %v, want 42", tok.ExecData["Count"])
	}
	// 真跑 sink 节点验 in.X 读取
	sinkNode := r.nodesByID["snk"]
	_, err = r.execNodeViaFramework(context.Background(), sinkNode, tok)
	if err != nil {
		t.Fatalf("sink dispatch error: %v", err)
	}
	if !tdSinkRecorded.Saw {
		t.Fatal("sink Run not called")
	}
	if tdSinkRecorded.Path != "/foo" {
		t.Errorf("sink saw Path = %q, want /foo", tdSinkRecorded.Path)
	}
	if tdSinkRecorded.Count != 42 {
		t.Errorf("sink saw Count = %d, want 42", tdSinkRecorded.Count)
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
	// Error 路径 emit node-error 让 GUI 高亮失败节点.
	events := dt.eventsByName("container:node-error")
	if len(events) != 1 {
		t.Fatalf("got %d node-error events, want 1", len(events))
	}
	payload := events[0].Data.(map[string]any)
	if payload["nodeId"] != "n1" {
		t.Errorf("node-error nodeId got %v, want n1", payload["nodeId"])
	}
	if msg, _ := payload["message"].(string); !strings.Contains(msg, "deliberate test error") {
		t.Errorf("node-error message %q should contain underlying error", msg)
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
// Region runner 测试 (Loop + Subgraph)
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
				{From: "loop.Body", To: "body_n.In"},
				{From: "loop.Done", To: "done_n.In"},
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
	dt := newRegionTestLoop(t, map[string]any{"Mode": "count", "Count": 3})
	tokens, err := dt.r.execNodeAsRegionViaFramework(context.Background(), dt.node, ExecToken{NodeID: "loop", InPin: "in"})
	if err != nil {
		t.Fatalf("loop region dispatch: %v", err)
	}
	if got := tdHappyCounter.Load(); got != 3 {
		t.Errorf("body called %d times, want 3", got)
	}
	if len(tokens) != 1 || tokens[0].NodeID != "done_n" || tokens[0].InPin != "In" {
		t.Errorf("loop done token = %+v, want {done_n, In}", tokens)
	}
}

func TestExecNodeAsRegionViaFramework_LoopCount0(t *testing.T) {
	resetTdHappyCounter()
	dt := newRegionTestLoop(t, map[string]any{"Mode": "count", "Count": 0})
	tokens, err := dt.r.execNodeAsRegionViaFramework(context.Background(), dt.node, ExecToken{NodeID: "loop", InPin: "in"})
	if err != nil {
		t.Fatalf("loop count=0: %v", err)
	}
	if got := tdHappyCounter.Load(); got != 0 {
		t.Errorf("count=0 should not call body, got %d calls", got)
	}
	if len(tokens) != 1 || tokens[0].NodeID != "done_n" {
		t.Errorf("loop count=0 done = %+v, want {done_n, In}", tokens)
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
				{ID: "loop", Kind: "Loop", Config: map[string]any{"Mode": "forever"}},
				{ID: "body_n", Kind: tkHappyCounted},
				{ID: "break_n", Kind: "Break"},
				{ID: "done_n", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				{From: "loop.Body", To: "body_n.In"},
				{From: "body_n.Out", To: "break_n.In"},
				{From: "loop.Done", To: "done_n.In"},
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
		t.Errorf("done = %+v, want {done_n, In}", tokens)
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
				{From: "sub_in.Done", To: "sub_node.In"},
				{From: "sub_node.Out", To: "sub_out.In"},
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
				{From: "sg_call.Done", To: "done_n.In"},
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
		t.Errorf("subgraph Done = %+v, want {done_n, In}", tokens)
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
	dt := newRegionTestLoop(t, map[string]any{"Mode": "count", "Count": 2})
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

// ============================================================================
// Try 测试
// ============================================================================

// newTryTest 建主图 { try_n (Try w/ SubgraphID=trybody), out_n (Stop), catch_n (Stop) } +
// 子图 trybody 含 sub_in / inner_n (节点 Kind 由 caller 指定) / sub_out.
// 主图边: try_n.out → out_n.in, try_n.catch → catch_n.in.
// 子图边: sub_in.Done → inner_n.in, inner_n.<outPin> → sub_out.in (outPin caller 指定).
func newTryTest(t *testing.T, innerKind, innerOutPin string) *dispatchTestCtx {
	t.Helper()
	subgraph := container.Subgraph{
		ID:    "trybody",
		Label: "trybody",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "sub_in", Kind: "SubgraphInput"},
				{ID: "inner_n", Kind: innerKind},
				{ID: "sub_out", Kind: "SubgraphOutput"},
			},
			Edges: []container.GraphEdge{
				{From: "sub_in.Done", To: "inner_n.In"},
				{From: "inner_n." + innerOutPin, To: "sub_out.In"},
			},
		},
	}
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "test-try",
		Name:          "test-try",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "try_n", Kind: "Try", Config: map[string]any{"SubgraphID": "trybody"}},
				{ID: "out_n", Kind: "Stop"},
				{ID: "catch_n", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				{From: "try_n.Out", To: "out_n.In"},
				{From: "try_n.Catch", To: "catch_n.In"},
			},
		},
		Subgraphs: []container.Subgraph{subgraph},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopColorDetector{}, nil, nil, nil, nil, 0)
	r := NewContainerRunner(rt)
	dt := &dispatchTestCtx{rt: rt, r: r}
	dt.node = r.nodesByID["try_n"]
	rt.Emit = func(name string, data any) {
		dt.emitMu.Lock()
		defer dt.emitMu.Unlock()
		dt.emitted = append(dt.emitted, emittedEvent{Name: name, Data: data})
	}
	return dt
}

func TestExecNodeAsRegionViaFramework_TryDone(t *testing.T) {
	resetTdHappyCounter()
	// body 跑通 (tdHappyCounted Fire("Out") 然后 SubgraphOutput 终点) → out 出口 → out_n.in
	dt := newTryTest(t, tkHappyCounted, "Out")
	tokens, err := dt.r.execNodeAsRegionViaFramework(context.Background(), dt.node, ExecToken{NodeID: "try_n", InPin: "in"})
	if err != nil {
		t.Fatalf("try done: %v", err)
	}
	if got := tdHappyCounter.Load(); got != 1 {
		t.Errorf("inner called %d times, want 1", got)
	}
	if len(tokens) != 1 || tokens[0].NodeID != "out_n" {
		t.Errorf("try done token = %+v, want {out_n, In}", tokens)
	}
}

func TestExecNodeAsRegionViaFramework_TryCatch(t *testing.T) {
	// body 节点 tkError 返 error "deliberate test error" → catch 出口 → catch_n.in
	dt := newTryTest(t, tkError, "Out") // tkError Run 返 error 前不会 Fire, outPin 不相关
	tokens, err := dt.r.execNodeAsRegionViaFramework(context.Background(), dt.node, ExecToken{NodeID: "try_n", InPin: "in"})
	if err != nil {
		t.Fatalf("try catch: %v", err)
	}
	if len(tokens) != 1 || tokens[0].NodeID != "catch_n" {
		t.Errorf("try catch token = %+v, want {catch_n, In}", tokens)
	}
}

// tdThrow 在 body 里 panic-style 抛 Throw-like error. 复用 nodes/system 的 ThrowError.
// (ThrowError 是 exported type, Try.RunRegion 不区别处理 — 跟普通 error 同样走 catch.)
//
// 实际上 nodes/system 已经有 Throw 节点 — 直接用它. 它读 config["Message"] 触发 ThrowError.
// 见 nodes/system/throw.go.

func TestExecNodeAsRegionViaFramework_TryThrow(t *testing.T) {
	// body 用 Throw 节点 (config["Message"] = "fish escaped") → Try.RunRegion 截 → catch
	subgraph := container.Subgraph{
		ID: "trybody_throw",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "sub_in", Kind: "SubgraphInput"},
				{ID: "throw_n", Kind: "Throw", Config: map[string]any{"message": "fish escaped"}},
				{ID: "sub_out", Kind: "SubgraphOutput"},
			},
			Edges: []container.GraphEdge{
				{From: "sub_in.Done", To: "throw_n.In"},
				// throw_n 没 out edge (Throw 没 Output pin), sub-flow 在 Throw error 时已退.
			},
		},
	}
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "test-try-throw",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "try_n", Kind: "Try", Config: map[string]any{"SubgraphID": "trybody_throw"}},
				{ID: "catch_n", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				{From: "try_n.Catch", To: "catch_n.In"},
			},
		},
		Subgraphs: []container.Subgraph{subgraph},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopColorDetector{}, nil, nil, nil, nil, 0)
	r := NewContainerRunner(rt)
	tryNode := r.nodesByID["try_n"]
	tokens, err := r.execNodeAsRegionViaFramework(context.Background(), tryNode, ExecToken{NodeID: "try_n", InPin: "in"})
	if err != nil {
		t.Fatalf("try throw: %v", err)
	}
	if len(tokens) != 1 || tokens[0].NodeID != "catch_n" {
		t.Errorf("try throw token = %+v, want {catch_n, In}", tokens)
	}
	// 验证 PopFrame
	if r.state.CurrentFrame == nil || r.state.CurrentFrame.Parent != nil {
		t.Errorf("PopFrame did not restore main frame after Try catch: %+v", r.state.CurrentFrame)
	}
}

// TestExecNodeAsRegionViaFramework_SubgraphPassesParams 验证:
// Subgraph 节点 Config["Params"] 是 JSON map, runner PushFrame 后 unpack 到
// frame.LocalParams. callee 子图内 GetParam 节点 (走 framework GetParam.Evaluate) 应能读到值.
//
// 拓扑:
//   主图: sg_call (Subgraph SubgraphID=paramSub Params={greeting:"hello"}) → done_n (Stop)
//   子图 paramSub: sub_in → sv1 (SetVar capturedGreeting <= GetParam.value) → sub_out
//   data wire: getp1.value (GetParam paramName=greeting) → sv1.value
//
// 验证: dispatch 完 rt.Vars()["capturedGreeting"] == "hello".
func TestExecNodeAsRegionViaFramework_SubgraphPassesParams(t *testing.T) {
	subgraph := container.Subgraph{
		ID: "paramSub",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "sub_in", Kind: "SubgraphInput"},
				{ID: "getp1", Kind: "GetParam", Config: map[string]any{"ParamName": "greeting"}},
				{ID: "sv1", Kind: "SetVar", Config: map[string]any{"VarName": "capturedGreeting", "Scope": "global"}},
				{ID: "sub_out", Kind: "SubgraphOutput"},
			},
			Edges: []container.GraphEdge{
				{From: "sub_in.Done", To: "sv1.In"},
				{From: "getp1.Value", To: "sv1.Value"},
				{From: "sv1.Done", To: "sub_out.In"},
			},
		},
		InputParams: []container.SubgraphInputParam{
			{Name: "greeting", Type: "string"},
		},
	}
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "test-subgraph-params",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "sg_call", Kind: "Subgraph", Config: map[string]any{
					"SubgraphID": "paramSub",
					"Params":     map[string]any{"greeting": "hello"},
				}},
				{ID: "done_n", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				{From: "sg_call.Done", To: "done_n.In"},
			},
		},
		Subgraphs: []container.Subgraph{subgraph},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopColorDetector{}, nil, nil, nil, nil, 0)
	r := NewContainerRunner(rt)
	sgNode := r.nodesByID["sg_call"]
	_, err := r.execNodeAsRegionViaFramework(context.Background(), sgNode, ExecToken{NodeID: "sg_call", InPin: "in"})
	if err != nil {
		t.Fatalf("subgraph dispatch with params: %v", err)
	}
	got := rt.Vars()["capturedGreeting"]
	if got != "hello" {
		t.Errorf("captured var = %v, want \"hello\" (Params not flow through to LocalParams → GetParam → SetVar)", got)
	}
}

// ============================================================================
// pull-eval — buildDataWireFor / resolveDataPinV5
// ============================================================================

// tdEcho 单输入回声节点 — 把 data-in pin "Value" 的值塞 OutputData.echo, 出口 Out 触发.
// 用于验证 buildDataWireFor 真把上游 pure-data 算出来的值塞进 in.
type tdEcho struct{}

const tkEcho = "test_dispatch_echo"

func (tdEcho) Spec() node.Spec {
	return node.Spec{
		Kind: tkEcho,
		Inputs: []node.InputSpec{
			{Name: "in", Type: "Exec"},
			{Name: "Value", Type: "*"},
		},
		Outputs: []node.OutputSpec{{Name: "Out", Type: "Exec"}},
	}
}
func (tdEcho) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	tdEchoMu.Lock()
	tdEchoLast = in.Raw("Value")
	tdEchoMu.Unlock()
	return ctx.Out("Out").Fire(), nil
}

// tdEchoLast captures last echoed Value (test-global, tests run serial).
var (
	tdEchoMu   sync.Mutex
	tdEchoLast any
)

func getTdEchoLast() any { tdEchoMu.Lock(); defer tdEchoMu.Unlock(); return tdEchoLast }
func resetTdEcho()       { tdEchoMu.Lock(); tdEchoLast = nil; tdEchoMu.Unlock() }

func init() { node.Register(&tdEcho{}) }

// TestBuildDataWireFor_UpstreamPureFuncViaFramework — Add 上游 → tdEcho.Value via data edge.
// 验证 buildDataWireFor 走 resolveDataPinV5 → 检测 Add IsPureData + Evaluator → 调 EvaluatePureData
// 拿到结果 5.0 → 塞 tdEcho dataWire. tdEcho.Run 通过 in.Raw 读 5.0.
//
// happy path — pure-data 节点 (Add) 在 framework 路径上被 evaluate.
func TestBuildDataWireFor_UpstreamPureFuncViaFramework(t *testing.T) {
	resetTdEcho()
	// 拓扑: add_n (Add a=2,b=3) → echo_n.Value (data edge); echo_n.Out → done_n.in
	// Add 输入靠 config literal (Add data-in 没上游 edge → resolveDataPinV5 走 pullDataPin →
	// 读 config["literal"]).
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "test-add-via-framework",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "add_n", Kind: "Add", Config: map[string]any{
					"literal": map[string]any{"A": 2.0, "B": 3.0},
				}},
				{ID: "echo_n", Kind: tkEcho},
				{ID: "done_n", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				{From: "add_n.Result", To: "echo_n.Value"},
				{From: "echo_n.Out", To: "done_n.In"},
			},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopColorDetector{}, nil, nil, nil, nil, 0)
	r := NewContainerRunner(rt)
	echoNode := r.nodesByID["echo_n"]

	tokens, err := r.execNodeViaFramework(context.Background(), echoNode, ExecToken{NodeID: "echo_n", InPin: "in"})
	if err != nil {
		t.Fatalf("echo via framework: %v", err)
	}
	if len(tokens) != 1 || tokens[0].NodeID != "done_n" {
		t.Errorf("tokens = %+v, want {done_n}", tokens)
	}
	got := getTdEchoLast()
	if got != 5.0 {
		t.Errorf("echoed value = %v (%T), want 5.0 (Add result via framework EvaluatePureData)", got, got)
	}
}

// TestBuildDataWireFor_UpstreamPureFuncRecursive — Add(Add(1,2), 3) → tdEcho.Value, 验证 resolveDataPinV5
// 递归: 内层 Add 走 framework, 外层 Add 也走 framework 拿到 6.0.
func TestBuildDataWireFor_UpstreamPureFuncRecursive(t *testing.T) {
	resetTdEcho()
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "test-recursive-add",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "inner", Kind: "Add", Config: map[string]any{
					"literal": map[string]any{"A": 1.0, "B": 2.0},
				}},
				{ID: "outer", Kind: "Add", Config: map[string]any{
					"literal": map[string]any{"B": 3.0}, // A 由上游 edge 给
				}},
				{ID: "echo_n", Kind: tkEcho},
				{ID: "done_n", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				{From: "inner.Result", To: "outer.A"},
				{From: "outer.Result", To: "echo_n.Value"},
				{From: "echo_n.Out", To: "done_n.In"},
			},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopColorDetector{}, nil, nil, nil, nil, 0)
	r := NewContainerRunner(rt)
	echoNode := r.nodesByID["echo_n"]

	_, err := r.execNodeViaFramework(context.Background(), echoNode, ExecToken{NodeID: "echo_n", InPin: "in"})
	if err != nil {
		t.Fatalf("recursive add via framework: %v", err)
	}
	got := getTdEchoLast()
	if got != 6.0 {
		t.Errorf("recursive Add(Add(1,2),3) = %v, want 6", got)
	}
}

// TestBuildDataWireFor_GetVarViaFramework — 上游是 GetVar (IsPureData + Evaluator) →
// resolveDataPinV5 走 nodepkg.EvaluatePureData, framework snapshot wrap 把 ctx.Vars()
// 替成 snapshot view, scope="global" 拿 frozen Vars (从 currentTick.Vars 读).
//
// 验证 dispatch 经 framework path 跨节点拉 GetVar 值.
func TestBuildDataWireFor_GetVarViaFramework(t *testing.T) {
	resetTdEcho()
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "test-fallback",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "gv1", Kind: "GetVar", Config: map[string]any{"VarName": "myvar", "Scope": "global"}},
				{ID: "echo_n", Kind: tkEcho},
				{ID: "done_n", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				{From: "gv1.Value", To: "echo_n.Value"},
				{From: "echo_n.Out", To: "done_n.In"},
			},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopColorDetector{}, nil, nil, nil, nil, 0)
	rt.SetVar("myvar", "from-old-path")
	r := NewContainerRunner(rt)
	// GetVar.Evaluate 经 snapshot wrap 读 frozen Vars; 主循环抓 snapshot 模拟之.
	ctx := withTickSnapshot(context.Background(), CaptureSnapshot(rt.Vars(), rt.Sys()))
	echoNode := r.nodesByID["echo_n"]

	_, err := r.execNodeViaFramework(ctx, echoNode, ExecToken{NodeID: "echo_n", InPin: "in"})
	if err != nil {
		t.Fatalf("GetVar via framework: %v", err)
	}
	got := getTdEchoLast()
	if got != "from-old-path" {
		t.Errorf("GetVar via framework = %v, want \"from-old-path\"", got)
	}
}

func TestExecNodeAsRegionViaFramework_TryMissingSubgraphID(t *testing.T) {
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "test-try-missing-id",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "try_n", Kind: "Try"}, // Config nil → SubgraphID missing
			},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopColorDetector{}, nil, nil, nil, nil, 0)
	r := NewContainerRunner(rt)
	tryNode := r.nodesByID["try_n"]
	_, err := r.execNodeAsRegionViaFramework(context.Background(), tryNode, ExecToken{NodeID: "try_n", InPin: "in"})
	if err == nil {
		t.Fatal("expected error for missing SubgraphID")
	}
	// makeBodyForTry → makeBodyForSubgraph 返 "missing SubgraphID" 在 framework Required check
	// 之前; 实际上 Required check 会先触发 validation. 接受任何路径错信号.
	if !strings.Contains(err.Error(), "SubgraphID") && !strings.Contains(err.Error(), "validation") {
		t.Errorf("error %q should mention SubgraphID or validation", err)
	}
}
