package androidadb

import (
	"context"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
)

func TestRegisterNodeAsyncSourceAndroidDevices(t *testing.T) {
	runner := &fakeRunner{
		outs: map[string][]byte{
			"|devices -l":                 []byte("List of devices attached\nemulator-5554 device model:Pixel_8\nBAD unauthorized\n"),
			"emulator-5554|shell wm size": []byte("Physical size: 1080x2400\n"),
		},
		errs: map[string]error{},
	}
	nodeSvc := node.NewService()
	RegisterNodeAsyncSource(nodeSvc, NewService(runner))

	opts, err := nodeSvc.AsyncOptions("node-1", "AndroidTarget", AsyncSourceDevices, nil)
	if err != nil {
		t.Fatalf("AsyncOptions error = %v", err)
	}
	if len(opts) != 1 {
		t.Fatalf("options = %#v, want 1 online device", opts)
	}
	if opts[0].Value != "emulator-5554" || opts[0].Label != "Pixel 8 (emulator-5554, 1080x2400)" {
		t.Fatalf("option = %#v", opts[0])
	}
	if opts[0].Meta["name"] != "Pixel 8" || opts[0].Meta["width"] != 1080 || opts[0].Meta["height"] != 2400 {
		t.Fatalf("option meta = %#v", opts[0].Meta)
	}
}

func TestRegisterNodeAsyncSourceAndroidApps(t *testing.T) {
	runner := &fakeRunner{
		outs: map[string][]byte{
			"|devices -l":                               []byte("List of devices attached\n127.0.0.1:16384 device model:SDY_AN00\n"),
			"127.0.0.1:16384|shell wm size":             []byte("Physical size: 720x1280\n"),
			"127.0.0.1:16384|shell dumpsys input":       []byte("SurfaceOrientation: 1\n"),
			"127.0.0.1:16384|shell dumpsys window":      []byte("mCurrentFocus=Window{f111 u0 com.RoamingStar.BlueArchive/com.yostar.sdk.bridge.YoStarUnityPlayerActivity}\n"),
			"127.0.0.1:16384|shell pm list packages -3": []byte("package:com.RoamingStar.BlueArchive\npackage:com.example.other\n"),
			"127.0.0.1:16384|shell cmd package query-activities -a android.intent.action.MAIN -c android.intent.category.LAUNCHER": []byte(`Activity #0:
  ActivityInfo:
    name=com.yostar.sdk.bridge.YoStarUnityPlayerActivity
    packageName=com.RoamingStar.BlueArchive
    ApplicationInfo:
      packageName=com.RoamingStar.BlueArchive
Activity #1:
  ActivityInfo:
    name=com.example.OtherActivity
    packageName=com.example.other
    nonLocalizedLabel=Other Game
    ApplicationInfo:
      packageName=com.example.other
`),
		},
		errs: map[string]error{},
	}
	nodeSvc := node.NewService()
	RegisterNodeAsyncSource(nodeSvc, NewService(runner))

	opts, err := nodeSvc.AsyncOptions("node-1", "AndroidStartApp", AsyncSourceApps, nil)
	if err != nil {
		t.Fatalf("AsyncOptions error = %v", err)
	}
	if len(opts) != 2 {
		t.Fatalf("options = %#v, want 2 apps", opts)
	}
	if opts[0].Value != "com.RoamingStar.BlueArchive" || opts[0].Meta["foreground"] != true {
		t.Fatalf("first option = %#v, want foreground BlueArchive", opts[0])
	}
	if opts[1].Value != "com.example.other" || opts[1].Label != "Other Game (com.example.other)" {
		t.Fatalf("second option = %#v", opts[1])
	}
}

func TestFormatDeviceLabelWithoutResolution(t *testing.T) {
	got := formatDeviceLabel(Device{Serial: "ABC", Model: "sdk_phone"})
	if got != "sdk phone (ABC)" {
		t.Fatalf("label = %q", got)
	}
}

func TestRunnerFunc(t *testing.T) {
	r := RunnerFunc(func(_ context.Context, serial string, args ...string) ([]byte, error) {
		if serial != "s1" || len(args) != 1 || args[0] != "devices" {
			t.Fatalf("args = %q %#v", serial, args)
		}
		return []byte("ok"), nil
	})
	out, err := r.Run(context.Background(), "s1", "devices")
	if err != nil || string(out) != "ok" {
		t.Fatalf("Run = %q %v", out, err)
	}
}
