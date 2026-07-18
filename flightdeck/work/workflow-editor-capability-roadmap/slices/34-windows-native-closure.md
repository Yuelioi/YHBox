---
slice: "34"
title: Windows 自动化真实纵向闭环
status: pending
---

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

Pending。Slice 20/26 的实现和 smoke 记录由本 Slice 重新验收后关闭。
