package inputclip

import (
	"errors"
	"fmt"
	"sort"
)

// TrimEvents returns a standalone carrier-safe event sequence for the inclusive
// time range. Held keyboard and mouse state is synthesized at both boundaries,
// timestamps are rebased to zero, and sequence numbers are normalized.
func TrimEvents(events []Event, startUs, endUs uint64) ([]Event, error) {
	if len(events) == 0 {
		return nil, errors.New("recording has no events")
	}
	ordered := append([]Event(nil), events...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].TUs != ordered[j].TUs {
			return ordered[i].TUs < ordered[j].TUs
		}
		return ordered[i].Seq < ordered[j].Seq
	})
	if startUs > endUs || endUs > ordered[len(ordered)-1].TUs {
		return nil, errors.New("trim range is outside the recording")
	}

	heldKeys := map[int32]struct{}{}
	heldButtons := map[int32]Event{}
	for _, event := range ordered {
		if event.TUs >= startUs {
			break
		}
		updateHeldInput(heldKeys, heldButtons, event)
	}

	selected := make([]Event, 0, len(ordered)+len(heldKeys)+len(heldButtons))
	for _, key := range sortedHeldKeys(heldKeys) {
		selected = append(selected, Event{TUs: startUs, Type: EventTypeKeyDown, A: key})
	}
	for _, button := range sortedHeldButtons(heldButtons) {
		down := heldButtons[button]
		selected = append(selected, Event{TUs: startUs, Type: EventTypeMouseBtnDown, A: button, B: down.B, C: down.C})
	}
	for _, event := range ordered {
		if event.TUs >= startUs && event.TUs <= endUs {
			selected = append(selected, event)
		}
	}
	if len(selected) == 0 {
		return nil, errors.New("trim range contains no events")
	}

	for index := len(heldKeys) + len(heldButtons); index < len(selected); index++ {
		event := selected[index]
		switch event.Type {
		case EventTypeKeyDown:
			if _, held := heldKeys[event.A]; held {
				return nil, fmt.Errorf("event %d presses an already-held key", index)
			}
			heldKeys[event.A] = struct{}{}
		case EventTypeKeyUp:
			if _, held := heldKeys[event.A]; !held {
				return nil, fmt.Errorf("event %d releases a key that is not held", index)
			}
			delete(heldKeys, event.A)
		case EventTypeMouseBtnDown:
			if _, held := heldButtons[event.A]; held {
				return nil, fmt.Errorf("event %d presses an already-held mouse button", index)
			}
			heldButtons[event.A] = event
		case EventTypeMouseBtnUp:
			if _, held := heldButtons[event.A]; !held {
				return nil, fmt.Errorf("event %d releases a mouse button that is not held", index)
			}
			delete(heldButtons, event.A)
		}
	}
	keys := sortedHeldKeys(heldKeys)
	for index := len(keys) - 1; index >= 0; index-- {
		selected = append(selected, Event{TUs: endUs, Type: EventTypeKeyUp, A: keys[index]})
	}
	buttons := sortedHeldButtons(heldButtons)
	for index := len(buttons) - 1; index >= 0; index-- {
		button := buttons[index]
		down := heldButtons[button]
		selected = append(selected, Event{TUs: endUs, Type: EventTypeMouseBtnUp, A: button, B: down.B, C: down.C})
	}
	baseUs := selected[0].TUs
	for index := range selected {
		selected[index].TUs -= baseUs
		selected[index].Seq = uint32(index)
	}
	return selected, nil
}

func updateHeldInput(keys map[int32]struct{}, buttons map[int32]Event, event Event) {
	switch event.Type {
	case EventTypeKeyDown:
		keys[event.A] = struct{}{}
	case EventTypeKeyUp:
		delete(keys, event.A)
	case EventTypeMouseBtnDown:
		buttons[event.A] = event
	case EventTypeMouseBtnUp:
		delete(buttons, event.A)
	}
}

func sortedHeldKeys(keys map[int32]struct{}) []int32 {
	result := make([]int32, 0, len(keys))
	for key := range keys {
		result = append(result, key)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func sortedHeldButtons(buttons map[int32]Event) []int32 {
	result := make([]int32, 0, len(buttons))
	for button := range buttons {
		result = append(result, button)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}
