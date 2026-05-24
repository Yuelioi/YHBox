package runtime

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"yhbox/internal/node"
	"yhbox/internal/services/container"
	"yhbox/internal/services/execution"
)

// ============================================================================
// Test-local fake nodes — 各覆盖一条 dispatch 路径.
// 在 init() 注册. 跟 production 节点 kind 不会冲突 (test_xxx 前缀).
// ============================================================================

const (
	tkHappy      = "test_dispatch_happy"
	tkValidation = "test_dispatch_validation"
	tkError      = "test_dispatch_error"
	tkPanic      = "test_dispatch_panic"
	tkDisplay    = "test_dispatch_display"
	tkNoExit     = "test_dispatch_noexit"
)

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
