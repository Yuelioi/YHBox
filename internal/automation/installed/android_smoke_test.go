package installed

import (
	"bytes"
	"context"
	"image/png"
	"os"
	"testing"
	"time"
)

func TestAndroidADBEmulatorSmoke(t *testing.T) {
	if os.Getenv("YOTTA_ADB_SMOKE") != "1" {
		t.Skip("set YOTTA_ADB_SMOKE=1 with an authorized emulator or device")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	devices, err := DiscoverAndroidDevices(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var selected *AndroidDeviceDescriptor
	for index := range devices {
		if devices[index].State == "device" && devices[index].Product != "" && devices[index].Model != "" && devices[index].Device != "" {
			selected = &devices[index]
			break
		}
	}
	if selected == nil {
		t.Fatal("no ready ADB device with complete identity")
	}
	profile, err := SealProfile(NewAndroidProfileDraft(AndroidProfilePayload{
		ADBSerial: selected.Serial, ADBProduct: selected.Product, ADBModel: selected.Model, ADBDevice: selected.Device,
		AndroidPackage: "com.android.settings", ResolveTimeoutMilliseconds: 5000,
	}))
	if err != nil {
		t.Fatal(err)
	}
	opened, err := newAndroidDriver(profile)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	resolved, err := opened.ResolveTarget(ctx)
	if err != nil || resolved.Resolution.W <= 0 || resolved.Resolution.H <= 0 {
		t.Fatalf("ResolveTarget = %#v, %v", resolved, err)
	}
	if err := opened.Execute(ctx, OperationActivate, struct{}{}); err != nil {
		t.Fatal(err)
	}
	data, err := opened.Capture(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := png.Decode(bytes.NewReader(data)); err != nil {
		t.Fatalf("capture is not PNG: %v", err)
	}
	center := Point{X: 0.5, Y: 0.5, Unit: "ratio"}
	requests := []struct {
		operation string
		request   any
	}{
		{OperationMove, MoveRequest{Point: center}},
		{OperationClick, ClickRequest{Point: center, Button: "left", DurationMilliseconds: 10}},
		{OperationDrag, DragRequest{From: Point{X: 0.5, Y: 0.55, Unit: "ratio"}, To: Point{X: 0.5, Y: 0.45, Unit: "ratio"}, Button: "left", DurationMilliseconds: 100}},
		{OperationScroll, ScrollRequest{Point: center, Notches: 1}},
		{OperationTypeText, TypeTextRequest{Text: "yotta"}},
	}
	for _, request := range requests {
		if err := opened.Execute(ctx, request.operation, request.request); err != nil {
			t.Fatalf("%s: %v", request.operation, err)
		}
	}
	if err := opened.Execute(ctx, OperationStopApp, struct{}{}); err != nil {
		t.Fatal(err)
	}
}
