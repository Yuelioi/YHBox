package recording

import (
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/services/inputclip"
)

func TestCanonicalizeSimpleRecordingUsesExplicitPolicyAndBalancesInput(t *testing.T) {
	result := &StopResult{
		Meta: inputclip.ClipMeta{
			RecordingMode: inputclip.RecordingModeSimple,
			MouseMode:     "absolute", BaseResolution: [2]int{1280, 720},
		},
		Events: []inputclip.Event{
			{TUs: 650, Seq: 5, Type: inputclip.EventTypeRawDelta, B: 12, C: -4},
			{TUs: 520, Seq: 2, Type: inputclip.EventTypeKeyUp, A: 'A'},
			{TUs: 100, Seq: 0, Type: inputclip.EventTypeMouseMove, B: 10, C: 20},
			{TUs: 600, Seq: 3, Type: inputclip.EventTypeMouseBtnDown, A: int32(HookBtnLeft), B: 640, C: 360},
			{TUs: 200, Seq: 1, Type: inputclip.EventTypeKeyUp, A: 'B'},
			{TUs: 500, Seq: 1, Type: inputclip.EventTypeKeyDown, A: 'A'},
			{TUs: 550, Seq: 2, Type: inputclip.EventTypeScroll, A: -2, B: 640, C: 360},
		},
	}
	if err := canonicalizeStopResult(result); err != nil {
		t.Fatal(err)
	}
	if len(result.Events) != 6 {
		t.Fatalf("canonical events = %#v", result.Events)
	}
	wantTypes := []inputclip.EventType{
		inputclip.EventTypeMouseMove, inputclip.EventTypeKeyDown, inputclip.EventTypeKeyUp, inputclip.EventTypeScroll,
		inputclip.EventTypeMouseBtnDown, inputclip.EventTypeMouseBtnUp,
	}
	for index, event := range result.Events {
		if event.Type != wantTypes[index] || event.Seq != uint32(index) {
			t.Fatalf("event %d = %#v", index, event)
		}
	}
	if result.Events[0].TUs != 0 || result.Events[1].TUs != 400 || result.Events[2].TUs != 420 || result.Events[3].TUs != 450 || result.Events[4].TUs != 500 || result.Events[5].TUs != 500 {
		t.Fatalf("normalized times = %#v", result.Events)
	}
}

func TestCanonicalizePreciseRecordingRetainsTrajectoryWithoutInferringMode(t *testing.T) {
	result := &StopResult{
		Meta: inputclip.ClipMeta{
			RecordingMode: inputclip.RecordingModePrecise, MouseMode: "mixed",
			BaseResolution: [2]int{1920, 1080}, MouseCounts360: 400,
		},
		Events: []inputclip.Event{
			{TUs: 1_000, Type: inputclip.EventTypeKeyDown, A: 'W'},
			{TUs: 1_100, Type: inputclip.EventTypeMouseMove, B: 960, C: 540},
			{TUs: 1_200, Type: inputclip.EventTypeRawDelta, B: 3, C: -2},
			{TUs: 1_300, Type: inputclip.EventTypeKeyUp, A: 'W'},
		},
	}
	if err := canonicalizeStopResult(result); err != nil {
		t.Fatal(err)
	}
	if len(result.Events) != 4 || result.Events[0].TUs != 0 || result.Events[3].TUs != 300 {
		t.Fatalf("canonical precise events = %#v", result.Events)
	}
	preview := recordingPreview(result)
	if preview.Mode != "precise" || preview.KeyActions != 2 || preview.PointerMoves != 1 || preview.RawDeltas != 1 || len(preview.Steps) != 1 || preview.Steps[0].Kind != "move-path" || preview.Steps[0].Samples != 2 {
		t.Fatalf("preview = %#v", preview)
	}
}

func TestCanonicalizeRejectsUncalibratedRelativeTrajectory(t *testing.T) {
	result := &StopResult{
		Meta: inputclip.ClipMeta{
			RecordingMode: inputclip.RecordingModePrecise,
			MouseMode:     "relative", BaseResolution: [2]int{1920, 1080},
		},
		Events: []inputclip.Event{{Type: inputclip.EventTypeRawDelta, B: 1}},
	}
	if err := canonicalizeStopResult(result); err == nil || !strings.Contains(err.Error(), "calibration") {
		t.Fatalf("canonicalize error = %v", err)
	}
}
