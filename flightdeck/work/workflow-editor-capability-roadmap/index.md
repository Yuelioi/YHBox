---
topic: workflow-editor-capability-roadmap
title: 工作流编辑器能力审计与升级路线
summary: 审计旧编辑器能力与 3.1 现状，按架构适配、用户价值和必要性决定恢复、重做、延期或删除，并分阶段恢复可靠的图编辑、运行认知与自动化创作能力。
---

## State

Stage 1 进行中。Slice 1 已完成并用真实 Vue Flow store 回归测试锁定；当前 Slice：类型感知的连线创作。

完整证据与决策见 [capability-audit.md](capability-audit.md)，外部交互调研见 [research.md](research.md)。

## Next

执行 [02-connection-authoring.md](02-connection-authoring.md)：从 EditorSession 当前连接校验提取单一兼容性判断，接入 handle hover、拖线落空候选菜单和原子“创建并连线”。

## Read now

- knowledge/architecture/feature-continuity-across-product-stack.md
- knowledge/frontend/ui.md
- knowledge/frontend/vue-flow-store-vmodel-shallow-sync.md
- knowledge/subgraph/autolayout-skips-subgraph-virtual-markers.md
- knowledge/nodes/debug-step-region-runner.md
- knowledge/frontend/display-preferences-must-not-gate-capabilities.md
- work/workflow-editor-capability-roadmap/capability-audit.md
- work/workflow-editor-capability-roadmap/research.md
- work/workflow-editor-capability-roadmap/02-connection-authoring.md

## Read if

- knowledge/build/code-style.md — 开始修改或验证产品代码
- knowledge/build/build.md — 到达阶段末批量验收或需要 Windows GUI smoke
- knowledge/frontend/vue-flow-delete-key-code-ignores-modifiers.md — 修改删除键或画布键盘行为
- knowledge/nodes/add-node.md — 开始恢复模板自动化节点
- work/workflow-editor-capability-roadmap/03-selection-layout.md — 开始多选与布局
- work/workflow-editor-capability-roadmap/04-diagnostics-run-trace.md — 开始运行认知
- work/workflow-editor-capability-roadmap/05-true-debugger.md — 开始真调试器
- work/workflow-editor-capability-roadmap/06-template-convenience-nodes.md — 开始模板复合节点
- work/workflow-editor-capability-roadmap/07-asset-authoring-integration.md — 开始资源预览与录制联动

## Progress

- Slice 1 根因已证实：外部 computed nodes 把 selected 与持久位置绑定，selection/source 刷新会调用 Vue Flow setNodes 并用旧坐标覆盖内部实时位置。
- 新增手势位置 overlay；拖拽期间 Source 刷新仍保留 event.node.position，结束时只提交一次 move-node。
- 节点拖拽面收窄到 header，正文和端口附近不再因默认 1px threshold 触发微移动。
- 真实 Vue Flow store 红灯连续三次复现旧回跳，修复后 selection/source-refresh 两个回归用例均通过；frontend typecheck 通过。
- 旧能力审计与三阶段七 Slice 路线保持不变；Stage 1 完成前不运行全量验收。
- 当前进入 Slice 2，恢复兼容节点提示、拖线落空创建和自动连线。

## Open questions

- 真调试器的输入输出和状态快照需要怎样的大小上限与脱敏策略？
- 断点只存本机编辑偏好，还是进入独立可共享调试配置；不得混入 Workflow Source。
- WaitTemplate、WaitTemplateGone、ClickTemplate 的交付顺序需用真实工作流使用频率确认。
