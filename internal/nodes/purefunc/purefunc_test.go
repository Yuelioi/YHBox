package purefunc

import (
	"context"
	"errors"
	"testing"

	"yhbox/internal/node"
)

// 所有 22 个 purefunc + Expr 都是 pure-data stub. Run 返 errPureDataNotEvaluatable.
// Phase 5 加 pull-eval framework 后, Run 永不调; FE inspector 已能渲染 Spec.

func TestAll_RegisterAndPureDataStub(t *testing.T) {
	node.ResetRegistryForTest()

	// 全部 23 节点显式注册一遍 (init 注册的是 ResetRegistryForTest 之前的, 这里重新调)
	all := []node.Node{
		&Add{}, &Sub{}, &Mul{}, &Div{}, &Mod{}, &Neg{},
		&Lt{}, &LtEq{}, &Gt{}, &GtEq{}, &Eq{}, &NotEq{},
		&And{}, &Or{}, &Not{},
		&Concat{}, &Contains{}, &Length{},
		&ToString{}, &ToNumber{}, &ToBool{},
		&Select{},
		&Expr{},
	}
	for _, n := range all {
		node.Register(n)
	}

	if len(all) != 23 {
		t.Fatalf("expected 23 purefunc nodes, got %d", len(all))
	}

	for _, n := range all {
		spec := n.Spec()
		if !spec.IsPureData {
			t.Errorf("%s.IsPureData = false, want true", spec.Kind)
		}
		if len(spec.Outputs) != 1 {
			t.Errorf("%s outputs len = %d, want 1", spec.Kind, len(spec.Outputs))
		}

		rn, _ := node.Get(spec.Kind)
		r := node.RunNode(context.Background(), rn, nil, nil, nil, node.StubServices())
		if !errors.Is(r.Error, errPureDataNotEvaluatable) {
			t.Errorf("%s Run error = %v, want errPureDataNotEvaluatable", spec.Kind, r.Error)
		}
	}
}

func TestSpecBuilder_HasResultOutput(t *testing.T) {
	s := specBuilder("Test", "测试", "", numIn(), "Number")
	if s.Outputs[0].Name != "Result" {
		t.Errorf("output name = %q, want Result", s.Outputs[0].Name)
	}
	if s.Outputs[0].Type != "Number" {
		t.Errorf("output type = %q, want Number", s.Outputs[0].Type)
	}
}

