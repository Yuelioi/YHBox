// internal/nodes/detect/check_template.go
// CheckTemplate — 第一个真节点. 展示节点作者完整模式: const + Spec + Run + Display + Validate + Dependencies.
// 所有后续 70 节点都照这个模式 (Phase 4 subagent template).
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
	ctInExec      = "in"
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
		Version:     1,
		Category:    "Detect",
		DisplayName: "检查模板",
		Description: "在当前帧 NCC 匹配模板. 命中走 Found 带坐标, 没命中走 NotFound.",
		Inputs: []node.InputSpec{
			{Name: ctInExec, Type: "Exec"},
			{Name: ctInTemplate, Type: "String", Semantic: "TemplateKey", Required: true,
				DisplayName: "模板", Doc: "命名空间.名 格式, e.g. fishing.hook_icon",
				Widget: node.WidgetSpec{Kind: "async-dropdown",
					Props: node.MarshalProps(node.AsyncDropdownProps{AsyncSource: "templateKeys"})}},
			{Name: ctInThreshold, Type: "Number", Default: json.Number("0.85"),
				DisplayName: "阈值", Doc: "NCC 阈值",
				Widget: node.WidgetSpec{Kind: "slider",
					Props: node.MarshalProps(node.SliderProps{Min: 0, Max: 1, Step: 0.01})}},
		},
		Outputs: []node.OutputSpec{
			{Name: ctOutFound, Type: "Exec", DisplayName: "命中",
				Data: []node.DataField{
					{Name: ctDataPoint, Type: "Point", Doc: "命中中心 (ratio)"},
					{Name: ctDataConf, Type: "Number", Doc: "实际匹配度"},
				}},
			{Name: ctOutNotFound, Type: "Exec", DisplayName: "未命中",
				Data: []node.DataField{
					{Name: ctDataConf, Type: "Number", Optional: true, Doc: "最高匹配度 (低于阈值)"},
				}},
		},
	}
}

// === Run: 执行逻辑. error 返回 + 单 Fire 语义 ===
func (CheckTemplate) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	key := in.String(ctInTemplate)
	threshold := in.Float64(ctInThreshold)
	pt, conf, err := ctx.Vision().Match(key, threshold)
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
