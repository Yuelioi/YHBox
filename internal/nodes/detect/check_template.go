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
	ctInMatchMode = "MatchMode"
	ctInThreshold = "Threshold"
	ctOutFound    = "Found"
	ctOutNotFound = "NotFound"
	ctDataPoint   = "Point"
	ctDataConf    = "Conf"
	ctCapFound    = "CaptureFound"
	ctCapPoint    = "CapturePoint"
)

// === Spec: declarative metadata ===
func (CheckTemplate) Spec() node.Spec {
	return node.Spec{
		Kind:        "CheckTemplate",
		Category:    "Detect",
		NeedsWindow: true,
		Inputs: []node.InputSpec{
			{Name: ctInExec, Type: "Exec"},
			{Name: ctInTemplates, Type: "String", Semantic: "TemplateKey", Required: true,
				Widget: node.WidgetSpec{Kind: "template-picker"}},
			{Name: ctInMatchMode, Type: "String", Default: "any", Advanced: true,
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{{Value: "any"}, {Value: "all"}}})}},
			{Name: ctInThreshold, Type: "Number", Default: json.Number("0.85"),
				Widget: node.WidgetSpec{Kind: "slider",
					Props: node.MarshalProps(node.SliderProps{Min: 0, Max: 1, Step: 0.01})}},
			{Name: ctCapFound, Type: "String", Advanced: true, Semantic: "capture",
				CaptureType: "bool", Widget: node.WidgetSpec{Kind: "text"}},
			{Name: ctCapPoint, Type: "String", Advanced: true, Semantic: "capture",
				CaptureType: "point", Widget: node.WidgetSpec{Kind: "text"}},
		},
		Outputs: []node.OutputSpec{
			{Name: ctOutFound, Type: "Exec",
				Data: []node.DataField{
					{Name: ctDataPoint, Type: "Point"},
					{Name: ctDataConf, Type: "Number"},
				}},
			{Name: ctOutNotFound, Type: "Exec",
				Data: []node.DataField{
					{Name: ctDataConf, Type: "Number", Optional: true},
				}},
		},
	}
}

// === Run: 执行逻辑. error 返回 + 单 Fire 语义 ===
func (CheckTemplate) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	keys := in.StringList(ctInTemplates)
	mode := in.String(ctInMatchMode)
	threshold := in.Float64(ctInThreshold)
	pt, conf, err := ctx.Vision().Match(ctx.Context(), keys, threshold, mode)
	if err != nil {
		return nil, node.Failf(node.CodeCaptureFailed, err, "vision match %s: %v", strings.Join(keys, "+"), err)
	}
	if pt != nil {
		node.Capture(ctx, in, ctCapFound, true)
		node.Capture(ctx, in, ctCapPoint, *pt)
		return ctx.Out(ctOutFound).Set(ctDataPoint, *pt).Set(ctDataConf, conf).Fire(), nil
	}
	node.Capture(ctx, in, ctCapFound, false) // miss 不写 point
	return ctx.Out(ctOutNotFound).Set(ctDataConf, conf).Fire(), nil
}

// === Validate: 可选 ===
func (CheckTemplate) Validate(in node.Inputs) []node.ValidationError {
	return validateTemplateKeys(in.StringList(ctInTemplates), ctInTemplates)
}

// === Dependencies: 可选 ===
func (CheckTemplate) Dependencies(in node.Inputs) []node.Dependency {
	return templateDeps(in.StringList(ctInTemplates))
}
