//go:build !windows

package recording

import (
	"sync"

	"github.com/yottaapp/yotta/internal/services/inputclip"
	"github.com/yottaapp/yotta/pkg/platform"
)

// Recorder preserves the recording lifecycle interface on unsupported hosts.
type Recorder struct {
	mu                   sync.Mutex
	active               bool
	mouseCounts360Getter func() int
}

// NewRecorder creates an inactive recorder adapter.
func NewRecorder() *Recorder { return &Recorder{} }

// SetMouseCounts360Getter stores the metadata provider for interface parity.
func (r *Recorder) SetMouseCounts360Getter(getter func() int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.mouseCounts360Getter = getter
}

// Active reports whether this adapter has an active recording.
func (r *Recorder) Active() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.active
}

// Pause is an idempotent no-op without an active native recording.
func (*Recorder) Pause() {}

// Resume is an idempotent no-op without an active native recording.
func (*Recorder) Resume() {}

// Start reports that native input recording is unavailable on this host.
func (*Recorder) Start(uintptr, inputclip.ClipMeta) (string, error) {
	return "", platform.NewUnsupportedError("native input recording")
}

// Stop reports the same inactive state as an idle Windows recorder.
func (*Recorder) Stop() (*StopResult, error) { return nil, ErrRecorderNotActive }

// Cancel is an idempotent no-op without an active native recording.
func (*Recorder) Cancel() {}

func SetActiveStopHotkey(uint32, func()) {}

func SetActivePauseHotkey(uint32, func()) {}
