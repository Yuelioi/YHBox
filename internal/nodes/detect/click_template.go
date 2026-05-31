// internal/nodes/detect/click_template.go
// ClickTemplate — 等模板出现 → 在命中位置鼠标点击. 命中或超时后单出口路由.
//
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
	clkInExec      = "In"
	clkInTemplates = "Templates"
	clkInMatchMode = "MatchMode"
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
		NeedsWindow: true,
		Inputs: []node.InputSpec{
			{Name: clkInExec, Type: "Exec"},
			{Name: clkInTemplates, Type: "String", Semantic: "TemplateKey", Required: true,
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
		},
		Outputs: []node.OutputSpec{
			{Name: clkOutDone, Type: "Exec",
				Data: []node.DataField{
					{Name: clkDataPoint, Type: "Point"},
					{Name: clkDataConf, Type: "Number"},
				}},
			{Name: clkOutTimeout, Type: "Exec",
				Data: []node.DataField{
					{Name: clkDataConf, Type: "Number", Optional: true},
				}},
		},
	}
}

func (ClickTemplate) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	keys := in.StringList(clkInTemplates)
	mode := in.String(clkInMatchMode)
	threshold := in.Float64(clkInThreshold)
	timeout := time.Duration(in.Int(clkInTimeoutMs)) * time.Millisecond
	btn := in.String(clkInButton)
	if btn == "" {
		btn = "left"
	}
	pt, conf, err := ctx.Vision().WaitMatch(ctx.Context(), keys, threshold, mode, timeout)
	if err != nil {
		return nil, fmt.Errorf("ClickTemplate wait %s: %w", strings.Join(keys, "+"), err)
	}
	if pt == nil {
		return ctx.Out(clkOutTimeout).Set(clkDataConf, conf).Fire(), nil
	}
	// 50ms duration 对齐老 runtime (execClickTemplate 硬编码 50).
	if err := ctx.Input().Click(pt.X, pt.Y, btn, 50); err != nil {
		return nil, fmt.Errorf("ClickTemplate click %s @ (%.3f,%.3f): %w", strings.Join(keys, "+"), pt.X, pt.Y, err)
	}
	return ctx.Out(clkOutDone).Set(clkDataPoint, *pt).Set(clkDataConf, conf).Fire(), nil
}

func (ClickTemplate) Display(in node.Inputs, exitName string, out node.OutputData) string {
	label := strings.Join(in.StringList(clkInTemplates), "+")
	switch exitName {
	case clkOutDone:
		pt := out.Point(clkDataPoint)
		return fmt.Sprintf("✓ click %s %s @ (%.2f,%.2f)",
			in.String(clkInButton), label, pt.X, pt.Y)
	case clkOutTimeout:
		return fmt.Sprintf("⌛ %s timeout (%dms)", label, in.Int(clkInTimeoutMs))
	}
	return ""
}

func (ClickTemplate) Validate(in node.Inputs) []node.ValidationError {
	errs := validateTemplateKeys(in.StringList(clkInTemplates), clkInTemplates)
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
