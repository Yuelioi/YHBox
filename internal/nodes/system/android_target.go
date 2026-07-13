// AndroidTarget switches the runtime active automation target to an ADB device.
package system

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yottaapp/yotta/internal/automation/target"
	"github.com/yottaapp/yotta/internal/node"
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

	androidADBDevicesSource = "androidADBDevices"
)

func (AndroidTarget) Spec() node.Spec {
	return node.Spec{
		Kind:                "AndroidTarget",
		Category:            "Target",
		RuntimeCapabilities: []node.RuntimeCapability{node.RuntimeCapabilityTarget},
		Inputs: []node.InputSpec{
			{Name: atInExec, Type: node.TypeExec},
			{Name: atInSerial, Type: "String", Required: true, Constraints: []node.InputConstraint{{Kind: node.InputConstraintNonBlank}}, Widget: node.WidgetSpec{Kind: "async-dropdown",
				Props: node.MarshalProps(node.AsyncDropdownProps{AsyncSource: androidADBDevicesSource, ApplyMeta: map[string]string{
					"name":   atInName,
					"width":  atInWidth,
					"height": atInHeight,
				}})}},
			{Name: atInName, Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "text"}},
			{Name: atInWidth, Type: "Number", Default: json.Number("1080"), Constraints: []node.InputConstraint{{Kind: node.InputConstraintNumberGreaterThan, Threshold: "0"}}, Widget: node.WidgetSpec{Kind: "number"}},
			{Name: atInHeight, Type: "Number", Default: json.Number("1920"), Constraints: []node.InputConstraint{{Kind: node.InputConstraintNumberGreaterThan, Threshold: "0"}}, Widget: node.WidgetSpec{Kind: "number"}},
		},
		Outputs: []node.OutputSpec{
			{Name: atOutDone, Type: node.TypeExec, Data: []node.DataField{
				{Name: "TargetID", Type: "String"},
				{Name: "Kind", Type: "String"},
			}},
		},
	}
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
	if err := ctx.Services().Target.SetActive(tg); err != nil {
		return nil, err
	}
	return ctx.Out(atOutDone).Set("TargetID", tg.ID).Set("Kind", tg.Kind).Fire(), nil
}
