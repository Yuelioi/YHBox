package tools

import "errors"

import "sync/atomic"

// windowSlot serializes one semantic window's open/close generations.
// Service.mu protects every field.
type windowSlot struct {
	window     Window
	opening    *windowOpenAttempt
	generation uint64
}

type windowOpenAttempt struct {
	generation uint64
	done       chan struct{}
	waiters    int
	window     Window
	opened     bool
	err        error
}

func (s *Service) openWindow(
	presenter Presenter,
	slot *windowSlot,
	request WindowRequest,
	onCurrentClose func(),
) (Window, bool, error) {
	s.mu.Lock()
	if slot.window != nil {
		window := slot.window
		s.mu.Unlock()
		return window, false, nil
	}
	if attempt := slot.opening; attempt != nil {
		attempt.waiters++
		s.mu.Unlock()
		<-attempt.done
		return attempt.window, attempt.opened, attempt.err
	}
	slot.generation++
	attempt := &windowOpenAttempt{
		generation: slot.generation,
		done:       make(chan struct{}),
	}
	slot.opening = attempt
	s.mu.Unlock()

	window, err := presenter.OpenWindow(request)
	if err == nil && window == nil {
		err = errors.New("presentation returned a nil window")
	}
	if err != nil {
		s.mu.Lock()
		cancelled := slot.generation != attempt.generation
		attempt.err = err
		if !cancelled {
			slot.opening = nil
		}
		close(attempt.done)
		s.mu.Unlock()
		if cancelled {
			if onCurrentClose != nil {
				onCurrentClose()
			}
			s.mu.Lock()
			if slot.opening == attempt {
				slot.opening = nil
			}
			s.mu.Unlock()
		}
		return nil, false, err
	}

	var closed atomic.Bool
	window.OnClosing(func() {
		closed.Store(true)
		s.mu.Lock()
		current := slot.generation == attempt.generation && slot.window != nil
		if current {
			slot.window = nil
		}
		s.mu.Unlock()
		if current && onCurrentClose != nil {
			onCurrentClose()
		}
	})

	s.mu.Lock()
	cancelled := slot.generation != attempt.generation || closed.Load()
	if !cancelled {
		slot.window = window
		attempt.window = window
		attempt.opened = true
	}
	s.mu.Unlock()

	if cancelled {
		s.mu.Lock()
		close(attempt.done)
		s.mu.Unlock()
		window.Close()
		if onCurrentClose != nil {
			onCurrentClose()
		}
		s.mu.Lock()
		if slot.opening == attempt {
			slot.opening = nil
		}
		s.mu.Unlock()
		return nil, false, nil
	}

	s.mu.Lock()
	slot.opening = nil
	close(attempt.done)
	s.mu.Unlock()
	return window, true, nil
}

// invalidateWindow prevents an older closing callback or in-flight open from
// mutating the next generation. The caller closes the returned window unlocked.
func (s *Service) invalidateWindow(slot *windowSlot) Window {
	s.mu.Lock()
	slot.generation++
	window := slot.window
	slot.window = nil
	s.mu.Unlock()
	return window
}

func (s *Service) currentWindow(slot *windowSlot) Window {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slot.window
}
