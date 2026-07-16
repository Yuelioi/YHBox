---
topic: major-upgrade-review
title: Yotta 3.1 major upgrade
summary: Implement and validate the AI-native destructive Yotta 3.1 architecture and release program.
---

## State

目标：完成并验证 AI-native、destructive 的 Yotta 3.1 架构与发布计划。

当前 Slice：plugin-hosts-sdk-conformance。stable-code-names-explicit-versions 已由 022bc360 完成；现在实现已签名 Node Package 到精确 Catalog/Program lock、隔离 host、SDK 与共享 conformance 的唯一执行链。

当前阶段验收边界：plugin-hosts-sdk-conformance 全部实现后，统一执行 task check、跨平台 build、真实 Windows WebView/plugin smoke；子任务内只做必要的定向开发反馈。

## Next

先冻结 common plugin execution contract、Store runtime projection 与 Catalog merge；再依次实现 Process/Wasm host、生成 SDK/WIT/Proto、示例包和共享 conformance，最后一次性接入 composition 与批量验收。

## Read now

- work/major-upgrade-review/slices/plugin-hosts-sdk-conformance.md
- knowledge/architecture/node-package-manifest.md
- knowledge/agent/codex-working-agreement.md
- knowledge/build/code-style.md
- knowledge/coding/comments.md

## Read if

- work/major-upgrade-review/slices/map.md — 重排 plugin 子任务或选择最终 acceptance 时
- work/major-upgrade-review/design.md — 修改 Plugin Host deep-module seam 或 composition 时
- work/major-upgrade-review/plan.md — 修改 Wave 11 completion criterion 时
- work/major-upgrade-review/research/script-isolation-2026-07-16.md — 修改 Windows sandbox、runner protocol、取消或资源预算时
- work/major-upgrade-review/slices/node-package-signing-trust.md — 修改 signing/trust contract 时
- work/major-upgrade-review/slices/stable-code-names-explicit-versions.md — 回查 Node identity contract 时

## Progress

- Workflow 唯一执行链、桌面启动边界、WebView smoke 与可信 global quality baseline 已完成。
- AI prompt/tool provenance、bounded agent、offline eval gate 与 reviewed authoring 已完成。
- Node Package manifest/archive/lifecycle/signing/trust 已由 a8c0cfb5、ba2efb65、53e6d8a9、ab57d572 完成。
- 022bc360 恢复稳定代码命名，并把 Node identity 冻结为稳定 nodeTypeId + SemVer version + semanticDigest。
- plugin-hosts-sdk-conformance 已进入实现；当前先建立 Store → Catalog/Program lock → host adapter 的可信深模块边界。
- 阶段验收尚未执行；完成全部 plugin 子任务后统一运行。

## Open questions

Process 与 Wasm 必须共享同一 invocation/result/error/status/budget 语义，但 engine-specific ABI 与取消强度保持显式；不得为了表面统一而引入弱协议或 ambient authority。
