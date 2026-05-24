// internal/nodes/mock/vision.go
package mock

import (
	"encoding/json"

	"yhbox/internal/node"
)

func init() {
	node.Register(&Vision{})
}

type Vision struct{}

const (
	visionInExec      = "in"
	visionInTemplate  = "Template"
	visionInROI       = "ROI"
	visionInThreshold = "Threshold"
	visionOutFound    = "Found"
	visionOutNotFound = "NotFound"
	visionDataPoint   = "Point"
	visionDataConf    = "Conf"
	visionDataPreview = "Preview"
)

func (Vision) Spec() node.Spec {
	return node.Spec{
		Kind:        "Mock_Vision",
		Version:     1,
		Category:    "Mock",
		DisplayName: "Mock 视觉",
		Description: "Phase 0 测试 — Point/Rect/Image + AsyncOptions dropdown + 多 exec out 含 Data 字段 (hover 展开 UI).",
		Inputs: []node.InputSpec{
			{Name: visionInExec, Type: "Exec"},
			{Name: visionInTemplate, Type: "String", Semantic: "TemplateKey", Required: true,
				DisplayName: "模板",
				Doc:         "async-dropdown — Phase 0 mock 返 hardcoded list, 接口对齐真 RPC",
				Widget: node.WidgetSpec{Kind: "async-dropdown",
					Props: node.MarshalProps(node.AsyncDropdownProps{AsyncSource: "templateKeys"})}},
			{Name: visionInROI, Type: "Rect",
				DisplayName: "搜索区域",
				Widget:      node.WidgetSpec{Kind: "rect-editor"}},
			{Name: visionInThreshold, Type: "Number", Default: json.Number("0.85"),
				DisplayName: "阈值",
				Widget: node.WidgetSpec{Kind: "slider",
					Props: node.MarshalProps(node.SliderProps{Min: 0, Max: 1, Step: 0.01})}},
		},
		Outputs: []node.OutputSpec{
			{Name: visionOutFound, Type: "Exec", DisplayName: "命中",
				Data: []node.DataField{
					{Name: visionDataPoint, Type: "Point", Doc: "命中中心"},
					{Name: visionDataConf, Type: "Number", Doc: "实际匹配度"},
					{Name: visionDataPreview, Type: "Image", Optional: true, Doc: "命中区域截图 (可选)"},
				}},
			{Name: visionOutNotFound, Type: "Exec", DisplayName: "未命中",
				Data: []node.DataField{
					{Name: visionDataConf, Type: "Number", Optional: true, Doc: "最高匹配度 (低于阈值)"},
				}},
		},
	}
}

// Run — Phase 0 mock 不真跑, 直接 Fire NotFound (安全 stub). 用于 framework 单测 / FE 视觉验证.
func (Vision) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	return ctx.Out(visionOutNotFound).Fire(), nil
}
