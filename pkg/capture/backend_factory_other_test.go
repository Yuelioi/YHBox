//go:build !windows

package capture

import (
	"errors"
	"testing"

	"github.com/yottaapp/yotta/pkg/platform"
)

func TestNewIBackendReportsUnsupportedPlatform(t *testing.T) {
	backend, warning, err := NewIBackend("auto")
	if backend != nil || warning != "" {
		t.Fatalf("backend = %v, warning = %q; want nil and empty", backend, warning)
	}
	if !errors.Is(err, platform.ErrUnsupported) {
		t.Fatalf("error = %v, want platform.ErrUnsupported", err)
	}
}

func TestNewIBackendRejectsUnknownNameBeforePlatformCheck(t *testing.T) {
	_, _, err := NewIBackend("missing")
	if err == nil || errors.Is(err, platform.ErrUnsupported) {
		t.Fatalf("error = %v, want unknown backend error", err)
	}
}
