//go:build !windows

package recording

import (
	"github.com/yottaapp/yotta/internal/services/inputclip"
	"github.com/yottaapp/yotta/pkg/platform"
)

// Recorder preserves the recording lifecycle interface on unsupported hosts.
type Recorder struct{}

// NewRecorder creates an inactive recorder adapter.
func NewRecorder() *Recorder { return &Recorder{} }

// Active reports whether this adapter has an active recording.
func (*Recorder) Active() bool { return false }

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

func setActiveStopHotkey(uint32, func()) {}

func setActivePauseHotkey(uint32, func()) {}
