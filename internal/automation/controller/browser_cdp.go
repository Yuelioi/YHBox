package controller

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image/png"
	"math"
	"time"

	"github.com/yottaapp/yotta/internal/automation/target"
	automationtrace "github.com/yottaapp/yotta/internal/automation/trace"
)

type CDPClient interface {
	Call(ctx context.Context, method string, params map[string]any) (map[string]any, error)
}

type CDPClientFunc func(ctx context.Context, method string, params map[string]any) (map[string]any, error)

func (f CDPClientFunc) Call(ctx context.Context, method string, params map[string]any) (map[string]any, error) {
	return f(ctx, method, params)
}

type BrowserCDPDeps struct {
	Client  CDPClient
	Trace   automationtrace.Recorder
	Backend string
}

type BrowserCDPController struct {
	target target.Target
	deps   BrowserCDPDeps
}

func NewBrowserCDPController(tg target.Target, deps BrowserCDPDeps) (*BrowserCDPController, error) {
	if err := tg.Validate(); err != nil {
		return nil, err
	}
	if tg.Kind != target.KindBrowserCDP {
		return nil, fmt.Errorf("browser cdp controller requires %s target, got %s", target.KindBrowserCDP, tg.Kind)
	}
	return &BrowserCDPController{target: tg, deps: deps}, nil
}

func (c *BrowserCDPController) Target() target.Target {
	return c.target
}

func (c *BrowserCDPController) Capabilities(context.Context) CapabilitySet {
	profile, ok := Profile(BackendBrowserCDP)
	if !ok {
		return CapabilitySet{}
	}
	return profile.Capabilities
}

func (c *BrowserCDPController) HealthCheck(context.Context) HealthReport {
	if err := c.target.Validate(); err != nil {
		return HealthReport{OK: false, Message: err.Error()}
	}
	if c.deps.Client == nil {
		return HealthReport{OK: false, Message: "cdp client is nil"}
	}
	return HealthReport{OK: true}
}

func (c *BrowserCDPController) Screenshot(ctx context.Context, req ScreenshotRequest) (Frame, error) {
	var frame Frame
	err := c.recordAction("screenshot", req, func() (any, error) {
		res, err := c.call(ctx, "Page.captureScreenshot", map[string]any{"format": "png"})
		if err != nil {
			return nil, err
		}
		data, _ := res["data"].(string)
		if data == "" {
			return nil, fmt.Errorf("cdp screenshot missing data")
		}
		raw, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			return nil, err
		}
		img, err := png.Decode(bytes.NewReader(raw))
		if err != nil {
			return nil, err
		}
		rgba, err := imageToRGBA(img)
		if err != nil {
			return nil, err
		}
		bounds := rgba.Bounds()
		frame = Frame{
			Image: rgba,
			Space: target.SpaceBrowserView,
			Size:  target.Size{W: bounds.Dx(), H: bounds.Dy()},
		}
		return map[string]any{"width": frame.Size.W, "height": frame.Size.H}, nil
	})
	return frame, err
}

func (c *BrowserCDPController) Click(ctx context.Context, req ClickRequest) error {
	steps, err := c.coordinateSteps(req.Point)
	if err != nil {
		return err
	}
	return c.recordActionWithSteps("click", req, steps, func() (any, error) {
		x, y, err := c.pointToViewport(req.Point)
		if err != nil {
			return nil, err
		}
		button := browserMouseButton(req.Button)
		if _, err := c.dispatchMouse(ctx, "mousePressed", x, y, map[string]any{"button": button, "clickCount": 1}); err != nil {
			return nil, err
		}
		_, err = c.dispatchMouse(ctx, "mouseReleased", x, y, map[string]any{"button": button, "clickCount": 1})
		return nil, err
	})
}

func (c *BrowserCDPController) Move(ctx context.Context, req MoveRequest) error {
	steps, err := c.coordinateSteps(req.Point)
	if err != nil {
		return err
	}
	return c.recordActionWithSteps("move", req, steps, func() (any, error) {
		x, y, err := c.pointToViewport(req.Point)
		if err != nil {
			return nil, err
		}
		_, err = c.dispatchMouse(ctx, "mouseMoved", x, y, nil)
		return nil, err
	})
}

func (c *BrowserCDPController) Scroll(ctx context.Context, req ScrollRequest) error {
	steps, err := c.coordinateSteps(req.Point)
	if err != nil {
		return err
	}
	return c.recordActionWithSteps("scroll", req, steps, func() (any, error) {
		x, y, err := c.pointToViewport(req.Point)
		if err != nil {
			return nil, err
		}
		delta := req.Notches * -120
		if delta == 0 {
			delta = -120
		}
		extra := map[string]any{"deltaX": 0, "deltaY": delta}
		if req.Horizontal {
			extra = map[string]any{"deltaX": delta, "deltaY": 0}
		}
		_, err = c.dispatchMouse(ctx, "mouseWheel", x, y, extra)
		return nil, err
	})
}

