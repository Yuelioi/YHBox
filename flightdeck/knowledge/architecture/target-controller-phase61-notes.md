# Target Controller Upgrade — Phase 61 Notes

SUMMARY: 破坏性移除旧 WindowTarget contract，统一 Windows HWND target selection 为 Win32WindowTarget
READ WHEN: 改 Win32WindowTarget / validator 缺目标错误 / 窗口捕获 API / MCP schema / 录制前置窗口解析 / Target palette 时
RECHECK WHEN: 引入 NeedsTarget、重命名 target kind、恢复旧容器 loader、改前端目标捕获事件或错误码时

---

## Completed

- 全仓统一 Windows 窗口目标 kind 为 `Win32WindowTarget`。
- 删除旧 contract 入口：
  - `WindowTarget`
  - `windowTarget`
  - `windowtarget`
  - `window-target`
  - `WINDOW_TARGET`
- 文件名同步改为 `win32_window_target*`。
- 前端自动修复、Inspector、事件订阅、toast/i18n key 改为 Win32WindowTarget 语义。
- 后端 validator、runtime、tools、MCP、recording、catalog、测试数据统一使用 `Win32WindowTarget` / `MISSING_WIN32_WINDOW_TARGET`。
- `window-vs-target-boundary.md` 移除 alias/兼容策略；Phase59/60 的兼容建议已标记 superseded。

## Boundary

这是破坏性更新。项目未上线，所以不保留旧容器兼容：

- 不注册旧 `WindowTarget` kind。
- 不在 loader 里把旧 kind 自动迁移成 `Win32WindowTarget`。
- 不保留旧 `windowtarget:captured` 事件。
- 不保留旧 `StartWindowTargetCapture` / `CancelWindowTargetCapture` API。
- 不保留旧 `MISSING_WINDOW_TARGET` / `INVALID_WINDOW_TARGET_*` 错误码。

用户可见文案仍写“Windows 窗口目标 / Windows window target”。代码与 contract 写 `Win32WindowTarget`。

## Follow-up

- 下一步继续做 `NeedsTarget(kind=win32-window, capabilities=...)`，不要回到旧 `WindowTarget` alias。
- 如果以后确实需要导入旧实验容器，应写一次性迁移工具，而不是把兼容层塞回 runtime/validator。

## Verification

- `go test ./...`
- `cd frontend && pnpm gen:node-i18n`
- `cd frontend && pnpm i18n:check`
- `cd frontend && pnpm vue-tsc --noEmit`
- `cd frontend && pnpm test`
- old live contract search:
  - `(?<!Win32)(?<!win32)WindowTarget`
  - `(?<!win32)windowTarget`
  - `(?<!win32)windowtarget`
  - `(?<!win32-)window-target`
  - `(?<!WIN32_)WINDOW_TARGET`
  - `(?<!win32_)window_target`
- `git diff --check`
