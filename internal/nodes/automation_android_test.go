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
	position, ok := builtins.Definition(GetPointerPositionNodeID)
	if !ok || !slices.Equal(position.Contract.Machine().ConfiguredTargets[0].TargetKinds, []string{installed.TargetKindDesktopWindow}) ||
		len(position.Contract.Machine().Ports.DataOutputs) != 1 || position.Contract.Machine().Ports.DataOutputs[0].ID != "point" {
		t.Fatalf("pointer position contract = %#v", position.Contract.Machine())
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

func TestMovePointerDefaultsWorkOnEveryAdvertisedTarget(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	move, ok := builtins.Definition(MovePointerNodeID)
	if !ok {
		t.Fatal("move pointer definition is missing")
	}
	if move.Contract.NodeRef().Version != "1.1.0" {
		t.Fatalf("move pointer version = %q", move.Contract.NodeRef().Version)
	}
	wantKinds := []string{installed.TargetKindAndroidDevice, installed.TargetKindBrowserCDP, installed.TargetKindDesktopWindow}
	if !slices.Equal(move.Contract.Machine().ConfiguredTargets[0].TargetKinds, wantKinds) {
		t.Fatalf("move pointer target kinds = %#v", move.Contract.Machine().ConfiguredTargets[0].TargetKinds)
	}
	defaults := make(map[string]string)
	for _, input := range move.Contract.Machine().Ports.DataInputs {
		if input.Default != nil {
			defaults[input.ID] = string(*input.Default)
		}
	}
	if defaults["duration"] != "0" || defaults["motion"] != `"instant"` {
		t.Fatalf("move pointer defaults = %#v", defaults)
	}
}
