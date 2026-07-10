package tools

import (
	"errors"
	"fmt"

	"github.com/yottaapp/yotta/internal/apperr"
	"github.com/yottaapp/yotta/internal/automation/target"
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

func (r targetToolRouter) OpenPicker(tg target.Target, req PickerRequest) error {
	adapter := r.adapters[tg.Kind]
	if adapter == nil {
		return fmt.Errorf("target picker for %q is not available", tg.Kind)
	}
	return adapter.OpenPicker(req)
}

func (r targetToolRouter) PixelAt(tg target.Target, req PixelSampleRequest) (PixelInfo, error) {
	adapter := r.adapters[tg.Kind]
	if adapter == nil {
		return PixelInfo{}, fmt.Errorf("target pixel sampler for %q is not available", tg.Kind)
	}
	return adapter.PixelAt(req)
}

type win32TargetToolAdapter struct {
	service *Service
}

func (a win32TargetToolAdapter) OpenPicker(req PickerRequest) error {
	return a.service.openScreenPickerWindow(req)
}

func (a win32TargetToolAdapter) PixelAt(req PixelSampleRequest) (PixelInfo, error) {
	return a.service.win32PixelAt(req.ContainerID, req.NodeID)
}

type androidTargetToolAdapter struct {
	service *Service
}

func (a androidTargetToolAdapter) OpenPicker(req PickerRequest) error {
	if a.service == nil {
		return fmt.Errorf("android target picker service is not available")
	}
	return a.service.openScreenPickerWindow(req)
}

func (androidTargetToolAdapter) PixelAt(PixelSampleRequest) (PixelInfo, error) {
	return PixelInfo{}, ErrAndroidTargetPixelNotImplemented
}

func (s *Service) openScreenPickerWindow(req PickerRequest) error {
	presenter := s.windowPresenter()
	if presenter == nil {
		return apperr.New(apperr.CodeWailsNotReady, nil)
	}
	s.mu.Lock()
	slot := s.pickerWindows[req.RequestID]
	if slot == nil {
		slot = &windowSlot{}
		s.pickerWindows[req.RequestID] = slot
	}
	s.mu.Unlock()

	w, opened, err := s.openWindow(presenter, slot, WindowRequest{
		Kind:        WindowScreenPicker,
		Mode:        req.Mode,
		RequestID:   req.RequestID,
		ContainerID: req.ContainerID,
		NodeID:      req.NodeID,
		ColorSpace:  req.ColorSpace,
		GUID:        req.GUID,
	}, func() {
		s.mu.Lock()
		if s.pickerWindows[req.RequestID] == slot {
			delete(s.pickerWindows, req.RequestID)
		}
		s.mu.Unlock()
	})
	if err != nil {
		s.mu.Lock()
		if s.pickerWindows[req.RequestID] == slot && slot.window == nil && slot.opening == nil {
			delete(s.pickerWindows, req.RequestID)
		}
		s.mu.Unlock()
		return err
	}
	if !opened && w != nil {
		w.Focus()
	}
	return nil
}
