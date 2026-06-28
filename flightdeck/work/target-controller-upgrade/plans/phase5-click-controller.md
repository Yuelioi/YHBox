# Phase 5 Click Controller Routing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Route `InputService.Click` through `Win32Controller` with runtime-owned trace.

**Architecture:** Reuse the Phase 4 runtime Win32 input wrapper. `inputAdapter.Click` still receives normalized client ratios from nodes, constructs a controller from the active runtime window, and delegates to `Win32Controller.Click`. This records a single `click` action while leaving `MoveTo`, drag, scroll, and text unchanged.

**Tech Stack:** Go, existing runtime service adapters, `internal/automation/controller`, `go test`.

---

## Scope

In scope:

- Route only `inputAdapter.Click`.
- Preserve existing `ClickAt` node API and point resolution behavior.
- Record trace action `click` with target id and backend label.
- Keep `MoveTo` outside this phase even though `ClickAt` calls it before clicking.

Out of scope:

- MoveTo, drag, mouse down/up, scroll, text, screenshot.
- Click coordinate conversion beyond existing normalized ratios.
- Node id / pin id trace metadata.
- UI trace viewer or persistence.

## Task 1: Plan Commit

- [ ] Save this plan to `flightdeck/work/target-controller-upgrade/plans/phase5-click-controller.md`.
- [ ] Update `flightdeck/work/target-controller-upgrade/index.md` Next to mention active Phase 5 click routing.
- [ ] Commit:

```powershell
git add flightdeck\work\target-controller-upgrade\plans\phase5-click-controller.md flightdeck\work\target-controller-upgrade\index.md
git commit -m "docs(architecture): plan click controller routing phase"
```

## Task 2: Failing Click Trace Test

- [ ] Extend `recordingRuntimeInput` in `internal/services/container/runtime/node_services_test.go` with click fields:

```go
clickHWND     []uintptr
clickX        []float64
clickY        []float64
clickButton   []string
clickDuration []int
```

- [ ] Replace its `Click` method with:

```go
func (r *recordingRuntimeInput) Click(hwnd win.HWND, xRatio, yRatio float64, button string, durMs int) error {
	r.clickHWND = append(r.clickHWND, uintptr(hwnd))
	r.clickX = append(r.clickX, xRatio)
	r.clickY = append(r.clickY, yRatio)
	r.clickButton = append(r.clickButton, button)
	r.clickDuration = append(r.clickDuration, durMs)
	return nil
}
```

- [ ] Add test:

```go
func TestInputAdapter_ClickRoutesThroughControllerTrace(t *testing.T) {
	rt := newAdapterTestRT(t, nil)
	rt.SetActiveWindow(winutil.WindowHandle{HWND: 99, Title: "After Effects", ClientW: 1920, ClientH: 1080})
	input := &recordingRuntimeInput{}
	rt.Input = input

	err := NewInputAdapter(rt).Click(0.25, 0.75, "right", 80)
	if err != nil {
		t.Fatalf("Click error = %v", err)
	}
	if len(input.clickHWND) != 1 || input.clickHWND[0] != 99 || input.clickX[0] != 0.25 || input.clickY[0] != 0.75 || input.clickButton[0] != "right" || input.clickDuration[0] != 80 {
		t.Fatalf("backend Click = hwnds %#v x %#v y %#v buttons %#v durations %#v", input.clickHWND, input.clickX, input.clickY, input.clickButton, input.clickDuration)
	}
	records := rt.TraceRecords()
	if len(records) != 1 {
		t.Fatalf("trace len = %d, want 1", len(records))
	}
	if records[0].Action != "click" || records[0].Target.ID != "win32:99" || records[0].Backend != "sendinput" {
		t.Fatalf("trace record = %#v", records[0])
	}
}
```

- [ ] Run and verify red:

```powershell
go test ./internal/services/container/runtime -run TestInputAdapter_ClickRoutesThroughControllerTrace -count=1
```

Expected: FAIL because trace records are still empty.

## Task 3: Route Click Through Controller

- [ ] Replace `inputAdapter.Click` in `internal/services/container/runtime/node_services.go`:

```go
func (a *inputAdapter) Click(xRatio, yRatio float64, button string, durationMs int) error {
	ctrl, err := a.controller()
	if err != nil {
		return err
	}
	return ctrl.Click(context.Background(), controller.ClickRequest{
		Point:      target.NewNormalizedPoint(xRatio, yRatio),
		Button:     button,
		DurationMs: durationMs,
		Policy: controller.ActionPolicy{
			ForegroundRequired: true,
		},
	})
}
```

- [ ] Add import to `node_services.go`:

```go
"yotta/internal/automation/target"
```

- [ ] Format and test:

```powershell
gofmt -w internal\services\container\runtime\node_services.go internal\services\container\runtime\node_services_test.go
go test ./internal/services/container/runtime -run "TestInputAdapter_ClickRoutesThroughControllerTrace|TestInputAdapter_KeyDownRoutesThroughControllerTrace|TestInputAdapter_KeyUpRoutesThroughControllerTrace" -count=1
```

Expected: PASS.

- [ ] Run related tests:

```powershell
go test ./internal/services/container/runtime -run "TestInputAdapter|TestWindowHandleToTarget|TestStateSETUP|TestClick" -count=1
```

Expected: PASS.

- [ ] Commit:

```powershell
git add internal\services\container\runtime\node_services.go internal\services\container\runtime\node_services_test.go
git commit -m "feat(runtime): route clicks through win32 controller"
```

## Task 4: Notes And Verification

- [ ] Create `flightdeck/knowledge/architecture/target-controller-phase5-notes.md`.
- [ ] Update `flightdeck/work/target-controller-upgrade/index.md` to mark Phase 5 complete and set Next to Phase 6 plan.
- [ ] Run:

```powershell
go test ./internal/automation/... -count=1
go test ./internal/services/container/runtime -run "TestInputAdapter|TestRuntimeContextTrace|TestWindowHandleToTarget|TestStateSETUP|TestStateIDLE" -count=1
```

- [ ] Commit:

```powershell
git add flightdeck\knowledge\architecture\target-controller-phase5-notes.md flightdeck\work\target-controller-upgrade\index.md
git commit -m "docs(architecture): record click controller routing phase"
```

## Acceptance

- `InputService.Click` delegates through `Win32Controller`.
- Existing backend still receives the same hwnd, normalized ratios, button, and duration.
- Runtime trace records a `click` action.
- Worktree is clean after commits.
