---
topic: major-upgrade-review
title: Yotta 3.1 major upgrade
summary: Implement and validate the AI-native destructive Yotta 3.1 architecture and release program.
---

## State

目标：完成并验证 AI-native、destructive 的 Yotta 3.1 架构与发布计划。

当前 Slice：ai-agent-budget-runtime。b674664c 已完成 PromptManifest/ToolSet trusted boundary；现在建立受 exact ToolSet、approval/capability 与多维 RunBudget 约束的 provider-native Agent tool loop。

## Next

冻结 provider-neutral AgentTurn、ToolCall、RunBudget 与 ToolExecutor contract；核对现有 provider Outcome/tool-call 表达和 runtime capability seam，再实现内置 Agent Node Contract 与 bounded continuation loop。

## Read now

- work/major-upgrade-review/slices/ai-agent-budget-runtime.md
- knowledge/agent/codex-working-agreement.md
- knowledge/architecture/provider-native-ai-installations.md
- knowledge/build/code-style.md
- knowledge/coding/comments.md

## Read if

- work/major-upgrade-review/slices/map.md — 选择下一 Slice、改变 blocker 或重排 frontier 时
- work/major-upgrade-review/research/ai-native-disposition-2026-07-17.md — 修改 AI remaining frontier 或核对处置证据时
- work/major-upgrade-review/ai-native-design.md — 修改 AI 产品/架构边界时
- work/major-upgrade-review/slices/ai-prompt-tool-provenance.md — 修改 PromptManifest/ToolSet trust seam 时
- work/major-upgrade-review/slices/node-package-signing-trust.md — AI frontier 暂停或完成后继续 Node Package trust 时
- work/major-upgrade-review/plan.md — 调整总体阶段或最终验收边界时
- work/major-upgrade-review/design.md — 修改 3.1 总体架构边界时

## Progress

- f3c83737、c3cab6e4、ab5b644f 分别恢复 Workflow 编辑交互、普通权限桌面启动与可重复 WebView 自调试 smoke。
- frontend check 为 27 files / 103 tests 全绿；隔离 production EXE 无 UAC 冷启动并由 PrintWindow 确认首屏正常。
- 27e01b17 通过 3.1 contract/compiler/service/authoring、Windows adapter/input/recording、Wails composition 与 vision 回归恢复 global quality gate。
- b25a0c6c 逐项处置 AI-native 设计：provider-native、slot/profile、OS credential、strict Extract 与 typed MCP 已完成。
- b674664c 完成 PromptManifest/ToolSet/rendered prompt canonical provenance，删除 workflow 任意 `instructions` override，并把 exact prompt/schema/toolset/model lineage 接入 provider/runtime。
- 2026-07-17 provenance 批次两次 `task check` 全绿，最终 global coverage 65.4%，frontend 27 files / 103 tests。
- AI 后续 frontier 为 Agent budget runtime → eval upgrade gate → authoring review/trace。
- Node Package signing trust 仍为独立 ready frontier；plugin hosts/SDK 在其后，最终 acceptance 等待所有实现 Slice。

## Open questions

首个 Agent ToolSet 应只包含显式注册、无 ambient authority 的宿主工具；任何 filesystem/network/process/window effect 都必须走现有 capability/approval seam，不能因模型请求而扩大 Run Grant。
