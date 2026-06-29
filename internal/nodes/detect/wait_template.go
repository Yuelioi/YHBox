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
	wtInExec           = "In"
	wtInTemplates      = "Templates"
	wtInTimeoutMs      = "TimeoutMs"
	wtInThreshold      = "Threshold"
	wtInROI            = "ROI"
	wtInPollIntervalMs = "PollIntervalMs"
	wtInSettleMs       = "SettleMs"
	wtOutFound         = "Found"
	wtOutTimeout       = "Timeout"
	wtDataPoint        = "Point"
	wtDataConf         = "Conf"
	wtDataMatched      = "Matched" // 命中与否 (bool) Data 字段 — 两出口都带, 供自动捕获 (Spec C)
)

func (WaitTemplate) Spec() node.Spec {
	return node.Spec{
		Kind:        "WaitTemplate",
		Category:    "Detect",
		NeedsTarget: true,
		TargetCapabilities: []node.TargetCapability{
			node.TargetCapabilityScreenshot,
		},
		Inputs: append([]node.InputSpec{
			{Name: wtInExec, Type: "Exec"},
			{Name: wtInTemplates, Type: "String", Semantic: "TemplateGUID", Required: true,
				Widget: node.WidgetSpec{Kind: "template-picker"}},
			{Name: wtInTimeoutMs, Type: "Number", Default: json.Number("5000"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: wtInThreshold, Type: "Number", Default: json.Number("0.85"),
				Widget: node.WidgetSpec{Kind: "slider",
					Props: node.MarshalProps(node.SliderProps{Min: 0, Max: 1, Step: 0.01})}},
			{Name: wtInROI, Type: "Geometry", Schema: node.GeometrySchema()},
			{Name: wtInPollIntervalMs, Type: "Number", Default: json.Number("100"), Advanced: true,
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: wtInSettleMs, Type: "Number", Default: json.Number("0"),
				Widget: node.WidgetSpec{Kind: "number"}},
		}, node.WindowInputSpec()),
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
			templateFailOutputSpec(),
		},
	}
}

func (WaitTemplate) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	keys := in.StringList(wtInTemplates)
	threshold := in.Float64(wtInThreshold)
	roi := in.Geometry(wtInROI)
	timeout := time.Duration(in.Int(wtInTimeoutMs)) * time.Millisecond
	poll := normalizePollInterval(time.Duration(in.Int(wtInPollIntervalMs)) * time.Millisecond)
	settle := time.Duration(in.Int(wtInSettleMs)) * time.Millisecond
	hit, err := waitForTemplate(ctx, keys, threshold, roi, timeout, poll)
	if err != nil {
		return fireTemplateFail(ctx, node.Failf(node.CodeCaptureFailed, err, "vision wait %s: %v", strings.Join(keys, "+"), err)), nil
	}
	if hit.Found {
		hit, err = settleAfterMatch(ctx, keys, threshold, roi, settle, hit)
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

func waitForTemplate(ctx node.Ctx, keys []string, threshold float64, roi node.Geometry, timeout, poll time.Duration) (node.MatchHit, error) {
	hit, err := matchOnce(ctx, keys, threshold, roi)
	if err != nil {
		return node.MatchHit{}, err
	}
	if hit.Found || timeout <= 0 {
		return hit, nil
	}
	deadline := ctx.Now().Add(timeout)
	bestConf := hit.Conf
	for {
		if err := waitOrCancel(ctx, poll); err != nil {
			return node.MatchHit{}, err
		}
		hit, err = matchOnce(ctx, keys, threshold, roi)
		if err != nil {
			return node.MatchHit{}, err
		}
		if hit.Conf > bestConf {
			bestConf = hit.Conf
		}
		if hit.Found {
			return hit, nil
		}
		if !ctx.Now().Before(deadline) {
			return node.MatchHit{Conf: bestConf}, nil
		}
	}
}
