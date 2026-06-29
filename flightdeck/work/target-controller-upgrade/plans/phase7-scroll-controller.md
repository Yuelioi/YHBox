# Phase 7 Scroll Controller Routing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Route `InputService.Scroll` through `Win32Controller` and record a minimal coordinate step for scroll points.

**Architecture:** Reuse the runtime Win32 input wrapper. `inputAdapter.Scroll` receives normalized client ratios and delegates to `Win32Controller.Scroll`. Controller trace records action `scroll`, request details, backend, target, and one normalized-to-window-client coordinate step.

**Tech Stack:** Go, `internal/automation/controller`, runtime service adapters, `go test`.

---

## Scope

In scope:

- Add coordinate-step trace metadata to `Win32Controller.Scroll`.
- Route only `inputAdapter.Scroll`.
- Preserve existing `Scroll` node point resolution, delta, and axis behavior.

Out of scope:

- Drag, mouse down/up, text, screenshot.
- Pixel-to-normalized conversion inside controller.
- Node id / pin id trace metadata.
- UI trace viewer or persistence.

## Tasks

1. Commit this plan and update `flightdeck/work/target-controller-upgrade/index.md`.
2. Add a failing controller test proving `Win32Controller.Scroll` records one coordinate step.
3. Implement scroll coordinate-step recording in controller.
4. Add a failing runtime test proving `InputService.Scroll` routes through controller trace.
5. Implement runtime `inputAdapter.Scroll` controller routing.
6. Add Phase 7 notes, update index, run verification, and commit docs.

## Verification

```powershell
go test ./internal/automation/... -count=1
go test ./internal/services/container/runtime -run "TestInputAdapter|TestRuntimeContextTrace|TestWindowHandleToTarget|TestScroll|TestClickAt|TestStateSETUP|TestStateIDLE" -count=1
```

## Acceptance

- `Win32Controller.Scroll` trace records include one coordinate step.
- `InputService.Scroll` delegates through `Win32Controller`.
- Existing backend still receives hwnd, normalized ratios, notches, and horizontal flag.
- Worktree is clean after commits.
