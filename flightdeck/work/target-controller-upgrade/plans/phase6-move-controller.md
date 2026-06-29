# Phase 6 Move Controller Routing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Route `InputService.MoveTo` through `Win32Controller` and add a minimal coordinate-step trace for normalized pointer movement.

**Architecture:** Keep node APIs unchanged. `inputAdapter.MoveTo` receives normalized client ratios from nodes and delegates to `Win32Controller.Move`. `Win32Controller.Move` records the request plus one coordinate step that states the normalized point accepted by the controller and the target window-client space it applies to.

**Tech Stack:** Go, existing `internal/automation/controller`, runtime service adapters, `go test`.

---

## Scope

In scope:

- Add coordinate-step trace metadata to `Win32Controller.Move`.
- Route only `inputAdapter.MoveTo` through `Win32Controller`.
- Preserve existing `ClickAt` behavior: `MoveTo` still runs before `Click`, but now both can be traced.
- Keep backend label from `pkg/input.Backend.Name()`.

Out of scope:

- Drag, mouse down/up, scroll, text, screenshot.
- Pixel-to-normalized conversion inside controller.
- Node id / pin id trace metadata.
- UI trace viewer or persistence.

## Task 1: Plan Commit

- [ ] Save this plan to `flightdeck/work/target-controller-upgrade/plans/phase6-move-controller.md`.
- [ ] Update `flightdeck/work/target-controller-upgrade/index.md` Next to mention active Phase 6 move routing.
- [ ] Commit:

```powershell
git add flightdeck\work\target-controller-upgrade\plans\phase6-move-controller.md flightdeck\work\target-controller-upgrade\index.md
git commit -m "docs(architecture): plan move controller routing phase"
```

## Task 2: Failing Controller Coordinate-Step Test

- [ ] Extend `fakeWin32Input` in `internal/automation/controller/win32_test.go` with move fields:

```go
moveHWND uintptr
moveX    float64
moveY    float64
```

- [ ] Replace `MoveTo` fake:

```go
func (f *fakeWin32Input) MoveTo(hwnd uintptr, xRatio, yRatio float64) error {
	f.moveHWND = hwnd
	f.moveX = xRatio
	f.moveY = yRatio
	return nil
}
```

- [ ] Add test:

```go
func TestWin32ControllerMoveRecordsCoordinateStep(t *testing.T) {
	in := &fakeWin32Input{}
	rec := automationtrace.NewMemoryRecorder()
	ctrl, err := NewWin32Controller(target.Target{
		ID:   "win32:42",
		Kind: target.KindWin32Window,
		Ref:  target.TargetRef{HWND: 42},
	}, Win32Deps{Input: in, Trace: rec})
	if err != nil {
		t.Fatalf("NewWin32Controller() error = %v", err)
	}
	if err := ctrl.Move(context.Background(), MoveRequest{Point: target.NewNormalizedPoint(0.25, 0.75)}); err != nil {
		t.Fatalf("Move() error = %v", err)
	}
	if in.moveHWND != 42 || in.moveX != 0.25 || in.moveY != 0.75 {
		t.Fatalf("delegate move = hwnd %d (%f,%f)", in.moveHWND, in.moveX, in.moveY)
	}
	records := rec.Records()
	if len(records) != 1 {
		t.Fatalf("trace records len = %d, want 1", len(records))
	}
	got := records[0]
	if got.Action != "move" {
		t.Fatalf("trace action = %q, want move", got.Action)
	}
	if len(got.CoordinateSteps) != 1 {
		t.Fatalf("coordinate steps len = %d, want 1", len(got.CoordinateSteps))
	}
	step := got.CoordinateSteps[0]
	if step.From != target.SpaceNormalized || step.To != target.SpaceWindowClient {
		t.Fatalf("coordinate step spaces = %s -> %s", step.From, step.To)
	}
}
```

- [ ] Run and verify red:

```powershell
go test ./internal/automation/controller -run TestWin32ControllerMoveRecordsCoordinateStep -count=1
```

Expected: FAIL because `CoordinateSteps` is empty.

## Task 3: Add Move Coordinate Step Recording

- [ ] Add helper in `internal/automation/controller/win32.go`:

```go
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
```

- [ ] Change existing `recordAction` to delegate:

```go
func (c *Win32Controller) recordAction(action string, request any, run func() error) error {
	return c.recordActionWithSteps(action, request, nil, run)
}
```

- [ ] Change `Move` to call `recordActionWithSteps`:

```go
func (c *Win32Controller) Move(ctx context.Context, req MoveRequest) error {
	steps := []automationtrace.CoordinateStep{{
		From:   req.Point.Space,
		To:     target.SpaceWindowClient,
		Input:  req.Point,
		Output: req.Point,
	}}
	if steps[0].From == "" {
		steps[0].From = target.SpaceNormalized
	}
	return c.recordActionWithSteps("move", req, steps, func() error {
		...
	})
}
```

- [ ] Run:

