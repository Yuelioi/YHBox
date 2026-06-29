package tools

import (
	"errors"
	"testing"

	"yotta/internal/automation/target"
	"yotta/pkg/winutil"
)

type recordingPickerAdapter struct {
	pickerCalls []PickerRequest
	pixelCalls  []PixelSampleRequest
	err         error
	pixel       PixelInfo
}

type fakeTargetKindResolver struct {
	targetKind string
	err        error
}

func (r fakeTargetKindResolver) ResolveWindowForNode(string, string) (winutil.WindowHandle, error) {
	return winutil.WindowHandle{}, nil
}

func (r fakeTargetKindResolver) ResolveEditorTargetKindForNode(string, string) (string, error) {
	return r.targetKind, r.err
}

func (r fakeTargetKindResolver) ResolveEditorTargetForNode(string, string) (target.Target, error) {
	return target.Target{Kind: r.targetKind}, r.err
}

func (r fakeTargetKindResolver) CaptureBackendFor(string) string { return "auto" }

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

	req := PickerRequest{Mode: "point", RequestID: "r1", ContainerID: "c1", NodeID: "n1"}
	if err := router.OpenPicker(target.KindAndroidADB, req); err != nil {
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
	err := router.OpenPicker("debug-replay", PickerRequest{Mode: "point", RequestID: "r1"})
	if err == nil || err.Error() != `target picker for "debug-replay" is not available` {
		t.Fatalf("err = %v", err)
	}
}

func TestAndroidTargetToolAdapter_PixelAtNotImplementedBoundary(t *testing.T) {
	_, err := androidTargetToolAdapter{}.PixelAt(PixelSampleRequest{ContainerID: "c1"})
	if !errors.Is(err, ErrAndroidTargetPixelNotImplemented) {
		t.Fatalf("err = %v, want ErrAndroidTargetPixelNotImplemented", err)
	}
}

func TestAndroidTargetToolAdapter_OpenPickerUsesSharedPickerWindow(t *testing.T) {
	err := androidTargetToolAdapter{service: NewService(nil)}.OpenPicker(PickerRequest{Mode: "rect", RequestID: "r1"})
	if errors.Is(err, ErrAndroidTargetPickerNotImplemented) {
		t.Fatalf("OpenPicker returned android not implemented boundary")
	}
	if err == nil {
		t.Fatalf("OpenPicker without Wails app should report not ready")
	}
}

func TestServiceOpenScreenPicker_ResolvesTargetKindAndRoutes(t *testing.T) {
	androidAdapter := &recordingPickerAdapter{}
	svc := NewService(fakeTargetKindResolver{targetKind: target.KindAndroidADB})
	svc.targetTools = newTargetToolRouter(map[string]TargetToolAdapter{
		target.KindAndroidADB: androidAdapter,
	})

	err := svc.OpenScreenPicker("rect", "req-1", "container-1", "node-1", "rgb", "asset-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(androidAdapter.pickerCalls) != 1 {
		t.Fatalf("android adapter calls = %+v", androidAdapter.pickerCalls)
	}
	got := androidAdapter.pickerCalls[0]
	if got.Mode != "rect" || got.RequestID != "req-1" || got.ContainerID != "container-1" || got.NodeID != "node-1" || got.ColorSpace != "rgb" || got.GUID != "asset-1" {
		t.Fatalf("picker request = %+v", got)
	}
}

func TestServicePixelAt_ResolvesTargetKindAndRoutes(t *testing.T) {
	androidAdapter := &recordingPickerAdapter{pixel: PixelInfo{OK: true, ClientX: 12}}
	svc := NewService(fakeTargetKindResolver{targetKind: target.KindAndroidADB})
	svc.targetTools = newTargetToolRouter(map[string]TargetToolAdapter{
		target.KindAndroidADB: androidAdapter,
	})

	got, err := svc.PixelAt("container-1", "node-1")
	if err != nil {
		t.Fatal(err)
	}
	if !got.OK || got.ClientX != 12 {
		t.Fatalf("pixel = %+v", got)
	}
	if len(androidAdapter.pixelCalls) != 1 || androidAdapter.pixelCalls[0].ContainerID != "container-1" || androidAdapter.pixelCalls[0].NodeID != "node-1" {
		t.Fatalf("pixel calls = %+v", androidAdapter.pixelCalls)
	}
}
