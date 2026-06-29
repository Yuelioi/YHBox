package detect

import (
	"encoding/json"
	"strconv"

	"yotta/internal/node"
)

func init() {
	node.Register(&PickMatchPoint{})
	node.Register(&PickBlobPoint{})
}

const (
	pmpInMatches = "Matches"
	pmpInIndex   = "Index"
	pmpInAnchor  = "Anchor"
	pmpInOffsetX = "OffsetX"
	pmpInOffsetY = "OffsetY"

	pbpInBlobs   = "Blobs"
	pbpInIndex   = "Index"
	pbpInAnchor  = "Anchor"
	pbpInOffsetX = "OffsetX"
	pbpInOffsetY = "OffsetY"
)

type PickMatchPoint struct{}

func (PickMatchPoint) Spec() node.Spec {
	return node.Spec{
		Kind:       "PickMatchPoint",
		Category:   "Detect",
		IsPureData: true,
		Inputs: []node.InputSpec{
			{Name: pmpInMatches, Type: "JSON"},
			{Name: pmpInIndex, Type: "Integer", Default: json.Number("0"), Widget: node.WidgetSpec{Kind: "number"}},
			anchorInput(pmpInAnchor),
			{Name: pmpInOffsetX, Type: "Number", Default: json.Number("0"), Advanced: true, Widget: node.WidgetSpec{Kind: "number"}},
			{Name: pmpInOffsetY, Type: "Number", Default: json.Number("0"), Advanced: true, Widget: node.WidgetSpec{Kind: "number"}},
		},
		Outputs: []node.OutputSpec{{Name: "Result", Type: "Point"}},
	}
}

func (PickMatchPoint) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	pt, bbox, ok := pickMatchPoint(in.Raw(pmpInMatches), in.Int(pmpInIndex))
	if !ok {
		return node.Point{}, nil
	}
	return anchoredPoint(pt, bbox, in.String(pmpInAnchor), in.Float64(pmpInOffsetX), in.Float64(pmpInOffsetY)), nil
}

type PickBlobPoint struct{}

func (PickBlobPoint) Spec() node.Spec {
	return node.Spec{
		Kind:       "PickBlobPoint",
		Category:   "Detect",
		IsPureData: true,
		Inputs: []node.InputSpec{
			{Name: pbpInBlobs, Type: "JSON"},
			{Name: pbpInIndex, Type: "Integer", Default: json.Number("0"), Widget: node.WidgetSpec{Kind: "number"}},
			anchorInput(pbpInAnchor),
			{Name: pbpInOffsetX, Type: "Number", Default: json.Number("0"), Advanced: true, Widget: node.WidgetSpec{Kind: "number"}},
			{Name: pbpInOffsetY, Type: "Number", Default: json.Number("0"), Advanced: true, Widget: node.WidgetSpec{Kind: "number"}},
		},
		Outputs: []node.OutputSpec{{Name: "Result", Type: "Point"}},
	}
}

func (PickBlobPoint) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	pt, bbox, ok := pickBlob(in.Raw(pbpInBlobs), in.Int(pbpInIndex))
	if !ok {
		return node.Point{}, nil
	}
	return anchoredPoint(pt, bbox, in.String(pbpInAnchor), in.Float64(pbpInOffsetX), in.Float64(pbpInOffsetY)), nil
}

func anchorInput(name string) node.InputSpec {
	return node.InputSpec{Name: name, Type: "String", Default: "center", Advanced: true,
		Widget: node.WidgetSpec{Kind: "dropdown",
			Props: node.MarshalProps(node.DropdownProps{
				Options: []node.EnumOption{
					{Value: "topLeft"}, {Value: "topCenter"}, {Value: "topRight"},
					{Value: "midLeft"}, {Value: "center"}, {Value: "midRight"},
					{Value: "botLeft"}, {Value: "botCenter"}, {Value: "botRight"},
				}})}}
}

