package nodes

import (
	"slices"
	"testing"

	"github.com/yottaapp/yotta/internal/automation/installed"
)

func TestAndroidAutomationCapabilitiesExposeOnlySupportedSemantics(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
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
	for _, nodeID := range []string{MovePointerRelativeNodeID, PressKeysNodeID} {
		definition, _ := builtins.Definition(nodeID)
		if definition.Contract.Machine().CapabilityRequirements[0].Capability.CapabilityID != AutomationDesktopInputCapabilityID {
			t.Fatalf("%s did not use desktop-only input authority", nodeID)
		}
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