func (c *BrowserCDPController) MouseDown(ctx context.Context, req MouseButtonRequest) error {
	steps, err := c.coordinateSteps(req.Point)
	if err != nil {
		return err
	}
	return c.recordActionWithSteps("mouse-down", req, steps, func() (any, error) {
		x, y, err := c.pointToViewport(req.Point)
		if err != nil {
			return nil, err
		}
		_, err = c.dispatchMouse(ctx, "mousePressed", x, y, map[string]any{"button": browserMouseButton(req.Button), "clickCount": 1})
		return nil, err
	})
}

func (c *BrowserCDPController) MouseUp(ctx context.Context, req MouseButtonRequest) error {
	return c.recordAction("mouse-up", req, func() (any, error) {
		_, err := c.dispatchMouse(ctx, "mouseReleased", 0, 0, map[string]any{"button": browserMouseButton(req.Button), "clickCount": 1})
		return nil, err
	})
}

func (c *BrowserCDPController) Drag(ctx context.Context, req DragRequest) error {
	steps, err := c.coordinateSteps(req.From, req.To)
	if err != nil {
		return err
	}
	return c.recordActionWithSteps("drag", req, steps, func() (any, error) {
		x1, y1, err := c.pointToViewport(req.From)
		if err != nil {
			return nil, err
		}
		x2, y2, err := c.pointToViewport(req.To)
		if err != nil {
			return nil, err
		}
		button := browserMouseButton(req.Button)
		if _, err := c.dispatchMouse(ctx, "mousePressed", x1, y1, map[string]any{"button": button, "clickCount": 1}); err != nil {
			return nil, err
		}
		if _, err := c.dispatchMouse(ctx, "mouseMoved", x2, y2, map[string]any{"button": button}); err != nil {
			return nil, err
		}
		_, err = c.dispatchMouse(ctx, "mouseReleased", x2, y2, map[string]any{"button": button, "clickCount": 1})
		return nil, err
	})
}

func (c *BrowserCDPController) MoveRelative(context.Context, RelativeMoveRequest) error {
	return fmt.Errorf("browser cdp controller does not support relative move")
}

func (c *BrowserCDPController) KeyChord(ctx context.Context, req KeyChordRequest) error {
	return c.recordAction("key-chord", req, func() (any, error) {
		modifiers := 0
		for _, key := range req.Keys {
			modifiers |= browserModifier(key)
			if err := c.dispatchKey(ctx, "keyDown", key, modifiers); err != nil {
				return nil, err
			}
		}
		for i := len(req.Keys) - 1; i >= 0; i-- {
			key := req.Keys[i]
			if err := c.dispatchKey(ctx, "keyUp", key, modifiers); err != nil {
				return nil, err
			}
			modifiers &^= browserModifier(key)
		}
		return nil, nil
	})
}

func (c *BrowserCDPController) KeyDown(ctx context.Context, req KeyRequest) error {
	return c.recordAction("key-down", req, func() (any, error) {
		return nil, c.dispatchKey(ctx, "keyDown", req.Key, browserModifier(req.Key))
	})
}

func (c *BrowserCDPController) KeyUp(ctx context.Context, req KeyRequest) error {
	return c.recordAction("key-up", req, func() (any, error) {
		return nil, c.dispatchKey(ctx, "keyUp", req.Key, browserModifier(req.Key))
	})
}

func (c *BrowserCDPController) Text(ctx context.Context, req TextRequest) error {
	return c.recordAction("text", req, func() (any, error) {
		_, err := c.call(ctx, "Input.insertText", map[string]any{"text": req.Text})
		return nil, err
	})
}

func (c *BrowserCDPController) StartApp(context.Context, StartAppRequest) error {
	return fmt.Errorf("browser cdp controller does not support start-app")
}

func (c *BrowserCDPController) StopApp(context.Context, StopAppRequest) error {
	return fmt.Errorf("browser cdp controller does not support stop-app")
}

func (c *BrowserCDPController) backend() string {
	if c.deps.Backend != "" {
		return c.deps.Backend
	}
	return string(BackendBrowserCDP)
}

func (c *BrowserCDPController) call(ctx context.Context, method string, params map[string]any) (map[string]any, error) {
	if c.deps.Client == nil {
		return nil, fmt.Errorf("cdp client is nil")
	}
	return c.deps.Client.Call(ctx, method, params)
}

func (c *BrowserCDPController) dispatchMouse(ctx context.Context, typ string, x, y int, extra map[string]any) (map[string]any, error) {
	params := map[string]any{
		"type": typ,
		"x":    x,
		"y":    y,
	}
	for k, v := range extra {
		params[k] = v
	}
	return c.call(ctx, "Input.dispatchMouseEvent", params)
}

