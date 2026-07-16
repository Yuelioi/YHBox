---
topic: major-upgrade-review
title: Yotta 3.1 major upgrade
summary: Implement and validate the AI-native destructive Yotta 3.1 architecture and release program.
---

## State

目标：完成并验证 AI-native、destructive 的 Yotta 3.1 架构与发布计划。

当前 Slice：final-contract-and-release-acceptance。全部实现 Slice 已完成；现在进入唯一的总审计阶段，不再新增未经 completion criterion 证明的功能。

本阶段要明确区分：3.1 工程完成、可构建发布候选、公开 stable 发布。许可证替换、签名证书、真实维护者权限和 owner 级仓库设置是公开发布外部前置，不得伪装成已完成，也不得阻止对工程完成度作出准确结论。

## Next

对照 plan、design、Slice registry 与仓库现状完成 contract/reference/golden、架构/规范 review、跨平台 CI 与 Windows 真实运行证据审计；集中修复结论中的工程缺口，阶段末只运行一次最终批量门禁并给出 major upgrade completion verdict。

## Read now

- work/major-upgrade-review/slices/final-contract-and-release-acceptance.md
- work/major-upgrade-review/plan.md
- work/major-upgrade-review/design.md
- knowledge/agent/codex-working-agreement.md
- knowledge/build/build.md

## Read if

- work/major-upgrade-review/slices/map.md — 回查完整 frontier
- work/major-upgrade-review/review.md — 对照原始审查问题
- work/major-upgrade-review/research/oss-governance.md — 审计公开发布、许可证、签名与治理
- knowledge/architecture/node-package-manifest.md — 审计 package/plugin lock
- work/major-upgrade-review/slices/plugin-hosts-sdk-conformance.md — 回查插件阶段证据

## Progress

- Workflow 唯一执行链、桌面启动边界、WebView smoke 与可信 global quality baseline 已完成。
- AI prompt/tool provenance、bounded agent、offline eval gate 与 reviewed authoring 已完成。
- Node Package manifest/archive/lifecycle/signing/trust 已由 a8c0cfb5、ba2efb65、53e6d8a9、ab57d572 完成。
- 022bc360 恢复稳定代码命名，并冻结稳定 nodeTypeId + SemVer version + semanticDigest。
- 310d8afd、613bc654、1483e908、623ebd44、b9871cf3 完成 runtime projection、strict protocol、Process/Wasm host、SDK、示例、conformance 与 composition。
- 625a1326 完成插件阶段集中修复与批量验收：task check 全绿（Go global 65.0%、frontend 28 files/106 tests）、Windows production build、真实 Process/Wasm plugin smoke、Linux/macOS portable core build、WebView smoke（100 catalog nodes、2 canvas nodes、AI review）及截图人工检查通过。
- Linux/macOS production GUI 的权威结果仍由现有原生 CI gui-build matrix 提供；Windows 本地不能冒充两个原生宿主。

## Open questions

- 公开 stable 发布仍受 LICENSE、签名证书、canonical public repository/identity、真实维护者权限与 owner 级设置阻塞；最终 Slice 必须把这些外部前置与工程缺口分开列明。
