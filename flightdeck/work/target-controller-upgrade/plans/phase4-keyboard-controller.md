# Phase 4 Keyboard Controller Routing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Route the first narrow runtime input path, `InputService.KeyDown/KeyUp`, through `Win32Controller` with runtime-owned trace.

**Architecture:** Keep node APIs unchanged. `inputAdapter` still implements `node.InputService`, but `KeyDown` and `KeyUp` construct a `Win32Controller` from the current runtime window target and delegate to controller keyboard state methods. `Win32Controller` records trace through `RuntimeContext.TraceRecorder()`.

**Tech Stack:** Go, existing `internal/automation/controller`, `internal/automation/target`, runtime node service adapters, `go test`.

---

## Scope

In scope:

- Add a runtime adapter from `pkg/input.Backend` to `controller.Win32Input`.
- Route `inputAdapter.KeyDown` and `inputAdapter.KeyUp` through `controller.Win32Controller`.
- Preserve existing `node.InputService` interface and node specs.
- Record trace actions `key-down` and `key-up` in `RuntimeContext`.
- Keep backend label from `rt.Input.Name()`.

Out of scope:

- Click, mouse move, scroll, drag, text, screenshot routing.
- Android/browser/CDP routing.
- Node id / pin id metadata in trace records.
- UI trace viewer or persistence.
- New shortcut/chord node design.

## Files

- Modify: `internal/services/container/runtime/automation_adapters.go`
  - Add a `runtimeWin32Input` wrapper that adapts `pkginput.Backend` `win.HWND` methods to `controller.Win32Input` `uintptr` methods.
- Modify: `internal/services/container/runtime/node_services.go`
  - Import `internal/automation/controller`.
  - Add `inputAdapter.controller()` helper.
  - Change `KeyDown` / `KeyUp` to use `Win32Controller`.
- Modify: `internal/services/container/runtime/node_services_test.go`
  - Add tests proving `KeyDown` / `KeyUp` delegate to the existing backend and create runtime trace records.
- Create: `flightdeck/knowledge/architecture/target-controller-phase4-notes.md`
  - Record the first migrated runtime path and remaining boundaries.
- Modify: `flightdeck/work/target-controller-upgrade/index.md`
  - Mark Phase 4 keyboard routing as complete after implementation.

## Task 1: Plan Commit

- [ ] **Step 1: Save this plan under flightdeck**

File: `flightdeck/work/target-controller-upgrade/plans/phase4-keyboard-controller.md`

- [ ] **Step 2: Update topic index to mention the active Phase 4 plan**

Modify `flightdeck/work/target-controller-upgrade/index.md`:

```markdown
## Next

Execute `plans/phase4-keyboard-controller.md`: route only `InputService.KeyDown/KeyUp` through `Win32Controller` and runtime trace. Do not migrate mouse/click/text/screenshot until a later plan.
```

- [ ] **Step 3: Commit**

```powershell
git add flightdeck\work\target-controller-upgrade\plans\phase4-keyboard-controller.md flightdeck\work\target-controller-upgrade\index.md
git commit -m "docs(architecture): plan keyboard controller routing phase"
```

## Task 2: Failing Runtime Adapter Tests

- [ ] **Step 1: Add a recording input backend to `node_services_test.go`**

Add this helper near existing test fakes:

```go
type recordingRuntimeInput struct {
	keyDownHWND []uintptr
	keyDownKeys []string
	keyUpHWND   []uintptr
	keyUpKeys   []string
}

func (r *recordingRuntimeInput) Name() string { return "sendinput" }
func (r *recordingRuntimeInput) Capabilities() pkginput.Capabilities { return pkginput.Capabilities{} }
func (r *recordingRuntimeInput) Click(win.HWND, float64, float64, string, int) error { return nil }
func (r *recordingRuntimeInput) KeyDown(hwnd win.HWND, key string) error {
	r.keyDownHWND = append(r.keyDownHWND, uintptr(hwnd))
	r.keyDownKeys = append(r.keyDownKeys, key)
	return nil
}
func (r *recordingRuntimeInput) KeyUp(hwnd win.HWND, key string) error {
	r.keyUpHWND = append(r.keyUpHWND, uintptr(hwnd))
	r.keyUpKeys = append(r.keyUpKeys, key)
	return nil
}
func (r *recordingRuntimeInput) MouseDown(win.HWND, float64, float64, string) error { return nil }
func (r *recordingRuntimeInput) MouseUp(win.HWND, string) error { return nil }
func (r *recordingRuntimeInput) MouseMoveRel(win.HWND, int, int, int) error { return nil }
func (r *recordingRuntimeInput) Scroll(win.HWND, float64, float64, int, bool) error { return nil }
func (r *recordingRuntimeInput) Drag(win.HWND, float64, float64, float64, float64, string, int) error { return nil }
func (r *recordingRuntimeInput) TypeText(win.HWND, string) error { return nil }
func (r *recordingRuntimeInput) MoveTo(win.HWND, float64, float64) error { return nil }
func (r *recordingRuntimeInput) CursorRatio(win.HWND) (float64, float64, error) { return 0, 0, nil }
func (r *recordingRuntimeInput) ReleaseAll() error { return nil }
```

