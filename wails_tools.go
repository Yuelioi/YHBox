package main

import (
	"errors"
	"fmt"
	"net/url"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/yottaapp/yotta/internal/services/tools"
)

// wailsToolsPresenter adapts the GUI runtime to the narrow tools presentation
// port. It exists in the executable layer so backend packages do not import Wails.
type wailsToolsPresenter struct {
	mu  sync.RWMutex
	app *application.App
}

func (p *wailsToolsPresenter) Attach(app *application.App) {
	p.mu.Lock()
	p.app = app
	p.mu.Unlock()
}

func (p *wailsToolsPresenter) Detach() {
	p.mu.Lock()
	p.app = nil
	p.mu.Unlock()
}

func (p *wailsToolsPresenter) Ready() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.app != nil
}

func (p *wailsToolsPresenter) OpenWindow(request tools.WindowRequest) (tools.Window, error) {
	p.mu.RLock()
	app := p.app
	p.mu.RUnlock()
	if app == nil {
		return nil, errors.New("wails application is not ready")
	}

	wailsOptions, err := wailsToolsWindowOptions(request)
	if err != nil {
		return nil, err
	}
	return &wailsToolsWindow{window: app.Window.NewWithOptions(wailsOptions)}, nil
}

func wailsToolsWindowOptions(request tools.WindowRequest) (application.WebviewWindowOptions, error) {
	query := url.Values{}
	withQuery := func(route string) string {
		if encoded := query.Encode(); encoded != "" {
			return route + "?" + encoded
		}
		return route
	}
	darkBackground := application.NewRGB(18, 18, 18)

	switch request.Kind {
	case tools.WindowMouseHUD:
		query.Set("containerID", request.ContainerID)
		return application.WebviewWindowOptions{
			Title: "鼠标位置", Width: 320, Height: 240, MinWidth: 260, MinHeight: 180,
			URL: withQuery("/#/tools/mouse-hud"), Frameless: true, AlwaysOnTop: true,
			BackgroundColour: darkBackground,
		}, nil
	case tools.WindowRecordingHUD:
		return application.WebviewWindowOptions{
			Title: "录制控制", Width: 360, Height: 200, URL: "/#/tools/recording-hud",
			Frameless: true, AlwaysOnTop: true, DisableResize: true,
			BackgroundColour: darkBackground,
		}, nil
	case tools.WindowLauncher:
		return application.WebviewWindowOptions{
			Title: "启动器", Width: 240, Height: 300, MinWidth: 140, MinHeight: 56,
			URL: "/#/tools/launcher", Frameless: true, AlwaysOnTop: true,
			BackgroundColour: darkBackground,
		}, nil
	case tools.WindowCalibratorHUD:
		query.Set("id", request.RequestID)
		return application.WebviewWindowOptions{
			Title: "鼠标校准", Width: 360, Height: 220, URL: withQuery("/#/tools/calibration-hud"),
			Frameless: true, AlwaysOnTop: true, DisableResize: true,
			BackgroundColour: darkBackground,
		}, nil
	case tools.WindowScreenPicker:
		query.Set("mode", request.Mode)
		query.Set("id", request.RequestID)
		query.Set("containerID", request.ContainerID)
		query.Set("nodeID", request.NodeID)
		query.Set("colorSpace", request.ColorSpace)
		query.Set("guid", request.GUID)
		return application.WebviewWindowOptions{
			Title: "选择屏幕位置", Width: 1280, Height: 800, MinWidth: 720, MinHeight: 480,
			URL: withQuery("/#/tools/screen-picker"), Frameless: true,
		}, nil
	default:
		return application.WebviewWindowOptions{}, fmt.Errorf("unsupported tools window kind %q", request.Kind)
	}
}

func (p *wailsToolsPresenter) Emit(name string, data any) {
	p.mu.RLock()
	app := p.app
	p.mu.RUnlock()
	if app != nil {
		app.Event.Emit(name, data)
	}
}

type wailsToolsWindow struct {
	window *application.WebviewWindow
}

func (w *wailsToolsWindow) Focus()                    { w.window.Focus() }
func (w *wailsToolsWindow) Show()                     { w.window.Show() }
func (w *wailsToolsWindow) Hide()                     { w.window.Hide() }
func (w *wailsToolsWindow) Close()                    { w.window.Close() }
func (w *wailsToolsWindow) SetAlwaysOnTop(on bool)    { w.window.SetAlwaysOnTop(on) }
func (w *wailsToolsWindow) SetSize(width, height int) { w.window.SetSize(width, height) }
func (w *wailsToolsWindow) OnClosing(callback func()) {
	w.window.OnWindowEvent(events.Common.WindowClosing, func(*application.WindowEvent) {
		callback()
	})
}
