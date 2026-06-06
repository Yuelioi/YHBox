// internal/nodes/detect/wait_template.go
// WaitTemplate — 阻塞等模板出现或超时. 节点本体是长轮询封装, 实际轮询在
// VisionService.WaitMatch 内做.
package detect

import (
	"encoding/json"
	"fmt"
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
	wtOutFound    = "Found"
	wtOutTimeout  = "Timeout"
	wtDataPoint   = "Point"
	wtDataConf    = "Conf"
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
			{Name: wtInTemplates, Type: "String", Semantic: "TemplateKey", Required: true,
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
			{Name: wtCapFound, Type: "String", Advanced: true, Semantic: "capture",
				Widget: node.WidgetSpec{Kind: "text"}},
			{Name: wtCapPoint, Type: "String", Advanced: true, Semantic: "capture",
				Widget: node.WidgetSpec{Kind: "text"}},
		},
		Outputs: []node.OutputSpec{
			{Name: wtOutFound, Type: "Exec",
				Data: []node.DataField{
					{Name: wtDataPoint, Type: "Point"},
					{Name: wtDataConf, Type: "Number"},
				}},
			{Name: wtOutTimeout, Type: "Exec",
				Data: []node.DataField{
					{Name: wtDataConf, Type: "Number", Optional: true},
				}},
		},
	}
}

func (WaitTemplate) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	keys := in.StringList(wtInTemplates)
	mode := in.String(wtInMatchMode)
	threshold := in.Float64(wtInThreshold)
	timeout := time.Duration(in.Int(wtInTimeoutMs)) * time.Millisecond
	pt, conf, err := ctx.Vision().WaitMatch(ctx.Context(), keys, threshold, mode, timeout)
	if err != nil {
		return nil, fmt.Errorf("vision wait %s: %w", strings.Join(keys, "+"), err)
	}
	if pt != nil {
		node.Capture(ctx, in, wtCapFound, true)
		node.Capture(ctx, in, wtCapPoint, *pt)
		return ctx.Out(wtOutFound).Set(wtDataPoint, *pt).Set(wtDataConf, conf).Fire(), nil
	}
	node.Capture(ctx, in, wtCapFound, false) // timeout 不写 point
	return ctx.Out(wtOutTimeout).Set(wtDataConf, conf).Fire(), nil
}

func (WaitTemplate) Validate(in node.Inputs) []node.ValidationError {
	return validateTemplateKeys(in.StringList(wtInTemplates), wtInTemplates)
}

func (WaitTemplate) Dependencies(in node.Inputs) []node.Dependency {
	return templateDeps(in.StringList(wtInTemplates))
}
