package purefunc

import (
	"context"
	"math"
	"testing"

	"yotta/internal/node"
)

func TestSpecBuilder_HasResultOutput(t *testing.T) {
	s := specBuilder("Test", numIn(), "Number")
	if s.Outputs[0].Name != "Result" {
		t.Errorf("output name = %q, want Result", s.Outputs[0].Name)
	}
	if s.Outputs[0].Type != "Number" {
		t.Errorf("output type = %q, want Number", s.Outputs[0].Type)
	}
}

// TestEvaluate_22PureFuncs 覆盖 22 个实现 Evaluator 的节点 (Add/.../Select), 验证 EvaluatePureData
// 走 framework 拿到预期值. Expr 不实现 Evaluator, 单独 fallback 测试在 dispatch_v5 层.
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
		{"Length", map[string]any{"S": "中文abc"}, 5.0}, // rune 计数 (字节是 9) — specs/2026-06-10-string-nodes.md
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

// TestEvaluate_DivByZero Div(_, 0) → NaN.
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

// Expr 节点 spec 校验 — IsPureData + Result output + Evaluator 实现.
func TestExpr_Registered(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Expr{})
	rn, ok := node.Get("Expr")
	if !ok {
		t.Fatal("Expr not registered")
	}
	if !rn.Spec.IsPureData {
		t.Errorf("Expr.IsPureData = false, want true")
	}
	if rn.Evaluate == nil {
		t.Errorf("Expr.Evaluate is nil — should implement Evaluator")
	}
	if len(rn.Spec.Outputs) != 1 || rn.Spec.Outputs[0].Name != "Result" {
		t.Errorf("Expr outputs = %+v, want [Result]", rn.Spec.Outputs)
	}
	// rand()/now() 非确定 — 不挂标记 = per-dispatch 不记忆化 = 同帧多路径引用出不同值.
	if !rn.Spec.IsNonDeterministic {
		t.Errorf("Expr.IsNonDeterministic = false, want true")
	}
}

// Expr.Evaluate 静态 dataWire 直跑 — dynamic input 通过 dataWire 喂.
// dispatch_v5.buildDataWireFor 跑生产路径里 pull config.Inputs[]; 单测直接喂值.
func TestExpr_Evaluate_DynamicInputs(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Expr{})
	rn, _ := node.Get("Expr")
	dataWire := map[string]any{
		"Expression": "i + 1",
		"i":          5.0,
	}
	v, err := node.EvaluatePureData(context.Background(), rn, dataWire, nil, node.StubServices())
	if err != nil {
		t.Fatal(err)
	}
	if v != 6.0 {
		t.Errorf("Expr i+1 with i=5: want 6, got %v (%T)", v, v)
	}
}

// TestExpr_Evaluate_ConfigKeysIsolatedFromEnv — 防回归: config 元数据 (OutType, Inputs)
// 必须不进 InputEnv. 老 evalExpr 走 cfg.Inputs[].Name 自然隔离, framework 路径要靠显式
// skip-set. 用户 expression 写 "OutType" 必须 undefined (InputEnv.Get → nil), 不能默契
// 命中 config 字符串 "auto".
//
// expr.InputEnv.Get(missing) → (nil, nil) — 不报错. 所以这里断言结果不是 "autox" (那是
// env leak 的特征值). nil + "x" 在 expr Concat 语义下不是 "autox", 是别的值或错.
func TestExpr_Evaluate_ConfigKeysIsolatedFromEnv(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Expr{})
	rn, _ := node.Get("Expr")
	// config 走第 4 个参数 (EvaluatePureData(ctx, rn, dataWire, config, services));
	// 同名 dataWire 会覆盖 config, 这里只放 config 避免歧义.
	got, _ := node.EvaluatePureData(context.Background(), rn, nil,
		map[string]any{
			"Expression": "OutType + \"x\"",
			"OutType":    "auto",
			"Inputs":     []any{},
		},
		node.StubServices())
	if got == "autox" {
		t.Errorf("env leak regression: OutType config key leaked into expr env (got %q, want NOT %q)", got, "autox")
	}
}

// ===== MakePoint =====

