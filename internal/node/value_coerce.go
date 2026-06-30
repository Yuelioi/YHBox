package node

import (
	"encoding/json"
	"strconv"
)

// CoerceInputMap materializes script/JSON-shaped pin values into the Go domain
// types that Inputs.Point/Rect/Geometry/Color expect.
func CoerceInputMap(spec *Spec, values map[string]any) map[string]any {
	if spec == nil || len(values) == 0 {
		return values
	}
	out := make(map[string]any, len(values))
	for k, v := range values {
		out[k] = v
	}
	for _, in := range spec.Inputs {
		if in.Type == TypeExec {
			continue
		}
		v, ok := out[in.Name]
		if !ok {
			continue
		}
		out[in.Name] = CoerceInputValue(v, in.Type)
	}
	return out
}

// CoerceInputValue converts JSON/JS literal shapes into node domain types.
// Scalar pins are intentionally left as-is because Inputs already handles their
// loose conversions.
func CoerceInputValue(v any, typ string) any {
	switch typ {
	case "Rect":
		if r, ok := coerceRect(v); ok {
			return r
		}
	case "Point":
		if p, ok := coercePoint(v); ok {
			return p
		}
	case "Geometry":
		if g, ok := coerceGeometry(v); ok {
			return g
		}
	case "Color":
		if c, ok := coerceColor(v); ok {
			return c
		}
	}
	return v
}

func coerceRect(v any) (Rect, bool) {
	switch t := v.(type) {
	case Rect:
		return t, true
	case map[string]any:
		return Rect{X: coerceFloat(t["x"]), Y: coerceFloat(t["y"]), W: coerceFloat(t["w"]), H: coerceFloat(t["h"])}, true
	case []any:
		if len(t) >= 4 {
			return Rect{X: coerceFloat(t[0]), Y: coerceFloat(t[1]), W: coerceFloat(t[2]), H: coerceFloat(t[3])}, true
		}
	}
	return Rect{}, false
}

func coercePoint(v any) (Point, bool) {
	switch t := v.(type) {
	case Point:
		return t, true
	case map[string]any:
		if _, ok := t["x"]; ok {
			u, _ := t["unit"].(string)
			return Point{X: coerceFloat(t["x"]), Y: coerceFloat(t["y"]), Unit: PointUnit(u)}, true
		}
	case []any:
		if len(t) >= 2 {
			return Point{X: coerceFloat(t[0]), Y: coerceFloat(t[1])}, true
		}
	}
	return Point{}, false
}

func coerceGeometry(v any) (Geometry, bool) {
	if g, ok := v.(Geometry); ok {
		return g, true
	}
	b, err := json.Marshal(v)
	if err != nil {
		return Geometry{}, false
	}
	var g Geometry
	if err := json.Unmarshal(b, &g); err != nil {
		return Geometry{}, false
	}
	return g, true
}

func coerceColor(v any) (Color, bool) {
	switch t := v.(type) {
	case Color:
		return t, true
	case map[string]any:
		return Color{H: coerceInt(t["h"]), S: coerceInt(t["s"]), V: coerceInt(t["v"])}, true
	}
	return Color{}, false
}

func coerceFloat(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case json.Number:
		f, _ := t.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(t, 64)
		return f
	default:
		return 0
	}
}

func coerceInt(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case json.Number:
		i, err := t.Int64()
		if err == nil {
			return int(i)
		}
		f, _ := t.Float64()
		return int(f)
	case string:
		i, _ := strconv.Atoi(t)
		return i
	default:
		return 0
	}
}
