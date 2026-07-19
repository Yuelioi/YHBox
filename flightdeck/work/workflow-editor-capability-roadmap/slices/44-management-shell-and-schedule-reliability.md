---
slice: "44"
title: 管理页壳层与计划编辑可靠性
status: in_progress
---

# Slice 44：管理页壳层与计划编辑可靠性

## Outcome / Question

让工作流、资源库、计划成为同一产品壳层下的高密度管理页面，把运行日志收回工作流运行工作台，并修复计划创建/编辑因 Vue Proxy 穿越 `structuredClone` 边界导致的黑屏。

## Completion criterion

- 工作流、资源库、计划使用同一 `workspace-page__*` 标题层级、图标标记、说明与操作区契约；页面主体仍保留各自信息架构。
- 应用全局壳层不挂载 LogPanel；日志只在工作流编辑器运行工作台中按需加载，管理首页不再被运行期信息挤占。
- 新建计划和编辑 Pinia 列表中的既有计划都不会把 Vue Proxy 直接传入 `structuredClone`；草稿编辑保持响应式且可保存/取消。
- Proxy 草稿、日志归属和三页壳层契约有回归测试；阶段批量门禁、包体预算和 production build 通过。
- 用户用最新 production build 真机确认三页标题观感一致，计划新建/编辑不再黑屏。

## Verification

- `ScheduleEditorPanel.component.spec.ts` 直接传入 reactive Schedule，断言编辑器可挂载。
- `AppShell.spec.ts` 断言全局壳层不再导入/渲染 LogPanel，运行工作台仍保留日志页签。
- `ManagementPageShell.spec.ts` 断言三个管理页使用同一组 `workspace-page__*` 视觉契约。
- `task check`：52 个前端测试文件、209 项测试通过，Go 全量门禁与 bundle budget 通过。
- `task build`：生成最新 `bin/Yotta.exe` 及辅助进程。

## Result

Implementation complete; awaiting user acceptance。App 全局日志面板已移除，编辑器日志改为打开日志页签时异步加载；工作流、资源库、计划标题统一到计划页的视觉层级。计划编辑状态改用 `shallowRef`，从 store 或 props 克隆前在 UI 边界解开外层 Proxy，创建和编辑两条路径均由回归覆盖。日志归属变化把原本由入口承担的 UI 共享依赖计入 editor closure，因此硬上限按实际归属从 200 KB 校准为 220 KB，125 KB 长期目标不变；当前 editor 为 210.8 KB。完整 `task check` 与 `task build` 通过，等待真实 WebView 视觉与计划交互接受。
