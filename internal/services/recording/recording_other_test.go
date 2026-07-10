//go:build !windows

package recording

import (
	"errors"
	"testing"

	"github.com/yottaapp/yotta/internal/automation/target"
	"github.com/yottaapp/yotta/internal/services/inputclip"
	"github.com/yottaapp/yotta/pkg/platform"
)

func TestRecordingAdaptersReportUnsupportedPlatform(t *testing.T) {
	if _, err := resolveRecordingWindow(target.WindowMatchSpec{}); !errors.Is(err, platform.ErrUnsupported) {
		t.Fatalf("resolveRecordingWindow() error = %v, want platform.ErrUnsupported", err)
	}
	if _, err := NewRecorder().Start(1, inputclip.ClipMeta{}); !errors.Is(err, platform.ErrUnsupported) {
		t.Fatalf("Recorder.Start() error = %v, want platform.ErrUnsupported", err)
	}
}
