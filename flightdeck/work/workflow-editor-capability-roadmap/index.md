---
topic: workflow-editor-capability-roadmap
title: 工作流编辑器能力审计与升级路线
summary: 审计旧编辑器能力与 3.1 现状，按架构适配、用户价值和必要性决定恢复、重做、延期或删除，并分阶段恢复可靠的图编辑、运行认知与自动化创作能力。
---

## State

Stage 1 进行中。Slice 1–2 已完成；当前 Slice：多选、批量编辑、对齐、分布与自动布局。

完整证据与决策见 [capability-audit.md](capability-audit.md)，外部交互调研见 [research.md](research.md)。

## Next

执行 [03-selection-layout.md](03-selection-layout.md)：建立多选事实与原子批量命令，先完成批量移动/Delete/clipboard，再接入对齐、等距分布和带 revision guard 的 ELK LR/TB 布局。

## Read now

- knowledge/architecture/feature-continuity-across-product-stack.md
- knowledge/frontend/ui.md
- knowledge/frontend/vue-flow-store-vmodel-shallow-sync.md
- knowledge/subgraph/autolayout-skips-subgraph-virtual-markers.md
- knowledge/nodes/debug-step-region-runner.md
- knowledge/frontend/display-preferences-must-not-gate-capabilities.md
- work/workflow-editor-capability-roadmap/capability-audit.md
- work/workflow-editor-capability-roadmap/research.md
- work/workflow-editor-capability-roadmap/03-selection-layout.md

## Read if

- knowledge/build/code-style.md — 开始修改或验证产品代码
- knowledge/build/build.md — 到达阶段末批量验收或需要 Windows GUI smoke
- knowledge/frontend/vue-flow-delete-key-code-ignores-modifiers.md — 修改删除键或画布键盘行为
- knowledge/nodes/add-node.md — 开始恢复模板自动化节点
- work/workflow-editor-capability-roadmap/04-diagnostics-run-trace.md — 开始运行认知
- work/workflow-editor-capability-roadmap/05-true-debugger.md — 开始真调试器
- work/workflow-editor-capability-roadmap/06-template-convenience-nodes.md — 开始模板复合节点
- work/workflow-editor-capability-roadmap/07-asset-authoring-integration.md — 开始资源预览与录制联动

## Progress

- Slice 1 已修复 selection/source 刷新覆盖 Vue Flow 实时位置，并把节点拖拽面收窄到 header。
- Slice 2 新增单一连接兼容性服务，候选、hover 和最终 connect 共用 data/exec/error、类型、carrier、instruction 与资源租约规则。
- 拖线落空会打开坐标内联菜单；多兼容端口显式列出，选择后新增节点+连线是一个 undo，失败不留孤儿节点。
- UI 已修正 error 边 target handle 被错误映射为 exec 的问题；“显示全部”只添加不兼容节点，不伪造连线。
- Slice 2 定向验证：4 个测试文件 18 项通过，typecheck、oxlint 和 i18n parity/compile/residue/refs 通过。
- 当前进入 Slice 3；Stage 1 三个 Slice 完成后才执行 task check、Windows build 与 GUI smoke。

## Open questions

- 真调试器的输入输出和状态快照需要怎样的大小上限与脱敏策略？
- 断点只存本机编辑偏好，还是进入独立可共享调试配置；不得混入 Workflow Source。
- WaitTemplate、WaitTemplateGone、ClickTemplate 的交付顺序需用真实工作流使用频率确认。
