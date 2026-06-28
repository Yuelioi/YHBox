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
	}
	if !reflect.DeepEqual(runner.calls, wantCalls) {
		t.Fatalf("calls = %#v, want %#v", runner.calls, wantCalls)
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
