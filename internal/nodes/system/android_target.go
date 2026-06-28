// AndroidTarget switches the runtime active automation target to an ADB device.
package system

import (
	"encoding/json"
	"fmt"
	"strings"

	"yotta/internal/automation/target"
	"yotta/internal/node"
)

func init() { node.Register(&AndroidTarget{}) }

type AndroidTarget struct{}

const (
	atInExec   = "In"
	atInSerial = "Serial"
	atInName   = "Name"
	atInWidth  = "Width"
	atInHeight = "Height"
	atOutDone  = "Done"
)

func (AndroidTarget) Spec() node.Spec {
	return node.Spec{
		Kind:     "AndroidTarget",
		Category: "Window",
		Inputs: []node.InputSpec{
			{Name: atInExec, Type: node.TypeExec},
			{Name: atInSerial, Type: "String", Required: true, Widget: node.WidgetSpec{Kind: "text"}},
			{Name: atInName, Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "text"}},
			{Name: atInWidth, Type: "Number", Default: json.Number("1080"), Widget: node.WidgetSpec{Kind: "number"}},
			{Name: atInHeight, Type: "Number", Default: json.Number("1920"), Widget: node.WidgetSpec{Kind: "number"}},
		},
		Outputs: []node.OutputSpec{
			{Name: atOutDone, Type: node.TypeExec, Data: []node.DataField{
				{Name: "TargetID", Type: "String"},
				{Name: "Kind", Type: "String"},
			}},
		},
	}
}

func (AndroidTarget) Validate(in node.Inputs) []node.ValidationError {
	var errs []node.ValidationError
	if strings.TrimSpace(in.String(atInSerial)) == "" {
		errs = append(errs, node.ValidationError{Code: "REQUIRED_FIELD_MISSING", Message: "required field \"Serial\" missing", Field: atInSerial})
	}
	if in.Int(atInWidth) <= 0 {
		errs = append(errs, node.ValidationError{Code: "INVALID_FIELD", Message: "Width must be greater than 0", Field: atInWidth})
	}
	if in.Int(atInHeight) <= 0 {
		errs = append(errs, node.ValidationError{Code: "INVALID_FIELD", Message: "Height must be greater than 0", Field: atInHeight})
	}
	return errs
}

func (AndroidTarget) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	serial := strings.TrimSpace(in.String(atInSerial))
	name := strings.TrimSpace(in.String(atInName))
	if name == "" {
		name = serial
	}
	tg := target.Target{
		ID:          fmt.Sprintf("android:%s", serial),
		Kind:        target.KindAndroidADB,
		DisplayName: name,
		Ref:         target.TargetRef{ADBSerial: serial},
		Resolution:  target.Size{W: in.Int(atInWidth), H: in.Int(atInHeight)},
	}
	if err := ctx.Target().SetActive(tg); err != nil {
		return nil, err
	}
	return ctx.Out(atOutDone).Set("TargetID", tg.ID).Set("Kind", tg.Kind).Fire(), nil
}
