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
	if startUs > endUs || endUs > events[len(events)-1].TUs {
		return nil, errors.New("trim range is outside the recording")
	}
	selected := make([]inputclip.Event, 0, len(events))
	for _, event := range events {
		if event.TUs >= startUs && event.TUs <= endUs {
			selected = append(selected, event)
		}
	}
	if len(selected) == 0 {
		return nil, errors.New("trim range contains no events")
	}
	sort.SliceStable(selected, func(i, j int) bool {
		if selected[i].TUs != selected[j].TUs {
			return selected[i].TUs < selected[j].TUs
		}
		return selected[i].Seq < selected[j].Seq
	})
	heldKeys := map[int32]struct{}{}
	heldButtons := map[int32]struct{}{}
	for index, event := range selected {
		switch event.Type {
		case inputclip.EventTypeKeyDown:
			if _, held := heldKeys[event.A]; held {
				return nil, fmt.Errorf("event %d presses an already-held key", index)
			}
			heldKeys[event.A] = struct{}{}
		case inputclip.EventTypeKeyUp:
			if _, held := heldKeys[event.A]; !held {
				return nil, fmt.Errorf("trim starts inside a held key interval at event %d", index)
			}
			delete(heldKeys, event.A)
		case inputclip.EventTypeMouseBtnDown:
			if _, held := heldButtons[event.A]; held {
				return nil, fmt.Errorf("event %d presses an already-held mouse button", index)
			}
			heldButtons[event.A] = struct{}{}
		case inputclip.EventTypeMouseBtnUp:
			if _, held := heldButtons[event.A]; !held {
				return nil, fmt.Errorf("trim starts inside a held mouse interval at event %d", index)
			}
			delete(heldButtons, event.A)
		}
	}
	if len(heldKeys) != 0 || len(heldButtons) != 0 {
		return nil, errors.New("trim ends while keyboard or mouse input is still held")
	}
	baseUs := selected[0].TUs
	for index := range selected {
		selected[index].TUs -= baseUs
	}
	return selected, nil
}
