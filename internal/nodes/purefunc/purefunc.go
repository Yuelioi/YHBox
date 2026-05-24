// Package purefunc 22 个纯函数节点 (Add/Sub/.../Select) + Expr.
// 全是 IsPureData=true — Phase 4 stub Run 返 errPureDataNotEvaluatable.
// Phase 5 加 pull-eval framework 后, 这些节点的 evaluator 由 framework 注册 (按 Kind 路由).
//
// 设计取舍: 22 节点用同一 spec shape (1 data-out "result"), 用 specBuilder 复用构造代码.
// 每节点仍独立 Go type — 符合 "1 type 1 node" 原则, builder 只复用 Spec 字段填充不引入抽象层.
// 这跟 GPT r4 #10 "禁 base class / shared validator" 不冲突 — 没有 nodeBase 类型 / shared logic.
package purefunc

import (
	"errors"

	"yhbox/internal/node"
)

// errPureDataNotEvaluatable Phase 4 stub Run 返这个. Phase 5 pull-eval framework 装好后, Run 永不调.
var errPureDataNotEvaluatable = errors.New("pure-data node — evaluated by pull mechanism (Phase 5+)")

// specBuilder 构造单 result 数据出口的 pure-data Spec.
// 不是抽象, 是 Spec 字段填充 helper — 节点作者用法跟手写完全一样.
func specBuilder(kind, displayName, doc string, inputs []node.InputSpec, resultType string) node.Spec {
	return node.Spec{
		Kind:        kind,
		Version:     1,
		Category:    "PureFunc",
		DisplayName: displayName,
		Description: doc,
		Inputs:      inputs,
		Outputs: []node.OutputSpec{
			{Name: "result", Type: resultType, DisplayName: "结果"},
		},
		IsPureData: true,
	}
}

// numIn 2 个 number 输入 (a, b).
func numIn() []node.InputSpec {
	return []node.InputSpec{
		{Name: "a", Type: "Number", Default: 0.0, DisplayName: "A", Widget: node.WidgetSpec{Kind: "number"}},
		{Name: "b", Type: "Number", Default: 0.0, DisplayName: "B", Widget: node.WidgetSpec{Kind: "number"}},
	}
}

// anyIn 2 个 wildcard 输入.
func anyIn() []node.InputSpec {
	return []node.InputSpec{
		{Name: "a", Type: "*", DisplayName: "A"},
		{Name: "b", Type: "*", DisplayName: "B"},
	}
}

// strIn 2 个 string 输入.
func strIn() []node.InputSpec {
	return []node.InputSpec{
		{Name: "a", Type: "String", Default: "", DisplayName: "A", Widget: node.WidgetSpec{Kind: "text"}},
		{Name: "b", Type: "String", Default: "", DisplayName: "B", Widget: node.WidgetSpec{Kind: "text"}},
	}
}

// boolIn 2 个 bool 输入.
func boolIn() []node.InputSpec {
	return []node.InputSpec{
		{Name: "a", Type: "Bool", Default: false, DisplayName: "A", Widget: node.WidgetSpec{Kind: "checkbox"}},
		{Name: "b", Type: "Bool", Default: false, DisplayName: "B", Widget: node.WidgetSpec{Kind: "checkbox"}},
	}
}

// stubRun 所有节点共用 — 返 errPureDataNotEvaluatable.
// 不抽 base class — 各节点 Run 方法显式调这个 func.
func stubRun() (node.Outputs, error) { return nil, errPureDataNotEvaluatable }

func init() {
	for _, n := range []node.Node{
		// 算术 (6)
		&Add{}, &Sub{}, &Mul{}, &Div{}, &Mod{}, &Neg{},
		// 比较 (6)
		&Lt{}, &LtEq{}, &Gt{}, &GtEq{}, &Eq{}, &NotEq{},
		// 逻辑 (3)
		&And{}, &Or{}, &Not{},
		// 字符串 (3)
		&Concat{}, &Contains{}, &Length{},
		// 转换 (3)
		&ToString{}, &ToNumber{}, &ToBool{},
		// 三元 (1)
		&Select{},
		// Expr (1) — IsPureData, dynamic inputs Phase 5
		&Expr{},
	} {
		node.Register(n)
	}
}

// ===== 算术 (6) =====

type Add struct{}

func (Add) Spec() node.Spec {
	return specBuilder("Add", "加", "a + b", numIn(), "Number")
}
func (Add) Run(_ node.Ctx, _ node.Inputs) (node.Outputs, error) { return stubRun() }

