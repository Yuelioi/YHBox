---
topic: workflow-editor-ui-polish
title: Workflow editor UI polish
summary: 收口工作流编辑器暗色图控、节点端口间距、离开确认弹窗和反馈层级，按阶段批量验收。
---

## State

四项编辑器 UI 回归已修复并完成阶段验收。Vue Flow Controls/Minimap 使用 semantic dark surface；节点 Handle 位于卡片边界且包围盒不再与标签相交；WorkflowEditorView 与 AIWorkflowReviewPanel 均接入共享 useConfirm/ConfirmDialog，全仓 frontend/src 直接 window.alert/confirm/prompt 为 0；编辑器保存/编译改为按钮原地短反馈，运行/调试/AI 接受依靠现有时间线或面板状态，错误 toast 保留。

## Next

完成 Flightdeck Topic，创建本地逻辑提交并向用户交付修改与验收证据。

## Read now

- knowledge/agent/codex-working-agreement.md
- knowledge/build/code-style.md
- knowledge/frontend/ui.md
- knowledge/build/build.md
- knowledge/frontend/headless-ui-verify.md

## Read if

- knowledge/frontend/nuxt-ui-icon-button-alignment.md — 若编辑器图控或工具按钮涉及 UButton 图标对齐
- knowledge/wails/wails-dev-fetch-transport-flattens-error.md — 若错误 toast 依赖 Wails dev fetch transport 的错误形态

## Progress

- 真机红灯基线同时复现图控浅色、8 个 Handle 重叠、原生 confirm、共享确认框缺失、保存成功 toast。
- Controls 按钮、Minimap 表面/节点/遮罩全部改用 Nuxt UI semantic tokens。
- Handle 从端口行内缩位置移到卡片边界，data 与 signal 形状分别保留正确 transform。
- 两处 window.confirm 改为共享 Nuxt UI ConfirmDialog；静态审计直接浏览器 dialog API 为 0。
- 保存与编译按钮原地显示约 1.6 秒成功状态；编辑器成功 toast 清零，失败继续 toast。
- WebView smoke 新增图控背景、Handle 包围盒、原生 confirm、共享 modal、保存反馈断言，并修复 PowerShell 吞子进程退出码的假绿。
- task check 全绿：global coverage 65.0%，28 files / 106 Vitest，typecheck/i18n/bundle 全通过。
- 真机 WebView smoke 全绿，截图 .task/workflow-editor-smoke/20260717-092950/workflow-editor.png 已目检通过。

## Open questions

- 无。
