# Phase 21 — Runtime Active Target

## Goal

Introduce an active `target.Target` in `RuntimeContext` so runtime services can eventually route to Win32, Android ADB, or Browser CDP controllers instead of assuming every active target is a Win32 HWND.

## Scope

- Add active target storage to `RuntimeContext`.
- Keep `SetActiveWindow` behavior, but make it also set the active target via `windowHandleToTarget`.
- Add `SetActiveTarget` / `ActiveTarget` helpers.
- Update input/capture controller construction to read the active target.
- Preserve current Win32 behavior.
- Add tests for active target stickiness and window-to-target sync.

## Non-goals

- Add Android/Browser target-selection nodes.
- Wire Android/Browser controllers into runtime execution.
- Remove existing `WindowHandle` APIs.

## Verification

- `go test ./internal/services/container/runtime -run "TestRuntimeContext|TestInputAdapter|TestCaptureAdapter|TestWindowHandleToTarget" -count=1`
- `go test ./internal/services/container/runtime -count=1`
