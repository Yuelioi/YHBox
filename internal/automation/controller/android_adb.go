package controller

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/yottaapp/yotta/internal/adbexec"
	"github.com/yottaapp/yotta/internal/automation/pointermotion"
	"github.com/yottaapp/yotta/internal/automation/target"
	automationtrace "github.com/yottaapp/yotta/internal/automation/trace"
)

type ADBRunner interface {
	Run(ctx context.Context, serial string, args ...string) ([]byte, error)
}

type ADBRunnerFunc func(ctx context.Context, serial string, args ...string) ([]byte, error)

func (f ADBRunnerFunc) Run(ctx context.Context, serial string, args ...string) ([]byte, error) {
	return f(ctx, serial, args...)
}

type execADBRunner struct{}

var androidSurfaceOrientationPattern = regexp.MustCompile(`SurfaceOrientation:\s*(\d+)`)

func (execADBRunner) Run(ctx context.Context, serial string, args ...string) ([]byte, error) {
	argv := []string{}
	if serial != "" {
		argv = append(argv, "-s", serial)
	}
	argv = append(argv, args...)
	return adbexec.CommandContext(ctx, argv...).Output()
}

type AndroidADBDeps struct {
	Runner  ADBRunner
	Trace   automationtrace.Recorder
	Backend string
}

type AndroidADBController struct {
	target target.Target
	deps   AndroidADBDeps
}

func NewAndroidADBController(tg target.Target, deps AndroidADBDeps) (*AndroidADBController, error) {
	if err := tg.Validate(); err != nil {
		return nil, err
	}
	if tg.Kind != target.KindAndroidADB {
		return nil, fmt.Errorf("android adb controller requires %s target, got %s", target.KindAndroidADB, tg.Kind)
	}
	return &AndroidADBController{target: tg, deps: deps}, nil
}

func (c *AndroidADBController) Target() target.Target {
	return c.target
}

func (c *AndroidADBController) Capabilities(context.Context) CapabilitySet {
	profile, ok := Profile(BackendAndroidADB)
	if !ok {
		return CapabilitySet{}
	}
	return profile.Capabilities
}

func (c *AndroidADBController) HealthCheck(context.Context) HealthReport {
	if err := c.target.Validate(); err != nil {
		return HealthReport{OK: false, Message: err.Error()}
	}
	if c.runner() == nil {
		return HealthReport{OK: false, Message: "adb runner is nil"}
	}
	return HealthReport{OK: true}
}

func (c *AndroidADBController) Screenshot(ctx context.Context, req ScreenshotRequest) (Frame, error) {
	var frame Frame
	err := c.recordAction("screenshot", req, func() (any, error) {
		out, err := c.runner().Run(ctx, c.serial(), "exec-out", "screencap", "-p")
		if err != nil {
			return nil, err
		}
		img, err := png.Decode(bytes.NewReader(out))
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
			Space: target.SpaceAndroidDevice,
			Size:  target.Size{W: bounds.Dx(), H: bounds.Dy()},
		}
		return map[string]any{"width": frame.Size.W, "height": frame.Size.H}, nil
	})
	return frame, err
}

func (c *AndroidADBController) Click(ctx context.Context, req ClickRequest) error {
	return c.recordInputAction(ctx, "click", req, []target.Point{req.Point}, func(x, y int) error {
		_, err := c.runner().Run(ctx, c.serial(), "shell", "input", "tap", fmt.Sprint(x), fmt.Sprint(y))
		return err
	})
}

func (c *AndroidADBController) Move(ctx context.Context, req MoveRequest) error {
	if req.Motion != pointermotion.Instant || req.DurationMs != 0 {
		return fmt.Errorf("android adb smooth pointer move is unsupported")
	}
	return c.recordInputAction(ctx, "move", req, []target.Point{req.Point}, func(x, y int) error {
		_, err := c.runner().Run(ctx, c.serial(), "shell", "input", "swipe", fmt.Sprint(x), fmt.Sprint(y), fmt.Sprint(x), fmt.Sprint(y), "0")
		return err
	})
}

