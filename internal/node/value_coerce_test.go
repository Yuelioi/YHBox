package node

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestCoerceInputMap_CopiesAndCoercesDeclaredDomainPins(t *testing.T) {
	spec := &Spec{Inputs: []InputSpec{
		{Name: "In", Type: TypeExec},
		{Name: "Point", Type: "Point"},
		{Name: "ROI", Type: "Rect"},
		{Name: "Color", Type: "Color"},
		{Name: "Count", Type: "Integer"},
	}}
	originalPoint := map[string]any{"x": 12, "y": json.Number("34"), "unit": "px"}
	values := map[string]any{
		"In":      "exec-token",
		"Point":   originalPoint,
		"ROI":     []any{0.1, 0.2, 0.3, 0.4},
		"Color":   map[string]any{"h": json.Number("120"), "s": 80.0, "v": "90"},
		"Count":   json.Number("3"),
		"Unknown": "preserved",
	}

	got := CoerceInputMap(spec, values)
	if got == nil {
		t.Fatal("CoerceInputMap returned nil")
	}
	if got["Point"] != (Point{X: 12, Y: 34, Unit: UnitPx}) {
		t.Fatalf("Point = %#v", got["Point"])
	}
	if got["ROI"] != (Rect{X: 0.1, Y: 0.2, W: 0.3, H: 0.4}) {
		t.Fatalf("ROI = %#v", got["ROI"])
	}
	if got["Color"] != (Color{H: 120, S: 80, V: 90}) {
		t.Fatalf("Color = %#v", got["Color"])
	}
	if got["In"] != "exec-token" || got["Count"] != json.Number("3") || got["Unknown"] != "preserved" {
		t.Fatalf("non-domain values changed: %#v", got)
	}
	if _, ok := values["Point"].(map[string]any); !ok {
		t.Fatal("CoerceInputMap mutated the caller's map")
	}
	originalPoint["x"] = 99
	if got["Point"].(Point).X != 12 {
		t.Fatal("coerced result still aliases the source point map")
	}
}

func TestCoerceInputMap_NoWorkReturnsOriginalMap(t *testing.T) {
	values := map[string]any{"x": 1}
	if got := CoerceInputMap(nil, values); got["x"] != 1 {
		t.Fatalf("nil spec result = %#v", got)
	}
	if got := CoerceInputMap(&Spec{}, nil); got != nil {
		t.Fatalf("nil values result = %#v", got)
	}
}

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

func TestCoerceInputValue_DomainShapesAndInvalidFallbacks(t *testing.T) {
	tests := []struct {
		name string
		in   any
		typ  string
		want any
	}{
		{name: "typed rect", in: Rect{X: 1, Y: 2, W: 3, H: 4}, typ: "Rect", want: Rect{X: 1, Y: 2, W: 3, H: 4}},
		{name: "rect map numeric variants", in: map[string]any{"x": float32(1.5), "y": int64(2), "w": "3.5", "h": json.Number("4.5")}, typ: "Rect", want: Rect{X: 1.5, Y: 2, W: 3.5, H: 4.5}},
		{name: "short rect unchanged", in: []any{1, 2, 3}, typ: "Rect", want: []any{1, 2, 3}},
		{name: "typed point", in: Point{X: 1, Y: 2}, typ: "Point", want: Point{X: 1, Y: 2}},
		{name: "point slice", in: []any{json.Number("1.25"), int64(2)}, typ: "Point", want: Point{X: 1.25, Y: 2}},
		{name: "point map without x unchanged", in: map[string]any{"y": 2}, typ: "Point", want: map[string]any{"y": 2}},
		{name: "typed geometry", in: Geometry{Pct: Rect{W: 1, H: 1}}, typ: "Geometry", want: Geometry{Pct: Rect{W: 1, H: 1}}},
		{name: "geometry marshal failure unchanged", in: make(chan int), typ: "Geometry", want: make(chan int)},
		{name: "geometry decode failure unchanged", in: "not-an-object", typ: "Geometry", want: "not-an-object"},
		{name: "typed color", in: Color{H: 1, S: 2, V: 3}, typ: "Color", want: Color{H: 1, S: 2, V: 3}},
		{name: "color fractional json number", in: map[string]any{"h": json.Number("12.9"), "s": int64(34), "v": 56}, typ: "Color", want: Color{H: 12, S: 34, V: 56}},
		{name: "invalid color unchanged", in: []any{1, 2, 3}, typ: "Color", want: []any{1, 2, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CoerceInputValue(tt.in, tt.typ)
			if tt.name == "geometry marshal failure unchanged" {
				if reflect.ValueOf(got).Pointer() != reflect.ValueOf(tt.in).Pointer() {
					t.Fatalf("fallback = %#v, want original channel", got)
				}
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("CoerceInputValue(%#v, %q) = %#v, want %#v", tt.in, tt.typ, got, tt.want)
			}
		})
	}
}

func TestCoerceNumericHelpers_InvalidValuesBecomeZero(t *testing.T) {
	if got := coerceFloat(struct{}{}); got != 0 {
		t.Fatalf("coerceFloat(unsupported) = %v", got)
	}
	if got := coerceFloat("not-a-number"); got != 0 {
		t.Fatalf("coerceFloat(invalid string) = %v", got)
	}
	if got := coerceInt(json.Number("not-a-number")); got != 0 {
		t.Fatalf("coerceInt(invalid json.Number) = %v", got)
	}
	if got := coerceInt("not-a-number"); got != 0 {
		t.Fatalf("coerceInt(invalid string) = %v", got)
	}
	if got := coerceInt(struct{}{}); got != 0 {
		t.Fatalf("coerceInt(unsupported) = %v", got)
	}
}
