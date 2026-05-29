package runtime

import (
	"testing"

	nodepkg "yhbox/internal/node"
	"yhbox/internal/services/expr"
)

func TestCoerceToType_Rect(t *testing.T) {
	// map {x,y,w,h} → node.Rect (画布 rect-editor / 屏幕拾取 literal 形态)
	got := coerceToType(map[string]any{"x": 0.1, "y": 0.2, "w": 0.3, "h": 0.4}, "Rect")
	r, ok := got.(nodepkg.Rect)
	if !ok || r != (nodepkg.Rect{X: 0.1, Y: 0.2, W: 0.3, H: 0.4}) {
		t.Fatalf("Rect from map = %#v (ok=%v), want {0.1,0.2,0.3,0.4}", got, ok)
	}
	// []any len>=4 → node.Rect
	if r2, ok := coerceToType([]any{1.0, 2.0, 3.0, 4.0}, "Rect").(nodepkg.Rect); !ok || r2 != (nodepkg.Rect{X: 1, Y: 2, W: 3, H: 4}) {
		t.Fatalf("Rect from array wrong: %#v", r2)
	}
}

func TestCoerceToType_Point(t *testing.T) {
	// expr.Point → node.Point (toExprValue 已把纯 {x,y} 转 expr.Point)
	if p, ok := coerceToType(expr.Point{X: 0.5, Y: 0.6}, "Point").(nodepkg.Point); !ok || p != (nodepkg.Point{X: 0.5, Y: 0.6}) {
		t.Fatalf("Point from expr.Point wrong: %#v", p)
	}
	// map {x,y} → node.Point
	if p, ok := coerceToType(map[string]any{"x": 0.1, "y": 0.2}, "Point").(nodepkg.Point); !ok || p != (nodepkg.Point{X: 0.1, Y: 0.2}) {
		t.Fatalf("Point from map wrong: %#v", p)
	}
}

func TestCoerceToType_ScalarPassthrough(t *testing.T) {
	if coerceToType("hello", "String") != "hello" {
		t.Error("String 应原样返")
	}
	if coerceToType(5.0, "Number") != 5.0 {
		t.Error("Number 应原样返")
	}
}

// 关键回归: {x,y,w,h} 不能被 toExprValue 当 Point (会丢 w/h → Rect coerce 拿不到 w/h)。
func TestToExprValue_RectMapNotMangledToPoint(t *testing.T) {
	if _, isPoint := toExprValue(map[string]any{"x": 0.1, "y": 0.2, "w": 0.3, "h": 0.4}).(expr.Point); isPoint {
		t.Fatal("{x,y,w,h} 被误判成 expr.Point (丢 w/h)")
	}
	if _, isPoint := toExprValue(map[string]any{"x": 0.1, "y": 0.2}).(expr.Point); !isPoint {
		t.Error("纯 {x,y} 应转 expr.Point")
	}
}
