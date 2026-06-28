# Target Controller Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Introduce a first-class Target/Controller abstraction and wrap the existing Win32 input/capture/window services without changing node graph behavior.

**Architecture:** Phase 1 is an adapter layer, not a node rewrite. Existing `Window` and runtime services keep working, while new `internal/automation` types become the stable boundary for future Android, browser, trace, and Rust native controller work.

**Tech Stack:** Go, existing `pkg/input`, `pkg/capture`, `pkg/winutil`, `internal/services/container/runtime`, standard `testing` package.

---

## Scope

This plan implements only Phase 1 from `flightdeck/knowledge/architecture/target-controller-upgrade-guide.md`:

- Add target identity and coordinate-space value types.
- Add controller capability interfaces.
- Add a Win32 controller that wraps existing capture/input/window helpers.
- Add tests proving the controller delegates to existing backends.
- Add runtime bridge helpers, but do not migrate every node.

Explicitly out of scope:

- Android ADB controller.
- Browser CDP controller.
- Rust native DLL.
- Trace UI.
- New Target nodes.
- Container JSON migration.

## File Structure

- Create `internal/automation/target/types.go`: `Target`, `TargetRef`, `Rect`, `Size`, `DPIMeta`, `CoordinateSpace`, `Point`.
- Create `internal/automation/controller/interfaces.go`: capability interfaces and request/response structs.
- Create `internal/automation/controller/capabilities.go`: capability set helpers.
- Create `internal/automation/controller/win32.go`: `Win32Controller` implementation over injected dependencies.
- Create `internal/automation/controller/win32_test.go`: tests with fake input/capture/window dependencies.
- Modify `internal/services/container/runtime/node_services.go`: add narrow helper constructors that can later consume a controller; do not replace current adapters in this phase.
- Create `flightdeck/knowledge/architecture/target-controller-phase1-notes.md`: short implementation notes and migration status.

## Task 1: Target Value Types

**Files:**
- Create: `internal/automation/target/types.go`
- Test: `internal/automation/target/types_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/automation/target/types_test.go`:

```go
package target

import "testing"

func TestTargetIsValid(t *testing.T) {
	t.Run("valid win32 target", func(t *testing.T) {
		tg := Target{
			ID:          "win32:100",
			Kind:        KindWin32Window,
			DisplayName: "After Effects",
			Ref:         TargetRef{HWND: 100},
			Resolution:  Size{W: 1920, H: 1080},
		}
		if err := tg.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
	})

	t.Run("missing id", func(t *testing.T) {
		tg := Target{Kind: KindWin32Window, Ref: TargetRef{HWND: 100}}
		if err := tg.Validate(); err == nil {
			t.Fatalf("Validate() expected error")
		}
	})

	t.Run("win32 target requires hwnd", func(t *testing.T) {
		tg := Target{ID: "win32:0", Kind: KindWin32Window}
		if err := tg.Validate(); err == nil {
			t.Fatalf("Validate() expected error")
		}
	})
}

func TestPointSpaceDefaults(t *testing.T) {
	p := NewNormalizedPoint(0.25, 0.75)
	if p.Space != SpaceNormalized {
		t.Fatalf("space = %q, want %q", p.Space, SpaceNormalized)
	}
	if p.X != 0.25 || p.Y != 0.75 {
		t.Fatalf("point = %#v", p)
	}
}
```

- [ ] **Step 2: Run the test and verify it fails**

Run:

```powershell
go test ./internal/automation/target -count=1
```

Expected: package or symbols do not exist.

- [ ] **Step 3: Add the target types**

Create `internal/automation/target/types.go`:

