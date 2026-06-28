# Phase 62 — Target/Window Contract Guards Implementation Plan

> **For agentic workers:** Execute directly; this is a narrow guard-only slice after Phase61.

**Goal:** 把 “Window = Win32 HWND 操作，Target = 自动化对象选择” 固化成注册节点级别测试，避免 Android/Browser target 或未来 target 节点误用 `WindowService` / `NeedsWindow` / `NeedsForeground`。

**Architecture:** 不改 runtime 行为。只在 `internal/nodes/all` 增加全集注册节点 guard：

- `Category=="Target"` 只允许 target selection nodes。
- `Win32WindowTarget` / `AndroidTarget` / `BrowserTarget` 必须在 Target 分组。
- target selection nodes 不得声明 `NeedsWindow` / `NeedsForeground`。
- Android/Browser target 不得暴露 `Window` input/output data。
- `Category=="Window"` 只允许明确列出的 Win32 HWND 操作节点。
- `NeedsForeground` 必须 imply `NeedsWindow`，因为 foreground 是 Win32 SendInput 语义。

**Files:**

- Add/modify: `internal/nodes/all/doc_test.go`
- Add: `flightdeck/knowledge/architecture/target-controller-phase62-notes.md`
- Modify: `flightdeck/work/target-controller-upgrade/index.md`

## Tasks

- [x] Add all-node target/window boundary guard.
- [x] Run focused Go tests.
- [x] Update Flightdeck notes/index.

## Verification

- [x] `go test ./internal/nodes/all ./internal/nodes/system ./internal/services/container/runtime`
- [x] `git diff --check`
