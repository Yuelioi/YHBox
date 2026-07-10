//go:build !windows

package winutil

import (
	"context"
	"errors"
	"testing"

	"github.com/yottaapp/yotta/pkg/platform"
)

func TestResolveWindowReportsUnsupportedPlatform(t *testing.T) {
	_, err := ResolveWindow(context.Background(), MatchSpec{Title: "window"}, 0, 0)
	if !errors.Is(err, platform.ErrUnsupported) {
		t.Fatalf("error = %v, want platform.ErrUnsupported", err)
	}
}

func TestWindowControlReportsUnsupportedPlatform(t *testing.T) {
	if err := Maximize(1); !errors.Is(err, platform.ErrUnsupported) {
		t.Fatalf("error = %v, want platform.ErrUnsupported", err)
	}
}
