package main

import (
	"fmt"
	"net/url"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"yhbox/internal/services/container"
)

// containerWindowAdapter 给每个 containerID 开一个独立 Frameless 子窗口。
// 同 id 重复打开 → focus 已有窗口。窗口 Close 事件清掉 map。
type containerWindowAdapter struct {
	wailsApp *application.App
	mu       sync.Mutex
	windows  map[string]*application.WebviewWindow
}

func newContainerWindowAdapter(w *application.App) *containerWindowAdapter {
	return &containerWindowAdapter{
		wailsApp: w,
		windows:  map[string]*application.WebviewWindow{},
	}
}

// OpenEditor 实现 container.WindowOpener 接口。
func (a *containerWindowAdapter) OpenEditor(containerID string) error {
	if a.wailsApp == nil {
		return fmt.Errorf("wails app 未初始化")
	}
	a.mu.Lock()
	if existing, ok := a.windows[containerID]; ok {
		a.mu.Unlock()
		existing.Focus()
		return nil
	}
	a.mu.Unlock()

	hashURL := "/#/container-editor?id=" + url.QueryEscape(containerID)
	w := a.wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     "编辑容器",
		Width:     1280,
		Height:    820,
		MinWidth:  960,
		MinHeight: 600,
		URL:       hashURL,
		Frameless: true,
	})

	a.mu.Lock()
	a.windows[containerID] = w
	a.mu.Unlock()

	w.OnWindowEvent(events.Common.WindowClosing, func(_ *application.WindowEvent) {
		a.mu.Lock()
		delete(a.windows, containerID)
		a.mu.Unlock()
		if app := a.wailsApp; app != nil {
			app.Event.Emit("container:editing-released", map[string]any{"id": containerID})
		}
	})

	a.wailsApp.Event.Emit("container:editing-acquired", map[string]any{"id": containerID})
	return nil
}

// 编译时接口检查
var _ container.WindowOpener = (*containerWindowAdapter)(nil)
