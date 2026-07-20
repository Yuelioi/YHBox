# 3.1 产品创作体验与运行工作台优化

## Goal

在稳定的 3.1 架构上恢复专业工作流创作能力，修正编辑器状态表达、节点发现、资源编辑和运行工作台体验。

## Status

Open

## Current

Stage A–I 已完成。最近的 Stage I 交付包括紧凑节点投影、工作流内 Macro 编辑与三路安全退出、
Tab 分类/搜索快速添加、持久 Snippet 快捷键，以及保留列表上下文的计划 Modal。真实 WebView
旅程无 JS 错误，最终 `task check` 通过；当前没有尚未实现的已批准 Stage。

## Next

收到下一轮真机反馈或新的产品范围后，先复现具体用户旅程并把相邻问题组合成一个新 Stage 写入
`plan.md`；不要把已经验收的 Stage I 重新拆成零散补丁。若没有新反馈，只报告当前等待状态。

## Progress

- 恢复专业画布多选、Source-native 子图、Macro/InputClip、资源工作区和真实调试链路。
- 建立 typed Authoring Surface、复杂 Editor Adapter、视觉分析配方和黄金用户旅程。
- 统一画布相机 ownership，复杂节点只保留少量高频摘要，完整配置回到 Inspector。
- 删除硬编码 Recipes，恢复 durable Snippets、节点上下文菜单和模板创作闭环。
- Stage I 恢复 Tab 快速添加、Snippet 快捷键、Macro 原地编辑和三路安全退出。
- 计划创建与编辑改用共享 Modal，继续由唯一 ScheduleStore 持有事实。
- WebView smoke 与 `task check` 全绿，编辑器 gzip 为 219922/220000 bytes。

## References

- [3.0/3.1 editor audit](references/current-vs-3.0-editor-audit.md) — 能力连续性和不恢复旧 runtime 的依据。
- [Editor discovery and modal decisions](references/editor-discovery-and-modal-decisions.md) — Stage I 研究与取舍。
- [Node density and optional pins](references/14-node-density-and-optional-pins.md) — 紧凑节点投影证据。
- [Workflow resource editing](references/15-workflow-resource-edit-and-safe-exit.md) — Macro 编辑与退出语义。
- [Tab and Snippet flow](references/16-tab-menu-and-snippet-shortcuts.md) — 快速添加与快捷键实现边界。
- [Schedule modal flow](references/17-schedule-modal-flow.md) — 计划 Modal 的验收记录。
- [Canvas authoring boundary](../../knowledge/frontend/canvas-node-authoring-boundary.md) — 当前画布创作规则。
- [Build and acceptance](../../knowledge/build/build.md) — 完整门禁的触发条件。
