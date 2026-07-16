---
topic: major-upgrade-review
title: Yotta 3.1 major upgrade
summary: Implement and validate the AI-native destructive Yotta 3.1 architecture and release program.
---

## State

目标：完成并验证 AI-native、destructive 的 Yotta 3.1 架构与发布计划。

当前 Slice：node-package-signing-trust。c71cc19f 已完成 reviewed AI authoring；现在建立内容寻址签名 envelope、publisher key/namespace authority，以及可持久化且 fail-closed 的 revocation/quarantine。

## Next

盘点现有 Node Package manifest、archive verifier、local immutable generation/store 与 registry authority seam；读取 manifest/resource-broker threat knowledge，冻结 canonical signing preimage、publisher namespace ownership 和 trust-state reopen/rollback contract 后实现。

## Read now

- work/major-upgrade-review/slices/node-package-signing-trust.md
- knowledge/agent/codex-working-agreement.md
- knowledge/architecture/node-package-manifest.md
- knowledge/architecture/resource-broker-open-revocation.md
- knowledge/build/code-style.md
- knowledge/coding/comments.md

## Read if

- work/major-upgrade-review/slices/map.md — 选择下一 Slice、改变 blocker 或重排 frontier 时
- work/major-upgrade-review/research/ai-native-disposition-2026-07-17.md — 修改 AI remaining frontier 或核对处置证据时
- work/major-upgrade-review/ai-native-design.md — 修改 remaining frontier 或最终 acceptance 边界时
- work/major-upgrade-review/slices/ai-authoring-review-trace.md — 修改 package-owned trusted prompt 的 authoring lineage 时
- work/major-upgrade-review/plan.md — 调整总体阶段或最终验收边界时
- work/major-upgrade-review/design.md — 修改 3.1 总体架构边界时

## Progress

- f3c83737、c3cab6e4、ab5b644f 分别恢复 Workflow 编辑交互、普通权限桌面启动与可重复 WebView 自调试 smoke。
- 27e01b17 恢复 global quality gate；b25a0c6c 完成 AI-native 设计处置。
- b674664c 完成 PromptManifest/ToolSet canonical provenance 与 trusted instruction boundary。
- d22b5bd5 完成 exact ToolSet、pure/capability authority、OpenAI/Anthropic native continuation、copy-on-write opaque state 与多维 RunBudget。
- cfa12703 完成 strict EvalSuite/EvalReport、8-case deterministic corpus、exact upgrade candidate、CLI/Task gate 与 evaluation apply/revoke/downgrade。
- c71cc19f 完成 bounded AI authoring：pure typed tools 生成 opaque PreparedPatch，review 后只提交 exact candidate；revision/hash 漂移 fail stale。
- AuthoringReview 展示 normalized changes、diagnostics、capability/credential/target delta、risk、usage 与脱敏 trace；敏感输入只记录 trust class/digest/size。
- 编辑器 AI review panel 覆盖 save-first、accept/reject/retry/audit/stale 与权限扩大确认；未经接受不产生 durable mutation。
- authoring 后 exact eval candidate 纳入 Authoring PromptManifest/ToolSet；tracked corpus 更新为 8/8、safety 0。
- 2026-07-17 Authoring 批次 task check 全绿：global coverage 65.8%，internal/ai 74.1%，internal/aiauthoring 62.5%，frontend 28 files / 106 tests，Wails 14 services / 95 methods / 109 models。
- 真实 Windows Wails/WebView smoke 全绿并验证 AI review panel；截图位于 ignored .task artifact。
- AI implementation frontier 已完成；Node Package signing trust 为当前 frontier，plugin hosts/SDK 在其后，最终 acceptance 等待所有实现 Slice。

## Open questions

Node Package 的 local exact-digest approval 只能授权该 artifact，不能推导 publisher namespace ownership。当前 Slice 必须明确 trust root/key distribution、namespace 冲突、撤销/隔离持久化，以及 rollback/reopen 的 fail-closed 语义。
