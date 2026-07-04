package node

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestWindowTypeRegistered(t *testing.T) {
	found := false
	for _, ts := range AllTypes() {
		if ts.Tag == "Window" {
			found = true
			if ts.Color != "#22d3ee" || ts.WidgetKind != "preview" {
				t.Fatalf("Window TypeSpec 错: %+v", ts)
			}
		}
	}
	if !found {
		t.Fatal("Window 类型未注册")
	}
}

func TestJSONTypeRegisteredAsAny(t *testing.T) {
	found := false
	for _, ts := range AllTypes() {
		if ts.Tag == "JSON" {
			found = true
			if ts.GoType != "any" {
				t.Fatalf("JSON GoType = %q, want any", ts.GoType)
			}
			if ts.WidgetKind != "json" {
				t.Fatalf("JSON WidgetKind = %q, want json", ts.WidgetKind)
			}
		}
	}
	if !found {
		t.Fatal("JSON 类型未注册")
	}
}

func TestInputsWindow(t *testing.T) {
	w := Window{HWND: 123, Title: "记事本"}
	in := newInputs(map[string]any{"W": w}, nil, nil, nil)
	got, ok := in.Window("W")
	if !ok || got.HWND != 123 {
		t.Fatalf("Window 取值失败: %v %v", got, ok)
	}
	if _, ok := in.Window("missing"); ok {
		t.Fatal("缺失 pin 应返 false")
	}
}

func TestInputsJSONValue(t *testing.T) {
	arr := []any{"a", float64(1)}
	obj := map[string]any{"ok": true}
	in := newInputs(map[string]any{
		"Arr":    arr,
		"Obj":    obj,
		"Scalar": "x",
	}, nil, nil, nil)

	if got := in.JSONValue("Arr"); len(got.([]any)) != 2 {
		t.Fatalf("JSONValue array = %#v, want original array", got)
	}
	if got := in.JSON("Obj"); got["ok"] != true {
		t.Fatalf("JSON object helper = %#v, want map", got)
	}
	if got := in.JSON("Arr"); got != nil {
		t.Fatalf("JSON object helper for array = %#v, want nil", got)
	}
	if got := in.JSONValue("Scalar"); got != "x" {
		t.Fatalf("JSONValue scalar = %#v, want x", got)
	}
}

func TestGeometry_JSONRoundTrip(t *testing.T) {
	g := Geometry{
		Pct: Rect{X: 0.125, Y: 0.8, W: 0.75, H: 0.06},
		Overrides: []GeoOverride{{
			Resolution: Resolution{W: 1920, H: 1080},
			Px:         PixelRect{X: 240, Y: 864, W: 1440, H: 65},
		}},
	}
	b, _ := json.Marshal(g)
	if !strings.Contains(string(b), `"resolution":{"w":1920,"h":1080}`) {
		t.Fatalf("resolution 必须小写 w/h, got %s", b)
	}
	if !strings.Contains(string(b), `"px":{"x":240,"y":864,"w":1440,"h":65}`) {
		t.Fatalf("px 小写, got %s", b)
	}
	var g2 Geometry
	if err := json.Unmarshal(b, &g2); err != nil || g2.Overrides[0].Resolution.W != 1920 {
		t.Fatalf("round-trip 失败: %v %+v", err, g2)
	}
}
