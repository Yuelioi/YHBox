package recording

import (
	"errors"
	"fmt"

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
	return inputclip.TrimEvents(events, startUs, endUs)
}