```go
package target

import "fmt"

const (
	KindWin32Window = "win32-window"
	KindWin32Screen = "win32-screen"
	KindAndroidADB  = "android-adb"
	KindBrowserCDP  = "browser-cdp"
	KindDebugReplay = "debug-replay"
	KindMock        = "mock"
)

const (
	SpaceNormalized    CoordinateSpace = "normalized"
	SpaceScreen        CoordinateSpace = "screen"
	SpaceWindowClient  CoordinateSpace = "window-client"
	SpaceCaptureFrame  CoordinateSpace = "capture-frame"
	SpaceAndroidDevice CoordinateSpace = "android-device"
	SpaceBrowserView   CoordinateSpace = "browser-viewport"
)

type CoordinateSpace string

type Point struct {
	X     float64
	Y     float64
	Space CoordinateSpace
}

func NewNormalizedPoint(x, y float64) Point {
	return Point{X: x, Y: y, Space: SpaceNormalized}
}

type Rect struct {
	X int
	Y int
	W int
	H int
}

type Size struct {
	W int
	H int
}

type DPIMeta struct {
	ScaleX float64
	ScaleY float64
}

type TargetRef struct {
	HWND        uintptr
	ADBSerial   string
	BrowserID   string
	ReplayPath  string
	MockImageID string
}

type Target struct {
	ID          string
	Kind        string
	DisplayName string
	Ref         TargetRef
	Bounds      Rect
	Resolution  Size
	DPI         DPIMeta
	Metadata    map[string]any
}

func (t Target) Validate() error {
	if t.ID == "" {
		return fmt.Errorf("target id is required")
	}
	if t.Kind == "" {
		return fmt.Errorf("target kind is required")
	}
	switch t.Kind {
	case KindWin32Window:
		if t.Ref.HWND == 0 {
			return fmt.Errorf("win32-window target requires hwnd")
		}
	case KindAndroidADB:
		if t.Ref.ADBSerial == "" {
			return fmt.Errorf("android-adb target requires adb serial")
		}
	case KindBrowserCDP:
		if t.Ref.BrowserID == "" {
			return fmt.Errorf("browser-cdp target requires browser id")
		}
	}
	return nil
}
```

- [ ] **Step 4: Run the target tests**

Run:

```powershell
go test ./internal/automation/target -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit the task**

```powershell
git add internal/automation/target
git commit -m "feat: add automation target types"
```

## Task 2: Controller Interfaces

**Files:**
- Create: `internal/automation/controller/interfaces.go`
- Create: `internal/automation/controller/capabilities.go`
- Test: `internal/automation/controller/capabilities_test.go`

- [ ] **Step 1: Write capability tests**

Create `internal/automation/controller/capabilities_test.go`:

```go
package controller

import "testing"

func TestCapabilitySetHas(t *testing.T) {
	caps := CapabilitySet{
		Screenshot: true,
		Click:      true,
		KeyChord:   false,
	}
	if !caps.Has(CapabilityScreenshot) {
		t.Fatalf("expected screenshot capability")
	}
	if caps.Has(CapabilityKeyChord) {
		t.Fatalf("did not expect key chord capability")
	}
}
```

- [ ] **Step 2: Run the test and verify it fails**

Run:

```powershell
go test ./internal/automation/controller -count=1
```

Expected: package or symbols do not exist.

- [ ] **Step 3: Add interfaces**

Create `internal/automation/controller/interfaces.go`:

```go
package controller

import (
	"context"
	"image"

	"yotta/internal/automation/target"
)

type Controller interface {
	Target() target.Target
	Capabilities(context.Context) CapabilitySet
	HealthCheck(context.Context) HealthReport
}

type Screenshotter interface {
	Screenshot(context.Context, ScreenshotRequest) (Frame, error)
}

type PointerInput interface {
	Click(context.Context, ClickRequest) error
	Move(context.Context, MoveRequest) error
	Scroll(context.Context, ScrollRequest) error
}

type KeyboardInput interface {
	KeyChord(context.Context, KeyChordRequest) error
	KeyDown(context.Context, KeyRequest) error
	KeyUp(context.Context, KeyRequest) error
	Text(context.Context, TextRequest) error
}

type AppLifecycle interface {
	StartApp(context.Context, StartAppRequest) error
	StopApp(context.Context, StopAppRequest) error
}

type HealthReport struct {
	OK      bool
	Message string
}

type Frame struct {
	Image *image.RGBA
	Space target.CoordinateSpace
	Size  target.Size
}

