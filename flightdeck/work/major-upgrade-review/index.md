---
topic: major-upgrade-review
title: Yotta 3.1 major upgrade
summary: Implement and validate the AI-native destructive Yotta 3.1 architecture and release program.
---

## State

目标：完成并验证 AI-native、destructive 的 Yotta 3.1 架构与发布计划。

当前 Slice：ai-eval-upgrade-gate。d22b5bd5 已完成 bounded provider-native Agent runtime；现在建立版本化离线 suite、确定性 grader、regression report 与 installation approval gate。

## Next

盘点现有 ModelProfile evaluation metadata、settings/install admission 与仓库工具模式；冻结 EvalSuite/EvalReport artifact 及 approval identity，再实现最小离线 corpus、deterministic grader、CLI/Task 和 fail-closed installation gate。

## Read now

- work/major-upgrade-review/slices/ai-eval-upgrade-gate.md
- knowledge/agent/codex-working-agreement.md
- knowledge/architecture/provider-native-ai-installations.md
- knowledge/build/code-style.md
- knowledge/coding/comments.md

## Read if

- work/major-upgrade-review/slices/map.md — 选择下一 Slice、改变 blocker 或重排 frontier 时
- work/major-upgrade-review/research/ai-native-disposition-2026-07-17.md — 修改 AI remaining frontier 或核对处置证据时
- work/major-upgrade-review/ai-native-design.md — 修改 eval 产品/架构边界时
- work/major-upgrade-review/slices/ai-agent-budget-runtime.md — 修改 Agent/runtime 与 eval 的衔接边界时
- work/major-upgrade-review/slices/ai-authoring-review-trace.md — eval 完成后恢复 authoring review/trace 时
- work/major-upgrade-review/slices/node-package-signing-trust.md — AI frontier 暂停或完成后继续 Node Package trust 时
- work/major-upgrade-review/plan.md — 调整总体阶段或最终验收边界时
- work/major-upgrade-review/design.md — 修改 3.1 总体架构边界时

## Progress

- f3c83737、c3cab6e4、ab5b644f 分别恢复 Workflow 编辑交互、普通权限桌面启动与可重复 WebView 自调试 smoke。
- 27e01b17 恢复 global quality gate；b25a0c6c 完成 AI-native 设计处置。
- b674664c 完成 PromptManifest/ToolSet canonical provenance 与 trusted instruction boundary。
- d22b5bd5 完成 exact ToolSet、pure/capability authority、OpenAI/Anthropic native continuation、copy-on-write opaque state 与多维 RunBudget。
- Agent Node 只安装显式 pure text_length；Agent/Generate capability scope 隔离，模型不能扩张 ambient authority。
- Agent 批次 task check 全绿：global coverage 65.8%，internal/ai 74.0%，frontend 27 files / 103 tests，Wails 100 models。
- AI 当前 frontier 为 eval upgrade gate → authoring review/trace。
- Node Package signing trust 仍为独立 ready frontier；plugin hosts/SDK 在其后，最终 acceptance 等待所有实现 Slice。

## Open questions

Eval approval 应绑定 suite digest、candidate artifact identity、baseline identity、grader version 与阈值报告；需要确认现有 ModelProfile EvaluationSuite 字段能否直接承载 approved report，还是必须显式区分 suite/report digest。
