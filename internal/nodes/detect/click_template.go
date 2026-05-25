// internal/nodes/detect/click_template.go
// ClickTemplate — 等模板出现 → 在命中位置鼠标点击. 命中或超时后单出口路由.
//
// 老 runtime: internal/services/container/runtime/nodes.go::execClickTemplate.
// 250ms 内部轮询 + InputBus.Lock 独占 + 50ms click duration.
package detect

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"yhbox/internal/node"
)

func init() { node.Register(&ClickTemplate{}) }

type ClickTemplate struct{}

const (
	clkInExec      = "in"
	clkInTemplate  = "Template"
	clkInTimeoutMs = "TimeoutMs"
	clkInThreshold = "Threshold"
	clkInButton    = "Button"
	clkOutDone     = "Done"
	clkOutTimeout  = "Timeout"
	clkDataPoint   = "Point"
	clkDataConf    = "Conf"
)

func (ClickTemplate) Spec() node.Spec {
	return node.Spec{
		Kind:        "ClickTemplate",
		Category:    "Detect",
		DisplayName: "点击模板",
		Description: "等模板在 timeoutMs 内出现, 命中后在中心点鼠标点击. 超时走 Timeout.",
		Inputs: []node.InputSpec{
			{Name: clkInExec, Type: "Exec"},
			{Name: clkInTemplate, Type: "String", Semantic: "TemplateKey", Required: true,
				DisplayName: "模板", Doc: "命名空间.名 格式, e.g. fishing.start_fish",
				Widget: node.WidgetSpec{Kind: "async-dropdown",
					Props: node.MarshalProps(node.AsyncDropdownProps{AsyncSource: "templateKeys"})}},
			{Name: clkInTimeoutMs, Type: "Number", Default: json.Number("5000"),
				DisplayName: "超时 (ms)",
				Widget:      node.WidgetSpec{Kind: "number"}},
			{Name: clkInThreshold, Type: "Number", Default: json.Number("0.85"),
				DisplayName: "阈值",
				Widget: node.WidgetSpec{Kind: "slider",
					Props: node.MarshalProps(node.SliderProps{Min: 0, Max: 1, Step: 0.01})}},
			{Name: clkInButton, Type: "String", Default: "left",
				DisplayName: "按键",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "left", Label: "左键"},
							{Value: "right", Label: "右键"},
							{Value: "middle", Label: "中键"},
						}})}},
		},
		Outputs: []node.OutputSpec{
			{Name: clkOutDone, Type: "Exec", DisplayName: "完成",
				Data: []node.DataField{
					{Name: clkDataPoint, Type: "Point", Doc: "点击点 (ratio)"},
					{Name: clkDataConf, Type: "Number", Doc: "实际匹配度"},
				}},
			{Name: clkOutTimeout, Type: "Exec", DisplayName: "超时",
				Data: []node.DataField{
					{Name: clkDataConf, Type: "Number", Optional: true, Doc: "最高匹配度 (低于阈值)"},
				}},
		},
	}
}

func (ClickTemplate) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	key := in.String(clkInTemplate)
	threshold := in.Float64(clkInThreshold)
	timeout := time.Duration(in.Int(clkInTimeoutMs)) * time.Millisecond
	btn := in.String(clkInButton)
	if btn == "" {
		btn = "left"
	}
	pt, conf, err := ctx.Vision().WaitMatch(ctx.Context(), key, threshold, timeout)
	if err != nil {
		return nil, fmt.Errorf("ClickTemplate wait %q: %w", key, err)
	}
	if pt == nil {
		return ctx.Out(clkOutTimeout).Set(clkDataConf, conf).Fire(), nil
	}
	// 50ms duration 对齐老 runtime (execClickTemplate 硬编码 50).
	if err := ctx.Input().Click(pt.X, pt.Y, btn, 50); err != nil {
		return nil, fmt.Errorf("ClickTemplate click %q @ (%.3f,%.3f): %w", key, pt.X, pt.Y, err)
	}
	return ctx.Out(clkOutDone).Set(clkDataPoint, *pt).Set(clkDataConf, conf).Fire(), nil
}

func (ClickTemplate) Display(in node.Inputs, exitName string, out node.OutputData) string {
	switch exitName {
	case clkOutDone:
		pt := out.Point(clkDataPoint)
		return fmt.Sprintf("✓ click %s %s @ (%.2f,%.2f)",
			in.String(clkInButton), in.String(clkInTemplate), pt.X, pt.Y)
	case clkOutTimeout:
		return fmt.Sprintf("⌛ %s timeout (%dms)", in.String(clkInTemplate), in.Int(clkInTimeoutMs))
	}
	return ""
}

func (ClickTemplate) Validate(in node.Inputs) []node.ValidationError {
	var errs []node.ValidationError
	key := in.String(clkInTemplate)
	if key != "" && !strings.Contains(key, ".") {
		errs = append(errs, node.ValidationError{
			Code:    "INVALID_TEMPLATE_KEY",
			Message: fmt.Sprintf("template key 必须 namespace.name 格式, got %q", key),
			Field:   clkInTemplate,
		})
	}
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
	return []node.Dependency{{Kind: "template", Key: in.String(clkInTemplate)}}
}
