package actions

import (
	"encoding/json"
	"testing"
)

func TestAction_JSONRoundTrip(t *testing.T) {
	src := Action{
		SchemaVersion:    2,
		ID:               "id-1",
		Name:             "test",
		RecordingContext: RecordingContext{Resolution: [2]int{1920, 1080}, DPIScale: 1.0},
		Params:           []ParamDecl{{Name: "pos", Type: "point", Default: nil}},
		Steps: []Step{
			{ID: "s1", Kind: StepClickLeft, XRatio: 0.5, YRatio: 0.5, DurationMs: 50},
			{ID: "s2", Kind: StepSleep, DurationMs: 1000},
		},
	}
	b, err := json.Marshal(src)
	if err != nil {
		t.Fatal(err)
	}
	var got Action
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Name != "test" || len(got.Steps) != 2 || got.Params[0].Name != "pos" {
		t.Errorf("round-trip lost: %+v", got)
	}
}

func TestIsClickStep(t *testing.T) {
	if !IsClickStep(StepClickLeft) || !IsClickStep(StepClickMiddle) || !IsClickStep(StepClickRight) {
		t.Error("click kinds should be click steps")
	}
	if IsClickStep(StepKey) || IsClickStep(StepSleep) {
		t.Error("non-click kinds should not be click steps")
	}
}