- [ ] **Step 2: Write failing tests**

Add tests:

```go
func TestInputAdapter_KeyDownRoutesThroughControllerTrace(t *testing.T) {
	rt := newAdapterTestRT(t, nil)
	rt.SetActiveWindow(winutil.WindowHandle{HWND: 77, Title: "After Effects", ClientW: 1920, ClientH: 1080})
	input := &recordingRuntimeInput{}
	rt.Input = input

	err := NewInputAdapter(rt).KeyDown("ctrl")
	if err != nil {
		t.Fatalf("KeyDown error = %v", err)
	}
	if len(input.keyDownHWND) != 1 || input.keyDownHWND[0] != 77 || input.keyDownKeys[0] != "ctrl" {
		t.Fatalf("backend KeyDown = hwnds %#v keys %#v, want hwnd 77 key ctrl", input.keyDownHWND, input.keyDownKeys)
	}
	records := rt.TraceRecords()
	if len(records) != 1 {
		t.Fatalf("trace len = %d, want 1", len(records))
	}
	if records[0].Action != "key-down" || records[0].Target.ID != "win32:77" || records[0].Backend != "sendinput" {
		t.Fatalf("trace record = %#v", records[0])
	}
}

func TestInputAdapter_KeyUpRoutesThroughControllerTrace(t *testing.T) {
	rt := newAdapterTestRT(t, nil)
	rt.SetActiveWindow(winutil.WindowHandle{HWND: 88, Title: "After Effects", ClientW: 1920, ClientH: 1080})
	input := &recordingRuntimeInput{}
	rt.Input = input

	err := NewInputAdapter(rt).KeyUp("n")
	if err != nil {
		t.Fatalf("KeyUp error = %v", err)
	}
	if len(input.keyUpHWND) != 1 || input.keyUpHWND[0] != 88 || input.keyUpKeys[0] != "n" {
		t.Fatalf("backend KeyUp = hwnds %#v keys %#v, want hwnd 88 key n", input.keyUpHWND, input.keyUpKeys)
	}
	records := rt.TraceRecords()
	if len(records) != 1 {
		t.Fatalf("trace len = %d, want 1", len(records))
	}
	if records[0].Action != "key-up" || records[0].Target.ID != "win32:88" || records[0].Backend != "sendinput" {
		t.Fatalf("trace record = %#v", records[0])
	}
}
```

- [ ] **Step 3: Run tests and verify red**

Run:

```powershell
go test ./internal/services/container/runtime -run "TestInputAdapter_KeyDownRoutesThroughControllerTrace|TestInputAdapter_KeyUpRoutesThroughControllerTrace" -count=1
```

Expected: FAIL because trace records are still empty.

## Task 3: Runtime Win32 Controller Routing

- [ ] **Step 1: Add runtime input wrapper**

Modify `internal/services/container/runtime/automation_adapters.go`:

```go
import (
	"fmt"

	"github.com/lxn/win"

	"yotta/internal/automation/target"
	pkginput "yotta/pkg/input"
	"yotta/pkg/winutil"
)

type runtimeWin32Input struct {
	backend pkginput.Backend
}

func (a runtimeWin32Input) Click(hwnd uintptr, xRatio, yRatio float64, button string, durMs int) error {
	return a.backend.Click(win.HWND(hwnd), xRatio, yRatio, button, durMs)
}

func (a runtimeWin32Input) KeyDown(hwnd uintptr, key string) error {
	return a.backend.KeyDown(win.HWND(hwnd), key)
}

func (a runtimeWin32Input) KeyUp(hwnd uintptr, key string) error {
	return a.backend.KeyUp(win.HWND(hwnd), key)
}

func (a runtimeWin32Input) TypeText(hwnd uintptr, text string) error {
	return a.backend.TypeText(win.HWND(hwnd), text)
}

func (a runtimeWin32Input) MoveTo(hwnd uintptr, xRatio, yRatio float64) error {
	return a.backend.MoveTo(win.HWND(hwnd), xRatio, yRatio)
}

func (a runtimeWin32Input) Scroll(hwnd uintptr, xRatio, yRatio float64, notches int, horizontal bool) error {
	return a.backend.Scroll(win.HWND(hwnd), xRatio, yRatio, notches, horizontal)
}
```

Keep the existing `windowHandleToTarget` function below the new wrapper.

- [ ] **Step 2: Add controller helper and route KeyDown/KeyUp**

