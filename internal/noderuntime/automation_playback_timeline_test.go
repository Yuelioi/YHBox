package noderuntime

import (
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/services/inputclip"
)

func TestPlaybackTimelineUsesAbsoluteDeadlines(t *testing.T) {
	const eventCount = 1500
	events := make([]inputclip.Event, eventCount)
	for index := range events {
		events[index] = inputclip.Event{
			TUs:  uint64(index) * uint64((16*time.Millisecond)/time.Microsecond),
			Seq:  uint32(index),
			Type: inputclip.EventTypeRawDelta,
			B:    1,
		}
	}

	started := time.Unix(0, 0)
	now := started
	commands := playbackCommands{
		Now: func() time.Time { return now },
		Wait: func(duration time.Duration) error {
			now = now.Add(duration)
			return nil
		},
		Play: func(installed.PlaybackEvent) error {
			// Simulate resource/provider/target/injection work for every event.
			now = now.Add(2 * time.Millisecond)
			return nil
		},
	}
	meta := inputclip.ClipMeta{MouseCounts360: 400, BaseResolution: [2]int{1920, 1080}}
	if err := playInputClipTimeline(events, meta, commands); err != nil {
		t.Fatal(err)
	}

	nominal := time.Duration(events[len(events)-1].TUs) * time.Microsecond
	if elapsed := now.Sub(started); elapsed != nominal+2*time.Millisecond {
		t.Fatalf("elapsed=%s, want nominal %s plus final injection cost", elapsed, nominal)
	}
}

func TestPlaybackTimelineCarriesSourceCalibrationOnlyOnRelativeMotion(t *testing.T) {
	events := []inputclip.Event{
		{Type: inputclip.EventTypeKeyDown, A: 0x57},
		{Seq: 1, Type: inputclip.EventTypeRawDelta, B: 3, C: -2},
	}
	var played []installed.PlaybackEvent
	commands := playbackCommands{
		Now:  time.Now,
		Wait: func(time.Duration) error { return nil },
		Play: func(event installed.PlaybackEvent) error {
			played = append(played, event)
			return nil
		},
	}
	meta := inputclip.ClipMeta{MouseCounts360: 400, BaseResolution: [2]int{1920, 1080}}
	if err := playInputClipTimeline(events, meta, commands); err != nil {
		t.Fatal(err)
	}
	if len(played) != 2 || played[0].SourceCounts360 != 0 || played[1].SourceCounts360 != 400 {
		t.Fatalf("played events = %#v", played)
	}
}
