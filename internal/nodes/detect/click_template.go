// internal/nodes/detect/click_template.go
// ClickTemplate — 等模板出现 → 在命中位置鼠标点击. 命中或超时后单出口路由.
//
// 100ms 内部轮询 (见 visionWaitPollMs) + 可选 SettleMs 命中后稳定延迟 + 50ms click duration.
// 可选 MaxAttempts>1: 点完验证模板消失没, 没消失就重点 (每下间隔 RetryIntervalMs), 点满还在走 Timeout(Matched=true).
package detect

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"yotta/internal/node"
)

func init() { node.Register(&ClickTemplate{}) }

type ClickTemplate struct{}

const (
	clkInExec            = "In"
	clkInTemplates       = "Templates"
	clkInMatchMode       = "MatchMode"
	clkInTimeoutMs       = "TimeoutMs"
	clkInThreshold       = "Threshold"
	clkInButton          = "Button"
	clkInSettleMs        = "SettleMs"
	clkInMaxAttempts     = "MaxAttempts"
	clkInRetryIntervalMs = "RetryIntervalMs"
	clkOutDone           = "Done"
	clkOutTimeout        = "Timeout"
	clkDataPoint         = "Point"
	clkDataConf          = "Conf"
	clkDataMatched       = "Matched" // 命中与否 (bool) Data 字段 — 两出口都带, 供自动捕获 (Spec C)
)

func (ClickTemplate) Spec() node.Spec {
	return node.Spec{
		Kind:        "ClickTemplate",
		Category:    "Detect",
		NeedsWindow: true,
		Inputs: []node.InputSpec{
			{Name: clkInExec, Type: "Exec"},
			{Name: clkInTemplates, Type: "String", Semantic: "TemplateGUID", Required: true,
				Widget: node.WidgetSpec{Kind: "template-picker"}},
			{Name: clkInMatchMode, Type: "String", Default: "any", Advanced: true,
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{{Value: "any"}, {Value: "all"}}})}},
			{Name: clkInTimeoutMs, Type: "Number", Default: json.Number("5000"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: clkInThreshold, Type: "Number", Default: json.Number("0.85"),
				Widget: node.WidgetSpec{Kind: "slider",
					Props: node.MarshalProps(node.SliderProps{Min: 0, Max: 1, Step: 0.01})}},
			{Name: clkInButton, Type: "String", Default: "left",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "left"},
							{Value: "right"},
							{Value: "middle"},
						}})}},
			{Name: clkInSettleMs, Type: "Number", Default: json.Number("0"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: clkInMaxAttempts, Type: "Number", Default: json.Number("1"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: clkInRetryIntervalMs, Type: "Number", Default: json.Number("500"),
				Widget: node.WidgetSpec{Kind: "number"}},
		},
		Outputs: []node.OutputSpec{
			{Name: clkOutDone, Type: "Exec",
				Data: []node.DataField{
					{Name: clkDataPoint, Type: "Point"},
					{Name: clkDataConf, Type: "Number"},
					{Name: clkDataMatched, Type: "Bool"},
				}},
			{Name: clkOutTimeout, Type: "Exec",
				Data: []node.DataField{
					{Name: clkDataConf, Type: "Number", Optional: true},
					{Name: clkDataMatched, Type: "Bool"},
				}},
		},
	}
}

func (ClickTemplate) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	keys := in.StringList(clkInTemplates)
	mode := in.String(clkInMatchMode)
	threshold := in.Float64(clkInThreshold)
	timeout := time.Duration(in.Int(clkInTimeoutMs)) * time.Millisecond
	settle := time.Duration(in.Int(clkInSettleMs)) * time.Millisecond
	maxAttempts := in.Int(clkInMaxAttempts)
	retryInterval := time.Duration(in.Int(clkInRetryIntervalMs)) * time.Millisecond
	btn := in.String(clkInButton)
	if btn == "" {
		btn = "left"
	}
	pt, conf, err := ctx.Vision().WaitMatch(ctx.Context(), keys, threshold, mode, timeout)
	if err != nil {
		return nil, node.Failf(node.CodeCaptureFailed, err, "ClickTemplate wait %s: %v", strings.Join(keys, "+"), err)
	}
	if pt == nil {
		return ctx.Out(clkOutTimeout).Set(clkDataConf, conf).Set(clkDataMatched, false).Fire(), nil
	}
	// 命中后可选稳定延迟 + 重定位 (SettleMs): 防"刚出现就点、点空了"。settle=0 → 行为同旧。详见 settleAfterMatch。
	pt, conf, err = settleAfterMatch(ctx, keys, threshold, mode, settle, pt, conf)
	if err != nil {
		return nil, err // settle 期间被取消 (graph stop) → 优雅 halt
	}
	if err := clickAt(ctx, keys, pt, btn); err != nil {
		return nil, err
	}
	// MaxAttempts<=1 → 旧行为: 点一次即放行, 不验证。
	if maxAttempts <= 1 {
		return ctx.Out(clkOutDone).Set(clkDataPoint, *pt).Set(clkDataConf, conf).Set(clkDataMatched, true).Fire(), nil
	}
	// 点完验证 (MaxAttempts>1): 等 RetryIntervalMs → 模板消失=点成了走 Done; 还在且没点满=重定位再点;
	// 还在且点满=没点掉走 Timeout (Matched=true, 跟"压根没出现"的 Matched=false 区分)。
	clicks := 1
	for {
		if err := waitOrCancel(ctx, retryInterval); err != nil {
			return nil, err // 间隔期间被取消 (graph stop) → 优雅 halt
		}
		pt2, conf2, err := matchOnce(ctx, keys, threshold, mode)
		if err != nil {
			return nil, node.Failf(node.CodeCaptureFailed, err, "ClickTemplate recheck %s: %v", strings.Join(keys, "+"), err)
		}
		if pt2 == nil {
			return ctx.Out(clkOutDone).Set(clkDataPoint, *pt).Set(clkDataConf, conf).Set(clkDataMatched, true).Fire(), nil
		}
		if clicks >= maxAttempts {
			return ctx.Out(clkOutTimeout).Set(clkDataConf, conf2).Set(clkDataMatched, true).Fire(), nil
		}
		pt, conf = pt2, conf2 // 重定位到最新坐标 (元素还在动也跟得上)
		if err := clickAt(ctx, keys, pt, btn); err != nil {
			return nil, err
		}
		clicks++
	}
}

// clickAt 在命中点鼠标点击 (50ms duration), 失败包成 CodeCaptureFailed。首次点击 + 重试共用。
func clickAt(ctx node.Ctx, keys []string, pt *node.Point, btn string) error {
	if err := ctx.Input().Click(pt.X, pt.Y, btn, 50); err != nil {
		return node.Failf(node.CodeCaptureFailed, err, "ClickTemplate click %s @ (%.3f,%.3f): %v", strings.Join(keys, "+"), pt.X, pt.Y, err)
	}
	return nil
}

func (ClickTemplate) Validate(in node.Inputs) []node.ValidationError {
	var errs []node.ValidationError
	btn := in.String(clkInButton)
	if btn != "" && btn != "left" && btn != "right" && btn != "middle" {
		errs = append(errs, node.ValidationError{
			Code:    "INVALID_MOUSE_BUTTON",
			Message: fmt.Sprintf("button %q not in left/right/middle", btn),
			Field:   clkInButton,
		})
	}
	return errs
}

func (ClickTemplate) Dependencies(in node.Inputs) []node.Dependency {
	return templateDeps(in.StringList(clkInTemplates))
}
