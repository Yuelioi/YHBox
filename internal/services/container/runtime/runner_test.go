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
// ClientW/ClientH 默认 1920x1080 (匹配 fishing-v2 ROI table 主分辨率), 让 ColorBarTrack
// 等需要 ClientSize 的节点能 pick ROI.
func stubRuntimeWindowAndInput(rt *RuntimeContext) {
	rt.Window = winutil.WindowHandle{HWND: 1, ClientW: 1920, ClientH: 1080}
	rt.Input = &fakeInputBackend{}
}

func TestRunner_StartSleep(t *testing.T) {
	c := newTestContainer(
		[]container.GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "s1", Kind: "Sleep", Config: map[string]any{"Duration": "10"}},
		},
		[]container.GraphEdge{
			{From: "start.Done", To: "s1.In"},
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

// SetVar/IncVar 显式 scope=local 时，值写入 ExecState.LocalVars，rt.vars 保持 Default
// 不变。GetVar(scope=auto) 读取应该看到 local 值（覆盖 rt.vars 默认）。
// 默认 auto 行为另在 TestRunner_SetVarAutoDefaultFindOrCreate.
func TestRunner_SetVarLocalScopeIsolation(t *testing.T) {
	// 用 GetVar(scope=auto) + Eq + If data-flow 验证 local 写入对后续 read 可见.
	// auto scope 优先 frame.LocalVars (其中 set 写了 42), 兜底 rt.vars (默认 0).
	c := newTestContainer(
		[]container.GraphNode{
			{ID: "start", Kind: "Start"},
			// 显式 scope = local — 锁定隔离行为
			{ID: "set", Kind: "SetVar", Config: map[string]any{"VarName": "x", "Scope": "local", "literal": map[string]any{"Value": 42.0}}},
			// data nodes: GetVar(x, auto) → Eq(==42) → If.condition
			{ID: "getx", Kind: "GetVar", Config: map[string]any{"VarName": "x", "Scope": "auto"}},
			{ID: "eq", Kind: "Eq", Config: map[string]any{"literal": map[string]any{"b": 42.0}}},
			{ID: "if", Kind: "If"},
			{ID: "markThen", Kind: "SetVar", Config: map[string]any{"VarName": "branch", "Scope": "global", "literal": map[string]any{"Value": "then"}}},
			{ID: "markElse", Kind: "SetVar", Config: map[string]any{"VarName": "branch", "Scope": "global", "literal": map[string]any{"Value": "else"}}},
		},
		[]container.GraphEdge{
			{From: "start.Done", To: "set.In"},
			{From: "set.Done", To: "if.In"},
			{From: "if.True", To: "markThen.In"},
			{From: "if.False", To: "markElse.In"},
			// data flow into If.condition
			{From: "getx.Value", To: "eq.a"},
			{From: "eq.result", To: "if.Condition"},
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

// 默认 scope=auto 行为锁定:
// SetVar 默认 scope=auto — 当前 frame.LocalVars 没 name → 写 rt.vars (find-or-create-global,
// 跟 Container.Vars 面板默认值合流). GetVar 默认 scope=auto — frame chain → rt.vars fallback.
// 用户场景: 容器面板声明 x=1, y=2; SetVar(x, 默认 auto, 5); GetVar(x, 默认 auto) + GetVar(y, 默认 auto)
// → 5, 2; Add = 7.
func TestRunner_SetVarAutoDefaultFindOrCreate(t *testing.T) {
	c := newTestContainer(
		[]container.GraphNode{
			{ID: "start", Kind: "Start"},
			// 默认 scope (auto): rt.vars[x] 启动时 = 1 (Container.Vars 默认); frame.LocalVars 空.
			// auto SetVar 写 5 → frame 没 x → 走 rt.vars[x]=5.
			{ID: "set", Kind: "SetVar", Config: map[string]any{"VarName": "x", "literal": map[string]any{"Value": 5.0}}},
			// 默认 GetVar auto — frame 没 y, fallback rt.vars[y]=2.
			{ID: "gety", Kind: "GetVar", Config: map[string]any{"VarName": "y"}},
			// 写 gety 值到 captured (global) — 验证 GetVar 默认 auto 读容器变量
			{ID: "capture", Kind: "SetVar", Config: map[string]any{"VarName": "captured", "Scope": "global"}},
		},
		[]container.GraphEdge{
			{From: "start.Done", To: "set.In"},
			{From: "set.Done", To: "capture.In"},
			{From: "gety.Value", To: "capture.Value"},
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
			{ID: "set", Kind: "SetVar", Config: map[string]any{"VarName": "x", "Scope": "global", "literal": map[string]any{"Value": 10.0}}},
			{ID: "inc", Kind: "IncVar", Config: map[string]any{"VarName": "x", "Scope": "global", "literal": map[string]any{"Delta": 5.0}}},
		},
		[]container.GraphEdge{
			{From: "start.Done", To: "set.In"},
			{From: "set.Done", To: "inc.In"},
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
	// GetVar(y, global) → Eq(==1) → If.condition data flow.
	c := newTestContainer(
		[]container.GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "set1", Kind: "SetVar", Config: map[string]any{"VarName": "y", "Scope": "global", "literal": map[string]any{"Value": 1.0}}},
			{ID: "gety", Kind: "GetVar", Config: map[string]any{"VarName": "y", "Scope": "global"}},
			{ID: "eq", Kind: "Eq", Config: map[string]any{"literal": map[string]any{"b": 1.0}}},
			{ID: "if", Kind: "If"},
			{ID: "setThen", Kind: "SetVar", Config: map[string]any{"VarName": "branch", "Scope": "global", "literal": map[string]any{"Value": "then"}}},
			{ID: "setElse", Kind: "SetVar", Config: map[string]any{"VarName": "branch", "Scope": "global", "literal": map[string]any{"Value": "else"}}},
		},
		[]container.GraphEdge{
			{From: "start.Done", To: "set1.In"},
			{From: "set1.Done", To: "if.In"},
			{From: "if.True", To: "setThen.In"},
			{From: "if.False", To: "setElse.In"},
			{From: "gety.Value", To: "eq.a"},
			{From: "eq.result", To: "if.Condition"},
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
	// Loop.count via inline literal.
	c := newTestContainer(
		[]container.GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "loop", Kind: "Loop", Config: map[string]any{"Mode": "count", "literal": map[string]any{"Count": 3.0}}},
			{ID: "inc", Kind: "IncVar", Config: map[string]any{"VarName": "i", "Scope": "global", "literal": map[string]any{"Delta": 1.0}}},
		},
		[]container.GraphEdge{
			{From: "start.Done", To: "loop.In"},
			{From: "loop.Body", To: "inc.In"},
			// 新 Loop 内部 for-loop 自驱迭代, body 终止 (inc.out 无下游) 即下一轮.
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
	// GetVar(i, global) → GtEq(b=2) → If.condition.
	c := newTestContainer(
		[]container.GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "loop", Kind: "Loop", Config: map[string]any{"Mode": "forever"}},
			{ID: "inc", Kind: "IncVar", Config: map[string]any{"VarName": "i", "Scope": "global", "literal": map[string]any{"Delta": 1.0}}},
			{ID: "geti", Kind: "GetVar", Config: map[string]any{"VarName": "i", "Scope": "global"}},
			{ID: "gte", Kind: "GtEq", Config: map[string]any{"literal": map[string]any{"B": 2.0}}},
			{ID: "if", Kind: "If"},
			{ID: "br", Kind: "Break"},
		},
		[]container.GraphEdge{
			{From: "start.Done", To: "loop.In"},
			{From: "loop.Body", To: "inc.In"},
			{From: "inc.Done", To: "if.In"},
			{From: "if.True", To: "br.In"},
			// if.False 无下游 → body iter 终止 → Loop 自驱下一轮.
			{From: "geti.Value", To: "gte.A"},
			{From: "gte.Result", To: "if.Condition"},
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
			{ID: "set1", Kind: "SetVar", Config: map[string]any{"VarName": "a", "Scope": "global", "literal": map[string]any{"Value": 1.0}}},
			{ID: "stop", Kind: "Stop"},
			{ID: "set2", Kind: "SetVar", Config: map[string]any{"VarName": "a", "Scope": "global", "literal": map[string]any{"Value": 2.0}}},
		},
		[]container.GraphEdge{
			{From: "start.Done", To: "set1.In"},
			{From: "set1.Done", To: "stop.In"},
			{From: "stop.Done", To: "set2.In"},
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

// PlayClip 回放 target (rt.MouseCounts360) 必须来自主图 MouseCalibration 节点值, 不是
// 构造期传入的 settings 本机值 — 否则改节点 Counts360 对 PlayClip 缩放无效 (容器2 实测整圈 bug).
// 无节点时保留 settings 兜底.
func TestRunner_PlayClipTargetUsesNodeCalibCounts(t *testing.T) {
	// 有 MouseCalibration 节点 (Counts360=2000): 节点值覆盖构造期 settings=9999.
	withNode := newTestContainer(
		[]container.GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "mc", Kind: "MouseCalibration", Config: map[string]any{"literal": map[string]any{"Counts360": 2000.0}}},
		},
		nil, nil,
	)
	rt := NewRuntimeContext(withNode, execution.NewInputBus(), NoopMatcher{}, NoopColorDetector{}, nil, nil, nil, nil, 9999)
	NewContainerRunner(rt)
	if rt.MouseCounts360 != 2000 {
		t.Errorf("有节点: rt.MouseCounts360 = %d, want 2000 (节点值覆盖 settings 9999)", rt.MouseCounts360)
	}

	// 无 MouseCalibration 节点: 保留构造期 settings 兜底 9999.
	noNode := newTestContainer(
		[]container.GraphNode{{ID: "start", Kind: "Start"}}, nil, nil,
	)
	rt2 := NewRuntimeContext(noNode, execution.NewInputBus(), NoopMatcher{}, NoopColorDetector{}, nil, nil, nil, nil, 9999)
	NewContainerRunner(rt2)
	if rt2.MouseCounts360 != 9999 {
		t.Errorf("无节点: rt.MouseCounts360 = %d, want 9999 (settings 兜底)", rt2.MouseCounts360)
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
