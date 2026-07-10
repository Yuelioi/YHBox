// internal/nodes/detect/wait_template_gone.go
// WaitTemplateGone — 等指定模板从画面消失 (无任何命中) 再放行; 超时仍在走 Timeout。
package detect

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/yottaapp/yotta/internal/node"
)

func init() { node.Register(&WaitTemplateGone{}) }

type WaitTemplateGone struct{}

const (
	wtgInExec           = "In"
	wtgInTemplates      = "Templates"
	wtgInTimeoutMs      = "TimeoutMs"
	wtgInThreshold      = "Threshold"
	wtgInROI            = "ROI"
	wtgInPollIntervalMs = "PollIntervalMs"
	wtgOutGone          = "Gone"
	wtgOutTimeout       = "Timeout"
	wtgDataConf         = "Conf"
)

func (WaitTemplateGone) Spec() node.Spec {
	return node.Spec{
		Kind:        "WaitTemplateGone",
		Category:    "Detect",
		NeedsTarget: true,
		TargetCapabilities: []node.TargetCapability{
			node.TargetCapabilityScreenshot,
		},
		Inputs: append([]node.InputSpec{
			{Name: wtgInExec, Type: "Exec"},
			{Name: wtgInTemplates, Type: "String", Semantic: "TemplateGUID", Required: true,
				Widget: node.WidgetSpec{Kind: "template-picker"}},
			{Name: wtgInTimeoutMs, Type: "Number", Default: json.Number("5000"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: wtgInThreshold, Type: "Number", Default: json.Number("0.85"),
				Widget: node.WidgetSpec{Kind: "slider",
					Props: node.MarshalProps(node.SliderProps{Min: 0, Max: 1, Step: 0.01})}},
			{Name: wtgInROI, Type: "Geometry", Schema: node.GeometrySchema()},
			{Name: wtgInPollIntervalMs, Type: "Number", Default: json.Number("100"), Advanced: true,
				Widget: node.WidgetSpec{Kind: "number"}},
		}, node.WindowInputSpec()),
		Outputs: []node.OutputSpec{
			{Name: wtgOutGone, Type: "Exec"},
			{Name: wtgOutTimeout, Type: "Exec",
				Data: []node.DataField{{Name: wtgDataConf, Type: "Number", Optional: true}}},
			templateFailOutputSpec(),
		},
	}
}

func (WaitTemplateGone) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	keys := in.StringList(wtgInTemplates)
	threshold := in.Float64(wtgInThreshold)
	roi := in.Geometry(wtgInROI)
	timeout := time.Duration(in.Int(wtgInTimeoutMs)) * time.Millisecond
	poll := normalizePollInterval(time.Duration(in.Int(wtgInPollIntervalMs)) * time.Millisecond)

	// 先查当前帧：模板已消失 → 立即走 Gone
	hit, err := matchOnce(ctx, keys, threshold, roi)
	if err != nil {
		return fireTemplateFail(ctx, node.Failf(node.CodeCaptureFailed, err, "WaitTemplateGone %s: %v", strings.Join(keys, "+"), err)), nil
	}
	if !hit.Found {
		return ctx.Out(wtgOutGone).Fire(), nil
	}
	// 单帧模式 (TimeoutMs<=0)：当帧仍在 → Timeout 带最后 Conf
	if timeout <= 0 {
		return ctx.Out(wtgOutTimeout).Set(wtgDataConf, hit.Conf).Fire(), nil
	}
	// 轮询直到模板消失或超时
	deadline := ctx.Now().Add(timeout)
	for {
		if err := waitOrCancel(ctx, poll); err != nil {
			return nil, err
		}
		hit, err = matchOnce(ctx, keys, threshold, roi)
		if err != nil {
			return fireTemplateFail(ctx, node.Failf(node.CodeCaptureFailed, err, "WaitTemplateGone recheck %s: %v", strings.Join(keys, "+"), err)), nil
		}
		if !hit.Found {
			return ctx.Out(wtgOutGone).Fire(), nil
		}
		if ctx.Now().After(deadline) {
			return ctx.Out(wtgOutTimeout).Set(wtgDataConf, hit.Conf).Fire(), nil
		}
	}
}

func (WaitTemplateGone) Dependencies(in node.Inputs) []node.Dependency {
	return templateDeps(in.StringList(wtgInTemplates))
}
