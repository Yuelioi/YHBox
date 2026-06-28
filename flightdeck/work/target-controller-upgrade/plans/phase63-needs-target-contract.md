# Phase 63 — NeedsTarget Contract Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Split cross-platform automation target requirements from Win32 HWND window requirements.

**Architecture:** Add `node.Spec.NeedsTarget` for nodes that use target-aware services (`Input`, `Capture`, `Vision`) and keep `NeedsWindow` only for nodes that directly require a Win32 HWND / `WindowService`. Validator accepts Android/Browser/Win32 target selection nodes for `NeedsTarget`, while still requiring `Win32WindowTarget` for real Win32 window operations. Runtime only initialises Win32 input/capture backends when a graph can actually use a Win32 target/window path.

**Tech Stack:** Go node spec, container validator, runtime setup, built-in node specs, Go tests, Flightdeck.

---

## Tasks

- [x] Write failing tests:
  - `AndroidTarget + ClickAt` must not report `MISSING_WIN32_WINDOW_TARGET`.
  - `AndroidTarget + Capture` must not initialise Win32 input/capture backends in `setupRuntime`.
- [x] Add `NeedsTarget` to `node.Spec` and catalog metadata.
- [x] Mark target-aware input/detect/image nodes with `NeedsTarget`.
- [x] Keep `NeedsWindow` on direct Win32 window operation nodes.
- [x] Change validator missing-target logic:
  - `NeedsWindow` still requires `Win32WindowTarget` unless a `Window` input is wired.
  - `NeedsTarget` requires any target selection node unless a `Window` input is wired.
  - Missing target currently reports `MISSING_WIN32_WINDOW_TARGET` so existing auto-fix remains the Windows default.
- [x] Change runtime setup logic:
  - Build Win32 input/capture backends for direct `NeedsWindow`.
  - Build them for `NeedsTarget` only when the graph has `Win32WindowTarget` or no explicit non-Win32 target.
  - Do not build them for Android/Browser-only target graphs.
- [x] Update all-node boundary guards for `NeedsTarget` / `NeedsForeground`.
- [x] Run verification and update Phase63 notes/index.

## Verification

- [x] `go test ./internal/services/container -run "TestValidate_(AndroidTargetWithInput_NoMissingWin32WindowTarget|UnwiredNeedsWindow_ReportsMissingWin32WindowTarget)" -count=1`
- [x] `go test ./internal/services/container/runtime -run "TestSetupRuntime_(AndroidTargetDoesNotBuildWin32Backends|BuildsBackendsWithoutResolvingWindow)" -count=1`
- [x] `go test ./internal/nodes/all ./internal/services/container ./internal/services/container/runtime ./internal/catalog`
- [x] `go test ./...`
- [x] `cd frontend && pnpm gen:node-i18n && pnpm i18n:check`
- [x] `git diff --check`
