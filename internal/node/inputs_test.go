package node

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"
	"time"
)

func TestInputs_PriorityOrder(t *testing.T) {
	in := newInputs(
		map[string]any{"X": "from-wire"},
		map[string]any{"X": "from-config", "Y": "from-config"},
		map[string]any{"X": "from-exec", "Y": "from-exec", "Z": "from-exec"},
		map[string]any{"X": "default", "Y": "default", "Z": "default", "W": "default"},
	)
	if in.String("X") != "from-wire" {
		t.Errorf("X = %q, want from-wire", in.String("X"))
	}
	if in.String("Y") != "from-config" {
		t.Errorf("Y = %q, want from-config", in.String("Y"))
	}
	if in.String("Z") != "from-exec" {
		t.Errorf("Z = %q, want from-exec", in.String("Z"))
	}
	if in.String("W") != "default" {
		t.Errorf("W = %q, want default", in.String("W"))
	}
}

func TestInputs_Has(t *testing.T) {
	in := newInputs(nil, map[string]any{"X": 1}, nil, nil)
	if !in.Has("X") {
		t.Error("Has(X) should be true")
	}
	if in.Has("Y") {
		t.Error("Has(Y) should be false")
	}
}

func TestInputs_TypeCast(t *testing.T) {
	in := newInputs(nil, nil, nil, map[string]any{"i": 42, "f": 3.14})
	if in.Float64("i") != 42.0 {
		t.Error("int→float cast failed")
	}
	if in.Int("f") != 3 {
		t.Error("float→int cast failed")
	}
}

func TestInputs_Geometry(t *testing.T) {
	want := Geometry{Pct: Rect{X: 0.1, Y: 0.2, W: 0.5, H: 0.5}}
	in := newInputs(nil, map[string]any{"ROI": want}, nil, nil)
	got := in.Geometry("ROI")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Geometry(ROI) = %+v, want %+v", got, want)
	}
	// 缺失 → 零值
	if z := in.Geometry("missing"); !reflect.DeepEqual(z, Geometry{}) {
		t.Fatalf("missing Geometry should be zero, got %+v", z)
	}
}

func TestInputs_JsonNumberCoercion(t *testing.T) {
	// InputSpec.Default 用 json.Number 保精度 (DS r2 #9a). Float64/Int 必须能解.
	in := newInputs(nil, nil, nil, map[string]any{
		"th":  json.Number("0.85"),
		"cnt": json.Number("42"),
	})
	if in.Float64("th") != 0.85 {
		t.Errorf("Float64(json.Number 0.85) = %v, want 0.85", in.Float64("th"))
	}
	if in.Int("cnt") != 42 {
		t.Errorf("Int(json.Number 42) = %d, want 42", in.Int("cnt"))
	}
	// String fallback (e.g. config value 来自 JSON deserialize 是 string)
	in2 := newInputs(nil, map[string]any{"x": "3.14"}, nil, nil)
	if in2.Float64("x") != 3.14 {
		t.Errorf("Float64(string '3.14') = %v, want 3.14", in2.Float64("x"))
	}
}

