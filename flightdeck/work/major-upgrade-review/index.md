---
topic: major-upgrade-review
title: Yotta 3.1 major upgrade
summary: Implement and validate the AI-native destructive Yotta 3.1 architecture and release program.
---

## State

目标：完成并验证 AI-native、destructive 的 Yotta 3.1 架构与发布计划。

当前 Slice：ai-prompt-tool-provenance。b25a0c6c 已完成 AI-native 设计处置；下一步先建立 content-addressed PromptManifest/ToolSet 与 trusted instruction boundary，作为 Agent、eval 和 authoring review 的共同前置。

## Next

设计并实现 PromptManifest、ToolSet 与 rendered instruction artifacts；删除 AI 节点任意 `instructions` config，确保只有可信 manifest 能进入 OpenAI instructions / Anthropic system。

## Read now

- work/major-upgrade-review/slices/ai-prompt-tool-provenance.md
- knowledge/agent/codex-working-agreement.md
- knowledge/architecture/provider-native-ai-installations.md
- knowledge/build/code-style.md
- knowledge/coding/comments.md

## Read if

- work/major-upgrade-review/slices/map.md — 选择下一 Slice、改变 blocker 或重排 frontier 时
- work/major-upgrade-review/research/ai-native-disposition-2026-07-17.md — 修改 AI remaining frontier 或核对处置证据时
- work/major-upgrade-review/ai-native-design.md — 修改 AI 产品/架构边界时
- work/major-upgrade-review/slices/node-package-signing-trust.md — Go quality gate 恢复后继续 Node Package trust 时
- work/major-upgrade-review/plan.md — 调整总体阶段或最终验收边界时
- work/major-upgrade-review/design.md — 修改 3.1 总体架构边界时

## Progress

- f3c83737、c3cab6e4、ab5b644f 分别恢复 Workflow 编辑交互、普通权限桌面启动与可重复 WebView 自调试 smoke。
- frontend check 为 27 files / 103 tests 全绿；隔离 production EXE 无 UAC 冷启动并由 PrintWindow 确认首屏正常。
- 27e01b17 通过 3.1 contract/compiler/service/authoring、Windows adapter/input/recording、Wails composition 与 vision 回归恢复 global 65.2% quality gate。
- b25a0c6c 逐项处置 AI-native 设计：provider-native、slot/profile、OS credential、strict Extract 与 typed MCP 已完成。
- AI 真实剩余 frontier 是 Prompt/Tool provenance → Agent budget 与 eval gate → authoring review/trace；Session 与固定包目录不阻断 3.1。
- Node Package signing trust 仍为独立 ready frontier；plugin hosts/SDK 在其后，最终 acceptance 等待所有实现 Slice。

## Open questions

PromptManifest 是否只由内置 build artifact 提供，还是同时允许经过 Node Package signing trust 验证的 package-owned manifest，需在当前 Slice 保持 namespace/owner seam 可扩展但不提前实现第三方 prompt。