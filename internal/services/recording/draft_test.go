package recording

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/services/inputclip"
	"github.com/yottaapp/yotta/internal/services/macro"
)

func TestRecordingStateCloneKeepsPreviewCollectionsAsArrays(t *testing.T) {
	state := cloneRecordingState(RecordingState{Pending: &StopResultPayload{
		PendingID: "pending-session",
		Preview: RecordingPreview{
			Steps:  []RecordingPreviewStep{},
			Tracks: []RecordingTrack{},
		},
	}})
	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	for _, nullable := range []string{`"steps":null`, `"tracks":null`} {
		if strings.Contains(string(raw), nullable) {
			t.Fatalf("recording state leaked nullable collection %s: %s", nullable, raw)
		}
	}
}

func TestSimpleRecordingBuildsAtomicOverlappingMacro(t *testing.T) {
	result := &StopResult{
		Meta: inputclip.ClipMeta{RecordingMode: inputclip.RecordingModeSimple, BaseResolution: [2]int{1280, 720}},
		Events: []inputclip.Event{
			{TUs: 0, Type: inputclip.EventTypeKeyDown, A: 'W'},
			{TUs: 50_000, Type: inputclip.EventTypeKeyDown, A: 'D'},
			{TUs: 150_000, Type: inputclip.EventTypeKeyUp, A: 'W'},
			{TUs: 200_000, Type: inputclip.EventTypeKeyUp, A: 'D'},
		},
	}
	document, err := buildMacroDocument(result)
	if err != nil {
		t.Fatal(err)
	}
	want := []macro.ActionKind{
		macro.ActionKeyDown, macro.ActionSleep, macro.ActionKeyDown,
		macro.ActionSleep, macro.ActionKeyUp, macro.ActionSleep, macro.ActionKeyUp,
	}
	if len(document.Actions) != len(want) {
		t.Fatalf("actions = %+v", document.Actions)
	}
	for index, kind := range want {
		if document.Actions[index].Kind != kind {
			t.Fatalf("action %d = %s, want %s", index, document.Actions[index].Kind, kind)
		}
	}
	if document.Actions[0].Key != "W" || document.Actions[2].Key != "D" || document.Actions[4].Key != "W" || document.Actions[6].Key != "D" {
		t.Fatalf("overlapping key order = %+v", document.Actions)
	}
	if document.Actions[1].DurationUs != 50_000 || document.Actions[3].DurationUs != 100_000 || document.Actions[5].DurationUs != 50_000 {
		t.Fatalf("explicit sleeps = %+v", document.Actions)
	}
	if err := macro.Validate(document); err != nil {
		t.Fatalf("macro validation = %v", err)
	}

	preview := recordingPreview(result)
	if preview.Mode != "simple" || preview.KeyActions != 4 || len(preview.Steps) != len(want) {
		t.Fatalf("preview = %+v", preview)
	}
}

func TestPreciseRecordingPreviewRetainsMotionSummaryWithoutMacroProjection(t *testing.T) {
	result := &StopResult{
		Meta: inputclip.ClipMeta{RecordingMode: inputclip.RecordingModePrecise, BaseResolution: [2]int{1280, 720}},
		Events: []inputclip.Event{
			{TUs: 0, Type: inputclip.EventTypeKeyDown, A: 'W'},
			{TUs: 25_000, Type: inputclip.EventTypeRawDelta, B: 12, C: -4},
			{TUs: 50_000, Type: inputclip.EventTypeMouseMove, B: 640, C: 360},
			{TUs: 75_000, Type: inputclip.EventTypeKeyUp, A: 'W'},
		},
	}
	preview := recordingPreview(result)
	if preview.Mode != "precise" || preview.KeyActions != 2 || preview.RawDeltas != 1 || preview.PointerMoves != 1 {
		t.Fatalf("preview = %+v", preview)
	}
	if len(preview.Steps) != 1 || preview.Steps[0].Kind != "move-path" || preview.Steps[0].Samples != 2 {
		t.Fatalf("motion summary = %+v", preview.Steps)
	}
	if len(preview.Tracks) != 3 || preview.Tracks[0].Kind != "keyboard" || preview.Tracks[1].Kind != "absolute-motion" || preview.Tracks[2].Kind != "relative-motion" {
		t.Fatalf("tracks = %+v", preview.Tracks)
	}
	if _, err := buildMacroDocument(result); err == nil {
		t.Fatal("precise recording unexpectedly projected into a macro")
	}
}