Modify `internal/services/container/runtime/node_services.go`:

```go
import (
	// existing imports
	"yotta/internal/automation/controller"
)
```

Add helper on `inputAdapter`:

```go
func (a *inputAdapter) controller() (*controller.Win32Controller, error) {
	if err := a.ensure(); err != nil {
		return nil, err
	}
	wh := a.rt.WindowHandle()
	if wh.HWND == 0 {
		return nil, ErrNoActiveWindow
	}
	backend := ""
	if a.rt.Input != nil {
		backend = a.rt.Input.Name()
	}
	return controller.NewWin32Controller(windowHandleToTarget(wh), controller.Win32Deps{
		Input:   runtimeWin32Input{backend: a.rt.Input},
		Trace:   a.rt.TraceRecorder(),
		Backend: backend,
	})
}
```

Replace `KeyDown`:

```go
func (a *inputAdapter) KeyDown(vk string) error {
	ctrl, err := a.controller()
	if err != nil {
		return err
	}
	return ctrl.KeyDown(context.Background(), controller.KeyRequest{
		Key: vk,
		Policy: controller.ActionPolicy{
			ForegroundRequired: true,
		},
	})
}
```

Replace `KeyUp`:

```go
func (a *inputAdapter) KeyUp(vk string) error {
	ctrl, err := a.controller()
	if err != nil {
		return err
	}
	return ctrl.KeyUp(context.Background(), controller.KeyRequest{
		Key: vk,
		Policy: controller.ActionPolicy{
			ForegroundRequired: true,
		},
	})
}
```

- [ ] **Step 3: Format and run focused tests**

Run:

```powershell
gofmt -w internal\services\container\runtime\automation_adapters.go internal\services\container\runtime\node_services.go internal\services\container\runtime\node_services_test.go
go test ./internal/services/container/runtime -run "TestInputAdapter_KeyDownRoutesThroughControllerTrace|TestInputAdapter_KeyUpRoutesThroughControllerTrace" -count=1
```

Expected: PASS.

- [ ] **Step 4: Run related runtime tests**

Run:

```powershell
go test ./internal/services/container/runtime -run "TestInputAdapter|TestRuntimeContextTrace|TestWindowHandleToTarget|TestTryHookF|TestStateIDLE" -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add internal\services\container\runtime\automation_adapters.go internal\services\container\runtime\node_services.go internal\services\container\runtime\node_services_test.go
git commit -m "feat(runtime): route keyboard state through win32 controller"
```

## Task 4: Notes And Verification

- [ ] **Step 1: Create Phase 4 notes**

Create `flightdeck/knowledge/architecture/target-controller-phase4-notes.md`:

```markdown
# Target / Controller Phase 4 Notes

SUMMARY: Phase 4 routes runtime KeyDown/KeyUp through Win32Controller with runtime trace
READ WHEN: Continuing input-node migration / debugging keyboard trace / planning click or text controller routing
RECHECK WHEN: `inputAdapter`, `Win32Controller`, or `pkg/input.Backend` changes

---

Phase 4 migrates the first runtime action path:

- `inputAdapter.KeyDown` and `inputAdapter.KeyUp` now construct a `Win32Controller`.
- The controller target comes from the current active `WindowHandle`.
- The controller input dependency wraps the existing `pkg/input.Backend`.
- Trace records are written to the current `RuntimeContext`.

Still not migrated:

- Click, move, scroll, drag, text, screenshot.
- Node id / pin id trace metadata.
- Android/browser/CDP controllers.
- UI trace viewer and persistence.

Operational note:

- Existing nodes keep their public behavior and still call `node.InputService`.
- `KeyPress` emits two trace records, `key-down` then `key-up`.
- Backend labels come from `pkg/input.Backend.Name()`.
```

- [ ] **Step 2: Update topic index**

Modify `flightdeck/work/target-controller-upgrade/index.md` so State/Progress mention Phase 4 keyboard routing completed, and Next says a future Phase 5 plan should pick either click/coordinate routing or text/chord support.

- [ ] **Step 3: Run verification**

Run:

```powershell
go test ./internal/automation/... -count=1
go test ./internal/services/container/runtime -run "TestInputAdapter|TestRuntimeContextTrace|TestWindowHandleToTarget|TestTryHookF|TestStateIDLE" -count=1
```

Expected: PASS.

- [ ] **Step 4: Commit docs**

```powershell
git add flightdeck\knowledge\architecture\target-controller-phase4-notes.md flightdeck\work\target-controller-upgrade\index.md
git commit -m "docs(architecture): record keyboard controller routing phase"
```

## Acceptance

- Existing input nodes compile unchanged.
- `InputService.KeyDown/KeyUp` still delegate to the configured `pkg/input.Backend`.
- Runtime trace records are emitted for `key-down` and `key-up`.
- Worktree is clean after commits.
