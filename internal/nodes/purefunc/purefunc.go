// Package purefunc 22 个纯函数节点 (Add/Sub/.../Select) + Expr.
// 全是 IsPureData=true, 全部实现 node.Evaluator (EvaluatePureData 入口).
// Expr 节点定义在 expr.go (dynamic inputs 用 Inputs.Keys() 遍历).
//
// 设计取舍: 22 节点用同一 spec shape (1 data-out "result"), 用 specBuilder 复用构造代码.
// 每节点仍独立 Go type — 符合 "1 type 1 node" 原则, builder 只复用 Spec 字段填充不引入抽象层.
package purefunc

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"yhbox/internal/node"
)

// specBuilder 构造单 Result 数据出口的 pure-data Spec. pin name 用 PascalCase.
// 展示文案 (名/描述) 由 FE i18n 持有 (node.<kind>.*), 这里只出结构.
func specBuilder(kind string, inputs []node.InputSpec, resultType string) node.Spec {
	return node.Spec{
		Kind:     kind,
		Category: "PureFunc",
		Inputs:   inputs,
		Outputs: []node.OutputSpec{
			{Name: "Result", Type: resultType},
		},
		IsPureData: true,
	}
}

// numIn 2 个 number 输入 (A, B).
func numIn() []node.InputSpec {
	return []node.InputSpec{
		{Name: "A", Type: "Number", Default: json.Number("0"), Widget: node.WidgetSpec{Kind: "number"}},
		{Name: "B", Type: "Number", Default: json.Number("0"), Widget: node.WidgetSpec{Kind: "number"}},
	}
}

// anyIn 2 个 wildcard 输入 (A, B).
func anyIn() []node.InputSpec {
	return []node.InputSpec{
		{Name: "A", Type: "*"},
		{Name: "B", Type: "*"},
	}
}

// strIn 2 个 string 输入 (A, B).
func strIn() []node.InputSpec {
	return []node.InputSpec{
		{Name: "A", Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "text"}},
		{Name: "B", Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "text"}},
	}
}

// boolIn 2 个 bool 输入 (A, B).
func boolIn() []node.InputSpec {
	return []node.InputSpec{
		{Name: "A", Type: "Bool", Default: false, Widget: node.WidgetSpec{Kind: "checkbox"}},
		{Name: "B", Type: "Bool", Default: false, Widget: node.WidgetSpec{Kind: "checkbox"}},
	}
}

// asNumber Number 软转 — Inputs.Float64 已处理 float64/int/json.Number/string; bool 软转 true→1/false→0 这里加.
func asNumber(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case bool:
		if x {
			return 1, true
		}
		return 0, true
	case string:
		// strconv 已在 Inputs.Float64 内部走过, 这里冗余但不会错.
		return 0, false
	}
	return 0, false
}

// asBool 软转 bool: nil/0/""/false → false; 其它 truthy.
func asBool(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case float64:
		return x != 0
	case int:
		return x != 0
	case int64:
		return x != 0
	case string:
		return x != ""
	}
	return true
}

// formatValue 软转 string (Log/Concat 用).
func formatValue(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case bool:
		if x {
			return "true"
		}
		return "false"
	case float64:
		return strconv.FormatFloat(x, 'g', -1, 64)
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case string:
		return x
	}
	return fmt.Sprintf("%v", v)
}

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
	} {
		node.Register(n)
	}
}

// ===== 算术 (6) =====

type Add struct{}

func (Add) Spec() node.Spec {
	return specBuilder("Add", numIn(), "Number")
}
func (Add) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return in.Float64("A") + in.Float64("B"), nil
}

type Sub struct{}

func (Sub) Spec() node.Spec {
	return specBuilder("Sub", numIn(), "Number")
}
func (Sub) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return in.Float64("A") - in.Float64("B"), nil
}

type Mul struct{}

func (Mul) Spec() node.Spec {
	return specBuilder("Mul", numIn(), "Number")
}
func (Mul) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return in.Float64("A") * in.Float64("B"), nil
}

type Div struct{}

func (Div) Spec() node.Spec {
	return specBuilder("Div", numIn(), "Number")
}
func (Div) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	b := in.Float64("B")
	if b == 0 {
		return math.NaN(), nil
	}
	return in.Float64("A") / b, nil
}

type Mod struct{}

func (Mod) Spec() node.Spec {
	return specBuilder("Mod", numIn(), "Number")
}
func (Mod) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return math.Mod(in.Float64("A"), in.Float64("B")), nil
}

type Neg struct{}

func (Neg) Spec() node.Spec {
	return specBuilder("Neg", []node.InputSpec{
		{Name: "X", Type: "Number", Default: json.Number("0"), Widget: node.WidgetSpec{Kind: "number"}},
	}, "Number")
}
func (Neg) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return -in.Float64("X"), nil
}

// ===== 比较 (6) =====

type Lt struct{}

func (Lt) Spec() node.Spec {
	return specBuilder("Lt", numIn(), "Bool")
}
func (Lt) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return in.Float64("A") < in.Float64("B"), nil
}

