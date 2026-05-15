package actiontransform

import (
	"testing"

	"yhbox/internal/services/actions"
)

func TestMergeSleeps_KeepsSingleLongSleep(t *testing.T) {
	in := []actions.Step{
		{Kind: actions.StepSleep, DurationMs: 100},
	}
	out := MergeSleeps(in)
	if len(out) != 1 || out[0].DurationMs != 100 {
		t.Errorf("out=%+v want one 100ms sleep", out)
	}
}

func TestMergeSleeps_MergesAdjacentSleeps(t *testing.T) {
	in := []actions.Step{
		{Kind: actions.StepSleep, DurationMs: 60},
		{Kind: actions.StepSleep, DurationMs: 60},
	}
	out := MergeSleeps(in)
	if len(out) != 1 {
		t.Fatalf("len=%d want 1: %+v", len(out), out)
	}
	if out[0].DurationMs != 120 {
		t.Errorf("dur=%d want 120", out[0].DurationMs)
	}
}

func TestMergeSleeps_DropsShortSleep(t *testing.T) {
	in := []actions.Step{
		{Kind: actions.StepSleep, DurationMs: 30},
	}
	out := MergeSleeps(in)
	if len(out) != 0 {
		t.Errorf("out=%+v want empty (30ms sleep dropped)", out)
	}
}

func TestMergeSleeps_SleepKeySleepNotMerged(t *testing.T) {
	in := []actions.Step{
		{Kind: actions.StepSleep, DurationMs: 100},
		{Kind: actions.StepKey, Vk: "F", DurationMs: 50},
		{Kind: actions.StepSleep, DurationMs: 100},
	}
	out := MergeSleeps(in)
	if len(out) != 3 {
		t.Fatalf("len=%d want 3: %+v", len(out), out)
	}
	if out[0].Kind != actions.StepSleep || out[0].DurationMs != 100 {
		t.Errorf("step0=%+v want sleep 100", out[0])
	}
	if out[1].Kind != actions.StepKey {
		t.Errorf("step1=%+v want key", out[1])
	}
	if out[2].Kind != actions.StepSleep || out[2].DurationMs != 100 {
		t.Errorf("step2=%+v want sleep 100", out[2])
	}
}

func TestMergeSleeps_ShortSleepBetweenKeysDropped(t *testing.T) {
	in := []actions.Step{
		{Kind: actions.StepKey, Vk: "A", DurationMs: 50},
		{Kind: actions.StepSleep, DurationMs: 30}, // < 50 drop
		{Kind: actions.StepKey, Vk: "B", DurationMs: 50},
	}
	out := MergeSleeps(in)
	if len(out) != 2 {
		t.Fatalf("len=%d want 2: %+v", len(out), out)
	}
	if out[0].Vk != "A" || out[1].Vk != "B" {
		t.Errorf("out=%+v want [key A, key B]", out)
	}
}

func TestMergeSleeps_Empty(t *testing.T) {
	if got := MergeSleeps(nil); len(got) != 0 {
		t.Errorf("len=%d want 0", len(got))
	}
}
