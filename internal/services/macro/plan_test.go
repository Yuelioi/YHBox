package macro

import (
	"testing"

	"github.com/yottaapp/yotta/internal/automation/pointermotion"
)

func TestExecutionPlanAutoMovesBeforeClickAndCountsDuration(t *testing.T) {
	document := Document{
		SchemaVersion:  SchemaVersion,
		BaseResolution: [2]int{1000, 500},
		Meta:           Meta{AutoMove: AutoMove{Enabled: true, Mode: pointermotion.Bezier, DurationMilliseconds: 250}},
		Actions: []Action{{
			ID: "click", Kind: ActionClick, Button: "left",
			Point: &Point{X: 0.75, Y: 0.25, Unit: "ratio"}, DurationUs: 50_000,
		}},
	}

	plan, err := ExecutionPlan(document)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan) != 2 || plan[0].Kind != ActionMove || plan[0].Motion != pointermotion.Bezier || plan[0].DurationUs != 250_000 || plan[1].Kind != ActionClick {
		t.Fatalf("plan = %#v", plan)
	}
	if analysis := Analyze(document); len(analysis.Issues) != 0 || analysis.DurationUs != 300_000 {
		t.Fatalf("analysis = %#v", analysis)
	}
}

func TestExecutionPlanSkipsNearbyClickAndLeavesDragIndependent(t *testing.T) {
	document := Document{
		SchemaVersion:  SchemaVersion,
		BaseResolution: [2]int{1000, 500},
		Meta:           DefaultMeta(),
		Actions: []Action{
			{ID: "move", Kind: ActionMove, Point: &Point{X: 0.5, Y: 0.5, Unit: "ratio"}, Motion: pointermotion.Instant},
			{ID: "click", Kind: ActionClick, Button: "left", Point: &Point{X: 0.503, Y: 0.504, Unit: "ratio"}, DurationUs: 50_000},
			{ID: "drag", Kind: ActionDrag, Button: "left", From: &Point{X: 0.1, Y: 0.2, Unit: "ratio"}, Point: &Point{X: 0.9, Y: 0.8, Unit: "ratio"}, Motion: pointermotion.Linear, DurationUs: 400_000},
		},
	}

	plan, err := ExecutionPlan(document)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan) != 3 || plan[0].Kind != ActionMove || plan[1].Kind != ActionClick || plan[2].Kind != ActionDrag || plan[2].DurationUs != 400_000 {
		t.Fatalf("plan = %#v", plan)
	}
}

func TestExecutionPlanCanDisableAutoMove(t *testing.T) {
	document := Document{
		SchemaVersion:  SchemaVersion,
		BaseResolution: [2]int{1000, 500},
		Meta:           Meta{AutoMove: AutoMove{Enabled: false, Mode: pointermotion.Linear, DurationMilliseconds: 300}},
		Actions: []Action{{
			ID: "click", Kind: ActionClick, Button: "left", Point: &Point{X: 0.5, Y: 0.5, Unit: "ratio"}, DurationUs: 50_000,
		}},
	}

	plan, err := ExecutionPlan(document)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan) != 1 || plan[0].Kind != ActionClick {
		t.Fatalf("plan = %#v", plan)
	}
}
