package runtime

import (
	"context"
	"testing"

	"yhbox/internal/services/container"
	"yhbox/internal/services/execution"
	"yhbox/pkg/winutil"
)

// 构造小图工具：Start → SetVar → Stop。
func newTestContainer(nodes []container.GraphNode, edges []container.GraphEdge, vars []container.VarDecl) *container.Container {
	return &container.Container{
		SchemaVersion: 1,
		ID:            "test",
		Name:          "test",
		Vars:          vars,
		Graph: container.Graph{
			Nodes: nodes,
			Edges: edges,
		},
	}
}

// stubRuntimeWindowAndInput 把 rt.Window / rt.Input stub 成非零, 让 setupRuntime
// 走幂等跳过分支 — 测试不需要真 hwnd / 真 backend / 真 capture.
func stubRuntimeWindowAndInput(rt *RuntimeContext) {
	rt.Window = winutil.WindowHandle{HWND: 1}
	rt.Input = &fakeInputBackend{}
}

func TestRunner_StartSleep(t *testing.T) {
	c := newTestContainer(
		[]container.GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "s1", Kind: "Sleep", Config: map[string]any{"durationMs": "10"}},
		},
		[]container.GraphEdge{
			{From: "start.out", To: "s1.in"},
		},
		nil,
	)
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopColorDetector{}, nil /* game */, nil, nil, nil, 0)
	stubRuntimeWindowAndInput(rt)
	r := NewContainerRunner(rt)
	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

// B-10 regression: SetVar/IncVar 默认 scope=local 时，值写入 ExecState.LocalVars，
// rt.vars 保持 Default 不变。表达式 $vars.x 读取应该看到 local 值（覆盖 rt.vars 默认）。
func TestRunner_SetVarLocalScopeIsolation(t *testing.T) {
	c := newTestContainer(
		[]container.GraphNode{
			{ID: "start", Kind: "Start"},
			// 缺省 scope = local（spec §2.7）
			{ID: "set", Kind: "SetVar", Config: map[string]any{"varName": "x", "value": "42"}},
			// 用 $vars.x 读 → 应拿到 local 42（不是 rt.vars 默认 0）；If condition 真则走 then
			{ID: "if", Kind: "If", Config: map[string]any{"condition": "$vars.x == 42"}},
			{ID: "markThen", Kind: "SetVar", Config: map[string]any{"varName": "branch", "value": "\"then\"", "scope": "global"}},
			{ID: "markElse", Kind: "SetVar", Config: map[string]any{"varName": "branch", "value": "\"else\"", "scope": "global"}},
		},
		[]container.GraphEdge{
			{From: "start.out", To: "set.in"},
			{From: "set.out", To: "if.in"},
			{From: "if.then", To: "markThen.in"},
			{From: "if.else", To: "markElse.in"},
		},
		[]container.VarDecl{
			{Name: "x", Type: "number", Default: 0.0},
			{Name: "branch", Type: "string", Default: ""},
		},
	)
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopColorDetector{}, nil, nil, nil, nil, 0)
	stubRuntimeWindowAndInput(rt)
	r := NewContainerRunner(rt)
	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	// rt.vars["x"] 应保持 Default 0（local 写入不污染容器级 vars）
	if got := rt.Vars()["x"]; got != 0.0 {
		t.Errorf("rt.vars[x] = %v, want 0 (local scope writes to LocalVars only)", got)
	}
	// $vars.x 在表达式里应能读到 42（local 优先于 rt.vars）
	if got := rt.Vars()["branch"]; got != "then" {
		t.Errorf("branch = %v, want 'then' ($vars.x should resolve to 42 via LocalVars)", got)
	}
}

func TestRunner_SetVarThenIncVar(t *testing.T) {
	c := newTestContainer(
		[]container.GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "set", Kind: "SetVar", Config: map[string]any{"varName": "x", "value": "10", "scope": "global"}},
			{ID: "inc", Kind: "IncVar", Config: map[string]any{"varName": "x", "delta": "5", "scope": "global"}},
		},
		[]container.GraphEdge{
			{From: "start.out", To: "set.in"},
			{From: "set.out", To: "inc.in"},
		},
		[]container.VarDecl{{Name: "x", Type: "number", Default: 0.0}},
	)
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopColorDetector{}, nil /* game */, nil, nil, nil, 0)
	stubRuntimeWindowAndInput(rt)
	r := NewContainerRunner(rt)
	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	vars := rt.Vars()
	if vars["x"] != 15.0 {
		t.Errorf("x = %v, want 15", vars["x"])
	}
}