func TestInputs_List(t *testing.T) {
	cases := []struct {
		name string
		val  any
		want []any
	}{
		{"any_slice", []any{1.0, "a"}, []any{1.0, "a"}},
		{"string_slice", []string{"a", "b"}, []any{"a", "b"}},
		{"nil", nil, nil},
		{"bare_string_not_list", "a,b", nil}, // 与 StringList 区别: 不把裸 string 当一元列表
		{"number_not_list", 3.14, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := NewInputsFromConfig(map[string]any{"literal": map[string]any{"X": tc.val}})
			got := in.List("X")
			if len(got) != len(tc.want) {
				t.Fatalf("len = %d, want %d (%v)", len(got), len(tc.want), got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("[%d] = %v, want %v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestNewInputsFromConfig_LiteralOverridesTopLevelAndHidesMetaKey(t *testing.T) {
	in := NewInputsFromConfig(map[string]any{
		"X":       "top-level",
		"Y":       "top-level-only",
		"literal": map[string]any{"X": "literal", "Z": "literal-only"},
	})
	if got := in.String("X"); got != "literal" {
		t.Fatalf("X = %q, want literal", got)
	}
	if got := in.String("Y"); got != "top-level-only" {
		t.Fatalf("Y = %q", got)
	}
	if got := in.String("Z"); got != "literal-only" {
		t.Fatalf("Z = %q", got)
	}
	if in.Has("literal") {
		t.Fatal("literal meta key leaked into runtime inputs")
	}
}

func TestInputs_StringListAcceptsJSONAndConfigShapes(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want []string
	}{
		{name: "string slice", in: []string{"a", "b"}, want: []string{"a", "b"}},
		{name: "json slice filters non-strings and empty strings", in: []any{"a", "", 3.0, "b"}, want: []string{"a", "b"}},
		{name: "single string", in: "a", want: []string{"a"}},
		{name: "empty string", in: "", want: nil},
		{name: "unsupported", in: 3, want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := NewInputsFromConfig(map[string]any{"Items": tt.in})
			if got := in.StringList("Items"); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("StringList = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestInputs_DomainAndScalarAccessors(t *testing.T) {
	point := Point{X: 0.25, Y: 0.75}
	rect := Rect{X: 0.1, Y: 0.2, W: 0.3, H: 0.4}
	color := Color{H: 120, S: 80, V: 90}
	in := newInputs(nil, map[string]any{
		"bool":       true,
		"point":      point,
		"rect":       rect,
		"color":      color,
		"int":        "42",
		"intDecimal": json.Number("42.9"),
		"badInt":     "forty-two",
		"badFloat":   "three-point-one-four",
	}, nil, nil)

	if !in.Bool("bool") || in.Bool("missing") {
		t.Fatal("Bool did not preserve true or zero a missing input")
	}
	if got := in.Point("point"); got != point {
		t.Fatalf("Point = %#v", got)
	}
	if got := in.Point("missing"); got != (Point{}) {
		t.Fatalf("missing Point = %#v", got)
	}
	if got := in.Rect("rect"); got != rect {
		t.Fatalf("Rect = %#v", got)
	}
	if got := in.Rect("missing"); got != (Rect{}) {
		t.Fatalf("missing Rect = %#v", got)
	}
	if got := in.Color("color"); got != color {
		t.Fatalf("Color = %#v", got)
	}
	if got := in.Color("missing"); got != (Color{}) {
		t.Fatalf("missing Color = %#v", got)
	}
	if got := in.Int("int"); got != 42 {
		t.Fatalf("Int(string) = %d", got)
	}
	if got := in.Int("intDecimal"); got != 42 {
		t.Fatalf("Int(decimal json.Number) = %d", got)
	}
	if got := in.Int("badInt"); got != 0 {
		t.Fatalf("Int(invalid) = %d", got)
	}
	if got := in.Float64("badFloat"); got != 0 {
		t.Fatalf("Float64(invalid) = %v", got)
	}
}

func TestInputs_DurationAcceptsSerializedForms(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want time.Duration
	}{
		{name: "duration", in: 2 * time.Second, want: 2 * time.Second},
		{name: "float milliseconds", in: 12.5, want: 12 * time.Millisecond},
		{name: "int milliseconds", in: 25, want: 25 * time.Millisecond},
		{name: "integer json number", in: json.Number("30"), want: 30 * time.Millisecond},
		{name: "decimal json number", in: json.Number("30.9"), want: 30 * time.Millisecond},
		{name: "duration string", in: "1.5s", want: 1500 * time.Millisecond},
		{name: "millisecond string", in: "250", want: 250 * time.Millisecond},
		{name: "invalid string", in: "later", want: 0},
		{name: "unsupported", in: struct{}{}, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := NewInputsFromConfig(map[string]any{"Delay": tt.in})
			if got := in.Duration("Delay"); got != tt.want {
				t.Fatalf("Duration(%#v) = %s, want %s", tt.in, got, tt.want)
			}
		})
	}
}

func TestInputs_KeysDescribeMaterializedInputs(t *testing.T) {
	in := NewInputsFromConfig(map[string]any{
		"B":       2,
		"literal": map[string]any{"A": 1, "C": 3},
	})
	keys := in.Keys()
	sort.Strings(keys)
	if !reflect.DeepEqual(keys, []string{"A", "B", "C"}) {
		t.Fatalf("Keys = %v", keys)
	}
}

func TestListTypeRegistered(t *testing.T) {
	for _, ts := range AllTypes() {
		if ts.Tag == "List" {
			if ts.GoType != "[]any" || ts.Color != "#818cf8" || ts.WidgetKind != "list-preview" {
				t.Fatalf("List TypeSpec wrong: %+v", ts)
			}
			return
		}
	}
	t.Fatal("List type not registered")
}
