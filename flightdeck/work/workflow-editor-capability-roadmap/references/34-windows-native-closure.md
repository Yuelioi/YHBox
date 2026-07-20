# Slice 34：Windows 自动化真实纵向闭环

## Outcome / Question

在重建后的外围边界上完成 Windows application/window、UAC、F9、键鼠、窗口操作、截图、模板和录制回放的真实用户旅程。

## Completion criterion

- automation target UI 一行一职责，内容不被压缩；F9 临时全局注册并可靠释放。
- exact/regex、多窗口、动态标题、删除/重建和同进程 activation 可用。
- Press Keys/Type Text/click/move/drag/held input 与窗口操作真实生效。
- screenshot、template wait/click、simple/precise clip playback 真实生效。
- 普通、管理员、UnrealWindow/异环和多窗口应用有证据；unsupported 边界明确。
- clean data 与当前 workspace 都能启动和恢复，不要求删除整个 data。

## Blocked by

Slices 30–33。

## Verification

- G02–G08、G11、G13、G15 native smoke。
- 阶段末统一 Windows 聚合测试、`task check`、production build、manifest 检查和人工视觉验收。
- 通过后形成单一 Stage R2 commit，不为旅程内每个小修复单独跑全门禁。

## Out of scope

- 不规避 secure desktop、UIPI 或第三方反作弊。
- 不恢复 asInvoker/按需 runas 双权限历史。
- 不为旧未发布 workspace schema 添加迁移层。

## Result

Completed（2026-07-18）。

- adapter/runtime：held key/button 改为 Run-owned deep lease，并提供显式 Release Held Input；窗口 activate/close/move-resize/state/wait-present/wait-gone 共享 exact installation lease；`unique` 不再缓存陈旧 HWND。
- Win32：SendInput drag 全程使用 native injection；所有原语检查注入数量；`BringToFront` 固定 goroutine OS thread 并成对 AttachThreadInput。
- workspace：坏 Source 隔离到 store-owned recovery，支持修复/删除；Program 作为派生缓存遇到 catalog/compiler 漂移会丢弃重编；stale consent 仅撤销授权，不丢安装。
- native gate：`task windows:smoke:automation` 通过 exact 尾空格、regex 动态标题、多窗口 ambiguity、键盘/文本/点击/移动/相对移动/滚轮/原生 drag、held cleanup、playback、窗口状态/关闭、PNG capture、F9 生命周期和真实 recorder→codec→asset→reload。
- Unreal evidence：隔离 3.1 profile 精确绑定 `HTGame.exe / UnrealWindow / 异环··`，现有 `Run Start → ESC` Source 成功；Run `019f7556-279d-711a-9b98-db9bd616bf94`，record `sha256:a3dfebe52e35404c3afa73e9f14633cbe06a24e2070fd39ae52e3d6541f289f6`。
- UI/G13/G15：真实 Wails WebView 旅程隔离损坏 Source、执行 launcher workflow、隐藏后复用同一 target，并覆盖编辑器/调试/子图/AI/资源库；人工查看四张 PNG 后修复 selection toolbar 窄画布压缩。
- stage gate：`task check`（237.1s）、`task build`、相关 Linux/Darwin cross-compile 与 `requireAdministrator` manifest 检查通过；production SHA-256 `7652263517690B0A527DAE2F40810E456FB97AF60BA09B79A75E49536FAB136D`。

不恢复旧扁平 target schema；secure desktop、UIPI 更高完整性与反作弊仍按明确 unsupported/failure 边界处理。