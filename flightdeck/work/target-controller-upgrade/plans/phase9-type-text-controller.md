# Phase 9 TypeText Controller Routing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Route runtime `InputService.TypeText` through `Win32Controller.Text` so text input is traceable and carries Phase 8 source metadata.

**Architecture:** Keep the existing backend text injection implementation. `Win32Controller.Text` already delegates to `Win32Input.TypeText` and records a `text` action; runtime only needs to stop bypassing the controller.

**Tech Stack:** Go, runtime input adapter, Win32Controller, node dispatch tests.

---

## Scope

In scope:

- Change `inputAdapter.TypeText` to call `Win32Controller.Text`.
- Add runtime test coverage for `InputText` node trace action/source metadata.
- Preserve existing `InputText` node validation and backend delegation semantics.

Out of scope:

- IME policy, clipboard paste fallback, browser DOM text input, Android text input.
- Drag, MouseDown, MouseUp routing.
- UI trace viewer and persistence.

## Design Notes

- Do not change `pkg/input` behavior in this phase.
- SendInput vs PostMessage text behavior remains selected by the existing runtime input backend.
- `Win32Controller.Text` is the canonical trace boundary for text, even if the underlying backend has platform-specific behavior.

## Tasks

1. Commit this plan and update `flightdeck/work/target-controller-upgrade/index.md`.
2. Add failing runtime test proving `InputText` emits a `text` trace with node source.
3. Route `inputAdapter.TypeText` through `Win32Controller.Text`.
4. Run focused runtime/controller/node tests.
5. Add Phase 9 notes, update index, and commit docs.

## Verification

```powershell
go test ./internal/services/container/runtime -run "TestExecNodeViaFramework_InputTextTraceIncludesNodeSource|TestInputAdapter|TestRuntimeContextTrace" -count=1
go test ./internal/automation/controller -count=1
go test ./internal/nodes/input -run TestInputText -count=1
```

## Acceptance

- `InputText` still delegates to the selected runtime input backend.
- Runtime trace includes one `text` action for `InputText`.
- The `text` trace includes container/node/kind/in-pin source metadata when dispatched through the framework.
- Worktree is clean after commits.