type ScreenshotRequest struct {
	Space target.CoordinateSpace
	ROI   target.Rect
}

type ClickRequest struct {
	Point      target.Point
	Button     string
	DurationMs int
	Policy     ActionPolicy
}

type MoveRequest struct {
	Point  target.Point
	Policy ActionPolicy
}

type ScrollRequest struct {
	Point      target.Point
	Notches    int
	Horizontal bool
	Policy     ActionPolicy
}

type KeyChordRequest struct {
	Keys   []string
	Policy ActionPolicy
}

type KeyRequest struct {
	Key    string
	Policy ActionPolicy
}

type TextRequest struct {
	Text   string
	Policy ActionPolicy
}

type StartAppRequest struct {
	Intent string
}

type StopAppRequest struct {
	Intent string
}

type ActionPolicy struct {
	ForegroundRequired bool
	BackgroundAllowed  bool
	NoStealFocus       bool
}
```

Create `internal/automation/controller/capabilities.go`:

```go
package controller

type Capability string

const (
	CapabilityScreenshot Capability = "screenshot"
	CapabilityClick      Capability = "click"
	CapabilityMove       Capability = "move"
	CapabilityScroll     Capability = "scroll"
	CapabilityKeyChord   Capability = "key-chord"
	CapabilityKeyState   Capability = "key-state"
	CapabilityText       Capability = "text"
	CapabilityStartApp   Capability = "start-app"
	CapabilityStopApp    Capability = "stop-app"
)

type CapabilitySet struct {
	Screenshot bool
	Click      bool
	Move       bool
	Scroll     bool
	KeyChord   bool
	KeyState   bool
	Text       bool
	StartApp   bool
	StopApp    bool
}

func (s CapabilitySet) Has(c Capability) bool {
	switch c {
	case CapabilityScreenshot:
		return s.Screenshot
	case CapabilityClick:
		return s.Click
	case CapabilityMove:
		return s.Move
	case CapabilityScroll:
		return s.Scroll
	case CapabilityKeyChord:
		return s.KeyChord
	case CapabilityKeyState:
		return s.KeyState
	case CapabilityText:
		return s.Text
	case CapabilityStartApp:
		return s.StartApp
	case CapabilityStopApp:
		return s.StopApp
	default:
		return false
	}
}
```

- [ ] **Step 4: Run the controller tests**

Run:

```powershell
go test ./internal/automation/controller -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit the task**

```powershell
git add internal/automation/controller
git commit -m "feat: add automation controller interfaces"
```

## Task 3: Win32 Controller Dependency Shape

**Files:**
- Create: `internal/automation/controller/win32.go`
- Test: `internal/automation/controller/win32_test.go`

- [ ] **Step 1: Write tests for target and capabilities**

Create `internal/automation/controller/win32_test.go`:

```go
package controller

import (
	"context"
	"testing"

	"yotta/internal/automation/target"
)

func TestWin32ControllerTargetAndCapabilities(t *testing.T) {
	tg := target.Target{
		ID:          "win32:42",
		Kind:        target.KindWin32Window,
		DisplayName: "Test Window",
		Ref:         target.TargetRef{HWND: 42},
		Resolution:  target.Size{W: 1280, H: 720},
	}
	ctrl, err := NewWin32Controller(tg, Win32Deps{})
	if err != nil {
		t.Fatalf("NewWin32Controller() error = %v", err)
	}
	if got := ctrl.Target(); got.ID != tg.ID {
		t.Fatalf("target id = %q, want %q", got.ID, tg.ID)
	}
	caps := ctrl.Capabilities(context.Background())
	if !caps.Screenshot || !caps.Click || !caps.KeyState || !caps.Text {
		t.Fatalf("unexpected caps: %#v", caps)
	}
	if caps.StartApp || caps.StopApp {
		t.Fatalf("win32 phase1 should not expose app lifecycle: %#v", caps)
	}
}

func TestWin32ControllerRejectsNonWin32Target(t *testing.T) {
	_, err := NewWin32Controller(target.Target{
		ID:   "adb:device",
		Kind: target.KindAndroidADB,
		Ref:  target.TargetRef{ADBSerial: "device"},
	}, Win32Deps{})
	if err == nil {
		t.Fatalf("expected error for non-win32 target")
	}
}
```