func (c *BrowserCDPController) dispatchKey(ctx context.Context, typ string, raw string, modifiers int) error {
	key, code, virtualKey := browserKey(raw)
	_, err := c.call(ctx, "Input.dispatchKeyEvent", map[string]any{
		"type": typ, "key": key, "code": code, "modifiers": modifiers,
		"windowsVirtualKeyCode": virtualKey, "nativeVirtualKeyCode": virtualKey,
	})
	return err
}

func browserModifier(key string) int {
	switch key {
	case "ALT":
		return 1
	case "CTRL":
		return 2
	case "SHIFT":
		return 8
	default:
		return 0
	}
}

func browserKey(raw string) (key, code string, virtualKey int) {
	if len(raw) == 1 && raw[0] >= 'A' && raw[0] <= 'Z' {
		return string(raw[0] + ('a' - 'A')), "Key" + raw, int(raw[0])
	}
	if len(raw) == 1 && raw[0] >= '0' && raw[0] <= '9' {
		return raw, "Digit" + raw, int(raw[0])
	}
	switch raw {
	case "CTRL":
		return "Control", "ControlLeft", 17
	case "ALT":
		return "Alt", "AltLeft", 18
	case "SHIFT":
		return "Shift", "ShiftLeft", 16
	case "ENTER":
		return "Enter", "Enter", 13
	case "ESC":
		return "Escape", "Escape", 27
	case "SPACE":
		return " ", "Space", 32
	case "TAB":
		return "Tab", "Tab", 9
	case "BACKSPACE":
		return "Backspace", "Backspace", 8
	case "DELETE":
		return "Delete", "Delete", 46
	case "INSERT":
		return "Insert", "Insert", 45
	case "HOME":
		return "Home", "Home", 36
	case "END":
		return "End", "End", 35
	case "PGUP":
		return "PageUp", "PageUp", 33
	case "PGDN":
		return "PageDown", "PageDown", 34
	case "UP":
		return "ArrowUp", "ArrowUp", 38
	case "DOWN":
		return "ArrowDown", "ArrowDown", 40
	case "LEFT":
		return "ArrowLeft", "ArrowLeft", 37
	case "RIGHT":
		return "ArrowRight", "ArrowRight", 39
	case ",":
		return ",", "Comma", 188
	case ".":
		return ".", "Period", 190
	case "CAPSLOCK":
		return "CapsLock", "CapsLock", 20
	}
	if len(raw) >= 2 && raw[0] == 'F' {
		var function int
		if _, err := fmt.Sscanf(raw, "F%d", &function); err == nil && function >= 1 && function <= 12 {
			return raw, raw, 111 + function
		}
	}
	return raw, raw, 0
}

func (c *BrowserCDPController) recordAction(action string, request any, run func() (any, error)) error {
	return c.recordActionWithSteps(action, request, nil, run)
}

func (c *BrowserCDPController) recordActionWithSteps(action string, request any, steps []automationtrace.CoordinateStep, run func() (any, error)) error {
	started := time.Now()
	result, err := run()
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
			Result:          result,
			Status:          status,
			Error:           errMsg,
			CoordinateSteps: steps,
			StartedAt:       started,
			EndedAt:         time.Now(),
		})
	}
	return err
}

func (c *BrowserCDPController) coordinateSteps(points ...target.Point) ([]automationtrace.CoordinateStep, error) {
	steps := make([]automationtrace.CoordinateStep, 0, len(points))
	for _, point := range points {
		x, y, err := c.pointToViewport(point)
		if err != nil {
			return nil, err
		}
		from := point.Space
		if from == "" {
			from = target.SpaceNormalized
		}
		steps = append(steps, automationtrace.CoordinateStep{
			From:   from,
			To:     target.SpaceBrowserView,
			Input:  point,
			Output: target.Point{X: float64(x), Y: float64(y), Space: target.SpaceBrowserView},
		})
	}
	return steps, nil
}

func (c *BrowserCDPController) pointToViewport(point target.Point) (int, int, error) {
	switch point.Space {
	case "", target.SpaceNormalized:
		if point.X < 0 || point.X > 1 || point.Y < 0 || point.Y > 1 {
			return 0, 0, fmt.Errorf("normalized point out of range: (%f,%f)", point.X, point.Y)
		}
		if c.target.Resolution.W <= 0 || c.target.Resolution.H <= 0 {
			return 0, 0, fmt.Errorf("browser cdp target resolution is required for normalized coordinates")
		}
		x := int(math.Round(point.X * float64(c.target.Resolution.W)))
		y := int(math.Round(point.Y * float64(c.target.Resolution.H)))
		return clampInt(x, 0, c.target.Resolution.W-1), clampInt(y, 0, c.target.Resolution.H-1), nil
	case target.SpaceBrowserView:
		return int(math.Round(point.X)), int(math.Round(point.Y)), nil
	default:
		return 0, 0, fmt.Errorf("browser cdp point space %q is not supported", point.Space)
	}
}

func browserMouseButton(button string) string {
	switch button {
	case "right", "middle":
		return button
	default:
		return "left"
	}
}
