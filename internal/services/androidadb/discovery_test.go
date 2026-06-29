package androidadb

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type fakeRunner struct {
	calls []adbCall
	outs  map[string][]byte
	errs  map[string]error
}

type adbCall struct {
	serial string
	args   []string
}

func (f *fakeRunner) Run(_ context.Context, serial string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, adbCall{serial: serial, args: append([]string(nil), args...)})
	key := serial + "|" + stringsJoin(args)
	return f.outs[key], f.errs[key]
}

type queuedRunner struct {
	calls []adbCall
	outs  map[string][][]byte
	errs  map[string][]error
}

func (f *queuedRunner) Run(_ context.Context, serial string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, adbCall{serial: serial, args: append([]string(nil), args...)})
	key := serial + "|" + stringsJoin(args)
	var out []byte
	if q := f.outs[key]; len(q) > 0 {
		out = q[0]
		f.outs[key] = q[1:]
	}
	var err error
	if q := f.errs[key]; len(q) > 0 {
		err = q[0]
		f.errs[key] = q[1:]
	}
	return out, err
}

func stringsJoin(args []string) string {
	out := ""
	for i, arg := range args {
		if i > 0 {
			out += " "
		}
		out += arg
	}
	return out
}

func TestParseDevicesOutput(t *testing.T) {
	out := `List of devices attached
emulator-5554 device product:sdk_gphone64_x86_64 model:sdk_gphone64_x86_64 device:emu64xa
ABCDEF unauthorized

`
	got := ParseDevicesOutput(out)
	if len(got) != 2 {
		t.Fatalf("devices = %#v, want 2", got)
	}
	if got[0].Serial != "emulator-5554" || got[0].State != "device" || got[0].Model != "sdk_gphone64_x86_64" || got[0].Product == "" || got[0].Device == "" {
		t.Fatalf("first device = %#v", got[0])
	}
	if got[1].Serial != "ABCDEF" || got[1].State != "unauthorized" {
		t.Fatalf("second device = %#v", got[1])
	}
}

func TestParseWMSizeOutput(t *testing.T) {
	size, ok := ParseWMSizeOutput("Physical size: 1080x2400\n")
	if !ok || size.W != 1080 || size.H != 2400 {
		t.Fatalf("physical size = %#v ok=%v", size, ok)
	}
	size, ok = ParseWMSizeOutput("Physical size: 1440x3200\nOverride size: 1080x2400\n")
	if !ok || size.W != 1080 || size.H != 2400 {
		t.Fatalf("override size = %#v ok=%v", size, ok)
	}
	if _, ok := ParseWMSizeOutput("no size"); ok {
		t.Fatal("expected no parsed size")
	}
}

func TestServiceListDevicesQueriesResolutionForOnlineDevices(t *testing.T) {
	runner := &fakeRunner{
		outs: map[string][]byte{
			"|devices -l":                 []byte("List of devices attached\nemulator-5554 device model:Pixel_8\nBAD unauthorized\n"),
			"emulator-5554|shell wm size": []byte("Physical size: 1080x2400\n"),
		},
		errs: map[string]error{},
	}
	devices, err := NewService(runner).ListDevices(context.Background())
	if err != nil {
		t.Fatalf("ListDevices error = %v", err)
	}
	if len(devices) != 2 {
		t.Fatalf("devices = %#v, want 2", devices)
	}
	if devices[0].Resolution.W != 1080 || devices[0].Resolution.H != 2400 {
		t.Fatalf("resolution = %#v", devices[0].Resolution)
	}
	wantCalls := []adbCall{
		{serial: "", args: []string{"devices", "-l"}},
		{serial: "emulator-5554", args: []string{"shell", "wm", "size"}},
		{serial: "emulator-5554", args: []string{"shell", "dumpsys", "input"}},
	}
	if !reflect.DeepEqual(runner.calls, wantCalls) {
		t.Fatalf("calls = %#v, want %#v", runner.calls, wantCalls)
	}
}