- [ ] **Step 2: Run the tests and verify they fail**

Run:

```powershell
go test ./internal/automation/controller -count=1
```

Expected: `NewWin32Controller`, `Win32Deps`, or `Win32Controller` undefined.

- [ ] **Step 3: Add controller skeleton**

Create `internal/automation/controller/win32.go`:

```go
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
	Input  Win32Input
	Capture Win32Capture
	Window Win32WindowOps
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
		Screenshot: c.deps.Capture != nil,
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
```

- [ ] **Step 4: Run the tests**

Run:

```powershell
go test ./internal/automation/controller -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit the task**

```powershell
git add internal/automation/controller
git commit -m "feat: add win32 controller skeleton"
```

## Task 4: Win32 Controller Delegation

**Files:**
- Modify: `internal/automation/controller/win32.go`
- Modify: `internal/automation/controller/win32_test.go`

- [ ] **Step 1: Add delegation tests**

Append to `internal/automation/controller/win32_test.go`:

```go
type fakeWin32Input struct {
	clickHWND uintptr
	clickX    float64
	clickY    float64
	keyDown   []string
	keyUp     []string
	text      string
}

func (f *fakeWin32Input) Click(hwnd uintptr, xRatio, yRatio float64, button string, durMs int) error {
	f.clickHWND = hwnd
	f.clickX = xRatio
	f.clickY = yRatio
	return nil
}

func (f *fakeWin32Input) KeyDown(hwnd uintptr, key string) error {
	f.keyDown = append(f.keyDown, key)
	return nil
}

func (f *fakeWin32Input) KeyUp(hwnd uintptr, key string) error {
	f.keyUp = append(f.keyUp, key)
	return nil
}

func (f *fakeWin32Input) TypeText(hwnd uintptr, text string) error {
	f.text = text
	return nil
}

func (f *fakeWin32Input) MoveTo(hwnd uintptr, xRatio, yRatio float64) error { return nil }

func (f *fakeWin32Input) Scroll(hwnd uintptr, xRatio, yRatio float64, notches int, horizontal bool) error {
	return nil
}

func TestWin32ControllerClickDelegatesNormalizedPoint(t *testing.T) {
	in := &fakeWin32Input{}
	ctrl, err := NewWin32Controller(target.Target{
		ID:   "win32:42",
		Kind: target.KindWin32Window,
		Ref:  target.TargetRef{HWND: 42},
	}, Win32Deps{Input: in})
	if err != nil {
		t.Fatalf("NewWin32Controller() error = %v", err)
	}
	err = ctrl.Click(context.Background(), ClickRequest{
		Point:  target.NewNormalizedPoint(0.25, 0.75),
		Button: "left",
	})
	if err != nil {
		t.Fatalf("Click() error = %v", err)
	}
	if in.clickHWND != 42 || in.clickX != 0.25 || in.clickY != 0.75 {
		t.Fatalf("delegated click = hwnd %d (%f,%f)", in.clickHWND, in.clickX, in.clickY)
	}
}

func TestWin32ControllerKeyChordDelegatesDownReverseUp(t *testing.T) {
	in := &fakeWin32Input{}
	ctrl, err := NewWin32Controller(target.Target{
		ID:   "win32:42",
		Kind: target.KindWin32Window,
		Ref:  target.TargetRef{HWND: 42},
	}, Win32Deps{Input: in})
	if err != nil {
		t.Fatalf("NewWin32Controller() error = %v", err)
	}
	err = ctrl.KeyChord(context.Background(), KeyChordRequest{Keys: []string{"ctrl", "n"}})
	if err != nil {
		t.Fatalf("KeyChord() error = %v", err)
	}
	if got := in.keyDown; len(got) != 2 || got[0] != "ctrl" || got[1] != "n" {
		t.Fatalf("keyDown = %#v", got)
	}
	if got := in.keyUp; len(got) != 2 || got[0] != "n" || got[1] != "ctrl" {
		t.Fatalf("keyUp = %#v", got)
	}
}
```

- [ ] **Step 2: Run the tests and verify they fail**

Run:

```powershell
go test ./internal/automation/controller -count=1
```

Expected: `Click` and `KeyChord` methods undefined.

- [ ] **Step 3: Implement delegation**

Append to `internal/automation/controller/win32.go`:

```go
func (c *Win32Controller) hwnd() uintptr {
	return c.target.Ref.HWND
}

