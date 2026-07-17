package nodes

import (
	"slices"
	"testing"

	"github.com/yottaapp/yotta/internal/automation/installed"
)

func TestAutomationCapabilitiesExposeOnlySupportedTargetSemantics(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	for _, capabilityID := range []string{AutomationInputCapabilityID, AutomationCaptureCapabilityID} {
		definition, _ := builtins.Catalog.LookupCapability(capabilityID)
		if !slices.Contains(definition.Machine().TargetKinds, installed.TargetKindBrowserCDP) {
			t.Fatalf("browser target missing from shared capability %q", capabilityID)
		}
	}
	for _, capabilityID := range []string{AutomationInputCapabilityID, AutomationCaptureCapabilityID, AutomationWindowCapabilityID} {
		definition, ok := builtins.Catalog.LookupCapability(capabilityID)
		if !ok || !slices.Contains(definition.Machine().TargetKinds, installed.TargetKindDesktopWindow) || !slices.Contains(definition.Machine().TargetKinds, installed.TargetKindAndroidDevice) {
			t.Fatalf("shared capability %q = %#v", capabilityID, definition.Machine())
		}
	}
	desktopInput, ok := builtins.Catalog.LookupCapability(AutomationDesktopInputCapabilityID)
	if !ok || !slices.Equal(desktopInput.Machine().TargetKinds, []string{installed.TargetKindDesktopWindow}) {
		t.Fatalf("desktop input capability = %#v", desktopInput.Machine())
	}
	moveRelative, _ := builtins.Definition(MovePointerRelativeNodeID)
	if moveRelative.Contract.Machine().CapabilityRequirements[0].Capability.CapabilityID != AutomationDesktopInputCapabilityID {
		t.Fatal("relative movement did not retain desktop-only authority")
	}
	keyInput, ok := builtins.Catalog.LookupCapability(AutomationKeyInputCapabilityID)
	if !ok || !slices.Equal(keyInput.Machine().TargetKinds, []string{installed.TargetKindBrowserCDP, installed.TargetKindDesktopWindow}) {
		t.Fatalf("key input capability = %#v", keyInput.Machine())
	}
	pressKeys, _ := builtins.Definition(PressKeysNodeID)
	if pressKeys.Contract.Machine().CapabilityRequirements[0].Capability.CapabilityID != AutomationKeyInputCapabilityID {
		t.Fatal("key node did not use browser-compatible key authority")
	}
	stop, ok := builtins.Definition(StopTargetAppNodeID)
	if !ok || stop.Contract.Machine().CapabilityRequirements[0].Capability.CapabilityID != AutomationAppLifecycleCapabilityID || stop.Contract.Machine().CapabilityRequirements[0].Operations[0] != installed.OperationStopApp {
		t.Fatalf("stop target app = %#v", stop.Contract.Machine())
	}
	lifecycle, _ := builtins.Catalog.LookupCapability(AutomationAppLifecycleCapabilityID)
	if !slices.Equal(lifecycle.Machine().TargetKinds, []string{installed.TargetKindAndroidDevice}) {
		t.Fatalf("Android lifecycle target kinds = %v", lifecycle.Machine().TargetKinds)
	}
}
