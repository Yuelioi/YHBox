// Package calibration provides mouse-DPI calibration state and host adapters.
package calibration

import "sync/atomic"

var (
	absDx atomic.Int64
	absDy atomic.Int64
	live  atomic.Bool
)

// State is the current mouse-DPI calibration snapshot.
type State struct {
	Active bool  `json:"active"`
	AbsDx  int64 `json:"absDx"`
	AbsDy  int64 `json:"absDy"`
}

// Get returns a lock-free snapshot suitable for UI polling.
func Get() State {
	return State{
		Active: live.Load(),
		AbsDx:  absDx.Load(),
		AbsDy:  absDy.Load(),
	}
}

// Reset clears the accumulated relative mouse counts.
func Reset() {
	absDx.Store(0)
	absDy.Store(0)
}