func TestRunner_IfBranch(t *testing.T) {
	c := newTestContainer(
		[]container.GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "set1", Kind: "SetVar", Config: map[string]any{"varName": "y", "value": "1", "scope": "global"}},
			{ID: "if", Kind: "If", Config: map[string]any{"condition": "$vars.y == 1"}},
			{ID: "setThen", Kind: "SetVar", Config: map[string]any{"varName": "branch", "value": "\"then\"", "scope": "global"}},
			{ID: "setElse", Kind: "SetVar", Config: map[string]any{"varName": "branch", "value": "\"else\"", "scope": "global"}},
		},
		[]container.GraphEdge{
			{From: "start.out", To: "set1.in"},
			{From: "set1.out", To: "if.in"},
			{From: "if.then", To: "setThen.in"},
			{From: "if.else", To: "setElse.in"},
		},
		[]container.VarDecl{
			{Name: "y", Type: "number", Default: 0.0},
			{Name: "branch", Type: "string", Default: ""},
		},
	)
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopColorDetector{}, nil /* game */, nil, nil, nil, 0)
	stubRuntimeWindowAndInput(rt)
	r := NewContainerRunner(rt)
	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := rt.Vars()["branch"]; got != "then" {
		t.Errorf("branch = %v, want then", got)
	}
}

func TestRunner_LoopCount(t *testing.T) {
	c := newTestContainer(
		[]container.GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "loop", Kind: "Loop", Config: map[string]any{"mode": "count", "count": "3"}},
			{ID: "inc", Kind: "IncVar", Config: map[string]any{"varName": "i", "delta": "1", "scope": "global"}},
		},
		[]container.GraphEdge{
			{From: "start.out", To: "loop.in"},
			{From: "loop.body", To: "inc.in"},
			{From: "inc.out", To: "loop.loopback"},
		},
		[]container.VarDecl{{Name: "i", Type: "number", Default: 0.0}},
	)
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopColorDetector{}, nil /* game */, nil, nil, nil, 0)
	stubRuntimeWindowAndInput(rt)
	r := NewContainerRunner(rt)
	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := rt.Vars()["i"]; got != 3.0 {
		t.Errorf("i = %v, want 3", got)
	}
}

func TestRunner_BreakExitsLoop(t *testing.T) {
	c := newTestContainer(
		[]container.GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "loop", Kind: "Loop", Config: map[string]any{"mode": "forever"}},
			{ID: "inc", Kind: "IncVar", Config: map[string]any{"varName": "i", "delta": "1", "scope": "global"}},
			{ID: "if", Kind: "If", Config: map[string]any{"condition": "$vars.i >= 2"}},
			{ID: "br", Kind: "Break"},
		},
		[]container.GraphEdge{
			{From: "start.out", To: "loop.in"},
			{From: "loop.body", To: "inc.in"},
			{From: "inc.out", To: "if.in"},
			{From: "if.then", To: "br.in"},
			{From: "if.else", To: "loop.loopback"},
		},
		[]container.VarDecl{{Name: "i", Type: "number", Default: 0.0}},
	)
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopColorDetector{}, nil /* game */, nil, nil, nil, 0)
	stubRuntimeWindowAndInput(rt)
	r := NewContainerRunner(rt)
	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := rt.Vars()["i"]; got != 2.0 {
		t.Errorf("i = %v, want 2", got)
	}
}

func TestRunner_StopHalts(t *testing.T) {
	c := newTestContainer(
		[]container.GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "set1", Kind: "SetVar", Config: map[string]any{"varName": "a", "value": "1", "scope": "global"}},
			{ID: "stop", Kind: "Stop"},
			{ID: "set2", Kind: "SetVar", Config: map[string]any{"varName": "a", "value": "2", "scope": "global"}},
		},
		[]container.GraphEdge{
			{From: "start.out", To: "set1.in"},
			{From: "set1.out", To: "stop.in"},
			{From: "stop.out", To: "set2.in"},
		},
		[]container.VarDecl{{Name: "a", Type: "number", Default: 0.0}},
	)
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopColorDetector{}, nil /* game */, nil, nil, nil, 0)
	stubRuntimeWindowAndInput(rt)
	r := NewContainerRunner(rt)
	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := rt.Vars()["a"]; got != 1.0 {
		t.Errorf("Stop 后 a 应停留在 1，实际 %v", got)
	}
}

func TestRunner_NoStartNode(t *testing.T) {
	c := newTestContainer(
		[]container.GraphNode{{ID: "s", Kind: "Sleep"}}, nil, nil,
	)
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopColorDetector{}, nil /* game */, nil, nil, nil, 0)
	stubRuntimeWindowAndInput(rt)
	r := NewContainerRunner(rt)
	if err := r.Run(context.Background()); err == nil {
		t.Error("expected error for no Start node")
	}
}
