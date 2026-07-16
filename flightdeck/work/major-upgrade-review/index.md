---
topic: major-upgrade-review
title: Yotta 3.1 major upgrade
summary: Implement and validate the AI-native destructive Yotta 3.1 architecture and release program.
---

## State

目标：完成并验证 AI-native、destructive 的 Yotta 3.1 架构与发布计划。

当前 Slice：plugin-hosts-sdk-conformance。实现批次已由 310d8afd、613bc654、1483e908、623ebd44、b9871cf3 完成；现在只做阶段批量验收与集中修复。

当前阶段验收边界：统一执行 task check、跨平台 core/GUI build、真实 Windows Process/Wasm plugin smoke 与 WebView smoke；不得把单项通过拆成独立验收。

## Next

执行完整阶段验收矩阵，集中修复所有失败项并整体复验；通过后完成 plugin-hosts-sdk-conformance handoff，再选择 major upgrade 的下一未完成 Slice。

## Read now

- work/major-upgrade-review/slices/plugin-hosts-sdk-conformance.md
- knowledge/architecture/node-package-manifest.md
- knowledge/agent/codex-working-agreement.md
- knowledge/build/build.md

## Read if

- work/major-upgrade-review/slices/map.md — plugin 阶段验收通过后选择下一 Slice 时
- work/major-upgrade-review/design.md — 验收暴露 Plugin Host seam 或 composition 问题时
- work/major-upgrade-review/plan.md — 修改 Wave 11 completion criterion 时
- work/major-upgrade-review/research/script-isolation-2026-07-16.md — 验收暴露 sandbox、取消或资源预算问题时
- work/major-upgrade-review/slices/node-package-signing-trust.md — 验收暴露 signing/trust contract 问题时
- work/major-upgrade-review/slices/stable-code-names-explicit-versions.md — 回查 Node identity contract 时

## Progress

- Workflow 唯一执行链、桌面启动边界、WebView smoke 与可信 global quality baseline 已完成。
- AI prompt/tool provenance、bounded agent、offline eval gate 与 reviewed authoring 已完成。
- Node Package manifest/archive/lifecycle/signing/trust 已由 a8c0cfb5、ba2efb65、53e6d8a9、ab57d572 完成。
- 022bc360 恢复稳定代码命名，并冻结稳定 nodeTypeId + SemVer version + semanticDigest。
- 310d8afd 完成 enabled/trusted runtime projection 与 Catalog merge。
- 613bc654、1483e908 完成 strict Protobuf、Process host 与可复用 LPAC/AppContainer + Job sandbox。
- 623ebd44 完成无 WASI、受内存/deadline 约束且位于隔离 runner 进程内的 Wasm host。
- b9871cf3 完成 SDK/WIT/Proto generator、示例、共享 conformance、签名 fixtures、composition 与 Windows 双链路 smoke 入口。
- 阶段验收正在执行；实现期定向测试不计作阶段通过。

## Open questions

无设计阻塞。验收若暴露问题，按根因归属集中回到相应实现批次修复，不拆成重复小验收。
