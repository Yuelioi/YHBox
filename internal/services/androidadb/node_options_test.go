package androidadb

import (
	"context"
	"testing"

	"yotta/internal/node"
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