func (c *Win32Controller) Click(ctx context.Context, req ClickRequest) error {
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
}

func (c *Win32Controller) Move(ctx context.Context, req MoveRequest) error {
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
}

func (c *Win32Controller) Scroll(ctx context.Context, req ScrollRequest) error {
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
}

func (c *Win32Controller) KeyChord(ctx context.Context, req KeyChordRequest) error {
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
}

func (c *Win32Controller) KeyDown(ctx context.Context, req KeyRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if c.deps.Input == nil {
		return fmt.Errorf("win32 input dependency is nil")
	}
	return c.deps.Input.KeyDown(c.hwnd(), req.Key)
}

func (c *Win32Controller) KeyUp(ctx context.Context, req KeyRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if c.deps.Input == nil {
		return fmt.Errorf("win32 input dependency is nil")
	}
	return c.deps.Input.KeyUp(c.hwnd(), req.Key)
}

func (c *Win32Controller) Text(ctx context.Context, req TextRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if c.deps.Input == nil {
		return fmt.Errorf("win32 input dependency is nil")
	}
	return c.deps.Input.TypeText(c.hwnd(), req.Text)
}

func (c *Win32Controller) Screenshot(ctx context.Context, req ScreenshotRequest) (Frame, error) {
	if err := ctx.Err(); err != nil {
		return Frame{}, err
	}
	if c.deps.Capture == nil {
		return Frame{}, fmt.Errorf("win32 capture dependency is nil")
	}
	return c.deps.Capture.Frame(c.hwnd())
}
```

- [ ] **Step 4: Run the controller tests**

Run:

```powershell
go test ./internal/automation/controller -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit the task**

```powershell
git add internal/automation/controller
git commit -m "feat: delegate win32 controller actions"
```

## Task 5: Runtime Dependency Adapters

**Files:**
- Create: `internal/services/container/runtime/automation_adapters.go`
- Test: `internal/services/container/runtime/automation_adapters_test.go`

- [ ] **Step 1: Write adapter tests**

Create `internal/services/container/runtime/automation_adapters_test.go`:

```go
package runtime

import (
	"testing"

	"yotta/internal/automation/target"
	"yotta/pkg/winutil"
)

func TestWindowHandleToTarget(t *testing.T) {
	wh := winutil.WindowHandle{
		HWND:        123,
		Title:       "After Effects",
		Class:       "AE_CApplication",
		ProcessName: "AfterFX.exe",
		ClientW:     1920,
		ClientH:     1080,
	}
	tg := windowHandleToTarget(wh)
	if tg.ID != "win32:123" {
		t.Fatalf("target id = %q", tg.ID)
	}
	if tg.Kind != target.KindWin32Window {
		t.Fatalf("target kind = %q", tg.Kind)
	}
	if tg.Ref.HWND != 123 {
		t.Fatalf("hwnd = %d", tg.Ref.HWND)
	}
	if tg.Resolution.W != 1920 || tg.Resolution.H != 1080 {
		t.Fatalf("resolution = %#v", tg.Resolution)
	}
}
```

- [ ] **Step 2: Run the test and verify it fails**

Run:

```powershell
go test ./internal/services/container/runtime -run TestWindowHandleToTarget -count=1
```

Expected: `windowHandleToTarget` undefined.

- [ ] **Step 3: Add runtime helper**

Create `internal/services/container/runtime/automation_adapters.go`:

