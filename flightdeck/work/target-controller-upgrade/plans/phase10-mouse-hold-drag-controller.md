# Phase 10 Mouse Hold And Drag Controller Routing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Route runtime `MouseDown`, `MouseUp`, and `Drag` through `Win32Controller` so hold/drag input is traceable and carries source metadata.

**Architecture:** Extend the Win32 controller/input boundary with primitive mouse button requests and drag requests. Keep backend behavior unchanged; controller owns validation, trace action naming, target attribution, coordinate-step metadata, and source metadata via the Phase 8 recorder wrapper.

**Tech Stack:** Go, Win32Controller, runtime input adapter, input node dispatch tests.

---

## Scope

In scope:

- Add controller request types for mouse down, mouse up, and drag.
- Add `MouseDown`, `MouseUp`, and `Drag` methods to `Win32Controller`.
- Extend `Win32Input` and `runtimeWin32Input` to delegate to existing backend methods.
- Route `inputAdapter.MouseDown`, `inputAdapter.MouseUp`, and `inputAdapter.Drag` through the controller.
- Add runtime tests for `MouseHoldStart`, `MouseHoldStop`, and `Swipe` trace/source metadata.

Out of scope:

- Reimplement backend drag as down/move/up composition.
- Cursor trajectory recording beyond begin/end coordinate steps.
- MouseMoveRel controller routing.
- UI trace viewer and persistence.

## Design Notes

- `mouse-down` records a normalized point coordinate step.
- `mouse-up` records button only; current backend API does not accept a release point.
- `drag` records begin and end normalized point coordinate steps while still delegating to the backend drag primitive.
- Button defaults remain handled by existing nodes/backends.

## Tasks

1. Commit this plan and update `flightdeck/work/target-controller-upgrade/index.md`.
2. Add failing controller tests for mouse down/up/drag trace records.
3. Implement controller request types and methods.
4. Add failing runtime tests for `MouseHoldStart`, `MouseHoldStop`, and `Swipe` trace/source metadata.
5. Route runtime adapter methods through the controller.
6. Run focused runtime/controller/input-node tests.
7. Add Phase 10 notes, update index, and commit docs.

## Verification

```powershell
go test ./internal/automation/controller -count=1
go test ./internal/services/container/runtime -run "TestExecNodeViaFramework_MouseHold|TestExecNodeViaFramework_Swipe|TestInputAdapter|TestRuntimeContextTrace" -count=1
go test ./internal/nodes/input -run "TestMouseHold|TestSwipe" -count=1
```

## Acceptance

- `MouseHoldStart` emits `mouse-down` trace with source metadata.
- `MouseHoldStop` emits `mouse-up` trace with source metadata.
- `Swipe` emits `drag` trace with source metadata and begin/end coordinate steps.
- Existing backend behavior and node-level tests keep passing.
- Worktree is clean after commits.
