// internal/nodes/detect/find_color_signature.go
// FindColorSignature — 颜色签名: 锚点 + N 偏移点的颜色组合, 区域搜索返回首个锚点位置。spec §节点1。
package detect

import (
	"encoding/json"
	"fmt"

	"yotta/internal/node"
)

func init() { node.Register(&FindColorSignature{}) }

type FindColorSignature struct{}

const (
	fcsInExec      = "In"
	fcsInROI       = "ROI"
	fcsInSignature = "Signature"
	fcsInTolerance = "Tolerance"
	fcsOutFound    = "Found"
	fcsOutNotFound = "NotFound"
	fcsDataPoint   = "Point"
	fcsMaxPoints   = 64
)

// fcsSignatureSchema 颜色签名变长列表 schema — 每点 {dx,dy,r,g,b,tol}, 首点为锚点 (dx=dy=0).
// 给 FE StructuredInput 渲染逐点结构表单 (可加减) + 整组 JSON 双模式; tol 选填 (留空→用节点默认容差).
// 业务校验 (非空/≤64 点/锚点 dx=dy=0/tol≥0) 仍在 parseColorSignature + Validate, 不进 schema.
var fcsSignatureSchema = node.ArraySchema(node.ObjSchema(
	node.Field("dx", node.NumberSchema(), true),
	node.Field("dy", node.NumberSchema(), true),
	node.Field("r", node.NumberSchema(), true),
	node.Field("g", node.NumberSchema(), true),
	node.Field("b", node.NumberSchema(), true),
	node.Field("tol", node.NumberSchema(), false),
))

func (FindColorSignature) Spec() node.Spec {
	return node.Spec{
		Kind:        "FindColorSignature",
		Category:    "Detect",
		NeedsWindow: true,
		Inputs: append([]node.InputSpec{
			{Name: fcsInExec, Type: "Exec"},
			{Name: fcsInROI, Type: "Geometry", Schema: node.GeometrySchema()},
			{Name: fcsInSignature, Type: "JSON",
				Widget: node.WidgetSpec{Kind: "json", Props: node.MarshalProps(node.JSONProps{Rows: 4})},
				Schema: fcsSignatureSchema},
			{Name: fcsInTolerance, Type: "Number", Default: json.Number("16"),
				Widget: node.WidgetSpec{Kind: "number"}},
		}, node.WindowInputSpec()),
		Outputs: []node.OutputSpec{
			{Name: fcsOutFound, Type: "Exec", Data: []node.DataField{{Name: fcsDataPoint, Type: "Point"}}},
			{Name: fcsOutNotFound, Type: "Exec"},
		},
	}
}

func (FindColorSignature) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	sig, err := parseColorSignature(in.Raw(fcsInSignature))
	if err != nil {
		return nil, fmt.Errorf("FindColorSignature signature: %w", err)
	}
	tol := in.Int(fcsInTolerance)
	if tol < 0 {
		tol = 0
	} else if tol > 255 {
		tol = 255
	}
	found, pt, err := ctx.Vision().FindColorSignature(in.Geometry(fcsInROI), sig, tol)
	if err != nil {
		return nil, node.Failf(node.CodeCaptureFailed, err, "FindColorSignature: %v", err)
	}
	if found {
		// 输出捕获由 Spec C 的 config.capture 自动处理 (dispatch_v5.applyCaptures), 节点只 Set Data 字段。
		return ctx.Out(fcsOutFound).Set(fcsDataPoint, pt).Fire(), nil
	}
	return ctx.Out(fcsOutNotFound).Fire(), nil
}

func (FindColorSignature) Validate(in node.Inputs) []node.ValidationError {
	if _, err := parseColorSignature(in.Raw(fcsInSignature)); err != nil {
		return []node.ValidationError{{Code: "INVALID_COLOR_SIGNATURE", Message: err.Error(), Field: fcsInSignature}}
	}
	return nil
}

// parseColorSignature 解析 Raw (JSON 数组) → ColorSignature。
// 校验: 非空、≤64 点、首项锚点 dx=dy=0、tol≥0。tol 缺省→nil (用默认), 给值→*int。
func parseColorSignature(raw any) (node.ColorSignature, error) {
	arr, ok := raw.([]any)
	if !ok || len(arr) == 0 {
		return node.ColorSignature{}, fmt.Errorf("signature must be non-empty array")
	}
	if len(arr) > fcsMaxPoints {
		return node.ColorSignature{}, fmt.Errorf("signature > %d points", fcsMaxPoints)
	}
	pts := make([]node.ColorPoint, len(arr))
	for i, e := range arr {
		m, ok := e.(map[string]any)
		if !ok {
			return node.ColorSignature{}, fmt.Errorf("point[%d] not object", i)
		}
		p := node.ColorPoint{
			DX: jnInt(m["dx"]), DY: jnInt(m["dy"]),
			R: jnInt(m["r"]), G: jnInt(m["g"]), B: jnInt(m["b"]),
		}
		if i == 0 && (p.DX != 0 || p.DY != 0) {
			return node.ColorSignature{}, fmt.Errorf("anchor (point[0]) must have dx=dy=0")
		}
		if tv, has := m["tol"]; has && tv != nil {
			t := jnInt(tv)
			if t < 0 {
				return node.ColorSignature{}, fmt.Errorf("point[%d] tol < 0", i)
			}
			p.Tol = &t
		}
		pts[i] = p
	}
	return node.ColorSignature{Points: pts}, nil
}

// jnInt 把 JSON 数字 (float64 / json.Number / int) 转 int, 非数字→0。
func jnInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return int(i)
		}
	}
	return 0
}
