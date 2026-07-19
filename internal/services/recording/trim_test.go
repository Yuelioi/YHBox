package recording

import (
	"testing"

	"github.com/yottaapp/yotta/internal/services/inputclip"
)

func TestTrimPreciseEventsRebasesTimeAndPreservesSameTimestampSequence(t *testing.T) {
	events := []inputclip.Event{
		{TUs: 0, Seq: 0, Type: inputclip.EventTypeMouseMove, B: 10, C: 10},
		{TUs: 100, Seq: 1, Type: inputclip.EventTypeKeyDown, A: 'W'},
		{TUs: 100, Seq: 2, Type: inputclip.EventTypeKeyDown, A: 'D'},
		{TUs: 200, Seq: 3, Type: inputclip.EventTypeRawDelta, B: 2, C: -1},
		{TUs: 300, Seq: 4, Type: inputclip.EventTypeKeyUp, A: 'W'},
		{TUs: 300, Seq: 5, Type: inputclip.EventTypeKeyUp, A: 'D'},
		{TUs: 400, Seq: 6, Type: inputclip.EventTypeScroll, A: 1},
	}
	trimmed, err := trimPreciseEvents(events, 100, 300)
	if err != nil {
		t.Fatal(err)
	}
	if len(trimmed) != 5 || trimmed[0].TUs != 0 || trimmed[4].TUs != 200 {
		t.Fatalf("trimmed = %+v", trimmed)
	}
	for index, seq := range []uint32{1, 2, 3, 4, 5} {
		if trimmed[index].Seq != seq {
			t.Fatalf("event %d seq = %d, want %d", index, trimmed[index].Seq, seq)
		}
	}
}

func TestTrimPreciseEventsRejectsBoundariesInsideHeldInput(t *testing.T) {
	events := []inputclip.Event{
		{TUs: 0, Seq: 0, Type: inputclip.EventTypeKeyDown, A: 'W'},
		{TUs: 100, Seq: 1, Type: inputclip.EventTypeRawDelta, B: 2},
		{TUs: 200, Seq: 2, Type: inputclip.EventTypeKeyUp, A: 'W'},
	}
	if _, err := trimPreciseEvents(events, 100, 200); err == nil {
		t.Fatal("trim beginning inside held key succeeded")
	}
	if _, err := trimPreciseEvents(events, 0, 100); err == nil {
		t.Fatal("trim ending inside held key succeeded")
	}
}

func TestPendingEventsPagesPreciseRecordingWithinBudget(t *testing.T) {
	service := NewService(&resultRecorder{}, nil, nil, nil, nil)
	service.pending = &pendingRecording{result: &StopResult{
		TempID: "precise",
		Meta:   inputclip.ClipMeta{RecordingMode: inputclip.RecordingModePrecise},
		Events: []inputclip.Event{
			{TUs: 0, Type: inputclip.EventTypeMouseMove},
			{TUs: 1, Seq: 1, Type: inputclip.EventTypeRawDelta},
			{TUs: 2, Seq: 2, Type: inputclip.EventTypeScroll},
		},
	}}
	page, err := service.PendingEvents("pending-precise", 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 3 || len(page.Items) != 1 || page.Items[0].Seq != 1 {
		t.Fatalf("page = %+v", page)
	}
	if _, err := service.PendingEvents("pending-precise", 0, maxPendingEventPageSize+1); err == nil {
		t.Fatal("oversized page succeeded")
	}
}