func pickMatchPoint(raw any, idx int) (node.Point, [4]float64, bool) {
	if idx < 0 {
		return node.Point{}, [4]float64{}, false
	}
	switch v := raw.(type) {
	case []node.TemplateMatch:
		if idx >= len(v) {
			return node.Point{}, [4]float64{}, false
		}
		return v[idx].Point, v[idx].BBox, true
	case []any:
		if idx >= len(v) {
			return node.Point{}, [4]float64{}, false
		}
		return parseMatchItem(v[idx])
	}
	return node.Point{}, [4]float64{}, false
}

func pickBlob(raw any, idx int) (node.Point, [4]float64, bool) {
	if idx < 0 {
		return node.Point{}, [4]float64{}, false
	}
	switch v := raw.(type) {
	case []node.BlobEntry:
		if idx >= len(v) {
			return node.Point{}, [4]float64{}, false
		}
		b := v[idx]
		return node.Point{X: b.CenterX, Y: b.CenterY}, [4]float64{b.X, b.Y, b.W, b.H}, true
	case []any:
		if idx >= len(v) {
			return node.Point{}, [4]float64{}, false
		}
		return parseBlobItem(v[idx])
	}
	return node.Point{}, [4]float64{}, false
}

func parseMatchItem(raw any) (node.Point, [4]float64, bool) {
	switch v := raw.(type) {
	case node.TemplateMatch:
		return v.Point, v.BBox, true
	case map[string]any:
		pt, _ := pointFromAny(first(v, "point", "Point"))
		bbox, _ := bboxFromAny(first(v, "bbox", "BBox"))
		return pt, bbox, true
	}
	return node.Point{}, [4]float64{}, false
}

func parseBlobItem(raw any) (node.Point, [4]float64, bool) {
	switch v := raw.(type) {
	case node.BlobEntry:
		return node.Point{X: v.CenterX, Y: v.CenterY}, [4]float64{v.X, v.Y, v.W, v.H}, true
	case map[string]any:
		cx, _ := numberFromAny(first(v, "centerX", "CenterX"))
		cy, _ := numberFromAny(first(v, "centerY", "CenterY"))
		x, _ := numberFromAny(first(v, "x", "X"))
		y, _ := numberFromAny(first(v, "y", "Y"))
		w, _ := numberFromAny(first(v, "w", "W"))
		h, _ := numberFromAny(first(v, "h", "H"))
		return node.Point{X: cx, Y: cy}, [4]float64{x, y, w, h}, true
	}
	return node.Point{}, [4]float64{}, false
}

func anchoredPoint(center node.Point, bbox [4]float64, anchor string, offX, offY float64) node.Point {
	if bbox[2] > 0 && bbox[3] > 0 {
		return anchorPoint(bbox, anchor, offX, offY)
	}
	return node.Point{X: clamp01(center.X + offX), Y: clamp01(center.Y + offY)}
}

func first(m map[string]any, keys ...string) any {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			return v
		}
	}
	return nil
}

func pointFromAny(raw any) (node.Point, bool) {
	switch v := raw.(type) {
	case node.Point:
		return v, true
	case map[string]any:
		x, xok := numberFromAny(first(v, "x", "X"))
		y, yok := numberFromAny(first(v, "y", "Y"))
		return node.Point{X: x, Y: y}, xok && yok
	}
	return node.Point{}, false
}

func bboxFromAny(raw any) ([4]float64, bool) {
	switch v := raw.(type) {
	case [4]float64:
		return v, true
	case []float64:
		if len(v) < 4 {
			return [4]float64{}, false
		}
		return [4]float64{v[0], v[1], v[2], v[3]}, true
	case []any:
		if len(v) < 4 {
			return [4]float64{}, false
		}
		var out [4]float64
		for i := 0; i < 4; i++ {
			n, ok := numberFromAny(v[i])
			if !ok {
				return [4]float64{}, false
			}
			out[i] = n
		}
		return out, true
	}
	return [4]float64{}, false
}

func numberFromAny(raw any) (float64, bool) {
	switch v := raw.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		f, err := v.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(v, 64)
		return f, err == nil
	}
	return 0, false
}
