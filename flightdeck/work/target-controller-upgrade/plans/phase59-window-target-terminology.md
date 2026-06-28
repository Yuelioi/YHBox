# Phase 59 — Window / Target Terminology Boundary Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把用户可见的“窗口”和“目标”语义拆清楚：Win32 窗口能力继续叫 Window，Android/Browser 等自动化对象统一叫 Target。

**Architecture:** 保留现有序列化 kind `WindowTarget` 作为兼容入口，不直接破坏旧容器、MCP 示例和 validator。先改 UI/i18n/docs，把 `WindowTarget` 显示为 Windows 窗口目标；后续再加 `Win32WindowTarget` alias/migration 和 `NeedsTarget(capability)`。

**Tech Stack:** Go node spec/runtime, Vue frontend i18n, Flightdeck knowledge.

---

## Boundary Decision

- `WindowService` / `WindowHandle` / `WindowInputSpec` 表示 **Win32 HWND 窗口**。
- `TargetService` / `target.Target` 表示 **自动化目标**，包括 `win32-window`、`android-adb`、`browser-cdp`。
- `BringWindowForeground`、`WindowState`、`MoveResizeWindow`、`CloseWindow`、`GetWindow` 是 Windows 窗口能力，不应对 Android/Browser 复用同一含义。
- `WindowTarget` 当前继续作为旧容器兼容 kind；UI 展示应叫“Windows 窗口目标 / Windows window target”。
- 通用文案里的“目标窗口”要改为“当前目标 / 自动化目标”；只有 Win32 能力文案才写“Windows 窗口”。

## Files

- Modify: `frontend/src/i18n/zh.ts`
  - 把用户可见 `WindowTarget` label/description、输入类描述、错误提示改为 Win32-specific。
- Modify: `frontend/src/i18n/en.ts`
  - 英文保持 parity。
- Modify: `internal/node/spec.go`
  - 注释中明确 `NeedsWindow` 是 legacy Win32 HWND requirement，不代表 Android/Browser target。
- Modify: `internal/nodes/input/bring_foreground.go`
  - 注释明确该节点是 Win32-only foreground action。
- Create: `flightdeck/knowledge/architecture/window-vs-target-boundary.md`
  - 记录后续命名迁移约束。
- Modify: `flightdeck/work/target-controller-upgrade/index.md`
  - 加入 phase59 入口和完成记录。

## Task 1: Frontend copy boundary

- [x] **Step 1: Update Chinese i18n copy**

Edit `frontend/src/i18n/zh.ts`:

```ts
WindowTarget: {
  label: 'Windows 窗口目标',
  description: '指定接下来操作哪个 Windows 窗口...',
}
```

Also update:

- `toast.window_target_added_*`
- handbook input/system group descriptions
- `BringWindowForeground`
- `ClickAt` / `InputText`
- window control node descriptions
- validation/error messages that mention recording or missing WindowTarget
- container input backend hint

- [x] **Step 2: Update English i18n copy**

Mirror the same meaning in `frontend/src/i18n/en.ts`, using `Windows window target` for the node display name.

- [x] **Step 3: Verify i18n**

Run:

```powershell
cd frontend
pnpm i18n:check
```

Expected: parity, compile, residue, and refs all pass.

## Task 2: Backend comment boundary

- [x] **Step 1: Clarify `NeedsWindow` comments**

Edit `internal/node/spec.go` comments so future node authors do not treat `NeedsWindow` as a generic target flag.

- [x] **Step 2: Clarify `BringWindowForeground` comments**

Edit `internal/nodes/input/bring_foreground.go` comments so it is plainly a Win32 HWND foreground operation.

- [x] **Step 3: Verify Go tests**

Run:

```powershell
go test ./internal/node ./internal/nodes/input
```

Expected: pass.

## Task 3: Flightdeck knowledge

- [x] **Step 1: Add architecture knowledge**

Create `flightdeck/knowledge/architecture/window-vs-target-boundary.md` with:

- summary and read/recheck routing headers
- the compatibility decision
- forbidden future drift
- migration path from `WindowTarget` to `Win32WindowTarget` alias

- [x] **Step 2: Update topic index**

Update `flightdeck/work/target-controller-upgrade/index.md`:

- add `plans/phase59-window-target-terminology.md` to Read now
- add the new knowledge note to Read if
- add a Phase 59 progress bullet

## Task 4: Final verification

- [x] **Step 1: Run scoped checks**

Run:

```powershell
go test ./internal/node ./internal/nodes/input
cd frontend
pnpm i18n:check
```

- [x] **Step 2: Check worktree**

Run:

```powershell
git status --short
```

Expected: only the intentional phase59 files plus any pre-existing external changes, notably the current `go.mod` direct/indirect dependency movement.

## Follow-up Plan

Phase 60 should add a compatibility alias without breaking old JSON:

- new display-facing kind alias `Win32WindowTarget`
- migration loader maps old `WindowTarget` to canonical internal target kind
- validator reports Win32 target requirements as `NeedsTarget(kind=win32-window)` while still accepting old `WindowTarget`
- UI can later hide raw `WindowTarget` from new palettes if the alias is stable
