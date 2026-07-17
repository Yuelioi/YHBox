---
topic: workflow-editor-capability-roadmap
title: 工作流编辑器能力审计与升级路线
summary: 审计旧编辑器能力与 3.1 现状，按架构适配、用户价值和必要性决定恢复、重做、延期或删除，并分阶段恢复可靠的图编辑、运行认知与自动化创作能力。
---

## State

规划与代码审计已完成。当前 Slice：Stage 1 / Slice 1，先解决单击、选中和连线手势可能导致节点位置跑偏的问题，再恢复连线引导与布局效率。

完整证据与决策见 [capability-audit.md](capability-audit.md)，外部交互调研见 [research.md](research.md)。

## Next

执行 [01-interaction-correctness.md](01-interaction-correctness.md)：建立手势复现矩阵与开发期事件追踪，证实位置漂移链路后实现稳定的 Vue Flow / EditorSession 位置同步边界。

## Read now

- knowledge/architecture/feature-continuity-across-product-stack.md
- knowledge/frontend/ui.md
- knowledge/frontend/vue-flow-store-vmodel-shallow-sync.md
- knowledge/subgraph/autolayout-skips-subgraph-virtual-markers.md
- knowledge/nodes/debug-step-region-runner.md
- knowledge/frontend/display-preferences-must-not-gate-capabilities.md
- work/workflow-editor-capability-roadmap/capability-audit.md
- work/workflow-editor-capability-roadmap/research.md
- work/workflow-editor-capability-roadmap/01-interaction-correctness.md

## Read if

- knowledge/build/code-style.md — 开始修改或验证产品代码
- knowledge/build/build.md — 到达阶段末批量验收或需要 Windows GUI smoke
- knowledge/frontend/vue-flow-delete-key-code-ignores-modifiers.md — 修改删除键或画布键盘行为
- knowledge/nodes/add-node.md — 开始恢复模板自动化节点
- work/workflow-editor-capability-roadmap/02-connection-authoring.md — Slice 1 完成
- work/workflow-editor-capability-roadmap/03-selection-layout.md — 开始多选与布局
- work/workflow-editor-capability-roadmap/04-diagnostics-run-trace.md — 开始运行认知
- work/workflow-editor-capability-roadmap/05-true-debugger.md — 开始真调试器
- work/workflow-editor-capability-roadmap/06-template-convenience-nodes.md — 开始模板复合节点
- work/workflow-editor-capability-roadmap/07-asset-authoring-integration.md — 开始资源预览与录制联动

## Progress

- 定位到 9fce7870：旧 Container 产品栈被整体移除，连线候选、布局、吸附、上下文菜单和调试面板随之删除；未发现逐项产品废弃决策。
- 当前 3.1 已恢复起始节点、目录搜索、删除、状态、精确目标、资源与录制入口，但高级图编辑尚未迁移完整。
- 当前“Debug”仍调用普通 startRun，只增加前端 debugging 标志和时间线；它不是暂停、单步、断点或 watches。
- 节点跑偏最高概率风险是 Vue Flow 内部手势态与外部 computed nodes 重建之间的浅同步竞态；需 Slice 1 用事件与坐标证据确认。
- 决策按恢复并适配、重新设计、暂缓、明确不恢复分类；不引入 Container 双栈或第二调试运行时。
- 路线分三个阶段、七个相邻 Slices；阶段内最小定向检查，阶段完成后统一批量验收。

## Open questions

- 真调试器的输入输出和状态快照需要怎样的大小上限与脱敏策略？
- 断点只存本机编辑偏好，还是进入独立可共享调试配置；不得混入 Workflow Source。
- WaitTemplate、WaitTemplateGone、ClickTemplate 的交付顺序需用真实工作流使用频率确认。