func (c *AndroidADBController) Scroll(ctx context.Context, req ScrollRequest) error {
	return c.recordInputAction(ctx, "scroll", req, []target.Point{req.Point}, func(x, y int) error {
		delta := req.Notches * 240
		if delta == 0 {
			delta = 240
		}
		if req.Horizontal {
			_, err := c.runner().Run(ctx, c.serial(), "shell", "input", "swipe", fmt.Sprint(x), fmt.Sprint(y), fmt.Sprint(x-delta), fmt.Sprint(y), "120")
			return err
		}
		_, err := c.runner().Run(ctx, c.serial(), "shell", "input", "swipe", fmt.Sprint(x), fmt.Sprint(y), fmt.Sprint(x), fmt.Sprint(y-delta), "120")
		return err
	})
}

func (c *AndroidADBController) MouseDown(context.Context, MouseButtonRequest) error {
	return fmt.Errorf("android adb controller does not support mouse-down")
}

func (c *AndroidADBController) MouseUp(context.Context, MouseButtonRequest) error {
	return fmt.Errorf("android adb controller does not support mouse-up")
}

func (c *AndroidADBController) Drag(ctx context.Context, req DragRequest) error {
	if req.Motion == pointermotion.Bezier {
		return fmt.Errorf("android adb bezier drag is unsupported")
	}
	if !req.Motion.Valid() {
		return fmt.Errorf("android adb drag motion is invalid")
	}
	size := c.inputResolutionForPoints(ctx, req.From, req.To)
	steps, err := c.coordinateStepsWithSize(size, req.From, req.To)
	if err != nil {
		return err
	}
	return c.recordActionWithSteps("drag", req, steps, func() (any, error) {
		x1, y1, err := c.pointToDeviceWithSize(req.From, size)
		if err != nil {
			return nil, err
		}
		x2, y2, err := c.pointToDeviceWithSize(req.To, size)
		if err != nil {
			return nil, err
		}
		duration := req.DurationMs
		_, err = c.runner().Run(ctx, c.serial(), "shell", "input", "swipe", fmt.Sprint(x1), fmt.Sprint(y1), fmt.Sprint(x2), fmt.Sprint(y2), fmt.Sprint(duration))
		return nil, err
	})
}

func (c *AndroidADBController) MoveRelative(context.Context, RelativeMoveRequest) error {
	return fmt.Errorf("android adb controller does not support relative move")
}

func (c *AndroidADBController) KeyChord(context.Context, KeyChordRequest) error {
	return fmt.Errorf("android adb controller does not support key chords")
}

func (c *AndroidADBController) KeyDown(context.Context, KeyRequest) error {
	return fmt.Errorf("android adb controller does not support key-down")
}

func (c *AndroidADBController) KeyUp(context.Context, KeyRequest) error {
	return fmt.Errorf("android adb controller does not support key-up")
}

func (c *AndroidADBController) Text(ctx context.Context, req TextRequest) error {
	return c.recordAction("text", req, func() (any, error) {
		_, err := c.runner().Run(ctx, c.serial(), "shell", "input", "text", escapeADBInputText(req.Text))
		return nil, err
	})
}

func (c *AndroidADBController) StartApp(ctx context.Context, req StartAppRequest) error {
	return c.recordAction("start-app", req, func() (any, error) {
		_, err := c.runner().Run(ctx, c.serial(), "shell", "monkey", "-p", req.Intent, "-c", "android.intent.category.LAUNCHER", "1")
		return nil, err
	})
}

func (c *AndroidADBController) StopApp(ctx context.Context, req StopAppRequest) error {
	return c.recordAction("stop-app", req, func() (any, error) {
		_, err := c.runner().Run(ctx, c.serial(), "shell", "am", "force-stop", req.Intent)
		return nil, err
	})
}

func (c *AndroidADBController) serial() string {
	return c.target.Ref.ADBSerial
}

func (c *AndroidADBController) runner() ADBRunner {
	if c.deps.Runner != nil {
		return c.deps.Runner
	}
	return execADBRunner{}
}

func (c *AndroidADBController) backend() string {
	if c.deps.Backend != "" {
		return c.deps.Backend
	}
	return string(BackendAndroidADB)
}

