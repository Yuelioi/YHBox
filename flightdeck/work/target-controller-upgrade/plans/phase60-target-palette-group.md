# Phase 60 — Target Palette Group Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把目标选择节点从 `Window` palette 分组拆到 `Target` 分组，避免 Android/Browser 被误认为 Windows 窗口函数。

**Architecture:** 不改节点 kind、不改旧容器 JSON、不改 runtime 路由，只改 node spec category 和前端 category 映射。Win32 窗口控制节点继续留 `Window`；`WindowTarget` 作为 Win32 target selection node 进入 `Target`。

**Tech Stack:** Go node specs, Vue node registry adapter, frontend i18n, Flightdeck knowledge.

---

## Files

- Modify: `internal/nodes/system/window_target.go`
- Modify: `internal/nodes/system/android_target.go`
- Modify: `internal/nodes/system/browser_target.go`
- Create: `internal/nodes/system/target_category_test.go`
- Modify: `frontend/src/components/containers/nodeRegistry/index.ts`
- Modify: `frontend/src/components/containers/nodeRegistry/adapter.ts`
- Modify: `frontend/src/components/containers/visualRegistry.ts`
- Modify: `frontend/src/composables/editor/useNodeGroupColor.ts`
- Modify: `frontend/src/i18n/zh.ts`
- Modify: `frontend/src/i18n/en.ts`
- Modify: `flightdeck/work/target-controller-upgrade/index.md`
- Create: `flightdeck/knowledge/architecture/target-controller-phase60-notes.md`

## Task 1: Backend category

- [x] **Step 1: Move target selection specs to `Target`**

Change these specs:

```go
WindowTarget.Spec().Category = "Target"
AndroidTarget.Spec().Category = "Target"
BrowserTarget.Spec().Category = "Target"
```

- [x] **Step 2: Add guard test**

Create `internal/nodes/system/target_category_test.go` asserting those three specs use `Target`.

- [x] **Step 3: Run scoped Go tests**

Run:

```powershell
go test ./internal/nodes/system ./internal/catalog
```

Expected: pass.

## Task 2: Frontend group support

- [x] **Step 1: Add `target` NodeGroup**

Add `target` to:

- `NodeGroup` union
- `GROUP_MAP`
- `ALL_NODE_GROUPS`
- `GROUP_I18N_KEY`
- `GROUP_VISUAL`

- [x] **Step 2: Add i18n labels**

Add `nodeGroup.target` in Chinese and English.

- [x] **Step 3: Run frontend checks**

Run:

```powershell
cd frontend
pnpm i18n:check
pnpm vue-tsc --noEmit
```

Expected: pass.

## Task 3: Flightdeck recovery

- [x] **Step 1: Add phase notes**

Create `target-controller-phase60-notes.md`.

- [x] **Step 2: Update topic index**

Add plan and notes to `target-controller-upgrade/index.md`.
