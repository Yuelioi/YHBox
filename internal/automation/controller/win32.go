package controller

import (
	"context"
	"fmt"

	"yotta/internal/automation/target"
)

type Win32Input interface {
	Click(hwnd uintptr, xRatio, yRatio float64, button string, durMs int) error
	KeyDown(hwnd uintptr, key string) error
	KeyUp(hwnd uintptr, key string) error
	TypeText(hwnd uintptr, text string) error
	MoveTo(hwnd uintptr, xRatio, yRatio float64) error
	Scroll(hwnd uintptr, xRatio, yRatio float64, notches int, horizontal bool) error
}

type Win32Capture interface {
	Frame(hwnd uintptr) (Frame, error)
}

type Win32WindowOps interface {
	BringForeground(hwnd uintptr) bool
}

type Win32Deps struct {
	Input   Win32Input
	Capture Win32Capture
	Window  Win32WindowOps
}

type Win32Controller struct {
	target target.Target
	deps   Win32Deps
}

func NewWin32Controller(tg target.Target, deps Win32Deps) (*Win32Controller, error) {
	if err := tg.Validate(); err != nil {
		return nil, err
	}
	if tg.Kind != target.KindWin32Window {
		return nil, fmt.Errorf("win32 controller requires %s target, got %s", target.KindWin32Window, tg.Kind)
	}
	return &Win32Controller{target: tg, deps: deps}, nil
}

func (c *Win32Controller) Target() target.Target {
	return c.target
}

func (c *Win32Controller) Capabilities(context.Context) CapabilitySet {
	return CapabilitySet{
		Screenshot: true,
		Click:      true,
		Move:       true,
		Scroll:     true,
		KeyChord:   true,
		KeyState:   true,
		Text:       true,
	}
}

func (c *Win32Controller) HealthCheck(context.Context) HealthReport {
	if err := c.target.Validate(); err != nil {
		return HealthReport{OK: false, Message: err.Error()}
	}
	return HealthReport{OK: true}
}
