// internal/nodes/detect/wait_template_gone.go
// WaitTemplateGone — 等指定模板从画面消失 (无任何命中) 再放行; 超时仍在走 Timeout。
package detect

import (
	"encoding/json"
	"strings"
	"time"

	"yotta/internal/node"
)

func init() { node.Register(&WaitTemplateGone{}) }

type WaitTemplateGone struct{}

const (
	wtgInExec      = "In"
	wtgInTemplates = "Templates"
	wtgInTimeoutMs = "TimeoutMs"
	wtgInThreshold = "Threshold"
	wtgOutGone     = "Gone"
	wtgOutTimeout  = "Timeout"
	wtgDataConf    = "Conf"
)

func (WaitTemplateGone) Spec() node.Spec {
	return node.Spec{
		Kind:        "WaitTemplateGone",
		Category:    "Detect",
		NeedsWindow: true,
		Inputs: append([]node.InputSpec{
			{Name: wtgInExec, Type: "Exec"},
			{Name: wtgInTemplates, Type: "String", Semantic: "TemplateGUID", Required: true,
				Widget: node.WidgetSpec{Kind: "template-picker"}},
			{Name: wtgInTimeoutMs, Type: "Number", Default: json.Number("5000"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: wtgInThreshold, Type: "Number", Default: json.Number("0.85"),
				Widget: node.WidgetSpec{Kind: "slider",
					Props: node.MarshalProps(node.SliderProps{Min: 0, Max: 1, Step: 0.01})}},
		}, node.WindowInputSpec()),
		Outputs: []node.OutputSpec{
			{Name: wtgOutGone, Type: "Exec"},
			{Name: wtgOutTimeout, Type: "Exec",
				Data: []node.DataField{{Name: wtgDataConf, Type: "Number", Optional: true}}},
		},
	}
}

func (WaitTemplateGone) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	keys := in.StringList(wtgInTemplates)
	threshold := in.Float64(wtgInThreshold)
	timeout := time.Duration(in.Int(wtgInTimeoutMs)) * time.Millisecond

	// 先查当前帧：模板已消失 → 立即走 Gone
	hit, err := matchOnce(ctx, keys, threshold)
	if err != nil {
		return nil, node.Failf(node.CodeCaptureFailed, err, "WaitTemplateGone %s: %v", strings.Join(keys, "+"), err)
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
		if err := waitOrCancel(ctx, visionPollInterval); err != nil {
			return nil, err
		}
		hit, err = matchOnce(ctx, keys, threshold)
		if err != nil {
			return nil, node.Failf(node.CodeCaptureFailed, err, "WaitTemplateGone recheck %s: %v", strings.Join(keys, "+"), err)
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
