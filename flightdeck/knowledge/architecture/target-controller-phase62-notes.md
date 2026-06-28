# Target Controller Upgrade — Phase 62 Notes

SUMMARY: 用全集节点 guard 固化 Target/Window 分层，防止 Android/Browser 误用 Win32 窗口能力
READ WHEN: 新增 target selection 节点、窗口操作节点、NeedsWindow/NeedsForeground 节点，或调整 palette category 时
RECHECK WHEN: 引入 `NeedsTarget`、新增非 Win32 controller、把窗口操作迁到跨平台 target 能力时

---

## Completed

- `internal/nodes/all` 增加全集注册节点 guard。
- Guard 固化以下边界：
  - `Target` category 只允许 target selection nodes。
  - `Window` category 只允许明确列出的 Win32 HWND operation nodes。
  - target selection nodes 不得声明 `NeedsWindow` / `NeedsForeground`。
  - Android/Browser target 不得暴露 `Window` pin。
  - `NeedsForeground` 必须 imply `NeedsWindow`。

## Boundary

这刀不改变 runtime 行为，只把 Phase59-61 的命名/分类决策变成测试契约。

新增 Windows 窗口控制节点时，需要更新 Window allow-list。新增 Android/Browser/其他 target selection 节点时，需要更新 Target allow-list；这是一种有意的 friction，目的是强迫作者明确它属于 target selection 还是 Win32 HWND operation。

## Verification

- `go test ./internal/nodes/all ./internal/nodes/system ./internal/services/container/runtime`
- `git diff --check`
