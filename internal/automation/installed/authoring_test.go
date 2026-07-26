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
	generation := authoringTestGeneration(t, "editor", profile, provider)
	targets, err := NewAuthoringTargets(generation)
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
	if err != nil || backend != desktopPayload(t, profile).CaptureBackend {
		t.Fatalf("CaptureBackend() = %q, %v", backend, err)
	}
	if _, err := targets.ResolveWindow(context.Background(), "missing"); err == nil {
		t.Fatal("ResolveWindow accepted an uninstalled slot")
	}
}

func TestAuthoringTargetsRejectInvalidProjection(t *testing.T) {
	if _, err := NewAuthoringTargets(Generation{}); err == nil {
		t.Fatal("NewAuthoringTargets accepted invalid generation")
	}
}

func TestAuthoringTargetsReplacePublishesNewGenerationToExistingHandle(t *testing.T) {
	profile, _ := testProfile(t)
	first := &provider{profile: profile, driver: &fakeDriver{window: target.WindowHandle{HWND: 41, ClientW: 800, ClientH: 600}}}
	firstGeneration := authoringTestGeneration(t, "first", profile, first)
	targets, err := NewAuthoringTargets(firstGeneration)
	if err != nil {
		t.Fatal(err)
	}
	copyOfHandle := targets
	second := &provider{profile: profile, driver: &fakeDriver{window: target.WindowHandle{HWND: 42, ClientW: 1280, ClientH: 720}}}
	secondGeneration := authoringTestGeneration(t, "second", profile, second)
	if err := targets.Replace(secondGeneration); err != nil {
		t.Fatal(err)
	}
	if _, err := copyOfHandle.ResolveWindow(context.Background(), "first"); err == nil {
		t.Fatal("existing authoring handle retained a removed generation")
	}
	window, err := copyOfHandle.ResolveWindow(context.Background(), "second")
	if err != nil || window.HWND != 42 {
		t.Fatalf("replacement window = %+v, %v", window, err)
	}
}

func TestRecordingTargetLeasePinsExactGenerationUntilSessionRelease(t *testing.T) {
	profile, _ := testProfile(t)
	driver := &fakeDriver{window: target.WindowHandle{HWND: 42, ClientW: 1280, ClientH: 720}}
	provider := &provider{profile: profile, driver: driver}
	generation := authoringTestGeneration(t, "editor", profile, provider)
	targets, err := NewAuthoringTargets(generation)
	if err != nil {
		t.Fatal(err)
	}
	window, counts360, release, err := targets.AcquireRecordingTarget(context.Background(), "editor")
	if err != nil {
		t.Fatal(err)
	}
	if window.HWND != 42 || counts360 != int(desktopPayload(t, profile).MouseCounts360) || driver.operation != OperationActivate {
		t.Fatalf("recording target window=%+v counts360=%d operation=%q", window, counts360, driver.operation)
	}
	if err := generation.Retire(); err != nil {
		t.Fatal(err)
	}
	if closed, _ := generation.Closed(); closed {
		t.Fatal("recording generation closed before the session released its lease")
	}
	release()
	if closed, err := generation.Closed(); !closed || err != nil {
		t.Fatalf("recording generation closed=%v error=%v", closed, err)
	}
}

func authoringTestGeneration(t *testing.T, slot string, profile Profile, installedProvider *provider) Generation {
	t.Helper()
	providerArtifact, err := ProviderArtifactDigest(profile)
	if err != nil {
		t.Fatal(err)
	}
	registered, err := defaultAdapterRegistry().registration(profile)
	if err != nil {
		t.Fatal(err)
	}
	providerID := "automation-test-provider"
	manifest, err := sealInstallationManifest(slot, slot, TargetID(slot), providerID, providerArtifact, profile, registered)
	if err != nil {
		t.Fatal(err)
	}
	installations := Installations{state: &installationState{
		entries: []Installation{{
			Slot: slot, Profile: profile, Manifest: manifest, ProviderID: providerID,
			ProviderArtifact: providerArtifact, TargetID: TargetID(slot), Provider: installedProvider, Descriptor: manifest.Descriptor(),
		}},
		providers: []*provider{installedProvider},
	}}
	generation, err := NewGeneration(installations)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = generation.Retire() })
	return generation
}
