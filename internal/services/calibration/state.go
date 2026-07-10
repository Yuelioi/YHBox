package calibration

import "sync/atomic"

type calibrationState struct {
	absDx atomic.Int64
	absDy atomic.Int64
	live  atomic.Bool
}

var currentState calibrationState

// State is the current mouse-DPI calibration snapshot.
type State struct {
	Active bool  `json:"active"`
	AbsDx  int64 `json:"absDx"`
	AbsDy  int64 `json:"absDy"`
}

// Get returns a lock-free snapshot suitable for UI polling.
func Get() State {
	return currentState.snapshot()
}

// Reset clears the accumulated relative mouse counts.
func Reset() {
	currentState.reset()
}

func (s *calibrationState) snapshot() State {
	return State{
		Active: s.live.Load(),
		AbsDx:  s.absDx.Load(),
		AbsDy:  s.absDy.Load(),
	}
}

func (s *calibrationState) reset() {
	s.absDx.Store(0)
	s.absDy.Store(0)
}

func (s *calibrationState) setActive(active bool) {
	s.live.Store(active)
}

func (s *calibrationState) addRelative(dx, dy int64) {
	s.absDx.Add(dx)
	s.absDy.Add(dy)
}
