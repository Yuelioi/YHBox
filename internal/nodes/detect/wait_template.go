// internal/nodes/detect/wait_template.go
// WaitTemplate — 阻塞等模板出现或超时. 节点本体是长轮询封装 (IsYield 节点),
// 实际轮询在 VisionService.WaitMatch 内做.
//
// 老 runtime: internal/services/container/runtime/nodes.go::execWaitTemplate.
package detect

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"yhbox/internal/node"
)

func init() { node.Register(&WaitTemplate{}) }

type WaitTemplate struct{}

const (
	wtInExec      = "in"
	wtInTemplate  = "Template"
	wtInTimeoutMs = "TimeoutMs"
	wtInThreshold = "Threshold"
	wtOutFound    = "Found"
	wtOutTimeout  = "Timeout"
	wtDataPoint   = "Point"
	wtDataConf    = "Conf"
)

func (WaitTemplate) Spec() node.Spec {
	return node.Spec{
		Kind:        "WaitTemplate",
		Version:     1,
		Category:    "Detect",
		DisplayName: "等待模板",
		Description: "在 timeoutMs 内轮询匹配模板. 命中走 Found 带坐标, 超时走 Timeout.",
		Inputs: []node.InputSpec{
			{Name: wtInExec, Type: "Exec"},
			{Name: wtInTemplate, Type: "String", Semantic: "TemplateKey", Required: true,
				DisplayName: "模板", Doc: "命名空间.名 格式, e.g. fishing.hook_icon",
				Widget: node.WidgetSpec{Kind: "async-dropdown",
					Props: node.MarshalProps(node.AsyncDropdownProps{AsyncSource: "templateKeys"})}},
			{Name: wtInTimeoutMs, Type: "Number", Default: json.Number("5000"),
				DisplayName: "超时 (ms)",
				Widget:      node.WidgetSpec{Kind: "number"}},
			{Name: wtInThreshold, Type: "Number", Default: json.Number("0.85"),
				DisplayName: "阈值", Doc: "NCC 阈值",
				Widget: node.WidgetSpec{Kind: "slider",
					Props: node.MarshalProps(node.SliderProps{Min: 0, Max: 1, Step: 0.01})}},
		},
		Outputs: []node.OutputSpec{
			{Name: wtOutFound, Type: "Exec", DisplayName: "命中",
				Data: []node.DataField{
					{Name: wtDataPoint, Type: "Point", Doc: "命中中心 (ratio)"},
					{Name: wtDataConf, Type: "Number", Doc: "实际匹配度"},
				}},
			{Name: wtOutTimeout, Type: "Exec", DisplayName: "超时",
				Data: []node.DataField{
					{Name: wtDataConf, Type: "Number", Optional: true, Doc: "最高匹配度 (低于阈值)"},
				}},
		},
	}
}

func (WaitTemplate) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	key := in.String(wtInTemplate)
	threshold := in.Float64(wtInThreshold)
	timeout := time.Duration(in.Int(wtInTimeoutMs)) * time.Millisecond
	pt, conf, err := ctx.Vision().WaitMatch(ctx.Context(), key, threshold, timeout)
	if err != nil {
		return nil, fmt.Errorf("vision wait %q: %w", key, err)
	}
	if pt != nil {
		return ctx.Out(wtOutFound).Set(wtDataPoint, *pt).Set(wtDataConf, conf).Fire(), nil
	}
	return ctx.Out(wtOutTimeout).Set(wtDataConf, conf).Fire(), nil
}

func (WaitTemplate) Display(in node.Inputs, exitName string, out node.OutputData) string {
	switch exitName {
	case wtOutFound:
		pt := out.Point(wtDataPoint)
		return fmt.Sprintf("✓ %s conf=%.2f @ (%.2f,%.2f)",
			in.String(wtInTemplate), out.Float64(wtDataConf), pt.X, pt.Y)
	case wtOutTimeout:
		return fmt.Sprintf("⌛ %s timeout (%dms)", in.String(wtInTemplate), in.Int(wtInTimeoutMs))
	}
	return ""
}

func (WaitTemplate) Validate(in node.Inputs) []node.ValidationError {
	key := in.String(wtInTemplate)
	if key != "" && !strings.Contains(key, ".") {
		return []node.ValidationError{{
			Code:    "INVALID_TEMPLATE_KEY",
			Message: fmt.Sprintf("template key 必须 namespace.name 格式, got %q", key),
			Field:   wtInTemplate,
		}}
	}
	return nil
}

func (WaitTemplate) Dependencies(in node.Inputs) []node.Dependency {
	return []node.Dependency{{Kind: "template", Key: in.String(wtInTemplate)}}
}
