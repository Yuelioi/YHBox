package noderuntime

import (
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/automation/pointermotion"
	"github.com/yottaapp/yotta/internal/services/macro"
)

func TestPlayMacroClickUsesOneAtomicPlaybackEvent(t *testing.T) {
	var events []installed.PlaybackEvent
	err := playMacroAction(macro.Action{
		ID:         "click",
		Kind:       macro.ActionClick,
		Button:     "left",
		Point:      &macro.Point{X: 0.25, Y: 0.75, Unit: "ratio"},
		DurationUs: 94_000,
	}, playbackCommands{
		Wait: func(time.Duration) error { return nil },
		Play: func(event installed.PlaybackEvent) error {
			events = append(events, event)
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Kind != installed.PlaybackClick || events[0].DurationMilliseconds != 94 {
		t.Fatalf("macro click playback events = %#v, want one atomic click", events)
	}
}

func TestPlayMacroMoveAndDragPreserveSemanticMotion(t *testing.T) {
	var events []installed.PlaybackEvent
	commands := playbackCommands{
		Wait: func(time.Duration) error { return nil },
		Play: func(event installed.PlaybackEvent) error {
			events = append(events, event)
			return nil
		},
	}
	if err := playMacroAction(macro.Action{
		ID: "move", Kind: macro.ActionMove, Point: &macro.Point{X: 0.4, Y: 0.6, Unit: "ratio"},
		DurationUs: 150_001, Motion: pointermotion.Linear,
	}, commands); err != nil {
		t.Fatal(err)
	}
	if err := playMacroAction(macro.Action{
		ID: "drag", Kind: macro.ActionDrag, Button: "left",
		From: &macro.Point{X: 0.1, Y: 0.2, Unit: "ratio"}, Point: &macro.Point{X: 0.8, Y: 0.9, Unit: "ratio"},
		DurationUs: 300_000, Motion: pointermotion.Bezier,
	}, commands); err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Kind != installed.PlaybackMove || events[0].Motion != pointermotion.Linear || events[0].DurationMilliseconds != 151 {
		t.Fatalf("move event = %#v", events)
	}
	if events[1].Kind != installed.PlaybackDrag || events[1].Motion != pointermotion.Bezier || events[1].DurationMilliseconds != 300 ||
		events[1].From == nil || events[1].Point == nil || events[1].From.X != 0.1 || events[1].Point.X != 0.8 {
		t.Fatalf("drag event = %#v", events[1])
	}
}

func TestMacroExecutionPlanAddsConfiguredMoveBeforeClick(t *testing.T) {
	document := macro.Document{
		SchemaVersion:  macro.SchemaVersion,
		BaseResolution: [2]int{1920, 1080},
		Meta: macro.Meta{AutoMove: macro.AutoMove{
			Enabled: true, Mode: pointermotion.Bezier, DurationMilliseconds: 250,
		}},
		Actions: []macro.Action{{
			ID: "click", Kind: macro.ActionClick, Button: "left",
			Point: &macro.Point{X: 0.25, Y: 0.75, Unit: "ratio"}, DurationUs: 50_000,
		}},
	}
	plan, err := macro.ExecutionPlan(document)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan) != 2 || plan[0].Kind != macro.ActionMove || plan[0].Motion != pointermotion.Bezier || plan[0].DurationUs != 250_000 || plan[1].Kind != macro.ActionClick {
		t.Fatalf("execution plan = %#v", plan)
	}
}
