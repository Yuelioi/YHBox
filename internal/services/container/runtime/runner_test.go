package runtime

import (
	"context"
	"testing"

	"yhbox/internal/services/container"
	"yhbox/internal/services/execution"
	"yhbox/internal/services/expr"
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

// B-10 regression: SetVar/IncVar 显式 scope=local 时，值写入 ExecState.LocalVars，
// rt.vars 保持 Default 不变。GetVar(scope=auto) 读取应该看到 local 值（覆盖 rt.vars 默认）。
//
// 2026-05-19 默认 scope 从 "local" 改成 "auto" 后, 本测试改成显式 scope: "local" 来
// 锁定隔离行为. 默认 auto 行为另在 TestRunner_SetVarAutoDefaultFindOrCreate.
func TestRunner_SetVarLocalScopeIsolation(t *testing.T) {
	// v4: 用 GetVar(scope=auto) + Eq + If data-flow 验证 local 写入对后续 read 可见.
	// auto scope 优先 frame.LocalVars (其中 set 写了 42), 兜底 rt.vars (默认 0).
	c := newTestContainer(
		[]container.GraphNode{
			{ID: "start", Kind: "Start"},
			// 显式 scope = local — 锁定隔离行为
			{ID: "set", Kind: "SetVar", Config: map[string]any{"varName": "x", "scope": "local", "literal": map[string]any{"value": 42.0}}},
			// data nodes: GetVar(x, auto) → Eq(==42) → If.condition
			{ID: "getx", Kind: "GetVar", Config: map[string]any{"varName": "x", "scope": "auto"}},
			{ID: "eq", Kind: "Eq", Config: map[string]any{"literal": map[string]any{"b": 42.0}}},
			{ID: "if", Kind: "If"},
			{ID: "markThen", Kind: "SetVar", Config: map[string]any{"varName": "branch", "scope": "global", "literal": map[string]any{"value": "then"}}},
			{ID: "markElse", Kind: "SetVar", Config: map[string]any{"varName": "branch", "scope": "global", "literal": map[string]any{"value": "else"}}},
		},
		[]container.GraphEdge{
			{From: "start.out", To: "set.in"},
			{From: "set.out", To: "if.in"},
			{From: "if.then", To: "markThen.in"},
			{From: "if.else", To: "markElse.in"},
			// data flow into If.condition
			{From: "getx.value", To: "eq.a"},
			{From: "eq.result", To: "if.condition"},
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

// 2026-05-19 默认 scope 从 "local" 改 "auto" 后的新行为锁定:
// SetVar 默认 scope=auto — 当前 frame.LocalVars 没 name → 写 rt.vars (find-or-create-global,
// 跟 Container.Vars 面板默认值合流). GetVar 默认 scope=auto — frame chain → rt.vars fallback.
// 用户场景: 容器面板声明 x=1, y=2; SetVar(x, 默认 auto, 5); GetVar(x, 默认 auto) + GetVar(y, 默认 auto)
// → 5, 2; Add = 7. (image copy 4.png 验证的 bug 修复后行为.)
func TestRunner_SetVarAutoDefaultFindOrCreate(t *testing.T) {
	c := newTestContainer(
		[]container.GraphNode{
			{ID: "start", Kind: "Start"},
			// 默认 scope (auto): rt.vars[x] 启动时 = 1 (Container.Vars 默认); frame.LocalVars 空.
			// auto SetVar 写 5 → frame 没 x → 走 rt.vars[x]=5.
			{ID: "set", Kind: "SetVar", Config: map[string]any{"varName": "x", "literal": map[string]any{"value": 5.0}}},
			// 默认 GetVar auto — frame 没 y, fallback rt.vars[y]=2.
			{ID: "gety", Kind: "GetVar", Config: map[string]any{"varName": "y"}},
			// 写 gety 值到 captured (global) — 验证 GetVar 默认 auto 读容器变量
			{ID: "capture", Kind: "SetVar", Config: map[string]any{"varName": "captured", "scope": "global"}},
		},
		[]container.GraphEdge{
			{From: "start.out", To: "set.in"},
			{From: "set.out", To: "capture.in"},
			{From: "gety.value", To: "capture.value"},
		},
		[]container.VarDecl{
			{Name: "x", Type: "number", Default: 1.0},
			{Name: "y", Type: "number", Default: 2.0},
			{Name: "captured", Type: "number", Default: 0.0},
		},
	)
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopColorDetector{}, nil, nil, nil, nil, 0)
	stubRuntimeWindowAndInput(rt)
	r := NewContainerRunner(rt)
	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	// SetVar(x, default=auto) 应写到 rt.vars (frame 没 x → find-or-create-global)
	if got, _ := expr.AsNumber(rt.Vars()["x"]); got != 5.0 {
		t.Errorf("rt.vars[x] = %v, want 5 (auto find-or-create-global: frame 没 x → 写 rt.vars)", got)
	}
	// GetVar(y, default=auto) 应 fallback rt.vars[y]=2
	if got, _ := expr.AsNumber(rt.Vars()["captured"]); got != 2.0 {
		t.Errorf("captured = %v, want 2 (GetVar default auto fallback rt.vars)", got)
	}
}

func TestRunner_SetVarThenIncVar(t *testing.T) {
	c := newTestContainer(
		[]container.GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "set", Kind: "SetVar", Config: map[string]any{"varName": "x", "scope": "global", "literal": map[string]any{"value": 10.0}}},
			{ID: "inc", Kind: "IncVar", Config: map[string]any{"varName": "x", "scope": "global", "literal": map[string]any{"delta": 5.0}}},
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
	// v4: GetVar(y, global) → Eq(==1) → If.condition data flow.
	c := newTestContainer(
		[]container.GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "set1", Kind: "SetVar", Config: map[string]any{"varName": "y", "scope": "global", "literal": map[string]any{"value": 1.0}}},
			{ID: "gety", Kind: "GetVar", Config: map[string]any{"varName": "y", "scope": "global"}},
			{ID: "eq", Kind: "Eq", Config: map[string]any{"literal": map[string]any{"b": 1.0}}},
			{ID: "if", Kind: "If"},
			{ID: "setThen", Kind: "SetVar", Config: map[string]any{"varName": "branch", "scope": "global", "literal": map[string]any{"value": "then"}}},
			{ID: "setElse", Kind: "SetVar", Config: map[string]any{"varName": "branch", "scope": "global", "literal": map[string]any{"value": "else"}}},
		},
		[]container.GraphEdge{
			{From: "start.out", To: "set1.in"},
			{From: "set1.out", To: "if.in"},
			{From: "if.then", To: "setThen.in"},
			{From: "if.else", To: "setElse.in"},
			{From: "gety.value", To: "eq.a"},
			{From: "eq.result", To: "if.condition"},
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
	// v4: Loop.count via inline literal (no more v3 expr-string config).
	c := newTestContainer(
		[]container.GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "loop", Kind: "Loop", Config: map[string]any{"mode": "count", "literal": map[string]any{"count": 3.0}}},
			{ID: "inc", Kind: "IncVar", Config: map[string]any{"varName": "i", "scope": "global", "literal": map[string]any{"delta": 1.0}}},
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
	// v4: GetVar(i, global) → GtEq(b=2) → If.condition.
	c := newTestContainer(
		[]container.GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "loop", Kind: "Loop", Config: map[string]any{"mode": "forever"}},
			{ID: "inc", Kind: "IncVar", Config: map[string]any{"varName": "i", "scope": "global", "literal": map[string]any{"delta": 1.0}}},
			{ID: "geti", Kind: "GetVar", Config: map[string]any{"varName": "i", "scope": "global"}},
			{ID: "gte", Kind: "GtEq", Config: map[string]any{"literal": map[string]any{"b": 2.0}}},
			{ID: "if", Kind: "If"},
			{ID: "br", Kind: "Break"},
		},
		[]container.GraphEdge{
			{From: "start.out", To: "loop.in"},
			{From: "loop.body", To: "inc.in"},
			{From: "inc.out", To: "if.in"},
			{From: "if.then", To: "br.in"},
			{From: "if.else", To: "loop.loopback"},
			{From: "geti.value", To: "gte.a"},
			{From: "gte.result", To: "if.condition"},
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
			{ID: "set1", Kind: "SetVar", Config: map[string]any{"varName": "a", "scope": "global", "literal": map[string]any{"value": 1.0}}},
			{ID: "stop", Kind: "Stop"},
			{ID: "set2", Kind: "SetVar", Config: map[string]any{"varName": "a", "scope": "global", "literal": map[string]any{"value": 2.0}}},
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