```go
package runtime

import (
	"fmt"

	"yotta/internal/automation/target"
	"yotta/pkg/winutil"
)

func windowHandleToTarget(wh winutil.WindowHandle) target.Target {
	return target.Target{
		ID:          fmt.Sprintf("win32:%d", wh.HWND),
		Kind:        target.KindWin32Window,
		DisplayName: wh.Title,
		Ref:         target.TargetRef{HWND: wh.HWND},
		Resolution:  target.Size{W: wh.ClientW, H: wh.ClientH},
		Metadata: map[string]any{
			"class":   wh.Class,
			"process": wh.ProcessName,
			"pid":     wh.PID,
		},
	}
}
```

- [ ] **Step 4: Run the targeted runtime test**

Run:

```powershell
go test ./internal/services/container/runtime -run TestWindowHandleToTarget -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit the task**

```powershell
git add internal/services/container/runtime/automation_adapters.go internal/services/container/runtime/automation_adapters_test.go
git commit -m "feat: map runtime windows to automation targets"
```

## Task 6: Documentation and Verification

**Files:**
- Create: `flightdeck/knowledge/architecture/target-controller-phase1-notes.md`

- [ ] **Step 1: Add implementation notes**

Create `flightdeck/knowledge/architecture/target-controller-phase1-notes.md`:

```markdown
# Target / Controller Phase 1 Notes

SUMMARY: Phase 1 introduces Target and Controller types, wraps Win32 behavior, and keeps existing nodes compatible
READ WHEN: Continuing Target/Controller implementation / debugging why nodes still call old services / deciding when to migrate nodes to controller APIs
RECHECK WHEN: `internal/automation` or runtime service adapters change

---

Phase 1 is an adapter phase. It does not change stored container JSON, node specs, or frontend node UI.

New packages:

- `internal/automation/target`: target identity and coordinate-space value types.
- `internal/automation/controller`: controller capabilities and Win32 controller wrapper.

Compatibility rule:

- Existing runtime services remain authoritative for current nodes.
- New controller code wraps the same input/capture/window dependencies so behavior can be compared before node migration.

Next phase:

- Add trace records around controller calls.
- Route one narrow node path through the Win32 controller behind a feature flag.
- Use AE main-window to composition-dialog smoke as the first real target consistency test.
```

- [ ] **Step 2: Run focused tests**

Run:

```powershell
go test ./internal/automation/... -count=1
go test ./internal/services/container/runtime -run TestWindowHandleToTarget -count=1
```

Expected: both pass. Runtime package may still contain unrelated known redline tests when run without `-run`; do not use broad runtime package success as a Phase 1 gate.

- [ ] **Step 3: Run formatting**

Run:

```powershell
gofmt -w internal\\automation internal\\services\\container\\runtime\\automation_adapters.go internal\\services\\container\\runtime\\automation_adapters_test.go
```

Expected: no output.

- [ ] **Step 4: Run status check**

Run:

```powershell
git status --short
```

Expected: only files from this plan plus pre-existing unrelated dirty files.

- [ ] **Step 5: Commit the task**

```powershell
git add internal/automation internal/services/container/runtime/automation_adapters.go internal/services/container/runtime/automation_adapters_test.go flightdeck/knowledge/architecture/target-controller-phase1-notes.md
git commit -m "docs: record target controller phase one"
```

## Self-Review

Spec coverage:

- Target identity is covered by Task 1.
- Controller interfaces are covered by Task 2.
- Win32 wrapper is covered by Tasks 3 and 4.
- Runtime compatibility bridge is covered by Task 5.
- Documentation is covered by Task 6.

Placeholder scan:

- No open-ended implementation placeholders remain.
- Android, browser, Rust native, trace UI, and node migration are explicitly out of scope.

Type consistency:

- `TargetRef.HWND` is `uintptr` throughout the new abstraction.
- Phase 1 controller accepts only normalized points for Win32 input because existing `pkg/input.Backend` already takes ratio coordinates.
- Screenshot returns a controller `Frame` so later trace code can attach coordinate-space metadata without changing capture backends first.