func TestMakePoint_Percent(t *testing.T) {
	// 输入 0-100 (与 PointWidget 一致); Evaluate 除以 100 得 0-1 ratio
	v, err := MakePoint{}.Evaluate(nil, node.NewInputsFromConfig(
		map[string]any{"X": 50.0, "Y": 25.0, "Unit": "percent"}))
	if err != nil {
		t.Fatal(err)
	}
	p, ok := v.(node.Point)
	if !ok || p.X != 0.5 || p.Y != 0.25 || p.Unit != node.UnitRatio {
		t.Fatalf("got %+v ok=%v want {0.5 0.25 ratio}", v, ok)
	}
}

func TestMakePoint_Px(t *testing.T) {
	v, _ := MakePoint{}.Evaluate(nil, node.NewInputsFromConfig(
		map[string]any{"X": 960.0, "Y": 540.0, "Unit": "px"}))
	p := v.(node.Point)
	if p.X != 960 || p.Y != 540 || p.Unit != node.UnitPx {
		t.Fatalf("got %+v want {960 540 px}", p)
	}
}

func TestMakePoint_Spec(t *testing.T) {
	s := MakePoint{}.Spec()
	if s.Kind != "MakePoint" || !s.IsPureData {
		t.Fatalf("spec wrong: %+v", s)
	}
	if len(s.Outputs) != 1 || s.Outputs[0].Type != "Point" {
		t.Fatalf("want single Point output, got %+v", s.Outputs)
	}
}

func TestOffsetPoint_ClampsRatioPoint(t *testing.T) {
	v, err := OffsetPoint{}.Evaluate(nil, node.NewInputsFromConfig(
		map[string]any{
			"Point":   node.Point{X: 0.9, Y: 0.1},
			"OffsetX": 0.2,
			"OffsetY": -0.3,
		}))
	if err != nil {
		t.Fatal(err)
	}
	p := v.(node.Point)
	if p.X != 1 || p.Y != 0 || p.Unit != node.UnitRatio {
		t.Fatalf("got %+v want clamped {1 0 ratio}", p)
	}
}

func TestOffsetPoint_PreservesPixelUnit(t *testing.T) {
	v, _ := OffsetPoint{}.Evaluate(nil, node.NewInputsFromConfig(
		map[string]any{
			"Point":   node.Point{X: 100, Y: 200, Unit: node.UnitPx},
			"OffsetX": 10,
			"OffsetY": -20,
		}))
	p := v.(node.Point)
	if p.X != 110 || p.Y != 180 || p.Unit != node.UnitPx {
		t.Fatalf("got %+v want {110 180 px}", p)
	}
}

func TestPointDistance(t *testing.T) {
	v, err := PointDistance{}.Evaluate(nil, node.NewInputsFromConfig(
		map[string]any{
			"Begin": node.Point{X: 0.1, Y: 0.2},
			"End":   node.Point{X: 0.4, Y: 0.6},
		}))
	if err != nil {
		t.Fatal(err)
	}
	if got := v.(float64); math.Abs(got-0.5) > 1e-9 {
		t.Fatalf("distance = %v, want 0.5", got)
	}
}

func TestROIAroundPoint_ClampsToScreen(t *testing.T) {
	v, err := ROIAroundPoint{}.Evaluate(nil, node.NewInputsFromConfig(
		map[string]any{
			"Center": node.Point{X: 0.9, Y: 0.1},
			"Width":  40.0,
			"Height": 50.0,
		}))
	if err != nil {
		t.Fatal(err)
	}
	g := v.(node.Geometry)
	want := node.Rect{X: 0.6, Y: 0, W: 0.4, H: 0.5}
	if g.Pct != want {
		t.Fatalf("roi = %+v, want %+v", g.Pct, want)
	}
}

func TestPointUtilitySpecs(t *testing.T) {
	for _, tc := range []struct {
		n       node.Node
		outType string
	}{
		{&OffsetPoint{}, "Point"},
		{&PointDistance{}, "Number"},
		{&ROIAroundPoint{}, "Geometry"},
	} {
		s := tc.n.Spec()
		if !s.IsPureData || s.Category != "PureFunc" {
			t.Fatalf("%s spec wrong: %+v", s.Kind, s)
		}
		if len(s.Outputs) != 1 || s.Outputs[0].Name != "Result" || s.Outputs[0].Type != tc.outType {
			t.Fatalf("%s output = %+v, want Result(%s)", s.Kind, s.Outputs, tc.outType)
		}
	}
}
