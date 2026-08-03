package macro

import (
	"errors"
	"fmt"

	"github.com/yottaapp/yotta/internal/automation/pointermotion"
	"github.com/yottaapp/yotta/internal/services/inputclip"
)

const (
	semanticMoveThresholdPixels = int32(4)
	moveAnchorGapUs             = uint64(75_000)
)

func fromInputEvents(events []inputclip.Event, resolution [2]int) (Document, error) {
	document := Document{SchemaVersion: SchemaVersion, BaseResolution: resolution, Meta: DefaultMeta(), Actions: []Action{}}
	if !validResolution(resolution) {
		return Document{}, errors.New("recording resolution is invalid")
	}

	var cursorUs uint64
	appendAction := func(action Action) {
		action.ID = actionID(len(document.Actions))
		document.Actions = append(document.Actions, action)
	}
	appendSleepUntil := func(atUs uint64) {
		if atUs > cursorUs {
			appendAction(Action{Kind: ActionSleep, DurationUs: atUs - cursorUs})
		}
	}

	for index := 0; index < len(events); {
		event := events[index]
		if event.Type == inputclip.EventTypeMouseMove {
			next, startedUs, endedUs := recordedPointerTravel(events, index)
			if next < len(events) {
				appendSleepUntil(startedUs)
				cursorUs = endedUs
			}
			index = next
			continue
		}
		if event.Type == inputclip.EventTypeMouseBtnDown {
			if action, next, ok := recordedPointerGesture(events, index, resolution); ok {
				appendSleepUntil(event.TUs)
				appendAction(action)
				cursorUs = events[next-1].TUs
				index = next
				continue
			}
		}

		appendSleepUntil(event.TUs)
		action, err := recordedAtomicAction(event, resolution)
		if err != nil {
			return Document{}, fmt.Errorf("recording event %d: %w", index, err)
		}
		appendAction(action)
		cursorUs = event.TUs
		index++
	}
	if err := Validate(document); err != nil {
		return Document{}, err
	}
	return document, nil
}

func recordedPointerTravel(events []inputclip.Event, index int) (int, uint64, uint64) {
	start := events[index]
	next := index + 1
	for next < len(events) && events[next].Type == inputclip.EventTypeMouseMove {
		next++
	}
	end := events[next-1]
	if next < len(events) && events[next].TUs >= end.TUs && events[next].TUs-end.TUs <= moveAnchorGapUs {
		anchor := events[next]
		if anchor.Type == inputclip.EventTypeMouseBtnDown || anchor.Type == inputclip.EventTypeScroll {
			end.TUs, end.B, end.C = anchor.TUs, anchor.B, anchor.C
		}
	}
	return next, start.TUs, end.TUs
}

func recordedPointerGesture(events []inputclip.Event, index int, resolution [2]int) (Action, int, bool) {
	down := events[index]
	next := index + 1
	for next < len(events) && events[next].Type == inputclip.EventTypeMouseMove {
		next++
	}
	if next >= len(events) {
		return Action{}, index, false
	}
	up := events[next]
	if up.Type != inputclip.EventTypeMouseBtnUp || up.A != down.A || up.TUs <= down.TUs {
		return Action{}, index, false
	}
	button := buttonName(down.A)
	if button == "" {
		return Action{}, index, false
	}
	duration := up.TUs - down.TUs
	click := duration <= MaxClickDurationUs
	for sampleIndex := index + 1; click && sampleIndex <= next; sampleIndex++ {
		click = !meaningfulMove(down, events[sampleIndex])
	}
	if click {
		return Action{
			Kind: ActionClick, Button: button, Point: ratioPoint(down.B, down.C, resolution), DurationUs: duration,
		}, next + 1, true
	}
	if duration > MaxMotionDurationUs {
		return Action{}, index, false
	}
	return Action{
		Kind: ActionDrag, Button: button, From: ratioPoint(down.B, down.C, resolution), Point: ratioPoint(up.B, up.C, resolution),
		DurationUs: duration, Motion: pointermotion.Linear,
	}, next + 1, true
}

func recordedAtomicAction(event inputclip.Event, resolution [2]int) (Action, error) {
	action := Action{}
	switch event.Type {
	case inputclip.EventTypeKeyDown, inputclip.EventTypeKeyUp:
		action.Key = KeyName(uint32(event.A))
		if action.Key == "" {
			return Action{}, fmt.Errorf("unsupported key code %d", event.A)
		}
		if event.Type == inputclip.EventTypeKeyDown {
			action.Kind = ActionKeyDown
		} else {
			action.Kind = ActionKeyUp
		}
	case inputclip.EventTypeMouseBtnDown, inputclip.EventTypeMouseBtnUp:
		action.Button = buttonName(event.A)
		if action.Button == "" {
			return Action{}, fmt.Errorf("unsupported mouse button %d", event.A)
		}
		action.Point = ratioPoint(event.B, event.C, resolution)
		if event.Type == inputclip.EventTypeMouseBtnDown {
			action.Kind = ActionMouseDown
		} else {
			action.Kind = ActionMouseUp
		}
	case inputclip.EventTypeScroll:
		action.Kind = ActionScroll
		action.Notches = event.A
		action.Point = ratioPoint(event.B, event.C, resolution)
	default:
		return Action{}, fmt.Errorf("event type %q is not a macro event", event.Type)
	}
	return action, nil
}

func meaningfulMove(from, to inputclip.Event) bool {
	deltaX := from.B - to.B
	if deltaX < 0 {
		deltaX = -deltaX
	}
	deltaY := from.C - to.C
	if deltaY < 0 {
		deltaY = -deltaY
	}
	return deltaX >= semanticMoveThresholdPixels || deltaY >= semanticMoveThresholdPixels
}
