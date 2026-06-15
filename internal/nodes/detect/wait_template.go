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
	wtInMatchMode = "MatchMode"
	wtInTimeoutMs = "TimeoutMs"
	wtInThreshold = "Threshold"
	wtInSettleMs  = "SettleMs"
	wtOutFound    = "Found"
	wtOutTimeout  = "Timeout"
	wtDataPoint   = "Point"
	wtDataConf    = "Conf"
	wtDataMatched = "Matched" // 命中与否 (bool) Data 字段 — 两出口都带, 供自动捕获 (Spec C)
	wtCapFound    = "CaptureFound"
	wtCapPoint    = "CapturePoint"
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
			{Name: wtInMatchMode, Type: "String", Default: "any", Advanced: true,
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{{Value: "any"}, {Value: "all"}}})}},
			{Name: wtInTimeoutMs, Type: "Number", Default: json.Number("5000"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: wtInThreshold, Type: "Number", Default: json.Number("0.85"),
				Widget: node.WidgetSpec{Kind: "slider",
					Props: node.MarshalProps(node.SliderProps{Min: 0, Max: 1, Step: 0.01})}},
			{Name: wtInSettleMs, Type: "Number", Default: json.Number("0"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: wtCapFound, Type: "String", Advanced: true, Semantic: "capture",
				CaptureType: "bool", Widget: node.WidgetSpec{Kind: "text"}},
			{Name: wtCapPoint, Type: "String", Advanced: true, Semantic: "capture",
				CaptureType: "point", Widget: node.WidgetSpec{Kind: "text"}},
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
	mode := in.String(wtInMatchMode)
	threshold := in.Float64(wtInThreshold)
	timeout := time.Duration(in.Int(wtInTimeoutMs)) * time.Millisecond
	settle := time.Duration(in.Int(wtInSettleMs)) * time.Millisecond
	pt, conf, err := ctx.Vision().WaitMatch(ctx.Context(), keys, threshold, mode, timeout)
	if err != nil {
		return nil, node.Failf(node.CodeCaptureFailed, err, "vision wait %s: %v", strings.Join(keys, "+"), err)
	}
	if pt != nil {
		// 命中后可选稳定延迟 + 重定位 (SettleMs): 等画面就位再放行, 顺带更新输出/捕获的 Point。详见 settleAfterMatch。
		pt, conf, err = settleAfterMatch(ctx, keys, threshold, mode, settle, pt, conf)
		if err != nil {
			return nil, err // settle 期间被取消 (graph stop) → 优雅 halt
		}
		node.Capture(ctx, in, wtCapFound, true)
		node.Capture(ctx, in, wtCapPoint, *pt)
		return ctx.Out(wtOutFound).Set(wtDataPoint, *pt).Set(wtDataConf, conf).Set(wtDataMatched, true).Fire(), nil
	}
	node.Capture(ctx, in, wtCapFound, false) // timeout 不写 point
	return ctx.Out(wtOutTimeout).Set(wtDataConf, conf).Set(wtDataMatched, false).Fire(), nil
}

func (WaitTemplate) Validate(in node.Inputs) []node.ValidationError {
	// 模板引用现为 GUID, 合法性 = 存在性 (由 container validator_deps 校验), 节点级无格式校验.
	return nil
}

func (WaitTemplate) Dependencies(in node.Inputs) []node.Dependency {
	return templateDeps(in.StringList(wtInTemplates))
}
