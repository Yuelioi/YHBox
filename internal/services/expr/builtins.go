// builtins.go — 内置函数单一来源: evalCall 分发、validator 编辑期校验
// (EXPR_UNKNOWN_FUNCTION / EXPR_FN_ARITY)、前端补全元数据 parity 测试都对这张表.
package expr

import (
	"fmt"
	"maps"
	"math"
	"math/rand/v2"
	"slices"
	"strings"
	"time"
)

// Builtin 一个内置函数: arity 区间 + 签名展示串 + 实现. impl 收到的 args 已通过 arity gate.
type Builtin struct {
	MinArgs, MaxArgs int
	Sig              string // 补全下拉展示, 参数名通用英文 (跨 locale 不翻译)
	impl             func(args []Value, pos int) (Value, error)
}

// Builtins 返回函数表快照 (浅拷贝 — caller 改 map 不影响内部).
func Builtins() map[string]Builtin {
	out := make(map[string]Builtin, len(builtins))
	maps.Copy(out, builtins)
	return out
}

// FunctionSpec 函数元数据 DTO — NodeService.GetExprFunctions RPC 喂前端
// (补全下拉/未知函数检查), FE 不再手写函数表.
type FunctionSpec struct {
	Name    string `json:"name"`
	Sig     string `json:"sig"`
	MinArgs int    `json:"minArgs"`
	MaxArgs int    `json:"maxArgs"`
}

// Functions 按名排序的函数元数据列表.
func Functions() []FunctionSpec {
	out := make([]FunctionSpec, 0, len(builtins))
	for name, b := range builtins {
		out = append(out, FunctionSpec{Name: name, Sig: b.Sig, MinArgs: b.MinArgs, MaxArgs: b.MaxArgs})
	}
	slices.SortFunc(out, func(a, b FunctionSpec) int { return strings.Compare(a.Name, b.Name) })
	return out
}

// num1 包装单 number 参数函数.
func num1(name string, f func(float64) float64) func([]Value, int) (Value, error) {
	return func(args []Value, pos int) (Value, error) {
		x, ok := AsNumber(args[0])
		if !ok {
			return nil, fmt.Errorf("expr: %s() needs number at col %d", name, pos)
		}
		return f(x), nil
	}
}

// num2 包装双 number 参数函数.
func num2(name string, f func(a, b float64) float64) func([]Value, int) (Value, error) {
	return func(args []Value, pos int) (Value, error) {
		a, aok := AsNumber(args[0])
		b, bok := AsNumber(args[1])
		if !aok || !bok {
			return nil, fmt.Errorf("expr: %s() needs numbers at col %d", name, pos)
		}
		return f(a, b), nil
	}
}

var builtins = map[string]Builtin{
	"abs":   {1, 1, "abs(x)", num1("abs", math.Abs)},
	"floor": {1, 1, "floor(x)", num1("floor", math.Floor)},
	"ceil":  {1, 1, "ceil(x)", num1("ceil", math.Ceil)},
	"sqrt":  {1, 1, "sqrt(x)", num1("sqrt", math.Sqrt)},
	"min":   {2, 2, "min(a, b)", num2("min", math.Min)},
	"max":   {2, 2, "max(a, b)", num2("max", math.Max)},
	"pow":   {2, 2, "pow(x, y)", num2("pow", math.Pow)},
	"now": {0, 0, "now()", func(_ []Value, _ int) (Value, error) {
		return float64(time.Now().UnixMilli()), nil
	}},
	// rand/randint 非确定 — Expr Spec 必须挂 IsNonDeterministic, 让 per-dispatch
	// 记忆化保同帧多路径同值 (语义对齐 random 节点包).
	"rand": {0, 0, "rand()", func(_ []Value, _ int) (Value, error) {
		return rand.Float64(), nil
	}},
	// randint(min, max) — 整数闭区间, min>max 交换 (与 RandomInt 节点 uniform 路径一致).
	"randint": {2, 2, "randint(min, max)", func(args []Value, pos int) (Value, error) {
		a, aok := AsNumber(args[0])
		b, bok := AsNumber(args[1])
		if !aok || !bok {
			return nil, fmt.Errorf("expr: randint() needs numbers at col %d", pos)
		}
		lo, hi := int64(a), int64(b)
		if lo > hi {
			lo, hi = hi, lo
		}
		if lo == hi {
			return float64(lo), nil
		}
		return float64(lo + rand.Int64N(hi-lo+1)), nil
	}},
	// round(x) 取整 / round(x, digits) 带位数 — 与 Round 节点对齐, digits clamp [-15,15].
	"round": {1, 2, "round(x, digits?)", func(args []Value, pos int) (Value, error) {
		x, ok := AsNumber(args[0])
		if !ok {
			return nil, fmt.Errorf("expr: round() needs number at col %d", pos)
		}
		if len(args) == 1 {
			return math.Round(x), nil
		}
		d, ok := AsNumber(args[1])
		if !ok {
			return nil, fmt.Errorf("expr: round() digits needs number at col %d", pos)
		}
		d = math.Trunc(d)
		if d > 15 {
			d = 15
		}
		if d < -15 {
			d = -15
		}
		factor := math.Pow(10, d)
		return math.Round(x*factor) / factor, nil
	}},
	// clamp(x, lo, hi) — lo>hi 先交换, 与 Clamp 节点一致.
	"clamp": {3, 3, "clamp(x, min, max)", func(args []Value, pos int) (Value, error) {
		x, xok := AsNumber(args[0])
		lo, lok := AsNumber(args[1])
		hi, hok := AsNumber(args[2])
		if !xok || !lok || !hok {
			return nil, fmt.Errorf("expr: clamp() needs numbers at col %d", pos)
		}
		if lo > hi {
			lo, hi = hi, lo
		}
		switch {
		case x < lo:
			return lo, nil
		case x > hi:
			return hi, nil
		}
		return x, nil
	}},
}
