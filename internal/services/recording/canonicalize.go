package recording

import (
	"errors"
	"fmt"
	"sort"

	"github.com/yottaapp/yotta/internal/services/inputclip"
	"github.com/yottaapp/yotta/internal/services/macro"
)

// canonicalizeStopResult is the single boundary between lossy native delivery
// and the strict InputClip carrier. It applies the selected recording policy,
// normalizes time, and guarantees balanced input state before persistence.
func canonicalizeStopResult(result *StopResult) error {
	if result == nil {
		return errors.New("recording result is missing")
	}
	if !result.Meta.RecordingMode.Valid() {
		return errors.New("recording mode is invalid")
	}
	if len(result.Events) > inputclip.MaxInputClipEvents {
		return errors.New("recording exceeds the event budget")
	}

	events := append([]inputclip.Event(nil), result.Events...)
	sort.SliceStable(events, func(left, right int) bool {
		return events[left].Less(events[right])
	})

	activeKeys := make(map[int32]struct{})
	keyOrder := make([]int32, 0)
	activeButtons := make(map[int32]inputclip.Event)
	buttonOrder := make([]int32, 0)
	canonical := make([]inputclip.Event, 0, len(events)+len(activeKeys)+len(activeButtons))
	var firstUs uint64
	hasFirst := false

	appendEvent := func(event inputclip.Event) {
		if !hasFirst {
			firstUs = event.TUs
			hasFirst = true
		}
		event.TUs -= firstUs
		event.Seq = uint32(len(canonical))
		canonical = append(canonical, event)
	}

	for index, event := range events {
		if event.TUs > inputclip.MaxInputClipDurationUs {
			return fmt.Errorf("recording event %d exceeds the duration budget", index)
		}
		if result.Meta.RecordingMode == inputclip.RecordingModeSimple && !simpleEvent(event.Type) {
			continue
		}
		switch event.Type {
		case inputclip.EventTypeKeyDown:
			if event.A <= 0 || event.A > 255 {
				continue
			}
			if result.Meta.RecordingMode == inputclip.RecordingModeSimple && macro.KeyName(uint32(event.A)) == "" {
				continue
			}
			if _, exists := activeKeys[event.A]; exists {
				continue
			}
			activeKeys[event.A] = struct{}{}
			keyOrder = append(keyOrder, event.A)
			appendEvent(event)
		case inputclip.EventTypeKeyUp:
			if _, exists := activeKeys[event.A]; !exists {
				continue
			}
			delete(activeKeys, event.A)
			appendEvent(event)
		case inputclip.EventTypeMouseBtnDown:
			if !validRecordedButton(event.A) || !validRecordedPoint(event.B, event.C, result.Meta.BaseResolution) {
				continue
			}
			if _, exists := activeButtons[event.A]; exists {
				continue
			}
			activeButtons[event.A] = event
			buttonOrder = append(buttonOrder, event.A)
			appendEvent(event)
		case inputclip.EventTypeMouseBtnUp:
			if !validRecordedPoint(event.B, event.C, result.Meta.BaseResolution) {
				continue
			}
			if _, exists := activeButtons[event.A]; !exists {
				continue
			}
			delete(activeButtons, event.A)
			appendEvent(event)
		case inputclip.EventTypeMouseMove:
			if validRecordedPoint(event.B, event.C, result.Meta.BaseResolution) {
				appendEvent(event)
			}
		case inputclip.EventTypeRawDelta:
			if result.Meta.MouseCounts360 <= 0 {
				return errors.New("precise relative recording requires mouse calibration")
			}
			if event.B != 0 || event.C != 0 {
				appendEvent(event)
			}
		case inputclip.EventTypeScroll:
			if event.A != 0 && validRecordedPoint(event.B, event.C, result.Meta.BaseResolution) {
				appendEvent(event)
			}
		default:
			return fmt.Errorf("recording event %d has an unsupported type", index)
		}
	}

	if len(canonical) != 0 {
		lastUs := canonical[len(canonical)-1].TUs + firstUs
		for index := len(keyOrder) - 1; index >= 0; index-- {
			key := keyOrder[index]
			if _, active := activeKeys[key]; active {
				appendEvent(inputclip.Event{TUs: lastUs, Type: inputclip.EventTypeKeyUp, A: key})
			}
		}
		for index := len(buttonOrder) - 1; index >= 0; index-- {
			button := buttonOrder[index]
			if down, active := activeButtons[button]; active {
				appendEvent(inputclip.Event{TUs: lastUs, Type: inputclip.EventTypeMouseBtnUp, A: button, B: down.B, C: down.C})
			}
		}
	}
	if len(canonical) > inputclip.MaxInputClipEvents {
		return errors.New("canonical recording exceeds the event budget")
	}
	result.Events = canonical
	return nil
}

func simpleEvent(eventType inputclip.EventType) bool {
	return eventType == inputclip.EventTypeKeyDown || eventType == inputclip.EventTypeKeyUp ||
		eventType == inputclip.EventTypeMouseBtnDown || eventType == inputclip.EventTypeMouseBtnUp ||
		eventType == inputclip.EventTypeScroll
}

func validRecordedButton(button int32) bool {
	return button >= int32(HookBtnLeft) && button <= int32(HookBtnRight)
}

func validRecordedPoint(x, y int32, resolution [2]int) bool {
	return resolution[0] > 0 && resolution[1] > 0 && x >= 0 && y >= 0 &&
		int64(x) < int64(resolution[0]) && int64(y) < int64(resolution[1])
}