type Sub struct{}

func (Sub) Spec() node.Spec {
	return specBuilder("Sub", "减", "a - b", numIn(), "Number")
}
func (Sub) Run(_ node.Ctx, _ node.Inputs) (node.Outputs, error) { return stubRun() }

type Mul struct{}

func (Mul) Spec() node.Spec {
	return specBuilder("Mul", "乘", "a * b", numIn(), "Number")
}
func (Mul) Run(_ node.Ctx, _ node.Inputs) (node.Outputs, error) { return stubRun() }

type Div struct{}

func (Div) Spec() node.Spec {
	return specBuilder("Div", "除", "a / b (b=0 → 0)", numIn(), "Number")
}
func (Div) Run(_ node.Ctx, _ node.Inputs) (node.Outputs, error) { return stubRun() }

type Mod struct{}

func (Mod) Spec() node.Spec {
	return specBuilder("Mod", "取模", "a mod b", numIn(), "Number")
}
func (Mod) Run(_ node.Ctx, _ node.Inputs) (node.Outputs, error) { return stubRun() }

type Neg struct{}

func (Neg) Spec() node.Spec {
	return specBuilder("Neg", "取负", "-x", []node.InputSpec{
		{Name: "x", Type: "Number", Default: 0.0, DisplayName: "X", Widget: node.WidgetSpec{Kind: "number"}},
	}, "Number")
}
func (Neg) Run(_ node.Ctx, _ node.Inputs) (node.Outputs, error) { return stubRun() }

// ===== 比较 (6) =====

type Lt struct{}

func (Lt) Spec() node.Spec {
	return specBuilder("Lt", "小于", "a < b", numIn(), "Bool")
}
func (Lt) Run(_ node.Ctx, _ node.Inputs) (node.Outputs, error) { return stubRun() }

type LtEq struct{}

func (LtEq) Spec() node.Spec {
	return specBuilder("LtEq", "小于等于", "a <= b", numIn(), "Bool")
}
func (LtEq) Run(_ node.Ctx, _ node.Inputs) (node.Outputs, error) { return stubRun() }

type Gt struct{}

func (Gt) Spec() node.Spec {
	return specBuilder("Gt", "大于", "a > b", numIn(), "Bool")
}
func (Gt) Run(_ node.Ctx, _ node.Inputs) (node.Outputs, error) { return stubRun() }

type GtEq struct{}

func (GtEq) Spec() node.Spec {
	return specBuilder("GtEq", "大于等于", "a >= b", numIn(), "Bool")
}
func (GtEq) Run(_ node.Ctx, _ node.Inputs) (node.Outputs, error) { return stubRun() }

type Eq struct{}

func (Eq) Spec() node.Spec {
	return specBuilder("Eq", "等于", "a == b (wildcard)", anyIn(), "Bool")
}
func (Eq) Run(_ node.Ctx, _ node.Inputs) (node.Outputs, error) { return stubRun() }

type NotEq struct{}

func (NotEq) Spec() node.Spec {
	return specBuilder("NotEq", "不等于", "a != b (wildcard)", anyIn(), "Bool")
}
func (NotEq) Run(_ node.Ctx, _ node.Inputs) (node.Outputs, error) { return stubRun() }

// ===== 逻辑 (3) =====

type And struct{}

func (And) Spec() node.Spec {
	s := specBuilder("And", "逻辑与", "a && b", boolIn(), "Bool")
	// And default 跟老版本一致: true, true (短路初始化)
	for i := range s.Inputs {
		s.Inputs[i].Default = true
	}
	return s
}
func (And) Run(_ node.Ctx, _ node.Inputs) (node.Outputs, error) { return stubRun() }

type Or struct{}

func (Or) Spec() node.Spec {
	return specBuilder("Or", "逻辑或", "a || b", boolIn(), "Bool")
}
func (Or) Run(_ node.Ctx, _ node.Inputs) (node.Outputs, error) { return stubRun() }

type Not struct{}

func (Not) Spec() node.Spec {
	return specBuilder("Not", "逻辑非", "!x", []node.InputSpec{
		{Name: "x", Type: "Bool", Default: false, DisplayName: "X", Widget: node.WidgetSpec{Kind: "checkbox"}},
	}, "Bool")
}
func (Not) Run(_ node.Ctx, _ node.Inputs) (node.Outputs, error) { return stubRun() }

// ===== 字符串 (3) =====

type Concat struct{}

