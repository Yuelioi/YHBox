package nodes

import (
	"slices"
	"testing"

	"github.com/yottaapp/yotta/internal/automation/installed"
)

func TestAutomationNodesExposeOnlyConfiguredTargetSemantics(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	moveRelative, _ := builtins.Definition(MovePointerRelativeNodeID)
	if !slices.Equal(moveRelative.Contract.Machine().ConfiguredTargets[0].TargetKinds, []string{installed.TargetKindDesktopWindow}) {
		t.Fatal("relative movement is not desktop-only")
	}
	pressKeys, _ := builtins.Definition(PressKeysNodeID)
	if !slices.Contains(pressKeys.Contract.Machine().ConfiguredTargets[0].TargetKinds, installed.TargetKindBrowserCDP) {
		t.Fatal("key node is not browser compatible")
	}
	stop, ok := builtins.Definition(StopTargetAppNodeID)
	if !ok || !slices.Equal(stop.Contract.Machine().ConfiguredTargets[0].TargetKinds, []string{installed.TargetKindAndroidDevice}) {
		t.Fatalf("stop target app = %#v", stop.Contract.Machine())
	}
}
