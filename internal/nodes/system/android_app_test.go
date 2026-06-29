package system

import (
	"context"
	"testing"

	"yotta/internal/node"
)

type recordingAppLifecycle struct {
	starts []string
	stops  []string
	err    error
}

func (r *recordingAppLifecycle) StartApp(packageName string) error {
	r.starts = append(r.starts, packageName)
	return r.err
}

func (r *recordingAppLifecycle) StopApp(packageName string) error {
	r.stops = append(r.stops, packageName)
	return r.err
}

func TestAndroidStartAppSpecRequiresTargetCapability(t *testing.T) {
	spec := AndroidStartApp{}.Spec()
	if spec.Kind != "AndroidStartApp" || spec.Category != "IO" || !spec.NeedsTarget {
		t.Fatalf("spec = %#v", spec)
	}
	if len(spec.TargetCapabilities) != 1 || spec.TargetCapabilities[0] != node.TargetCapabilityStartApp {
		t.Fatalf("TargetCapabilities = %#v", spec.TargetCapabilities)
	}
}

func TestAndroidStartAppRun(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&AndroidStartApp{})
	rn, _ := node.Get("AndroidStartApp")

	app := &recordingAppLifecycle{}
	svc := node.StubServices()
	svc.App = app
	r := node.RunNode(context.Background(), rn, nil, map[string]any{
		androidAppInPackage: " com.nexon.bluearchive ",
	}, nil, svc, false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != androidAppOutDone {
		t.Fatalf("exit = %q, want Done", r.ExitName)
	}
	if len(app.starts) != 1 || app.starts[0] != "com.nexon.bluearchive" {
		t.Fatalf("starts = %#v", app.starts)
	}
}

func TestAndroidStopAppRun(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&AndroidStopApp{})
	rn, _ := node.Get("AndroidStopApp")

	app := &recordingAppLifecycle{}
	svc := node.StubServices()
	svc.App = app
	r := node.RunNode(context.Background(), rn, nil, map[string]any{
		androidAppInPackage: "com.nexon.bluearchive",
	}, nil, svc, false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if len(app.stops) != 1 || app.stops[0] != "com.nexon.bluearchive" {
		t.Fatalf("stops = %#v", app.stops)
	}
}

func TestAndroidAppValidateRequiresPackage(t *testing.T) {
	errs := AndroidStartApp{}.Validate(node.NewInputsFromConfig(nil))
	if len(errs) != 1 || errs[0].Field != androidAppInPackage {
		t.Fatalf("Validate = %#v", errs)
	}
}
