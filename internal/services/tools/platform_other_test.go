//go:build !windows

package tools

import (
	"errors"
	"testing"

	"github.com/yottaapp/yotta/pkg/platform"
)

func TestNativeToolsReportUnsupportedPlatform(t *testing.T) {
	service := NewService(nil, nil)
	if _, err := service.MousePos(""); !errors.Is(err, platform.ErrUnsupported) {
		t.Fatalf("MousePos() error = %v, want platform.ErrUnsupported", err)
	}
	if _, err := service.StartWin32WindowTargetCapture(); !errors.Is(err, platform.ErrUnsupported) {
		t.Fatalf("StartWin32WindowTargetCapture() error = %v, want platform.ErrUnsupported", err)
	}
	if _, err := service.win32PixelAt(""); !errors.Is(err, platform.ErrUnsupported) {
		t.Fatalf("win32PixelAt() error = %v, want platform.ErrUnsupported", err)
	}
}
