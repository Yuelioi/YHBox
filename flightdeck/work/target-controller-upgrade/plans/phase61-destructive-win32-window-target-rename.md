# Phase 61 — Destructive Win32WindowTarget Rename Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 破坏性移除旧 `WindowTarget` kind，统一改名为 `Win32WindowTarget`，不保留旧容器兼容。

**Architecture:** 全仓 kind/API/event/error-code 同步重命名，旧 `WindowTarget` 不再注册、不再校验、不再自动创建。Windows 窗口目标仍在 `Target` palette 分组；Windows 窗口操作节点继续在 `Window` 分组。

**Tech Stack:** Go node/runtime/services, Vue frontend, generated node i18n catalog, Flightdeck.

---

## Files

- Rename: `internal/nodes/system/window_target.go` → `internal/nodes/system/win32_window_target.go`
- Rename: `internal/nodes/system/window_target_test.go` → `internal/nodes/system/win32_window_target_test.go`
- Rename: `internal/services/container/runtime/windowtarget_dispatch_test.go` → `internal/services/container/runtime/win32_window_target_dispatch_test.go`
- Modify: all Go/TS/Vue/i18n/catalog files that mention `WindowTarget`, `windowtarget`, `window-target`, or `WINDOW_TARGET`
- Modify: Flightdeck phase notes to mark phase59/60 compatibility assumptions superseded

## Task 1: Mechanical rename

- [x] Replace identifiers and strings:
  - `WindowTarget` → `Win32WindowTarget`
  - `windowTarget` → `win32WindowTarget`
  - `windowtarget` → `win32windowtarget`
  - `window-target` → `win32-window-target`
  - `WINDOW_TARGET` → `WIN32_WINDOW_TARGET`

- [x] Rename files that encode the old kind name.

## Task 2: Source-specific cleanup

- [x] Ensure frontend Inspector uses `node.kind === 'Win32WindowTarget'`.
- [x] Ensure auto-fix creates `Win32WindowTarget`.
- [x] Ensure tools capture API/event names use `Win32WindowTarget`.
- [x] Ensure MCP schema examples use `Win32WindowTarget`.
- [x] Ensure error code i18n uses `*_WIN32_WINDOW_TARGET`.
- [x] Regenerate `internal/catalog/node-i18n.json`.

## Task 3: Flightdeck truth

- [x] Add phase61 notes.
- [x] Update target-controller topic index.
- [x] Update `window-vs-target-boundary.md` to remove compatibility language.

## Task 4: Verification

- [x] Run:

```powershell
go test ./...
cd frontend
pnpm i18n:check
pnpm vue-tsc --noEmit
pnpm test
```

- [x] Search must return no old live code references:

```powershell
Select-String -Path <tracked source files> -Pattern '(?<!Win32)(?<!win32)WindowTarget|(?<!win32)windowTarget|(?<!win32)windowtarget|(?<!win32-)window-target|(?<!WIN32_)WINDOW_TARGET|(?<!win32_)window_target'
```

Flightdeck historical notes may still mention the old decision if explicitly marked superseded.

Completed verification:

- `go test ./...`
- `cd frontend && pnpm gen:node-i18n`
- `cd frontend && pnpm i18n:check`
- `cd frontend && pnpm vue-tsc --noEmit`
- `cd frontend && pnpm test`
- `git diff --check`
