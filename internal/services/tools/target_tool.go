package tools

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"yotta/internal/apperr"
)

var (
	ErrAndroidTargetPickerNotImplemented = errors.New("android target picker preview is not implemented")
	ErrAndroidTargetPixelNotImplemented  = errors.New("android target pixel sampling is not implemented")
)

type PickerRequest struct {
	Mode        string
	RequestID   string
	ContainerID string
	NodeID      string
	ColorSpace  string
	GUID        string
}

type PixelSampleRequest struct {
	ContainerID string
	NodeID      string
}

type TargetToolAdapter interface {
	OpenPicker(req PickerRequest) error
	PixelAt(req PixelSampleRequest) (PixelInfo, error)
}

type targetToolRouter struct {
	adapters map[string]TargetToolAdapter
}

func newTargetToolRouter(adapters map[string]TargetToolAdapter) targetToolRouter {
	return targetToolRouter{adapters: adapters}
}

func (r targetToolRouter) OpenPicker(targetKind string, req PickerRequest) error {
	adapter := r.adapters[targetKind]
	if adapter == nil {
		return fmt.Errorf("target picker for %q is not available", targetKind)
	}
	return adapter.OpenPicker(req)
}

func (r targetToolRouter) PixelAt(targetKind string, req PixelSampleRequest) (PixelInfo, error) {
	adapter := r.adapters[targetKind]
	if adapter == nil {
		return PixelInfo{}, fmt.Errorf("target pixel sampler for %q is not available", targetKind)
	}
	return adapter.PixelAt(req)
}

type win32TargetToolAdapter struct {
	service *Service
}

func (a win32TargetToolAdapter) OpenPicker(req PickerRequest) error {
	return a.service.openWin32ScreenPicker(req)
}

func (a win32TargetToolAdapter) PixelAt(req PixelSampleRequest) (PixelInfo, error) {
	return a.service.win32PixelAt(req.ContainerID, req.NodeID)
}

type androidTargetToolAdapter struct{}

func (androidTargetToolAdapter) OpenPicker(PickerRequest) error {
	return ErrAndroidTargetPickerNotImplemented
}

func (androidTargetToolAdapter) PixelAt(PixelSampleRequest) (PixelInfo, error) {
	return PixelInfo{}, ErrAndroidTargetPixelNotImplemented
}

func (s *Service) openWin32ScreenPicker(req PickerRequest) error {
	app := s.wailsApp()
	if app == nil {
		return apperr.New(apperr.CodeWailsNotReady, nil)
	}
	s.mu.Lock()
	if existing, ok := s.pickerWindows[req.RequestID]; ok {
		s.mu.Unlock()
		existing.Focus()
		return nil
	}
	s.mu.Unlock()

	hashURL := "/#/tools/screen-picker?mode=" + url.QueryEscape(req.Mode) + "&id=" + url.QueryEscape(req.RequestID) + "&containerID=" + url.QueryEscape(req.ContainerID) + "&nodeID=" + url.QueryEscape(req.NodeID) + "&colorSpace=" + url.QueryEscape(req.ColorSpace) + "&guid=" + url.QueryEscape(req.GUID)
	w := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     "选择屏幕位置",
		Width:     1280,
		Height:    800,
		MinWidth:  720,
		MinHeight: 480,
		URL:       hashURL,
		Frameless: true,
	})
	s.mu.Lock()
	s.pickerWindows[req.RequestID] = w
	s.mu.Unlock()
	w.OnWindowEvent(events.Common.WindowClosing, func(_ *application.WindowEvent) {
		s.mu.Lock()
		delete(s.pickerWindows, req.RequestID)
		s.mu.Unlock()
	})
	return nil
}
