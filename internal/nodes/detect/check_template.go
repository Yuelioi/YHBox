// internal/nodes/detect/check_template.go
// CheckTemplate — 模板匹配检测, 命中与否分别走 Found / NotFound 出口.
package detect

import (
	"encoding/json"
	"fmt"
	"strings"

	"yhbox/internal/node"
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
		return nil, fmt.Errorf("vision match %s: %w", strings.Join(keys, "+"), err)
	}
	if pt != nil {
		return ctx.Out(ctOutFound).Set(ctDataPoint, *pt).Set(ctDataConf, conf).Fire(), nil
	}
	return ctx.Out(ctOutNotFound).Set(ctDataConf, conf).Fire(), nil
}

// === Display: 可选 (没实现 → 节点不出现在日志面板) ===
func (CheckTemplate) Display(in node.Inputs, exitName string, out node.OutputData) string {
	label := strings.Join(in.StringList(ctInTemplates), "+")
	switch exitName {
	case ctOutFound:
		pt := out.Point(ctDataPoint)
		return fmt.Sprintf("✓ %s conf=%.2f @ (%.2f,%.2f)",
			label, out.Float64(ctDataConf), pt.X, pt.Y)
	case ctOutNotFound:
		conf := out.Float64(ctDataConf)
		if conf > 0.5 {
			return fmt.Sprintf("· %s near-miss %.2f", label, conf)
		}
		return ""
	}
	return ""
}

// === Validate: 可选 ===
func (CheckTemplate) Validate(in node.Inputs) []node.ValidationError {
	return validateTemplateKeys(in.StringList(ctInTemplates), ctInTemplates)
}

// === Dependencies: 可选 ===
func (CheckTemplate) Dependencies(in node.Inputs) []node.Dependency {
	return templateDeps(in.StringList(ctInTemplates))
}
