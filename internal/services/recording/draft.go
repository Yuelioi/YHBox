package recording

import (
	"errors"

	"github.com/yottaapp/yotta/internal/services/inputclip"
	"github.com/yottaapp/yotta/internal/services/macro"
)

const maxPreviewSteps = 32

type RecordingPreview struct {
	Mode          string                 `json:"mode"`
	DurationUs    uint64                 `json:"durationUs"`
	EventCount    int                    `json:"eventCount"`
	KeyActions    int                    `json:"keyActions"`
	ClickActions  int                    `json:"clickActions"`
	PointerMoves  int                    `json:"pointerMoves"`
	RawDeltas     int                    `json:"rawDeltas"`
	ScrollActions int                    `json:"scrollActions"`
	Steps         []RecordingPreviewStep `json:"steps"`
	Tracks        []RecordingTrack       `json:"tracks"`
}

type RecordingTrack struct {
	Kind    string `json:"kind"`
	Count   int    `json:"count"`
	FirstUs uint64 `json:"firstUs"`
	LastUs  uint64 `json:"lastUs"`
}

type RecordingPreviewStep struct {
	Kind       string       `json:"kind"`
	AtUs       uint64       `json:"atUs"`
	DurationUs uint64       `json:"durationUs"`
	Key        string       `json:"key,omitempty"`
	Button     string       `json:"button,omitempty"`
	Point      *macro.Point `json:"point,omitempty"`
	Notches    int32        `json:"notches,omitempty"`
	Samples    int          `json:"samples,omitempty"`
}

func recordingPreview(result *StopResult) RecordingPreview {
	preview := RecordingPreview{
		Mode: string(result.Meta.RecordingMode), EventCount: len(result.Events), Steps: []RecordingPreviewStep{}, Tracks: []RecordingTrack{},
	}
	if len(result.Events) != 0 {
		preview.DurationUs = result.Events[len(result.Events)-1].TUs
	}
	if result.Meta.RecordingMode == inputclip.RecordingModeSimple {
		document, err := buildMacroDocument(result)
		if err != nil {
			return preview
		}
		preview.DurationUs = macro.Analyze(document).DurationUs
		var cursor uint64
		for _, action := range document.Actions {
			step := RecordingPreviewStep{
				Kind: string(action.Kind), AtUs: cursor, DurationUs: action.DurationUs,
				Key: action.Key, Button: action.Button, Notches: action.Notches,
			}
			if action.Point != nil {
				point := *action.Point
				step.Point = &point
			}
			switch action.Kind {
			case macro.ActionKeyDown, macro.ActionKeyUp:
				preview.KeyActions++
			case macro.ActionMouseDown, macro.ActionMouseUp, macro.ActionClick:
				preview.ClickActions++
			case macro.ActionMove:
				preview.PointerMoves++
			case macro.ActionDrag:
				preview.PointerMoves++
				preview.ClickActions++
			case macro.ActionScroll:
				preview.ScrollActions++
			}
			if len(preview.Steps) < maxPreviewSteps {
				preview.Steps = append(preview.Steps, step)
			}
			cursor += action.DurationUs
		}
		return preview
	}
	var firstMove, lastMove uint64
	hasMove := false
	tracks := map[string]*RecordingTrack{}
	for _, event := range result.Events {
		trackKind := ""
		switch event.Type {
		case inputclip.EventTypeKeyDown, inputclip.EventTypeKeyUp:
			preview.KeyActions++
			trackKind = "keyboard"
		case inputclip.EventTypeMouseBtnDown, inputclip.EventTypeMouseBtnUp:
			preview.ClickActions++
			trackKind = "mouse-buttons"
		case inputclip.EventTypeMouseMove:
			preview.PointerMoves++
			trackKind = "absolute-motion"
			if !hasMove {
				firstMove = event.TUs
				hasMove = true
			}
			lastMove = event.TUs
		case inputclip.EventTypeRawDelta:
			preview.RawDeltas++
			trackKind = "relative-motion"
			if !hasMove {
				firstMove = event.TUs
				hasMove = true
			}
			lastMove = event.TUs
		case inputclip.EventTypeScroll:
			preview.ScrollActions++
			trackKind = "scroll"
		}
		if trackKind != "" {
			track := tracks[trackKind]
			if track == nil {
				track = &RecordingTrack{Kind: trackKind, FirstUs: event.TUs}
				tracks[trackKind] = track
			}
			track.Count++
			track.LastUs = event.TUs
		}
	}
	for _, kind := range []string{"keyboard", "mouse-buttons", "absolute-motion", "relative-motion", "scroll"} {
		if track := tracks[kind]; track != nil {
			preview.Tracks = append(preview.Tracks, *track)
		}
	}
	if hasMove {
		preview.Steps = append(preview.Steps, RecordingPreviewStep{
			Kind: "move-path", AtUs: firstMove, DurationUs: lastMove - firstMove, Samples: preview.PointerMoves + preview.RawDeltas,
		})
	}
	return preview
}

func buildMacroDocument(result *StopResult) (macro.Document, error) {
	if result == nil || result.Meta.RecordingMode != inputclip.RecordingModeSimple {
		return macro.Document{}, errors.New("macro document requires a simple recording")
	}
	return macro.FromInputEvents(result.Events, result.Meta.BaseResolution)
}
