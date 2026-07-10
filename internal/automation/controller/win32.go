package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/yottaapp/yotta/internal/automation/target"
	automationtrace "github.com/yottaapp/yotta/internal/automation/trace"
)

type Win32Input interface {
	Click(hwnd uintptr, xRatio, yRatio float64, button string, durMs int) error
	MouseDown(hwnd uintptr, xRatio, yRatio float64, button string) error
	MouseUp(hwnd uintptr, button string) error
	Drag(hwnd uintptr, x1Ratio, y1Ratio, x2Ratio, y2Ratio float64, button string, durationMs int) error
	MouseMoveRel(hwnd uintptr, dx, dy, durationMs int) error
	KeyDown(hwnd uintptr, key string) error
	KeyUp(hwnd uintptr, key string) error
	TypeText(hwnd uintptr, text string) error
	MoveTo(hwnd uintptr, xRatio, yRatio float64) error
	Scroll(hwnd uintptr, xRatio, yRatio float64, notches int, horizontal bool) error
	CursorRatio(hwnd uintptr) (float64, float64, error)
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
	hasInput := c.deps.Input != nil
	return CapabilitySet{
		Screenshot:      c.deps.Capture != nil,
		Click:           hasInput,
		Move:            hasInput,
		Scroll:          hasInput,
		MouseButton:     hasInput,
		Drag:            hasInput,
		MoveRelative:    hasInput,
		PointerPosition: hasInput,
		KeyChord:        hasInput,
		KeyState:        hasInput,
		Text:            hasInput,
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

func (c *Win32Controller) MouseDown(ctx context.Context, req MouseButtonRequest) error {
	steps := []automationtrace.CoordinateStep{pointStep(req.Point)}
	return c.recordActionWithSteps("mouse-down", req, steps, func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if c.deps.Input == nil {
			return fmt.Errorf("win32 input dependency is nil")
		}
		if err := validateNormalizedPoint("mouse-down", req.Point); err != nil {
			return err
		}
		button := req.Button
		if button == "" {
			button = "left"
		}
		return c.deps.Input.MouseDown(c.hwnd(), req.Point.X, req.Point.Y, button)
	})
}

func (c *Win32Controller) MouseUp(ctx context.Context, req MouseButtonRequest) error {
	return c.recordAction("mouse-up", req, func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if c.deps.Input == nil {
			return fmt.Errorf("win32 input dependency is nil")
		}
		button := req.Button
		if button == "" {
			button = "left"
		}
		return c.deps.Input.MouseUp(c.hwnd(), button)
	})
}

func (c *Win32Controller) Drag(ctx context.Context, req DragRequest) error {
	steps := []automationtrace.CoordinateStep{pointStep(req.From), pointStep(req.To)}
	return c.recordActionWithSteps("drag", req, steps, func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if c.deps.Input == nil {
			return fmt.Errorf("win32 input dependency is nil")
		}
		if err := validateNormalizedPoint("drag from", req.From); err != nil {
			return err
		}
		if err := validateNormalizedPoint("drag to", req.To); err != nil {
			return err
		}
		button := req.Button
		if button == "" {
			button = "left"
		}
		return c.deps.Input.Drag(c.hwnd(), req.From.X, req.From.Y, req.To.X, req.To.Y, button, req.DurationMs)
	})
}

func (c *Win32Controller) MoveRelative(ctx context.Context, req RelativeMoveRequest) error {
	return c.recordAction("move-relative", req, func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if c.deps.Input == nil {
			return fmt.Errorf("win32 input dependency is nil")
		}
		return c.deps.Input.MouseMoveRel(c.hwnd(), req.Dx, req.Dy, req.DurationMs)
	})
}

func (c *Win32Controller) Move(ctx context.Context, req MoveRequest) error {
	steps := []automationtrace.CoordinateStep{pointStep(req.Point)}
	return c.recordActionWithSteps("move", req, steps, func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if c.deps.Input == nil {
			return fmt.Errorf("win32 input dependency is nil")
		}
		if err := validateNormalizedPoint("move", req.Point); err != nil {
			return err
		}
		return c.deps.Input.MoveTo(c.hwnd(), req.Point.X, req.Point.Y)
	})
}

func (c *Win32Controller) PointerPosition(ctx context.Context) (target.Point, error) {
	if err := ctx.Err(); err != nil {
		return target.Point{}, err
	}
	if c.deps.Input == nil {
		return target.Point{}, fmt.Errorf("win32 input dependency is nil")
	}
	x, y, err := c.deps.Input.CursorRatio(c.hwnd())
	if err != nil {
		return target.Point{}, err
	}
	return target.NewNormalizedPoint(x, y), nil
}

func (c *Win32Controller) Scroll(ctx context.Context, req ScrollRequest) error {
	steps := []automationtrace.CoordinateStep{pointStep(req.Point)}
	return c.recordActionWithSteps("scroll", req, steps, func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if c.deps.Input == nil {
			return fmt.Errorf("win32 input dependency is nil")
		}
		if err := validateNormalizedPoint("scroll", req.Point); err != nil {
			return err
		}
		return c.deps.Input.Scroll(c.hwnd(), req.Point.X, req.Point.Y, req.Notches, req.Horizontal)
	})
}

func pointStep(point target.Point) automationtrace.CoordinateStep {
	step := automationtrace.CoordinateStep{
		From:   point.Space,
		To:     target.SpaceWindowClient,
		Input:  point,
		Output: point,
	}
	if step.From == "" {
		step.From = target.SpaceNormalized
	}
	return step
}

func validateNormalizedPoint(action string, point target.Point) error {
	if point.Space != "" && point.Space != target.SpaceNormalized {
		return fmt.Errorf("win32 phase1 %s supports only normalized points, got %s", action, point.Space)
	}
	return nil
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
	return c.recordActionWithSteps(action, request, nil, run)
}

func (c *Win32Controller) recordActionWithSteps(action string, request any, steps []automationtrace.CoordinateStep, run func() error) error {
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
			Action:          action,
			Target:          c.target,
			Backend:         c.backend(),
			Request:         request,
			Status:          status,
			Error:           errMsg,
			CoordinateSteps: steps,
			StartedAt:       started,
			EndedAt:         time.Now(),
		})
	}
	return err
}