func (c *AndroidADBController) recordInputAction(ctx context.Context, action string, request any, points []target.Point, run func(x, y int) error) error {
	size := c.inputResolutionForPoints(ctx, points...)
	steps, err := c.coordinateStepsWithSize(size, points...)
	if err != nil {
		return err
	}
	return c.recordActionWithSteps(action, request, steps, func() (any, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		x, y, err := c.pointToDeviceWithSize(points[0], size)
		if err != nil {
			return nil, err
		}
		return nil, run(x, y)
	})
}

func (c *AndroidADBController) recordAction(action string, request any, run func() (any, error)) error {
	return c.recordActionWithSteps(action, request, nil, run)
}

func (c *AndroidADBController) recordActionWithSteps(action string, request any, steps []automationtrace.CoordinateStep, run func() (any, error)) error {
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

func (c *AndroidADBController) coordinateStepsWithSize(size target.Size, points ...target.Point) ([]automationtrace.CoordinateStep, error) {
	steps := make([]automationtrace.CoordinateStep, 0, len(points))
	for _, point := range points {
		x, y, err := c.pointToDeviceWithSize(point, size)
		if err != nil {
			return nil, err
		}
		from := point.Space
		if from == "" {
			from = target.SpaceNormalized
		}
		steps = append(steps, automationtrace.CoordinateStep{
			From:   from,
			To:     target.SpaceAndroidDevice,
			Input:  point,
			Output: target.Point{X: float64(x), Y: float64(y), Space: target.SpaceAndroidDevice},
		})
	}
	return steps, nil
}

func (c *AndroidADBController) pointToDevice(point target.Point) (int, int, error) {
	return c.pointToDeviceWithSize(point, c.target.Resolution)
}

func (c *AndroidADBController) pointToDeviceWithSize(point target.Point, size target.Size) (int, int, error) {
	switch point.Space {
	case "", target.SpaceNormalized:
		if point.X < 0 || point.X > 1 || point.Y < 0 || point.Y > 1 {
			return 0, 0, fmt.Errorf("normalized point out of range: (%f,%f)", point.X, point.Y)
		}
		if size.W <= 0 || size.H <= 0 {
			return 0, 0, fmt.Errorf("android adb target resolution is required for normalized coordinates")
		}
		x := int(math.Round(point.X * float64(size.W)))
		y := int(math.Round(point.Y * float64(size.H)))
		return clampInt(x, 0, size.W-1), clampInt(y, 0, size.H-1), nil
	case target.SpaceAndroidDevice:
		return int(math.Round(point.X)), int(math.Round(point.Y)), nil
	default:
		return 0, 0, fmt.Errorf("android adb point space %q is not supported", point.Space)
	}
}

func (c *AndroidADBController) inputResolutionForPoints(ctx context.Context, points ...target.Point) target.Size {
	for _, point := range points {
		if point.Space == "" || point.Space == target.SpaceNormalized {
			return c.currentInputResolution(ctx)
		}
	}
	return c.target.Resolution
}

func (c *AndroidADBController) currentInputResolution(ctx context.Context) target.Size {
	size := c.target.Resolution
	if size.W <= 0 || size.H <= 0 {
		return size
	}
	out, err := c.runner().Run(ctx, c.serial(), "shell", "dumpsys", "input")
	if err != nil {
		return size
	}
	orientation, ok := parseAndroidSurfaceOrientation(string(out))
	if !ok {
		return size
	}
	landscape := orientation%2 != 0
	if landscape && size.W < size.H || !landscape && size.W > size.H {
		size.W, size.H = size.H, size.W
	}
	return size
}

func parseAndroidSurfaceOrientation(out string) (int, bool) {
	m := androidSurfaceOrientationPattern.FindStringSubmatch(out)
	if len(m) != 2 {
		return 0, false
	}
	var orientation int
	if _, err := fmt.Sscanf(m[1], "%d", &orientation); err != nil {
		return 0, false
	}
	return orientation, true
}

func escapeADBInputText(text string) string {
	text = strings.ReplaceAll(text, "%", "\\%")
	text = strings.ReplaceAll(text, " ", "%s")
	return text
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func imageToRGBA(img image.Image) (*image.RGBA, error) {
	if rgba, ok := img.(*image.RGBA); ok {
		return rgba, nil
	}
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)
	return rgba, nil
}
