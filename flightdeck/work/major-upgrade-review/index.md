---
topic: major-upgrade-review
title: Yotta 3.1 major upgrade
summary: Implement and validate the AI-native destructive Yotta 3.1 architecture and release program.
---

## State

目标：完成并验证 AI-native、destructive 的 Yotta 3.1 架构与发布计划。

当前 Slice：stable-code-names-explicit-versions 已由 022bc360 完成；下一步按 slice map 恢复 plugin-hosts-sdk-conformance。

当前阶段验收边界：stable-code-names-explicit-versions 与 plugin-hosts-sdk-conformance 全部实现后，统一执行 task check、跨平台 build、真实 Windows WebView/plugin smoke；Slice 内只做必要的定向开发反馈。

## Next

读取 slices/map.md 与 plugin hosts/SDK/conformance 任务设计，确认当前缺口后直接实现插件 host、SDK 与 conformance，不提前运行阶段门禁。

## Read now

- work/major-upgrade-review/slices/map.md
- knowledge/agent/codex-working-agreement.md
- knowledge/build/code-style.md
- knowledge/coding/comments.md

## Read if

- work/major-upgrade-review/slices/stable-code-names-explicit-versions.md — 回查刚完成的稳定命名与 Node identity contract 时
- work/major-upgrade-review/slices/node-package-signing-trust.md — 修改 signing/trust contract 时
- work/major-upgrade-review/plan.md — 重排 major-upgrade frontier 时
- work/major-upgrade-review/design.md — 修改 plugin host 或最终 acceptance 边界时
- knowledge/architecture/node-package-manifest.md — 修改 package-owned Node identity 或 payload contract 时

## Progress

- Workflow 唯一执行链、桌面启动边界、WebView smoke 与可信 global quality baseline 已完成。
- AI prompt/tool provenance、bounded agent、offline eval gate 与 reviewed authoring 已由 b674664c、d22b5bd5、cfa12703、c71cc19f 完成。
- ab57d572 建立 Ed25519 signature envelope、explicit publisher namespace authority、monotonic trust policy 与 registry v2。
- 022bc360 删除 nodes31、nodes31runtime、workflow31、node31 与 app31/run31 等结构性 release 后缀；稳定职责名为 nodes、noderuntime、workflow 等。
- Node identity 现在由稳定 nodeTypeId、必填 SemVer version 与 semanticDigest 构成；/vN 不再编码 Node entity version，3.1 只保留为 artifact format generation。
- Go 全仓 compile-only、Wails binding generation、前端 typecheck 与 Node contract/package/catalog/schema 定向测试已作为开发反馈通过；尚未执行阶段验收。
- plugin hosts/SDK 紧随其后；完成后统一运行 task check、跨平台 build、真实 Windows smoke 与验收。

## Open questions

稳定命名与 Node version taxonomy 已冻结，无遗留开放问题。下一 Slice 的具体边界以 slices/map.md 和对应设计为准。
