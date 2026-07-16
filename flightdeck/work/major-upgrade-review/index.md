---
topic: major-upgrade-review
title: Yotta 3.1 major upgrade
summary: Implement and validate the AI-native destructive Yotta 3.1 architecture and release program.
---

## State

目标：完成并验证 AI-native、destructive 的 Yotta 3.1 架构与发布计划。

当前 Slice：stable-code-names-explicit-versions。ab57d572 已完成 Node Package signing/trust 实现；现在删除由产品 release 派生的结构性 31 命名，并把 Node 版本收回显式 contract identity。

当前阶段验收边界：stable-code-names-explicit-versions 与 plugin-hosts-sdk-conformance 全部实现后，统一执行 task check、跨平台 build、真实 Windows WebView/plugin smoke；Slice 内只做必要的定向开发反馈。

## Next

生成 nodes31/nodes31runtime/workflow31/node31 全仓 impact map，冻结产品 release、artifact format generation 与 Node entity version 三层 taxonomy；先改 NodeRef/schema/hash identity，再执行稳定职责名迁移。

## Read now

- work/major-upgrade-review/slices/stable-code-names-explicit-versions.md
- knowledge/agent/codex-working-agreement.md
- knowledge/architecture/content-addressed-workflow-artifacts.md
- knowledge/build/code-style.md
- knowledge/coding/comments.md

## Read if

- work/major-upgrade-review/slices/map.md — 选择下一 Slice、改变 blocker 或重排 frontier 时
- work/major-upgrade-review/slices/node-package-signing-trust.md — 修改刚完成的 signing/trust contract 时
- work/major-upgrade-review/plan.md — 修正会把产品 release 推导为代码代际的总体表述时
- work/major-upgrade-review/design.md — 修改 contract generation 或最终 acceptance 边界时
- knowledge/architecture/node-package-manifest.md — 命名恢复触及 package-owned Node identity 时

## Progress

- Workflow 唯一执行链、桌面启动边界、WebView smoke 与可信 global quality baseline 已完成。
- AI prompt/tool provenance、bounded agent、offline eval gate 与 reviewed authoring 已由 b674664c、d22b5bd5、cfa12703、c71cc19f 完成。
- ab57d572 建立 Ed25519 signature envelope、explicit publisher namespace authority、monotonic trust policy 与 registry v2。
- Node Package install/open/update/revocation/quarantine/rollback/reopen 均从同一 registry authority fail closed；local exact-digest approval 不再构成 Store admission。
- 用户确认产品 release 3.1 不应进入 Go/TS package、目录、文件或 service 名；污染最早由 64e371ed 引入并扩散。
- stable-code-names-explicit-versions 为当前恢复任务；plugin hosts/SDK 紧随其后。
- 当前扩展平台阶段完成后才做一次批量 task check、跨平台 build、真实 Windows smoke 与验收。

## Open questions

Node entity version 的新显式字段、稳定 nodeTypeId 形态和 semantic digest preimage 必须一起冻结；不得保留 /vN URI 尾段与字段双重事实，也不得用兼容 alias 保留 nodes31 import path。
