// BrowserTarget switches the runtime active automation target to a Chrome-compatible CDP page.
package system

import (
	"encoding/json"
	"fmt"
	"strings"

	"yotta/internal/automation/target"
	"yotta/internal/node"
	"yotta/internal/services/browsercdp"
)

func init() { node.Register(&BrowserTarget{}) }

type BrowserTarget struct{}

const (
	btInExec         = "In"
	btInEndpoint     = "Endpoint"
	btInBrowserID    = "BrowserID"
	btInName         = "Name"
	btInWebSocketURL = "WebSocketURL"
	btInWidth        = "Width"
	btInHeight       = "Height"
	btOutDone        = "Done"
)

func (BrowserTarget) Spec() node.Spec {
	return node.Spec{
		Kind:     "BrowserTarget",
		Category: "Window",
		Inputs: []node.InputSpec{
			{Name: btInExec, Type: node.TypeExec},
			{Name: btInEndpoint, Type: "String", Default: browsercdp.DefaultEndpoint, Widget: node.WidgetSpec{Kind: "text"}},
			{Name: btInBrowserID, Type: "String", Required: true, Widget: node.WidgetSpec{Kind: "async-dropdown",
				Props: node.MarshalProps(node.AsyncDropdownProps{AsyncSource: browsercdp.AsyncSourceTargets})}},
			{Name: btInName, Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "text"}},
			{Name: btInWebSocketURL, Type: "String", Default: "", Advanced: true, Widget: node.WidgetSpec{Kind: "text"}},
			{Name: btInWidth, Type: "Number", Default: json.Number("1280"), Widget: node.WidgetSpec{Kind: "number"}},
			{Name: btInHeight, Type: "Number", Default: json.Number("720"), Widget: node.WidgetSpec{Kind: "number"}},
		},
		Outputs: []node.OutputSpec{
			{Name: btOutDone, Type: node.TypeExec, Data: []node.DataField{
				{Name: "TargetID", Type: "String"},
				{Name: "Kind", Type: "String"},
			}},
		},
	}
}

func (BrowserTarget) Validate(in node.Inputs) []node.ValidationError {
	var errs []node.ValidationError
	if strings.TrimSpace(in.String(btInBrowserID)) == "" {
		errs = append(errs, node.ValidationError{Code: "REQUIRED_FIELD_MISSING", Message: "required field \"BrowserID\" missing", Field: btInBrowserID})
	}
	if in.Int(btInWidth) <= 0 {
		errs = append(errs, node.ValidationError{Code: "INVALID_FIELD", Message: "Width must be greater than 0", Field: btInWidth})
	}
	if in.Int(btInHeight) <= 0 {
		errs = append(errs, node.ValidationError{Code: "INVALID_FIELD", Message: "Height must be greater than 0", Field: btInHeight})
	}
	return errs
}

func (BrowserTarget) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	browserID := strings.TrimSpace(in.String(btInBrowserID))
	name := strings.TrimSpace(in.String(btInName))
	if name == "" {
		name = browserID
	}
	endpoint := strings.TrimSpace(in.String(btInEndpoint))
	if endpoint == "" {
		endpoint = browsercdp.DefaultEndpoint
	}
	wsURL := strings.TrimSpace(in.String(btInWebSocketURL))
	tg := target.Target{
		ID:          fmt.Sprintf("browser:%s", browserID),
		Kind:        target.KindBrowserCDP,
		DisplayName: name,
		Ref:         target.TargetRef{BrowserID: browserID},
		Resolution:  target.Size{W: in.Int(btInWidth), H: in.Int(btInHeight)},
		Metadata: map[string]any{
			"endpoint":             endpoint,
			"webSocketDebuggerUrl": wsURL,
		},
	}
	if err := ctx.Target().SetActive(tg); err != nil {
		return nil, err
	}
	return ctx.Out(btOutDone).Set("TargetID", tg.ID).Set("Kind", tg.Kind).Fire(), nil
}
