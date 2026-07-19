---
slice: "44"
title: 管理页壳层与计划编辑可靠性
status: in_progress
---

# Slice 44：管理页壳层与计划编辑可靠性

## Outcome / Question

让工作流、资源库、计划成为同一产品壳层下的高密度管理页面，把运行日志收回工作流运行工作台，并让计划创建/编辑恢复旧版熟悉的单栏表单节奏与可信保存语义。

## Completion criterion

- 工作流、资源库、计划使用同一 `workspace-page__*` 标题层级、图标标记与操作区契约；管理页标题下不再重复放说明文字，页面主体仍保留各自信息架构。
- 应用全局壳层不挂载 LogPanel；日志只在工作流编辑器运行工作台中按需加载，管理首页不再被运行期信息挤占。
- 新建计划和编辑 Pinia 列表中的既有计划都不会把 Vue Proxy 直接传入 `structuredClone`；草稿编辑保持响应式且可保存/取消。
- 计划编辑器为单栏分组表单，不再设置割裂的右侧行为预览；所有枚举控件走 `AdaptiveSelect`，显示默认值必须同步写入草稿，保存前给出就地校验。
- Proxy 草稿、日志归属和三页壳层契约有回归测试；阶段批量门禁、包体预算和 production build 通过。
- 用户用最新 production build 真机确认三页标题观感一致，计划新建/编辑不再黑屏。

## Verification

- `ScheduleEditorPanel.component.spec.ts` 直接传入 reactive Schedule，断言编辑器可挂载。
- `AppShell.spec.ts` 断言全局壳层不再导入/渲染 LogPanel，运行工作台仍保留日志页签。
- `ManagementPageShell.spec.ts` 断言三个管理页使用同一组 `workspace-page__*` 视觉契约。
- `ScheduleEditorPanel.component.spec.ts` 断言 interval 控件未发生用户输入时，界面默认值 `30` 仍会真实进入保存 payload。
- `task webview:smoke` 使用隔离 DEV host 与 loopback CDP，进入计划编辑器并生成 `schedules.png`；截图必须由执行者实际查看。`task dev` 支持 `Ctrl+Shift+I` DevTools，`task webview:screenshot` 可捕获正在运行的开发 WebView。
- `task check`：Go 全量门禁、覆盖率和架构契约通过；随后前端完整门禁 52 个测试文件、210 项测试、类型检查、i18n、lint 与 bundle budget 通过。
- `task build`：生成最新 `bin/Yotta.exe` 及辅助进程。

## Result

Implementation complete; awaiting user acceptance。App 全局日志面板已移除，编辑器日志改为打开日志页签时异步加载；工作流、资源库、计划标题统一视觉层级并移除标题下重复说明。计划编辑状态在 UI 边界解开外层 Proxy；编辑页按旧版产品节奏重构为居中的单栏分组表单，删除右侧行为预览，枚举统一使用 `AdaptiveSelect`。修复了 interval `30` 只显示但未写入数据导致后端收到 `0` 的根因，并把 daily/interval 等默认值初始化为真实草稿值、增加就地校验。WebView 调试能力不再藏在单一脚本：开发任务固定 loopback CDP、`Ctrl+Shift+I` 打开 DevTools、可单独截图当前页面，并将计划编辑器加入隔离 smoke 截图；执行者已实际查看最新 `schedules.png`，确认单栏布局、对齐和滚动正常。production build 继续禁用 DevTools/CDP。日志归属变化把原本由入口承担的 UI 共享依赖计入 editor closure，因此硬上限按实际归属从 200 KB 校准为 220 KB，125 KB 长期目标不变；当前 editor 为 211.1 KB。完整 `task check`、WebView smoke 与最新 production `task build` 通过，等待用户用真实数据完成计划交互接受。