func (Concat) Spec() node.Spec {
	return specBuilder("Concat", "拼接", "a + b (字符串)", strIn(), "String")
}
func (Concat) Run(_ node.Ctx, _ node.Inputs) (node.Outputs, error) { return stubRun() }

type Contains struct{}

func (Contains) Spec() node.Spec {
	return specBuilder("Contains", "包含", "haystack 含 needle", []node.InputSpec{
		{Name: "haystack", Type: "String", Default: "", DisplayName: "源串", Widget: node.WidgetSpec{Kind: "text"}},
		{Name: "needle", Type: "String", Default: "", DisplayName: "子串", Widget: node.WidgetSpec{Kind: "text"}},
	}, "Bool")
}
func (Contains) Run(_ node.Ctx, _ node.Inputs) (node.Outputs, error) { return stubRun() }

type Length struct{}

func (Length) Spec() node.Spec {
	return specBuilder("Length", "字符串长度", "len(s)", []node.InputSpec{
		{Name: "s", Type: "String", Default: "", DisplayName: "字符串", Widget: node.WidgetSpec{Kind: "text"}},
	}, "Number")
}
func (Length) Run(_ node.Ctx, _ node.Inputs) (node.Outputs, error) { return stubRun() }

// ===== 转换 (3) =====

type ToString struct{}

func (ToString) Spec() node.Spec {
	return specBuilder("ToString", "转字符串", "fmt.Sprint(x)", []node.InputSpec{
		{Name: "x", Type: "*", DisplayName: "X"},
	}, "String")
}
func (ToString) Run(_ node.Ctx, _ node.Inputs) (node.Outputs, error) { return stubRun() }

type ToNumber struct{}

func (ToNumber) Spec() node.Spec {
	return specBuilder("ToNumber", "转数字", "strconv.ParseFloat(x) 失败 → 0", []node.InputSpec{
		{Name: "x", Type: "*", DisplayName: "X"},
	}, "Number")
}
func (ToNumber) Run(_ node.Ctx, _ node.Inputs) (node.Outputs, error) { return stubRun() }

type ToBool struct{}

func (ToBool) Spec() node.Spec {
	return specBuilder("ToBool", "转布尔", "truthy: != 0 / 非空 / true", []node.InputSpec{
		{Name: "x", Type: "*", DisplayName: "X"},
	}, "Bool")
}
func (ToBool) Run(_ node.Ctx, _ node.Inputs) (node.Outputs, error) { return stubRun() }

// ===== 三元 (1) =====

type Select struct{}

func (Select) Spec() node.Spec {
	return specBuilder("Select", "三元选择", "cond ? a : b", []node.InputSpec{
		{Name: "cond", Type: "Bool", Default: true, DisplayName: "条件", Widget: node.WidgetSpec{Kind: "checkbox"}},
		{Name: "a", Type: "*", DisplayName: "A (cond=true)"},
		{Name: "b", Type: "*", DisplayName: "B (cond=false)"},
	}, "*")
}
func (Select) Run(_ node.Ctx, _ node.Inputs) (node.Outputs, error) { return stubRun() }

// ===== Expr (1) — dynamic inputs Phase 5 =====
//
// 老 spec 用 DataInDynamicFn 按 config.inputs[] 动态生成 data-in pin.
// 新 framework MVP 不支持 dynamic pin (Non-goals). 此 stub 仅注册基本 Spec, FE 编辑器无法
// 真显示 inputs[]; Phase 5+ 单独 spec 加 dynamic pin framework support.
type Expr struct{}

func (Expr) Spec() node.Spec {
	return node.Spec{
		Kind:        "Expr",
		Version:     1,
		Category:    "PureFunc",
		DisplayName: "表达式",
		Description: "求值表达式. Phase 4 stub: dynamic inputs (config.inputs[]) Phase 5 加 framework support.",
		Inputs: []node.InputSpec{
			{Name: "expr", Type: "String", Default: "", Required: true,
				DisplayName: "表达式",
				Doc:         "Go-like 表达式. Phase 5 加 dynamic inputs[] 后可引用配置 input.",
				Widget: node.WidgetSpec{Kind: "textarea",
					Props: node.MarshalProps(node.TextareaProps{Rows: 3})}},
		},
		Outputs: []node.OutputSpec{
			{Name: "value", Type: "*", DisplayName: "值"},
		},
		IsPureData: true,
	}
}
func (Expr) Run(_ node.Ctx, _ node.Inputs) (node.Outputs, error) { return stubRun() }
