//go:build !windows

package platform

import (
	"errors"
	"testing"
)

func TestKillProcessReportsUnsupportedPlatform(t *testing.T) {
	if err := KillProcess("1234"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("error = %v, want ErrUnsupported", err)
	}
}
