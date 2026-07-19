---
topic: v3.1-product-optimization
title: 3.1 产品创作体验与运行工作台优化
summary: 在稳定的 3.1 架构上修正编辑器状态表达，恢复专业级多选与子图创作，重新判定调试能力，并重构复杂节点的可理解交互。
---

## State

Stage I complete。四个 Slice 均已实现并通过阶段聚合验收：紧凑节点投影、工作流内 Macro 编辑与三路退出、Tab 快速添加与 Snippet 快捷键、计划 Modal。

## Next

等待下一轮真机反馈或新的 Stage 范围；不要把已完成的 Stage I 再拆回零散补丁。

## Read now

- knowledge/build/build.md
- work/v3.1-product-optimization/slices/14-node-density-and-optional-pins.md
- work/v3.1-product-optimization/slices/15-workflow-resource-edit-and-safe-exit.md
- work/v3.1-product-optimization/slices/16-tab-menu-and-snippet-shortcuts.md
- work/v3.1-product-optimization/slices/17-schedule-modal-flow.md
- work/v3.1-product-optimization/context/editor-discovery-and-modal-decisions.md
- knowledge/frontend/canvas-node-authoring-boundary.md
- knowledge/frontend/ui.md

## Read if

- work/v3.1-product-optimization/slices/map.md — 检查 Stage I 全部 Slice 与依赖
- work/v3.1-product-optimization/slices/14-node-density-and-optional-pins.md — 已完成节点低密度投影
- work/v3.1-product-optimization/slices/15-workflow-resource-edit-and-safe-exit.md — 已完成编辑器内宏编辑与保存退出
- work/v3.1-product-optimization/slices/16-tab-menu-and-snippet-shortcuts.md — 已完成 Tab 快速添加与 Snippet 快捷键
- work/v3.1-product-optimization/slices/17-schedule-modal-flow.md — 已完成计划 Modal
- work/v3.1-product-optimization/context/current-vs-3.0-editor-audit.md — 复查 3.0 能力连续性
- knowledge/build/code-style.md — 修改 Go/TypeScript/Vue
- knowledge/architecture/feature-continuity-across-product-stack.md — Stage I 黄金旅程验收

## Progress

- Stage A–H 已恢复专业画布、Source-native 子图、Macro/InputClip、资源工作区、真实调试、typed Authoring Surface、黄金旅程、Snippets 与节点上下文菜单。
- 源码确认点击模板的 timeout、poll interval、settle duration 属于 common 调优参数，却因类型级 inlinePriority 同时进入节点卡片。
- 3.0 已存在 Houdini 式 Tab Explorer 和 Snippet 快捷键；3.1 的缺失属于能力回归，不需要复制旧 runtime/store。
- 成熟编辑器对照支持“上下文 Tab 搜索 + 分类 + 键盘”和“参数独立面板”；饼菜单只适合少量固定熟练动作，不用于可增长节点全集。
- 工作流宏编辑器与计划编辑器均已有可复用实现；缺口是入口与容器形态，不另造第二套数据模型。
- Slice 14 已固定节点宽度、折叠非关键可选输入，并将多个 common inline candidate 退回 Inspector。
- Slice 15 已恢复工作流内 Macro 编辑，并将脏工作流退出改为取消、放弃、保存并退出三路决策。
- Slice 16 已恢复画布 Tab 分类/搜索快速添加和持久 Snippet 快捷键，服务端拒绝保留与冲突组合。
- Slice 17 已将计划创建/编辑改为覆盖列表上下文的 BaseModal，保存仍通过唯一 ScheduleStore。
- Stage I WebView smoke `20260720-043007` 已真实覆盖 Tab 搜索/添加、Snippet 快捷键保存与调用、节点上下文菜单、资源工作区、子图、调试和计划 Modal；无 WebView JS 错误。
- Stage I 最终 `task check` 通过：全仓 Go 测试与静态检查、全局覆盖率 65.6%、前端测试/契约/生产构建全部通过，编辑器 gzip 219922/220000 bytes。

## Open questions

- 无。饼菜单明确不纳入本阶段。