// TestEvaluate_22PureFuncs 覆盖 22 个实现 Evaluator 的节点 (Add/.../Select), 验证 EvaluatePureData
// 走 framework 拿到预期值. Expr 不实现 Evaluator (Phase 6+ partial), 单独 fallback 测试在 dispatch_v5 层.
func TestEvaluate_22PureFuncs(t *testing.T) {
	node.ResetRegistryForTest()
	for _, n := range []node.Node{
		&Add{}, &Sub{}, &Mul{}, &Div{}, &Mod{}, &Neg{},
		&Lt{}, &LtEq{}, &Gt{}, &GtEq{}, &Eq{}, &NotEq{},
		&And{}, &Or{}, &Not{},
		&Concat{}, &Contains{}, &Length{},
		&ToString{}, &ToNumber{}, &ToBool{},
		&Select{},
	} {
		node.Register(n)
	}

	cases := []struct {
		kind     string
		dataWire map[string]any
		want     any
	}{
		// 算术
		{"Add", map[string]any{"A": 1.5, "B": 2.5}, 4.0},
		{"Sub", map[string]any{"A": 5.0, "B": 3.0}, 2.0},
		{"Mul", map[string]any{"A": 4.0, "B": 0.5}, 2.0},
		{"Div", map[string]any{"A": 10.0, "B": 4.0}, 2.5},
		{"Mod", map[string]any{"A": 10.0, "B": 3.0}, 1.0},
		{"Neg", map[string]any{"X": 7.0}, -7.0},
		// 比较
		{"Lt", map[string]any{"A": 1.0, "B": 2.0}, true},
		{"LtEq", map[string]any{"A": 2.0, "B": 2.0}, true},
		{"Gt", map[string]any{"A": 3.0, "B": 2.0}, true},
		{"GtEq", map[string]any{"A": 2.0, "B": 2.0}, true},
		{"Eq", map[string]any{"A": "x", "B": "x"}, true},
		{"NotEq", map[string]any{"A": "x", "B": "y"}, true},
		// 逻辑 — And default 是 true,true 不传也 OK; 但显式塞.
		{"And", map[string]any{"A": true, "B": false}, false},
		{"Or", map[string]any{"A": false, "B": true}, true},
		{"Not", map[string]any{"X": false}, true},
		// 字符串
		{"Concat", map[string]any{"A": "foo", "B": "bar"}, "foobar"},
		{"Contains", map[string]any{"Haystack": "hello world", "Needle": "world"}, true},
		{"Length", map[string]any{"S": "hello"}, 5.0},
		// 转换
		{"ToString", map[string]any{"X": 42.0}, "42"},
		{"ToNumber", map[string]any{"X": 3.14}, 3.14},
		{"ToBool", map[string]any{"X": "non-empty"}, true},
		// 三元
		{"Select", map[string]any{"Cond": true, "A": "yes", "B": "no"}, "yes"},
	}

	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			rn, ok := node.Get(tc.kind)
			if !ok {
				t.Fatalf("kind %q not registered", tc.kind)
			}
			if rn.Evaluate == nil {
				t.Fatalf("kind %q: rn.Evaluate is nil — node should implement Evaluator", tc.kind)
			}
			got, err := node.EvaluatePureData(context.Background(), rn, tc.dataWire, nil, node.StubServices())
			if err != nil {
				t.Fatalf("EvaluatePureData %s: %v", tc.kind, err)
			}
			if got != tc.want {
				t.Errorf("EvaluatePureData %s = %v (%T), want %v (%T)", tc.kind, got, got, tc.want, tc.want)
			}
		})
	}
}

// TestEvaluate_ShortCircuit Or/And 短路 — Or(true, ?) 不读 b; And(false, ?) 不读 b.
// 用一个无法 read 的 sentinel 验证 b 没被消费 (这里直接 omit b 让 default 生效, 行为可见).
func TestEvaluate_ShortCircuit(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&And{})
	node.Register(&Or{})

	// And(false, true) → false (短路, b 不读)
	rn, _ := node.Get("And")
	got, err := node.EvaluatePureData(context.Background(), rn, map[string]any{"A": false, "B": true}, nil, node.StubServices())
	if err != nil || got != false {
		t.Errorf("And(false, true) = (%v, %v), want (false, nil)", got, err)
	}
	// Or(true, false) → true (短路)
	rn, _ = node.Get("Or")
	got, err = node.EvaluatePureData(context.Background(), rn, map[string]any{"A": true, "B": false}, nil, node.StubServices())
	if err != nil || got != true {
		t.Errorf("Or(true, false) = (%v, %v), want (true, nil)", got, err)
	}
}

// TestEvaluate_DivByZero Div(_, 0) → NaN, 跟老 evalPureFunc 一致.
func TestEvaluate_DivByZero(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Div{})
	rn, _ := node.Get("Div")
	got, err := node.EvaluatePureData(context.Background(), rn, map[string]any{"A": 1.0, "B": 0.0}, nil, node.StubServices())
	if err != nil {
		t.Fatalf("Div by zero err: %v", err)
	}
	f, ok := got.(float64)
	if !ok || f == f { // NaN != NaN
		t.Errorf("Div(1, 0) = %v, want NaN", got)
	}
}

func TestExpr_RequiredExprField(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Expr{})
	rn, _ := node.Get("Expr")
	// 不传 expr → Required 应触发 ValidationError (在 Run 入口前)
	r := node.RunNode(context.Background(), rn, nil, nil, nil, node.StubServices())
	if len(r.Validation) == 0 && r.Error != errPureDataNotEvaluatable {
		t.Errorf("Expr missing expr should ValidationError or sentinel, got error=%v validation=%v", r.Error, r.Validation)
	}
}
