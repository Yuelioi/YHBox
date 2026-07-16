---
topic: major-upgrade-review
title: Yotta 3.1 major upgrade
summary: Implement and validate the AI-native destructive Yotta 3.1 architecture and release program.
---

## State

目标：完成并验证 AI-native、destructive 的 Yotta 3.1 架构与发布计划。

当前 Slice：node-package-signing-trust。签名/trust 实现仍是当前未提交批次；用户指出的结构性 release-version 命名错误已拆成下一 Slice，不与当前供应链改动混写。

## Next

完成 Node Package signing/trust 的 threat-matrix tests、staticcheck/race/cross-platform 验证和文档同步；独立提交后切换到 stable-code-names-explicit-versions。

## Read now

- work/major-upgrade-review/slices/node-package-signing-trust.md
- knowledge/agent/codex-working-agreement.md
- knowledge/architecture/node-package-manifest.md
- knowledge/architecture/resource-broker-open-revocation.md
- knowledge/build/code-style.md
- knowledge/coding/comments.md

## Read if

- work/major-upgrade-review/slices/map.md — 选择下一 Slice、改变 blocker 或重排 frontier 时
- work/major-upgrade-review/slices/stable-code-names-explicit-versions.md — 当前 signing/trust 完成后进入下一任务时
- work/major-upgrade-review/research/ai-native-disposition-2026-07-17.md — 修改 AI remaining frontier 或核对处置证据时
- work/major-upgrade-review/ai-native-design.md — 修改 remaining frontier 或最终 acceptance 边界时
- work/major-upgrade-review/plan.md — 调整总体阶段或最终验收边界时
- work/major-upgrade-review/design.md — 修改 3.1 总体架构边界时

## Progress

- Workflow 3.1 唯一执行链、桌面启动边界、WebView smoke 与 global quality gate 已完成。
- AI prompt/tool provenance、bounded agent、offline eval gate 与 reviewed authoring 已分别由 b674664c、d22b5bd5、cfa12703、c71cc19f 完成。
- Authoring 批次 task check 与真实 Windows Wails/WebView smoke 全绿；AI implementation frontier 已关闭。
- Node Package signing/trust 已形成未提交实现：Ed25519 envelope、canonical trust policy、registry v2、签名安装及 revocation/quarantine/rollback/reopen tests；package test/race 曾通过，最终 staticcheck/full gate 尚待中断后重跑。
- 2026-07-17 用户确认产品 release 3.1 不应进入 Go/TS package、目录、文件或 service 名；污染最早由 64e371ed 引入并扩散到 nodes31、nodes31runtime、workflow31 与 node31.ts。
- stable-code-names-explicit-versions 已登记为 signing/trust 后的下一 Slice：稳定语义代码名，版本只进入显式 contract/manifest/schema/identity 属性。
- plugin hosts/SDK 排在命名恢复之后；最终 acceptance 等待全部实现 Slice。

## Open questions

当前 signing/trust 仍需确认本地 trust root 的产品入口和 key rotation UX；实现层必须保持 namespace authority、revocation/quarantine 与 registry commit 同一权威边界。
