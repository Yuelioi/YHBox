//go:build !windows

package input

import (
	"errors"
	"testing"

	"github.com/yottaapp/yotta/pkg/platform"
)

func TestNewBackendReportsUnsupportedPlatform(t *testing.T) {
	backend, err := NewBackend("postmessage")
	if backend != nil {
		t.Fatal("backend must be nil on unsupported platforms")
	}
	if !errors.Is(err, platform.ErrUnsupported) {
		t.Fatalf("error = %v, want platform.ErrUnsupported", err)
	}
}

func TestNewBackendRejectsUnknownNameBeforePlatformCheck(t *testing.T) {
	_, err := NewBackend("missing")
	if err == nil || errors.Is(err, platform.ErrUnsupported) {
		t.Fatalf("error = %v, want unknown backend error", err)
	}
}
