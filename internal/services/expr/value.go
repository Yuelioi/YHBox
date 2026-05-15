// Package expr 是 Container 节点 config 用的表达式引擎。Pratt parser + AST + Eval。
//
// 关键约束：字段访问限定在 $-rooted 路径里（Env.Get 收完整 dotted path），AST 只
// parse 变量路径 + 字面量 + 运算符；不支持任意表达式后接 .field。
// 不支持：lambda / 列表 / regex / 用户自定义函数。
package expr

import (
	"fmt"
	"strconv"
)

// Point 复合类型；$sys.lastTemplate.point 等返此结构。
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Value 是表达式 eval 结果。运行期类型 = float64 | bool | string | Point | nil。
// 类型转换 / 比较 / 算术 在 eval.go 里集中处理。
type Value = any

// AsNumber 软转 number。bool→0/1，string 尝试 strconv.ParseFloat；其它返 0 + ok=false。
func AsNumber(v Value) (float64, bool) {
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
		if f, err := strconv.ParseFloat(x, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

// AsBool null/0/""/false → false；其它 truthy。
func AsBool(v Value) bool {
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
	case Point:
		return true
	}
	return true
}

// FormatValue 给 Log / Toast 节点用的字符串化。
func FormatValue(v Value) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case bool:
		if x {
			return "true"
		}
		return "false"
	case float64:
		return fmt.Sprintf("%g", x)
	case string:
		return x
	case Point:
		return fmt.Sprintf("(%g,%g)", x.X, x.Y)
	}
	return fmt.Sprintf("%v", v)
}
