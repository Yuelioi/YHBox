package recording

import (
	"errors"
	"fmt"
	"sort"

	"github.com/yottaapp/yotta/internal/services/inputclip"
)

const maxPendingEventPageSize = 200

type PendingEventPage struct {
	Items  []inputclip.Event `json:"items"`
	Total  int               `json:"total"`
	Offset int               `json:"offset"`
	Limit  int               `json:"limit"`
}

// PendingEvents returns a bounded diagnostic page without copying the entire
// raw trajectory into the recording state event or mounting it in the DOM.
func (s *Service) PendingEvents(pendingID string, offset, limit int) (PendingEventPage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pending == nil || "pending-"+s.pending.result.TempID != pendingID {
		return PendingEventPage{}, fmt.Errorf("pending recording %q not found", pendingID)
	}
	if s.pending.result.Meta.RecordingMode != inputclip.RecordingModePrecise {
		return PendingEventPage{}, errors.New("raw event pages are available only for precise recordings")
	}
	if offset < 0 || limit <= 0 || limit > maxPendingEventPageSize {
		return PendingEventPage{}, errors.New("pending event page is outside the bounded range")
	}
	total := len(s.pending.result.Events)
	start := min(offset, total)
	end := min(start+limit, total)
	return PendingEventPage{
		Items: append([]inputclip.Event(nil), s.pending.result.Events[start:end]...),
		Total: total, Offset: start, Limit: limit,
	}, nil
}

func trimPreciseEvents(events []inputclip.Event, startUs, endUs uint64) ([]inputclip.Event, error) {
	if len(events) == 0 {
		return nil, errors.New("recording has no events")
	}
	ordered := append([]inputclip.Event(nil), events...)
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
	heldButtons := map[int32]inputclip.Event{}
	for _, event := range ordered {
		if event.TUs >= startUs {
			break
		}
		updateHeldInput(heldKeys, heldButtons, event)
	}

	selected := make([]inputclip.Event, 0, len(ordered)+len(heldKeys)+len(heldButtons))
	for _, key := range sortedHeldKeys(heldKeys) {
		selected = append(selected, inputclip.Event{TUs: startUs, Type: inputclip.EventTypeKeyDown, A: key})
	}
	for _, button := range sortedHeldButtons(heldButtons) {
		down := heldButtons[button]
		selected = append(selected, inputclip.Event{TUs: startUs, Type: inputclip.EventTypeMouseBtnDown, A: button, B: down.B, C: down.C})
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
		case inputclip.EventTypeKeyDown:
			if _, held := heldKeys[event.A]; held {
				return nil, fmt.Errorf("event %d presses an already-held key", index)
			}
			heldKeys[event.A] = struct{}{}
		case inputclip.EventTypeKeyUp:
			if _, held := heldKeys[event.A]; !held {
				return nil, fmt.Errorf("event %d releases a key that is not held", index)
			}
			delete(heldKeys, event.A)
		case inputclip.EventTypeMouseBtnDown:
			if _, held := heldButtons[event.A]; held {
				return nil, fmt.Errorf("event %d presses an already-held mouse button", index)
			}
			heldButtons[event.A] = event
		case inputclip.EventTypeMouseBtnUp:
			if _, held := heldButtons[event.A]; !held {
				return nil, fmt.Errorf("event %d releases a mouse button that is not held", index)
			}
			delete(heldButtons, event.A)
		}
	}
	keys := sortedHeldKeys(heldKeys)
	for index := len(keys) - 1; index >= 0; index-- {
		selected = append(selected, inputclip.Event{TUs: endUs, Type: inputclip.EventTypeKeyUp, A: keys[index]})
	}
	buttons := sortedHeldButtons(heldButtons)
	for index := len(buttons) - 1; index >= 0; index-- {
		button := buttons[index]
		down := heldButtons[button]
		selected = append(selected, inputclip.Event{TUs: endUs, Type: inputclip.EventTypeMouseBtnUp, A: button, B: down.B, C: down.C})
	}
	baseUs := selected[0].TUs
	for index := range selected {
		selected[index].TUs -= baseUs
		selected[index].Seq = uint32(index)
	}
	return selected, nil
}

func updateHeldInput(keys map[int32]struct{}, buttons map[int32]inputclip.Event, event inputclip.Event) {
	switch event.Type {
	case inputclip.EventTypeKeyDown:
		keys[event.A] = struct{}{}
	case inputclip.EventTypeKeyUp:
		delete(keys, event.A)
	case inputclip.EventTypeMouseBtnDown:
		buttons[event.A] = event
	case inputclip.EventTypeMouseBtnUp:
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

func sortedHeldButtons(buttons map[int32]inputclip.Event) []int32 {
	result := make([]int32, 0, len(buttons))
	for button := range buttons {
		result = append(result, button)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}
