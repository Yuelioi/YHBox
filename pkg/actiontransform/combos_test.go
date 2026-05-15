package actiontransform

import (
	"reflect"
	"testing"

	"yhbox/internal/services/actions"
)

func TestCollapseCombos_IdentityV1(t *testing.T) {
	in := []actions.Step{
		{Kind: actions.StepKeyDown, Vk: "Ctrl"},
		{Kind: actions.StepKey, Vk: "F", DurationMs: 50},
		{Kind: actions.StepKeyUp, Vk: "Ctrl"},
		{Kind: actions.StepSleep, DurationMs: 100},
	}
	out := CollapseCombos(in)
	if !reflect.DeepEqual(out, in) {
		t.Errorf("CollapseCombos v1 should be identity\ngot:  %+v\nwant: %+v", out, in)
	}
}

func TestCollapseCombos_Empty(t *testing.T) {
	if got := CollapseCombos(nil); len(got) != 0 {
		t.Errorf("len=%d want 0", len(got))
	}
}