func TestServiceListDevicesSwapsResolutionForLandscapeOrientation(t *testing.T) {
	runner := &fakeRunner{
		outs: map[string][]byte{
			"|devices -l":                   []byte("List of devices attached\n127.0.0.1:16384 device model:SDY_AN00\n"),
			"127.0.0.1:16384|shell wm size": []byte("Physical size: 720x1280\n"),
			"127.0.0.1:16384|shell dumpsys input": []byte(`
SurfaceOrientation: 1
`),
		},
		errs: map[string]error{},
	}
	devices, err := NewService(runner).ListDevices(context.Background())
	if err != nil {
		t.Fatalf("ListDevices error = %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("devices = %#v, want 1", devices)
	}
	if devices[0].Resolution.W != 1280 || devices[0].Resolution.H != 720 {
		t.Fatalf("resolution = %#v, want 1280x720", devices[0].Resolution)
	}
}

func TestServiceListDevicesKeepsDeviceWhenResolutionFails(t *testing.T) {
	runner := &fakeRunner{
		outs: map[string][]byte{
			"|devices -l": []byte("List of devices attached\nemulator-5554 device model:Pixel_8\n"),
		},
		errs: map[string]error{
			"emulator-5554|shell wm size": errors.New("wm failed"),
		},
	}
	devices, err := NewService(runner).ListDevices(context.Background())
	if err != nil {
		t.Fatalf("ListDevices error = %v", err)
	}
	if len(devices) != 1 || devices[0].Resolution.W != 0 {
		t.Fatalf("devices = %#v", devices)
	}
}

func TestServiceListDevicesAutoConnectsCommonEmulatorPortWhenNoOnlineDevice(t *testing.T) {
	runner := &queuedRunner{
		outs: map[string][][]byte{
			"|devices -l": {
				[]byte("List of devices attached\n"),
				[]byte("List of devices attached\n127.0.0.1:16384 device model:SDY_AN00 product:Sandy device:Sandy\n"),
			},
			"127.0.0.1:16384|shell wm size": {[]byte("Physical size: 1280x720\n")},
		},
		errs: map[string][]error{},
	}
	devices, err := NewService(runner).ListDevices(context.Background())
	if err != nil {
		t.Fatalf("ListDevices error = %v", err)
	}
	if len(devices) != 1 || devices[0].Serial != "127.0.0.1:16384" || devices[0].Resolution.W != 1280 {
		t.Fatalf("devices = %#v", devices)
	}
	wantPrefix := []adbCall{
		{serial: "", args: []string{"devices", "-l"}},
		{serial: "", args: []string{"connect", "127.0.0.1:16384"}},
	}
	if len(runner.calls) < len(wantPrefix) || !reflect.DeepEqual(runner.calls[:len(wantPrefix)], wantPrefix) {
		t.Fatalf("calls = %#v, want prefix %#v", runner.calls, wantPrefix)
	}
}

func TestServiceListAppsFallsBackToThirdPartyPackagesWhenLauncherQueryFails(t *testing.T) {
	runner := &fakeRunner{
		outs: map[string][]byte{
			"127.0.0.1:16384|shell dumpsys window":      []byte("mCurrentFocus=Window{f111 u0 com.example.game/.MainActivity}\n"),
			"127.0.0.1:16384|shell pm list packages -3": []byte("package:com.example.game\npackage:com.example.tool\n"),
		},
		errs: map[string]error{
			"127.0.0.1:16384|shell cmd package query-activities -a android.intent.action.MAIN -c android.intent.category.LAUNCHER": errors.New("query failed"),
		},
	}
	apps, err := NewService(runner).ListApps(context.Background(), "127.0.0.1:16384")
	if err != nil {
		t.Fatalf("ListApps error = %v", err)
	}
	if len(apps) != 2 {
		t.Fatalf("apps = %#v, want 2", apps)
	}
	if apps[0].Package != "com.example.game" || !apps[0].Foreground {
		t.Fatalf("first app = %#v, want foreground game", apps[0])
	}
	if apps[1].Package != "com.example.tool" {
		t.Fatalf("second app = %#v", apps[1])
	}
}
