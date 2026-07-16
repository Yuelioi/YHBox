---
topic: major-upgrade-review
title: Yotta 3.1 major upgrade
summary: Implement and validate the AI-native destructive Yotta 3.1 architecture and release program.
---

## State

目标：完成并验证 AI-native、destructive 的 Yotta 3.1 架构与发布计划。

当前 Slice：ai-authoring-review-trace。cfa12703 已完成 offline eval/upgrade gate；现在建立只经 typed Application commands 的 AI authoring loop、review artifact、permission delta 与 redacted provenance trace。

## Next

盘点现有 MCP/Application typed authoring commands、revision/compile/preview 返回、permission proposal 与编辑器 review seam；冻结 AuthoringReview/RedactedTrace artifact 后实现 bounded AI tool loop 与 accept/reject UI。

## Read now

- work/major-upgrade-review/slices/ai-authoring-review-trace.md
- knowledge/agent/codex-working-agreement.md
- knowledge/architecture/provider-native-ai-installations.md
- knowledge/build/code-style.md
- knowledge/coding/comments.md

## Read if

- work/major-upgrade-review/slices/map.md — 选择下一 Slice、改变 blocker 或重排 frontier 时
- work/major-upgrade-review/research/ai-native-disposition-2026-07-17.md — 修改 AI remaining frontier 或核对处置证据时
- work/major-upgrade-review/ai-native-design.md — 修改 eval 产品/架构边界时
- work/major-upgrade-review/slices/ai-agent-budget-runtime.md — 修改 Agent/runtime 与 eval 的衔接边界时
- work/major-upgrade-review/slices/ai-eval-upgrade-gate.md — 修改 eval/report 与 authoring lineage 衔接时
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
- mandatory corpus 覆盖中英 authoring、catalog、minimal patch、diagnostic repair、strict extraction、injection 与 unauthorized capability；canonical report 为 8/8、safety 0。
- Eval candidate 绑定 model subject、三个 PromptManifest、Agent ToolSet 与三个 AI Node Contract semantic digest；report digest 进入 ModelProfile/consent lineage。
- unverified/rejected/stale candidate 不发布到 Host Profile；profile edit 自动撤销 evaluation/consent，Apply/RevokeEvaluation 提供显式导入与撤销。
- 2026-07-17 Eval 批次 task check 全绿：global coverage 65.9%，internal/ai 74.1%，frontend 27/103，Wails 91 methods / 101 models。
- cfa12703 完成 strict EvalSuite/EvalReport、8-case deterministic corpus、exact upgrade candidate、CLI/Task gate 与 evaluation apply/revoke/downgrade。
- AI 当前 frontier 为 authoring review/trace。
- Node Package signing trust 仍为独立 ready frontier；plugin hosts/SDK 在其后，最终 acceptance 等待所有实现 Slice。

## Open questions

Authoring review/trace 必须复用现有 typed MCP patch 与 compiler diagnostics；AI 不得直接写 Source，且 review payload/trace 只能记录 canonical diff、permission delta 与脱敏 lineage。
