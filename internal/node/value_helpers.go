// value_helpers.go — 跨包共享的宽松值语义 (Eq/NotEq/ListContains/Join/Concat/ToString 共用).
// 从 purefunc 提升 (specs/2026-06-10-collection-nodes.md A.4). 行为唯一修正: 不可比类型防护
// (A' 审计: 同类型 slice/map 直比是运行时 panic, 被 EvaluatePureData recover 吞成静默 nil).
package node

import (
	"fmt"
	"reflect"
	"strconv"
)

// FormatValue 软转任意值为 string. 与 expr 包数值/Point 比较是两套语义, 勿混.
func FormatValue(v any) string {
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

// LooseEqual 宽松等值 — 同类型直比, 跨类型 FormatValue 串比.
// 同类型但不可比 (slice/map) → FormatValue 串比 (直比 panic, 防护性退化).
// nil 语义: LooseEqual(nil,nil)=true; LooseEqual(nil,"")=false (FormatValue(nil)="null").
func LooseEqual(a, b any) bool {
	if looseSameType(a, b) {
		if a == nil || reflect.TypeOf(a).Comparable() {
			return a == b
		}
		return FormatValue(a) == FormatValue(b)
	}
	return FormatValue(a) == FormatValue(b)
}

func looseSameType(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return fmt.Sprintf("%T", a) == fmt.Sprintf("%T", b)
}
