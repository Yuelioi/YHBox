package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/yottaapp/yotta/internal/automation/target"
)

type recordingPickerAdapter struct {
	pickerCalls []PickerRequest
	pixelCalls  []PixelSampleRequest
	err         error
	pixel       PixelInfo
}

type fakeTargetResolver struct {
	target target.Target
	err    error
}

func (r fakeTargetResolver) ResolveWindow(context.Context, string) (target.WindowHandle, error) {
	return target.WindowHandle{}, nil
}

func (r fakeTargetResolver) ResolveTarget(context.Context, string) (target.Target, error) {
	return r.target, r.err
}

func (r fakeTargetResolver) CaptureBackend(string) (string, error) { return "auto", nil }

func (a *recordingPickerAdapter) OpenPicker(req PickerRequest) error {
	a.pickerCalls = append(a.pickerCalls, req)
	return a.err
}

func (a *recordingPickerAdapter) PixelAt(req PixelSampleRequest) (PixelInfo, error) {
	a.pixelCalls = append(a.pixelCalls, req)
	return a.pixel, a.err
}

func TestTargetToolRouter_RoutesPickerByTargetKind(t *testing.T) {
	winAdapter := &recordingPickerAdapter{}
	androidAdapter := &recordingPickerAdapter{}
	router := newTargetToolRouter(map[string]TargetToolAdapter{
		target.KindWin32Window: winAdapter,
		target.KindAndroidADB:  androidAdapter,
	})

	req := PickerRequest{Mode: "point", RequestID: "r1", TargetSlot: "editor"}
	if err := router.OpenPicker(target.Target{Kind: target.KindAndroidADB}, req); err != nil {
		t.Fatal(err)
	}
	if len(androidAdapter.pickerCalls) != 1 || androidAdapter.pickerCalls[0].RequestID != "r1" {
		t.Fatalf("android calls = %+v", androidAdapter.pickerCalls)
	}
	if len(winAdapter.pickerCalls) != 0 {
		t.Fatalf("win adapter should not be called, got %+v", winAdapter.pickerCalls)
	}
}

func TestTargetToolRouter_UnknownTargetKindFails(t *testing.T) {
	router := newTargetToolRouter(nil)
	err := router.OpenPicker(target.Target{Kind: target.KindDebugReplay}, PickerRequest{Mode: "point", RequestID: "r1"})
	if err == nil || err.Error() != `target picker for "debug-replay" is not available` {
		t.Fatalf("err = %v", err)
	}
}

func TestAndroidTargetToolAdapter_PixelAtNotImplementedBoundary(t *testing.T) {
	_, err := androidTargetToolAdapter{}.PixelAt(PixelSampleRequest{TargetSlot: "editor"})
	if !errors.Is(err, ErrAndroidTargetPixelNotImplemented) {
		t.Fatalf("err = %v, want ErrAndroidTargetPixelNotImplemented", err)
	}
}

func TestAndroidTargetToolAdapter_OpenPickerUsesSharedPickerWindow(t *testing.T) {
	err := androidTargetToolAdapter{service: NewService(nil, nil)}.OpenPicker(PickerRequest{Mode: "rect", RequestID: "r1"})
	if errors.Is(err, ErrAndroidTargetPickerNotImplemented) {
		t.Fatalf("OpenPicker returned android not implemented boundary")
	}
	if err == nil {
		t.Fatalf("OpenPicker without Wails app should report not ready")
	}
}

func TestServiceOpenScreenPicker_ResolvesTargetKindAndRoutes(t *testing.T) {
	androidAdapter := &recordingPickerAdapter{}
	svc := NewService(fakeTargetResolver{target: target.Target{Kind: target.KindAndroidADB}}, nil)
	svc.targetTools = newTargetToolRouter(map[string]TargetToolAdapter{
		target.KindAndroidADB: androidAdapter,
	})

	err := svc.OpenScreenPicker("rect", "req-1", "editor", "rgb", "asset-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(androidAdapter.pickerCalls) != 1 {
		t.Fatalf("android adapter calls = %+v", androidAdapter.pickerCalls)
	}
	got := androidAdapter.pickerCalls[0]
	if got.Mode != "rect" || got.RequestID != "req-1" || got.TargetSlot != "editor" || got.ColorSpace != "rgb" || got.GUID != "asset-1" {
		t.Fatalf("picker request = %+v", got)
	}
}

func TestServicePixelAt_ResolvesTargetKindAndRoutes(t *testing.T) {
	androidAdapter := &recordingPickerAdapter{pixel: PixelInfo{OK: true, ClientX: 12}}
	svc := NewService(fakeTargetResolver{target: target.Target{Kind: target.KindAndroidADB}}, nil)
	svc.targetTools = newTargetToolRouter(map[string]TargetToolAdapter{
		target.KindAndroidADB: androidAdapter,
	})

	got, err := svc.PixelAt("editor")
	if err != nil {
		t.Fatal(err)
	}
	if !got.OK || got.ClientX != 12 {
		t.Fatalf("pixel = %+v", got)
	}
	if len(androidAdapter.pixelCalls) != 1 || androidAdapter.pixelCalls[0].TargetSlot != "editor" {
		t.Fatalf("pixel calls = %+v", androidAdapter.pixelCalls)
	}
}