type LtEq struct{}

func (LtEq) Spec() node.Spec {
	return specBuilder("LtEq", numIn(), "Bool")
}
func (LtEq) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return in.Float64("A") <= in.Float64("B"), nil
}

type Gt struct{}

func (Gt) Spec() node.Spec {
	return specBuilder("Gt", numIn(), "Bool")
}
func (Gt) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return in.Float64("A") > in.Float64("B"), nil
}

type GtEq struct{}

func (GtEq) Spec() node.Spec {
	return specBuilder("GtEq", numIn(), "Bool")
}
func (GtEq) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return in.Float64("A") >= in.Float64("B"), nil
}

type Eq struct{}

func (Eq) Spec() node.Spec {
	return specBuilder("Eq", anyIn(), "Bool")
}
func (Eq) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return equalAny(in.Raw("A"), in.Raw("B")), nil
}

type NotEq struct{}

func (NotEq) Spec() node.Spec {
	return specBuilder("NotEq", anyIn(), "Bool")
}
func (NotEq) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return !equalAny(in.Raw("A"), in.Raw("B")), nil
}

// equalAny same-type direct compare; cross-type via formatValue (镜像老 evalEq).
func equalAny(a, b any) bool {
	if sameType(a, b) {
		return a == b
	}
	return formatValue(a) == formatValue(b)
}

func sameType(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return typeNameOf(a) == typeNameOf(b)
}

// typeNameOf 用 fmt.Sprintf("%T", v) 取类型名. 仅用于 sameType.
func typeNameOf(v any) string {
	return fmt.Sprintf("%T", v)
}

// ===== 逻辑 (3) =====

type And struct{}

func (And) Spec() node.Spec {
	s := specBuilder("And", boolIn(), "Bool")
	// And 输入默认 true (恒等元, 未接线的输入不影响结果)
	for i := range s.Inputs {
		s.Inputs[i].Default = true
	}
	return s
}
func (And) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	if !asBool(in.Raw("A")) {
		return false, nil
	}
	return asBool(in.Raw("B")), nil
}

type Or struct{}

func (Or) Spec() node.Spec {
	return specBuilder("Or", boolIn(), "Bool")
}
func (Or) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	if asBool(in.Raw("A")) {
		return true, nil
	}
	return asBool(in.Raw("B")), nil
}

type Not struct{}

func (Not) Spec() node.Spec {
	return specBuilder("Not", []node.InputSpec{
		{Name: "X", Type: "Bool", Default: false, Widget: node.WidgetSpec{Kind: "checkbox"}},
	}, "Bool")
}
func (Not) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return !asBool(in.Raw("X")), nil
}

// ===== 字符串 (3) =====

type Concat struct{}

func (Concat) Spec() node.Spec {
	return specBuilder("Concat", strIn(), "String")
}
func (Concat) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return formatValue(in.Raw("A")) + formatValue(in.Raw("B")), nil
}

type Contains struct{}

func (Contains) Spec() node.Spec {
	return specBuilder("Contains", []node.InputSpec{
		{Name: "Haystack", Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "text"}},
		{Name: "Needle", Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "text"}},
	}, "Bool")
}
func (Contains) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return strings.Contains(formatValue(in.Raw("Haystack")), formatValue(in.Raw("Needle"))), nil
}

type Length struct{}

func (Length) Spec() node.Spec {
	return specBuilder("Length", []node.InputSpec{
		{Name: "S", Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "text"}},
	}, "Number")
}
func (Length) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return float64(len(formatValue(in.Raw("S")))), nil
}

// ===== 转换 (3) =====

type ToString struct{}

func (ToString) Spec() node.Spec {
	return specBuilder("ToString", []node.InputSpec{
		{Name: "X", Type: "*"},
	}, "String")
}
func (ToString) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return formatValue(in.Raw("X")), nil
}

type ToNumber struct{}

func (ToNumber) Spec() node.Spec {
	return specBuilder("ToNumber", []node.InputSpec{
		{Name: "X", Type: "*"},
	}, "Number")
}
func (ToNumber) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	if f := in.Float64("X"); f != 0 {
		return f, nil
	}
	if f, ok := asNumber(in.Raw("X")); ok {
		return f, nil
	}
	return float64(0), nil
}

type ToBool struct{}

func (ToBool) Spec() node.Spec {
	return specBuilder("ToBool", []node.InputSpec{
		{Name: "X", Type: "*"},
	}, "Bool")
}
func (ToBool) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return asBool(in.Raw("X")), nil
}

// ===== 三元 (1) =====

type Select struct{}

func (Select) Spec() node.Spec {
	return specBuilder("Select", []node.InputSpec{
		{Name: "Cond", Type: "Bool", Default: true, Widget: node.WidgetSpec{Kind: "checkbox"}},
		{Name: "A", Type: "*"},
		{Name: "B", Type: "*"},
	}, "*")
}
func (Select) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	if asBool(in.Raw("Cond")) {
		return in.Raw("A"), nil
	}
	return in.Raw("B"), nil
}
