// internal/nodes/detect/check_template.go
// CheckTemplate — 模板匹配检测, 命中与否分别走 Found / NotFound 出口.
package detect

import (
	"encoding/json"
	"strings"

	"yotta/internal/node"
)

func init() {
	node.Register(&CheckTemplate{})
}

type CheckTemplate struct{}

// === 字符串 const (选项 A: 节点级常量, typo 编译期捕获) ===
const (
	ctInExec      = "In"
	ctInTemplates = "Templates"
	ctInThreshold = "Threshold"
	ctOutFound    = "Found"
	ctOutNotFound = "NotFound"
	ctDataPoint   = "Point"
	ctDataConf    = "Conf"
	ctDataMatched = "Matched" // 命中与否 (bool) Data 字段 — 两出口都带, 供自动捕获 (Spec C)
)

// === Spec: declarative metadata ===
func (CheckTemplate) Spec() node.Spec {
	return node.Spec{
		Kind:        "CheckTemplate",
		Category:    "Detect",
		NeedsTarget: true,
		TargetCapabilities: []node.TargetCapability{
			node.TargetCapabilityScreenshot,
		},
		Inputs: append([]node.InputSpec{
			{Name: ctInExec, Type: "Exec"},
			{Name: ctInTemplates, Type: "String", Semantic: "TemplateGUID", Required: true,
				Widget: node.WidgetSpec{Kind: "template-picker"}},
			{Name: ctInThreshold, Type: "Number", Default: json.Number("0.85"),
				Widget: node.WidgetSpec{Kind: "slider",
					Props: node.MarshalProps(node.SliderProps{Min: 0, Max: 1, Step: 0.01})}},
		}, node.WindowInputSpec()),
		Outputs: []node.OutputSpec{
			{Name: ctOutFound, Type: "Exec",
				Data: []node.DataField{
					{Name: ctDataPoint, Type: "Point"},
					{Name: ctDataConf, Type: "Number"},
					{Name: ctDataMatched, Type: "Bool"},
				}},
			{Name: ctOutNotFound, Type: "Exec",
				Data: []node.DataField{
					{Name: ctDataConf, Type: "Number", Optional: true},
					{Name: ctDataMatched, Type: "Bool"},
				}},
		},
	}
}

// === Run: 执行逻辑. error 返回 + 单 Fire 语义 ===
func (CheckTemplate) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	keys := in.StringList(ctInTemplates)
	threshold := in.Float64(ctInThreshold)
	hit, err := ctx.Vision().Match(ctx.Context(), keys, threshold, node.Geometry{})
	if err != nil {
		return nil, node.Failf(node.CodeCaptureFailed, err, "vision match %s: %v", strings.Join(keys, "+"), err)
	}
	if hit.Found {
		return ctx.Out(ctOutFound).Set(ctDataPoint, hit.Point).Set(ctDataConf, hit.Conf).Set(ctDataMatched, true).Fire(), nil
	}
	return ctx.Out(ctOutNotFound).Set(ctDataConf, hit.Conf).Set(ctDataMatched, false).Fire(), nil
}

// === Validate: 可选 ===
func (CheckTemplate) Validate(in node.Inputs) []node.ValidationError {
	// 模板引用现为 GUID, 合法性 = 存在性 (由 container validator_deps 校验), 节点级无格式校验.
	return nil
}

// === Dependencies: 可选 ===
func (CheckTemplate) Dependencies(in node.Inputs) []node.Dependency {
	return templateDeps(in.StringList(ctInTemplates))
}
