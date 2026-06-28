# Phase 12 Capture Controller Routing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Route runtime `CaptureService.Capture` and `CaptureROI` through `Win32Controller.Screenshot` so image capture is target-aware, traceable, and source-attributed.

**Architecture:** Add a runtime capture adapter for the controller capture dependency. Keep existing PNG encoding and ROI crop behavior in `captureAdapter`; only the full-frame acquisition boundary moves to `Win32Controller.Screenshot`. Copy `ServiceBundle` per node and replace `Capture` with a source-aware adapter alongside `Input`.

**Tech Stack:** Go, Win32Controller, runtime capture adapter, Capture node dispatch tests.

---

## Scope

In scope:

- Add `runtimeWin32Capture` to adapt `pkg/capture.IBackend` to `controller.Win32Capture`.
- Add optional source metadata to `captureAdapter`.
- Replace per-node `Capture` service in `bundleForNode`.
- Route `Capture()` and `CaptureROI()` full-frame acquisition through `Win32Controller.Screenshot`.
- Preserve existing ROI crop and PNG encode behavior.
- Add runtime test coverage for `Capture` node trace/source metadata.

Out of scope:

- Controller-native ROI crop.
- Capture cache / vision adapter migration.
- Screenshot UI and trace persistence.
- Browser/Android screenshot implementations.

## Design Notes

- `CaptureROI` still crops after the full frame is captured, so output compatibility is unchanged.
- `screenshot` trace source should point to the node that requested capture.
- Direct `NewCaptureAdapter(rt)` keeps empty source metadata, matching `NewInputAdapter(rt)`.

## Tasks

1. Commit this plan and update `flightdeck/work/target-controller-upgrade/index.md`.
2. Add failing runtime test proving `Capture` node emits `screenshot` trace/source metadata.
3. Implement runtime capture dependency adapter.
4. Make `captureAdapter` controller-backed and source-aware.
5. Replace per-node `Capture` service in `bundleForNode`.
6. Run focused runtime/controller/image-node tests.
7. Add Phase 12 notes, update index, and commit docs.

## Verification

```powershell
go test ./internal/services/container/runtime -run "TestExecNodeViaFramework_Capture|TestInputAdapter|TestRuntimeContextTrace" -count=1
go test ./internal/automation/controller -run TestWin32Controller -count=1
go test ./internal/nodes/image -run TestCapture -count=1
```

## Acceptance

- `Capture` emits one `screenshot` trace with source metadata.
- Existing PNG/JPEG capture node behavior remains unchanged.
- ROI crop still happens through runtime geometry resolution.
- Worktree is clean after commits.
