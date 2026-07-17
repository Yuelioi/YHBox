---
topic: workflow-editor-capability-roadmap
title: 工作流编辑器能力审计与升级路线
summary: 审计旧编辑器能力与 3.1 现状，按架构适配、用户价值和必要性决定恢复、重做、延期或删除，并分阶段恢复可靠的图编辑、运行认知与自动化创作能力。
---

## State

Stage 1 三个实现 Slice 已完成，正在执行阶段级批量验收：聚合测试、task check、Windows build 与真实 GUI 手势/视觉 smoke。

完整证据与决策见 [capability-audit.md](capability-audit.md)，外部交互调研见 [research.md](research.md)。

## Next

完成 Stage 1 acceptance gate：运行 task check 和 task build，启动最新 bin/Yotta.exe 检查点击/连线不漂移、拖线候选、多选批量编辑、吸附/对齐/分布与 LR/TB 自动布局。

## Read now

- knowledge/architecture/feature-continuity-across-product-stack.md
- knowledge/frontend/ui.md
- knowledge/frontend/vue-flow-store-vmodel-shallow-sync.md
- knowledge/subgraph/autolayout-skips-subgraph-virtual-markers.md
- knowledge/nodes/debug-step-region-runner.md
- knowledge/frontend/display-preferences-must-not-gate-capabilities.md
- knowledge/build/build.md
- work/workflow-editor-capability-roadmap/capability-audit.md
- work/workflow-editor-capability-roadmap/research.md
- work/workflow-editor-capability-roadmap/03-selection-layout.md

## Read if

- knowledge/build/code-style.md — 开始修改或验证产品代码
- knowledge/frontend/vue-flow-delete-key-code-ignores-modifiers.md — 修改删除键或画布键盘行为
- knowledge/nodes/add-node.md — 开始恢复模板自动化节点
- work/workflow-editor-capability-roadmap/04-diagnostics-run-trace.md — Stage 1 验收完成，开始运行认知
- work/workflow-editor-capability-roadmap/05-true-debugger.md — 开始真调试器
- work/workflow-editor-capability-roadmap/06-template-convenience-nodes.md — 开始模板复合节点
- work/workflow-editor-capability-roadmap/07-asset-authoring-integration.md — 开始资源预览与录制联动

## Progress

- Slice 1 已修复 selection/source 刷新覆盖 Vue Flow 实时位置，加入 live gesture overlay，并把拖拽面收窄到 header。
- Slice 2 已恢复共用权威规则的连接 hover、拖线落空候选、按端口选择和原子新增+连线。
- Slice 3 已恢复多选/框选、批量移动/Delete、copy/cut/paste/duplicate、六向对齐、双向等距和上下文工具条。
- 拖拽吸附使用实际节点尺寸与 flow/screen 坐标转换，Alt 临时关闭；批量拖拽作为一个 move-nodes 历史条目。
- ELK LR/TB 异步布局保持图中心，以发起时 Source 身份阻止过期结果写回；整个布局是一个 undo。
- Stage 1 定向验证已通过：5 个测试文件 24 项、typecheck、定向 oxlint、i18n parity/compile/residue/refs；当前开始一次性完整验收。

## Open questions

- 真调试器的输入输出和状态快照需要怎样的大小上限与脱敏策略？
- 断点只存本机编辑偏好，还是进入独立可共享调试配置；不得混入 Workflow Source。
- WaitTemplate、WaitTemplateGone、ClickTemplate 的交付顺序需用真实工作流使用频率确认。
