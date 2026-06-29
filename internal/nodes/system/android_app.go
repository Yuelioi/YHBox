package system

import (
	"fmt"
	"strings"

	"yotta/internal/node"
)

func init() {
	node.Register(&AndroidStartApp{})
	node.Register(&AndroidStopApp{})
}

const (
	androidAppInExec    = "In"
	androidAppInPackage = "Package"
	androidAppOutDone   = "Done"

	androidADBAppsSource = "androidADBApps"
)

type AndroidStartApp struct{}

func (AndroidStartApp) Spec() node.Spec {
	return androidAppSpec("AndroidStartApp", node.TargetCapabilityStartApp)
}

func (AndroidStartApp) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	packageName, err := androidPackageInput(in)
	if err != nil {
		return nil, err
	}
	if ctx.App() == nil {
		return nil, fmt.Errorf("AndroidStartApp: app lifecycle service is not available")
	}
	if err := ctx.App().StartApp(packageName); err != nil {
		return nil, node.Failf(node.CodeLaunchFailed, err, "AndroidStartApp %q failed: %v", packageName, err)
	}
	return ctx.Out(androidAppOutDone).Fire(), nil
}

func (AndroidStartApp) Validate(in node.Inputs) []node.ValidationError {
	return validateAndroidPackage(in)
}

type AndroidStopApp struct{}

func (AndroidStopApp) Spec() node.Spec {
	return androidAppSpec("AndroidStopApp", node.TargetCapabilityStopApp)
}

func (AndroidStopApp) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	packageName, err := androidPackageInput(in)
	if err != nil {
		return nil, err
	}
	if ctx.App() == nil {
		return nil, fmt.Errorf("AndroidStopApp: app lifecycle service is not available")
	}
	if err := ctx.App().StopApp(packageName); err != nil {
		return nil, node.Failf(node.CodeError, err, "AndroidStopApp %q failed: %v", packageName, err)
	}
	return ctx.Out(androidAppOutDone).Fire(), nil
}

func (AndroidStopApp) Validate(in node.Inputs) []node.ValidationError {
	return validateAndroidPackage(in)
}

func androidAppSpec(kind string, capability node.TargetCapability) node.Spec {
	return node.Spec{
		Kind:               kind,
		Category:           "IO",
		NeedsTarget:        true,
		TargetCapabilities: []node.TargetCapability{capability},
		Inputs: []node.InputSpec{
			{Name: androidAppInExec, Type: node.TypeExec},
			{Name: androidAppInPackage, Type: "String", Required: true, Widget: node.WidgetSpec{Kind: "async-dropdown",
				Props: node.MarshalProps(node.AsyncDropdownProps{AsyncSource: androidADBAppsSource})}},
		},
		Outputs: []node.OutputSpec{
			{Name: androidAppOutDone, Type: node.TypeExec},
		},
	}
}

func androidPackageInput(in node.Inputs) (string, error) {
	packageName := strings.TrimSpace(in.String(androidAppInPackage))
	if packageName == "" {
		return "", fmt.Errorf("Package is required")
	}
	return packageName, nil
}

func validateAndroidPackage(in node.Inputs) []node.ValidationError {
	if strings.TrimSpace(in.String(androidAppInPackage)) == "" {
		return []node.ValidationError{{
			Code:    "REQUIRED_FIELD_MISSING",
			Message: "required field \"Package\" missing",
			Field:   androidAppInPackage,
		}}
	}
	return nil
}
