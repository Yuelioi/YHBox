# Phase 11 Relative Move Controller Routing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Route runtime `MouseMoveRel` through `Win32Controller` as a relative/raw-input pointer action.

**Architecture:** Add a dedicated `RelativeMoveRequest` and `Win32Controller.MoveRelative` method. This action is not a normalized coordinate move; it records dx/dy/duration in the request and emits a `move-relative` trace without coordinate steps.

**Tech Stack:** Go, Win32Controller, runtime input adapter, input node dispatch tests.

---

## Scope

In scope:

- Add `RelativeMoveRequest` with `Dx`, `Dy`, and `DurationMs`.
- Add `MoveRelative` to controller pointer input and Win32 controller.
- Extend `Win32Input` / `runtimeWin32Input` to delegate to existing backend `MouseMoveRel`.
- Route `inputAdapter.MouseMoveRel` through `Win32Controller.MoveRelative`.
- Add runtime test coverage for `MouseMoveRel` trace/source metadata.

Out of scope:

- Mouse calibration math.
- Raw input smoothing or trajectory details.
- Reworking backend SendInput/PostMessage semantics.
- UI trace viewer and persistence.

## Design Notes

- `move-relative` intentionally has no coordinate steps.
- Relative movement is a raw delta action, not a point action.
- This phase completes controller routing for current `InputService` methods.

## Tasks

1. Commit this plan and update `flightdeck/work/target-controller-upgrade/index.md`.
2. Add failing controller test for `MoveRelative`.
3. Implement request type, interface method, and Win32 controller method.
4. Add failing runtime test for `MouseMoveRel` trace/source metadata.
5. Route runtime adapter through the controller.
6. Run focused runtime/controller/input-node tests.
7. Add Phase 11 notes, update index, and commit docs.

## Verification

```powershell
go test ./internal/automation/controller -count=1
go test ./internal/services/container/runtime -run "TestExecNodeViaFramework_MouseMoveRel|TestInputAdapter|TestRuntimeContextTrace" -count=1
go test ./internal/nodes/input -run TestMouseMoveRel -count=1
```

## Acceptance

- `MouseMoveRel` emits one `move-relative` trace with source metadata.
- Backend `MouseMoveRel` receives the same hwnd, dx, dy, and duration.
- The trace has no coordinate steps.
- All current runtime `InputService` methods that can use controller now route through it.
