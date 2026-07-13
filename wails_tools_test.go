package main

import (
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/services/tools"
)

func TestWailsToolsWindowOptionsOwnPresentationPolicy(t *testing.T) {
	mouse, err := wailsToolsWindowOptions(tools.WindowRequest{
		Kind:        tools.WindowMouseHUD,
		ContainerID: "container with spaces",
	})
	if err != nil {
		t.Fatal(err)
	}
	if mouse.Title != "鼠标位置" || mouse.Width != 320 || mouse.Height != 240 || !mouse.Frameless || !mouse.AlwaysOnTop {
		t.Fatalf("mouse options = %+v", mouse)
	}
	if mouse.URL != "/#/tools/mouse-hud?containerID=container+with+spaces" {
		t.Fatalf("mouse URL = %q", mouse.URL)
	}

	picker, err := wailsToolsWindowOptions(tools.WindowRequest{
		Kind: tools.WindowScreenPicker, RequestID: "request-1", Mode: "rect", ColorSpace: "rgb",
	})
	if err != nil {
		t.Fatal(err)
	}
	if picker.Title != "选择屏幕位置" || picker.MinWidth != 720 || picker.MinHeight != 480 || !strings.HasPrefix(picker.URL, "/#/tools/screen-picker?") {
		t.Fatalf("picker options = %+v", picker)
	}

	if _, err := wailsToolsWindowOptions(tools.WindowRequest{Kind: "unknown"}); err == nil {
		t.Fatal("unknown window kind succeeded")
	}
}

func TestMainWindowOptionsEnforceEditorMinimumWidth(t *testing.T) {
	options := mainWindowOptions(1100, 720)
	if options.MinWidth != 1180 || options.Width < options.MinWidth {
		t.Fatalf("main window width = %d, minimum = %d", options.Width, options.MinWidth)
	}
}
