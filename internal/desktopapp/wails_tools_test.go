package desktopapp

import (
	"strings"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/yottaapp/yotta/internal/services/tools"
)

func TestWailsToolsWindowOptionsOwnPresentationPolicy(t *testing.T) {
	mouse, err := wailsToolsWindowOptions(tools.WindowRequest{
		Kind:       tools.WindowMouseHUD,
		TargetSlot: "target slot",
	})
	if err != nil {
		t.Fatal(err)
	}
	if mouse.Title != "鼠标位置" || mouse.Width != 340 || mouse.Height != 300 || mouse.MinWidth != 300 || mouse.MinHeight != 240 || !mouse.Frameless || !mouse.AlwaysOnTop {
		t.Fatalf("mouse options = %+v", mouse)
	}
	if mouse.URL != "/#/tools/mouse-hud?targetSlot=target+slot" {
		t.Fatalf("mouse URL = %q", mouse.URL)
	}

	picker, err := wailsToolsWindowOptions(tools.WindowRequest{
		Kind: tools.WindowScreenPicker, RequestID: "request-1", Mode: "rect", ColorSpace: "rgb",
	})
	if err != nil {
		t.Fatal(err)
	}
	if picker.Title != "选择屏幕位置" || picker.Width != 1360 || picker.Height != 860 || picker.MinWidth != 760 || picker.MinHeight != 520 || !strings.HasPrefix(picker.URL, "/#/tools/screen-picker?") {
		t.Fatalf("picker options = %+v", picker)
	}

	recording, err := wailsToolsWindowOptions(tools.WindowRequest{Kind: tools.WindowRecordingHUD})
	if err != nil {
		t.Fatal(err)
	}
	if recording.Width != 380 || recording.Height != 240 || !recording.DisableResize {
		t.Fatalf("recording options = %+v", recording)
	}

	launcher, err := wailsToolsWindowOptions(tools.WindowRequest{Kind: tools.WindowLauncher})
	if err != nil {
		t.Fatal(err)
	}
	if launcher.Width != 300 || launcher.Height != 360 || launcher.MinWidth != 220 || launcher.MinHeight != 120 {
		t.Fatalf("launcher options = %+v", launcher)
	}

	calibrator, err := wailsToolsWindowOptions(tools.WindowRequest{Kind: tools.WindowCalibratorHUD})
	if err != nil {
		t.Fatal(err)
	}
	if calibrator.Width != 380 || calibrator.Height != 260 || !calibrator.DisableResize {
		t.Fatalf("calibrator options = %+v", calibrator)
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
	if options.URL != "/#/workflows" {
		t.Fatalf("main window URL = %q, want Workflow 3.1 list", options.URL)
	}
}

func TestWailsToolsPresenterDelegatesWindowLifecycleBeforeNativeAttach(t *testing.T) {
	presenter := &wailsToolsPresenter{}
	app := application.New(application.Options{Name: "test"})
	presenter.Attach(app)
	if _, err := presenter.OpenWindow(tools.WindowRequest{Kind: "unknown"}); err == nil {
		t.Fatal("OpenWindow accepted unknown kind")
	}
	created, err := presenter.OpenWindow(tools.WindowRequest{Kind: tools.WindowLauncher})
	if err != nil {
		t.Fatal(err)
	}
	if created == nil {
		t.Fatal("OpenWindow returned nil")
	}
	window := &wailsToolsWindow{window: &application.WebviewWindow{}}
	window.Focus()
	window.Show()
	window.Hide()
	window.SetAlwaysOnTop(true)
	window.SetSize(320, 240)
	window.Close()
	created.OnClosing(func() {})
	presenter.Emit("test:event", map[string]any{"ok": true})
	presenter.Detach()
	if presenter.Ready() {
		t.Fatal("detached presenter remained ready")
	}

	options := mainWindowOptions(1400, 900)
	if options.Width != 1400 || options.Height != 900 {
		t.Fatalf("main window options = %+v", options)
	}
}