```powershell
gofmt -w internal\automation\controller\win32.go internal\automation\controller\win32_test.go
go test ./internal/automation/controller -run TestWin32ControllerMoveRecordsCoordinateStep -count=1
```

Expected: PASS.

- [ ] Commit:

```powershell
git add internal\automation\controller\win32.go internal\automation\controller\win32_test.go
git commit -m "feat(automation): trace win32 move coordinate steps"
```

## Task 4: Failing Runtime MoveTo Test

- [ ] Extend `recordingRuntimeInput` in `internal/services/container/runtime/node_services_test.go` with move fields:

```go
moveHWND []uintptr
moveX    []float64
moveY    []float64
```

- [ ] Replace `MoveTo` fake:

```go
func (r *recordingRuntimeInput) MoveTo(hwnd win.HWND, xRatio, yRatio float64) error {
	r.moveHWND = append(r.moveHWND, uintptr(hwnd))
	r.moveX = append(r.moveX, xRatio)
	r.moveY = append(r.moveY, yRatio)
	return nil
}
```

- [ ] Add test:

```go
func TestInputAdapter_MoveToRoutesThroughControllerTrace(t *testing.T) {
	rt := newAdapterTestRT(t, nil)
	rt.SetActiveWindow(winutil.WindowHandle{HWND: 66, Title: "After Effects", ClientW: 1920, ClientH: 1080})
	input := &recordingRuntimeInput{}
	rt.Input = input

	err := NewInputAdapter(rt).MoveTo(0.4, 0.6)
	if err != nil {
		t.Fatalf("MoveTo error = %v", err)
	}
	if len(input.moveHWND) != 1 || input.moveHWND[0] != 66 || input.moveX[0] != 0.4 || input.moveY[0] != 0.6 {
		t.Fatalf("backend MoveTo = hwnds %#v x %#v y %#v", input.moveHWND, input.moveX, input.moveY)
	}
	records := rt.TraceRecords()
	if len(records) != 1 {
		t.Fatalf("trace len = %d, want 1", len(records))
	}
	if records[0].Action != "move" || records[0].Target.ID != "win32:66" || records[0].Backend != "sendinput" {
		t.Fatalf("trace record = %#v", records[0])
	}
	if len(records[0].CoordinateSteps) != 1 {
		t.Fatalf("coordinate steps len = %d, want 1", len(records[0].CoordinateSteps))
	}
}
```

- [ ] Run and verify red:

```powershell
go test ./internal/services/container/runtime -run TestInputAdapter_MoveToRoutesThroughControllerTrace -count=1
```

Expected: FAIL because runtime `MoveTo` still calls backend directly without trace.

## Task 5: Route Runtime MoveTo

- [ ] Replace `inputAdapter.MoveTo` in `internal/services/container/runtime/node_services.go`:

```go
func (a *inputAdapter) MoveTo(xRatio, yRatio float64) error {
	ctrl, err := a.controller()
	if err != nil {
		return err
	}
	return ctrl.Move(context.Background(), controller.MoveRequest{
		Point: target.NewNormalizedPoint(xRatio, yRatio),
		Policy: controller.ActionPolicy{
			ForegroundRequired: true,
		},
	})
}
```

- [ ] Run:

```powershell
gofmt -w internal\services\container\runtime\node_services.go internal\services\container\runtime\node_services_test.go
go test ./internal/services/container/runtime -run "TestInputAdapter_MoveToRoutesThroughControllerTrace|TestInputAdapter_ClickRoutesThroughControllerTrace|TestInputAdapter_KeyDownRoutesThroughControllerTrace|TestInputAdapter_KeyUpRoutesThroughControllerTrace" -count=1
```

Expected: PASS.

- [ ] Run related tests:

```powershell
go test ./internal/services/container/runtime -run "TestInputAdapter|TestWindowHandleToTarget|TestClickAt|TestStateSETUP" -count=1
```

Expected: PASS.

- [ ] Commit:

```powershell
git add internal\services\container\runtime\node_services.go internal\services\container\runtime\node_services_test.go
git commit -m "feat(runtime): route pointer move through win32 controller"
```

## Task 6: Notes And Verification

- [ ] Create `flightdeck/knowledge/architecture/target-controller-phase6-notes.md`.
- [ ] Update `flightdeck/work/target-controller-upgrade/index.md` to mark Phase 6 complete and set Next to Phase 7 plan.
- [ ] Run:

```powershell
go test ./internal/automation/... -count=1
go test ./internal/services/container/runtime -run "TestInputAdapter|TestRuntimeContextTrace|TestWindowHandleToTarget|TestClickAt|TestStateSETUP|TestStateIDLE" -count=1
```

- [ ] Commit:

```powershell
git add flightdeck\knowledge\architecture\target-controller-phase6-notes.md flightdeck\work\target-controller-upgrade\index.md
git commit -m "docs(architecture): record move controller routing phase"
```

## Acceptance

- `Win32Controller.Move` trace records include one coordinate step.
- `InputService.MoveTo` delegates through `Win32Controller`.
- Existing backend still receives the same hwnd and normalized ratios.
- Worktree is clean after commits.
