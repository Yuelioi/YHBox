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
	ctInTemplate  = "Template"
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
			{Name: ctInTemplate, Type: "String", Semantic: "TemplateKey", Required: true,
				Widget: node.WidgetSpec{Kind: "async-dropdown",
					Props: node.MarshalProps(node.AsyncDropdownProps{AsyncSource: "templateKeys"})}},
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
	key := in.String(ctInTemplate)
	threshold := in.Float64(ctInThreshold)
	pt, conf, err := ctx.Vision().Match(ctx.Context(), key, threshold)
	if err != nil {
		return nil, fmt.Errorf("vision match %q: %w", key, err)
	}
	if pt != nil {
		return ctx.Out(ctOutFound).Set(ctDataPoint, *pt).Set(ctDataConf, conf).Fire(), nil
	}
	return ctx.Out(ctOutNotFound).Set(ctDataConf, conf).Fire(), nil
}

// === Display: 可选 (没实现 → 节点不出现在日志面板) ===
func (CheckTemplate) Display(in node.Inputs, exitName string, out node.OutputData) string {
	switch exitName {
	case ctOutFound:
		pt := out.Point(ctDataPoint)
		return fmt.Sprintf("✓ %s conf=%.2f @ (%.2f,%.2f)",
			in.String(ctInTemplate), out.Float64(ctDataConf), pt.X, pt.Y)
	case ctOutNotFound:
		conf := out.Float64(ctDataConf)
		if conf > 0.5 {
			return fmt.Sprintf("· %s near-miss %.2f", in.String(ctInTemplate), conf)
		}
		return ""
	}
	return ""
}

// === Validate: 可选 ===
func (CheckTemplate) Validate(in node.Inputs) []node.ValidationError {
	key := in.String(ctInTemplate)
	if key != "" && !strings.Contains(key, ".") {
		return []node.ValidationError{{
			Code:    "INVALID_TEMPLATE_KEY",
			Message: fmt.Sprintf("template key 必须 namespace.name 格式, got %q", key),
			Field:   ctInTemplate,
		}}
	}
	return nil
}

// === Dependencies: 可选 ===
func (CheckTemplate) Dependencies(in node.Inputs) []node.Dependency {
	return []node.Dependency{{Kind: "template", Key: in.String(ctInTemplate)}}
}
