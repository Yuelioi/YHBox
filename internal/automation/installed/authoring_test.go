package installed

import (
	"context"
	"testing"

	"github.com/yottaapp/yotta/internal/automation/target"
)

func TestAuthoringTargetsUseExactInstalledSlot(t *testing.T) {
	profile, _ := testProfile(t)
	driver := &fakeDriver{
		capture: []byte("png"),
		window:  target.WindowHandle{HWND: 42, Title: "Editor", ClientW: 1280, ClientH: 720},
	}
	provider := &provider{profile: profile, driver: driver}
	installations := Installations{state: &installationState{entries: []Installation{{
		Slot: "editor", Profile: profile, Provider: provider,
	}}}}
	targets, err := NewAuthoringTargets(installations)
	if err != nil {
		t.Fatal(err)
	}
	window, err := targets.ResolveWindow(context.Background(), "editor")
	if err != nil || window.HWND != 42 || window.ClientW != 1280 {
		t.Fatalf("ResolveWindow() = %+v, %v", window, err)
	}
	resolved, err := targets.ResolveTarget(context.Background(), "editor")
	if err != nil || resolved.Kind != target.KindWin32Window || resolved.Ref.HWND != 42 {
		t.Fatalf("ResolveTarget() = %+v, %v", resolved, err)
	}
	png, err := targets.CapturePNG(context.Background(), "editor")
	if err != nil || string(png) != "png" {
		t.Fatalf("CapturePNG() = %q, %v", png, err)
	}
	if err := targets.Activate(context.Background(), "editor"); err != nil || driver.operation != OperationActivate {
		t.Fatalf("Activate() operation = %q, error = %v", driver.operation, err)
	}
	backend, err := targets.CaptureBackend("editor")
	if err != nil || backend != profile.Machine().CaptureBackend {
		t.Fatalf("CaptureBackend() = %q, %v", backend, err)
	}
	if _, err := targets.ResolveWindow(context.Background(), "missing"); err == nil {
		t.Fatal("ResolveWindow accepted an uninstalled slot")
	}
}

func TestAuthoringTargetsRejectInvalidProjection(t *testing.T) {
	if _, err := NewAuthoringTargets(Installations{}); err == nil {
		t.Fatal("NewAuthoringTargets accepted invalid installations")
	}
}
