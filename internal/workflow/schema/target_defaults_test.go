package schema

import "testing"

func TestTargetDefaultsRemainSortedAndUnique(t *testing.T) {
	source := WorkflowSource{}
	if err := SetTargetDefault(&source, "target", "window-target"); err != nil {
		t.Fatal(err)
	}
	if err := SetTargetDefault(&source, "application", "game-app"); err != nil {
		t.Fatal(err)
	}
	if err := SetTargetDefault(&source, "target", "other-target"); err != nil {
		t.Fatal(err)
	}
	if len(source.TargetDefaults) != 2 || source.TargetDefaults[0].Target != "application" || source.TargetDefaults[1].Slot != "other-target" {
		t.Fatalf("target defaults = %+v", source.TargetDefaults)
	}
	if err := validateTargetDefaults(source.TargetDefaults); err != nil {
		t.Fatal(err)
	}
}
