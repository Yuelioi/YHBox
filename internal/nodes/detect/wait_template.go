// internal/nodes/detect/wait_template.go
// WaitTemplate — 阻塞等模板出现或超时. 节点本体是长轮询封装, 实际轮询在
// VisionService.WaitMatch 内做.
package detect

import (
	"encoding/json"
	"strings"
	"time"

	"yotta/internal/node"
)

func init() { node.Register(&WaitTemplate{}) }

type WaitTemplate struct{}

const (
	wtInExec      = "In"
	wtInTemplates = "Templates"
	wtInTimeoutMs = "TimeoutMs"
	wtInThreshold = "Threshold"
	wtInSettleMs  = "SettleMs"
	wtOutFound    = "Found"
	wtOutTimeout  = "Timeout"
	wtDataPoint   = "Point"
	wtDataConf    = "Conf"
	wtDataMatched = "Matched" // 命中与否 (bool) Data 字段 — 两出口都带, 供自动捕获 (Spec C)
)

func (WaitTemplate) Spec() node.Spec {
	return node.Spec{
		Kind:        "WaitTemplate",
		Category:    "Detect",
		NeedsWindow: true,
		Inputs: []node.InputSpec{
			{Name: wtInExec, Type: "Exec"},
			{Name: wtInTemplates, Type: "String", Semantic: "TemplateGUID", Required: true,
				Widget: node.WidgetSpec{Kind: "template-picker"}},
			{Name: wtInTimeoutMs, Type: "Number", Default: json.Number("5000"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: wtInThreshold, Type: "Number", Default: json.Number("0.85"),
				Widget: node.WidgetSpec{Kind: "slider",
					Props: node.MarshalProps(node.SliderProps{Min: 0, Max: 1, Step: 0.01})}},
			{Name: wtInSettleMs, Type: "Number", Default: json.Number("0"),
				Widget: node.WidgetSpec{Kind: "number"}},
		},
		Outputs: []node.OutputSpec{
			{Name: wtOutFound, Type: "Exec",
				Data: []node.DataField{
					{Name: wtDataPoint, Type: "Point"},
					{Name: wtDataConf, Type: "Number"},
					{Name: wtDataMatched, Type: "Bool"},
				}},
			{Name: wtOutTimeout, Type: "Exec",
				Data: []node.DataField{
					{Name: wtDataConf, Type: "Number", Optional: true},
					{Name: wtDataMatched, Type: "Bool"},
				}},
		},
	}
}

func (WaitTemplate) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	keys := in.StringList(wtInTemplates)
	threshold := in.Float64(wtInThreshold)
	timeout := time.Duration(in.Int(wtInTimeoutMs)) * time.Millisecond
	settle := time.Duration(in.Int(wtInSettleMs)) * time.Millisecond
	hit, err := ctx.Vision().WaitMatch(ctx.Context(), keys, threshold, node.Geometry{}, timeout)
	if err != nil {
		return nil, node.Failf(node.CodeCaptureFailed, err, "vision wait %s: %v", strings.Join(keys, "+"), err)
	}
	if hit.Found {
		hit, err = settleAfterMatch(ctx, keys, threshold, settle, hit)
		if err != nil {
			return nil, err
		}
		return ctx.Out(wtOutFound).Set(wtDataPoint, hit.Point).Set(wtDataConf, hit.Conf).Set(wtDataMatched, true).Fire(), nil
	}
	return ctx.Out(wtOutTimeout).Set(wtDataConf, hit.Conf).Set(wtDataMatched, false).Fire(), nil
}

func (WaitTemplate) Validate(in node.Inputs) []node.ValidationError {
	// 模板引用现为 GUID, 合法性 = 存在性 (由 container validator_deps 校验), 节点级无格式校验.
	return nil
}

func (WaitTemplate) Dependencies(in node.Inputs) []node.Dependency {
	return templateDeps(in.StringList(wtInTemplates))
}
