package node

import (
	"reflect"
	"testing"
)

func TestCoerceInputValue_PointMap(t *testing.T) {
	got := CoerceInputValue(map[string]any{"x": 12.0, "y": 34.0, "unit": "px"}, "Point")
	want := Point{X: 12, Y: 34, Unit: UnitPx}
	if got != want {
		t.Fatalf("CoerceInputValue Point = %#v, want %#v", got, want)
	}
}

func TestCoerceInputValue_GeometryMap(t *testing.T) {
	got := CoerceInputValue(map[string]any{
		"pct": map[string]any{"x": 0.1, "y": 0.2, "w": 0.3, "h": 0.4},
	}, "Geometry")
	want := Geometry{Pct: Rect{X: 0.1, Y: 0.2, W: 0.3, H: 0.4}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CoerceInputValue Geometry = %#v, want %#v", got, want)
	}
}

func TestCoerceInputValue_StringPassthrough(t *testing.T) {
	if got := CoerceInputValue("template-guid", "String"); got != "template-guid" {
		t.Fatalf("String passthrough = %#v", got)
	}
}
