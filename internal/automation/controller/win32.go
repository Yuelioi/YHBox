package controller

import (
	"context"
	"fmt"
	"time"

	"yotta/internal/automation/target"
	automationtrace "yotta/internal/automation/trace"
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
	Trace   automationtrace.Recorder
	Backend string
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

func (c *Win32Controller) hwnd() uintptr {
	return c.target.Ref.HWND
}

func (c *Win32Controller) Click(ctx context.Context, req ClickRequest) error {
	return c.recordAction("click", req, func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if c.deps.Input == nil {
			return fmt.Errorf("win32 input dependency is nil")
		}
		if req.Point.Space != "" && req.Point.Space != target.SpaceNormalized {
			return fmt.Errorf("win32 phase1 click supports only normalized points, got %s", req.Point.Space)
		}
		button := req.Button
		if button == "" {
			button = "left"
		}
		return c.deps.Input.Click(c.hwnd(), req.Point.X, req.Point.Y, button, req.DurationMs)
	})
}

func (c *Win32Controller) Move(ctx context.Context, req MoveRequest) error {
	return c.recordAction("move", req, func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if c.deps.Input == nil {
			return fmt.Errorf("win32 input dependency is nil")
		}
		if req.Point.Space != "" && req.Point.Space != target.SpaceNormalized {
			return fmt.Errorf("win32 phase1 move supports only normalized points, got %s", req.Point.Space)
		}
		return c.deps.Input.MoveTo(c.hwnd(), req.Point.X, req.Point.Y)
	})
}

func (c *Win32Controller) Scroll(ctx context.Context, req ScrollRequest) error {
	return c.recordAction("scroll", req, func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if c.deps.Input == nil {
			return fmt.Errorf("win32 input dependency is nil")
		}
		if req.Point.Space != "" && req.Point.Space != target.SpaceNormalized {
			return fmt.Errorf("win32 phase1 scroll supports only normalized points, got %s", req.Point.Space)
		}
		return c.deps.Input.Scroll(c.hwnd(), req.Point.X, req.Point.Y, req.Notches, req.Horizontal)
	})
}

func (c *Win32Controller) KeyChord(ctx context.Context, req KeyChordRequest) error {
	return c.recordAction("key-chord", req, func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if c.deps.Input == nil {
			return fmt.Errorf("win32 input dependency is nil")
		}
		for _, key := range req.Keys {
			if err := c.deps.Input.KeyDown(c.hwnd(), key); err != nil {
				return err
			}
		}
		for i := len(req.Keys) - 1; i >= 0; i-- {
			if err := c.deps.Input.KeyUp(c.hwnd(), req.Keys[i]); err != nil {
				return err
			}
		}
		return nil
	})
}

func (c *Win32Controller) KeyDown(ctx context.Context, req KeyRequest) error {
	return c.recordAction("key-down", req, func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if c.deps.Input == nil {
			return fmt.Errorf("win32 input dependency is nil")
		}
		return c.deps.Input.KeyDown(c.hwnd(), req.Key)
	})
}

func (c *Win32Controller) KeyUp(ctx context.Context, req KeyRequest) error {
	return c.recordAction("key-up", req, func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if c.deps.Input == nil {
			return fmt.Errorf("win32 input dependency is nil")
		}
		return c.deps.Input.KeyUp(c.hwnd(), req.Key)
	})
}

func (c *Win32Controller) Text(ctx context.Context, req TextRequest) error {
	return c.recordAction("text", req, func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if c.deps.Input == nil {
			return fmt.Errorf("win32 input dependency is nil")
		}
		return c.deps.Input.TypeText(c.hwnd(), req.Text)
	})
}

func (c *Win32Controller) Screenshot(ctx context.Context, req ScreenshotRequest) (Frame, error) {
	var frame Frame
	err := c.recordAction("screenshot", req, func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if c.deps.Capture == nil {
			return fmt.Errorf("win32 capture dependency is nil")
		}
		var err error
		frame, err = c.deps.Capture.Frame(c.hwnd())
		return err
	})
	return frame, err
}

func (c *Win32Controller) backend() string {
	if c.deps.Backend != "" {
		return c.deps.Backend
	}
	return "win32"
}

func (c *Win32Controller) recordAction(action string, request any, run func() error) error {
	started := time.Now()
	err := run()
	if c.deps.Trace != nil {
		status := automationtrace.StatusSuccess
		errMsg := ""
		if err != nil {
			status = automationtrace.StatusError
			errMsg = err.Error()
		}
		c.deps.Trace.Record(automationtrace.ActionRecord{
			Action:    action,
			Target:    c.target,
			Backend:   c.backend(),
			Request:   request,
			Status:    status,
			Error:     errMsg,
			StartedAt: started,
			EndedAt:   time.Now(),
		})
	}
	return err
}
