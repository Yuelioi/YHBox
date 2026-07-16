---
topic: major-upgrade-review
title: Yotta 3.1 major upgrade
summary: Implement and validate the AI-native destructive Yotta 3.1 architecture and release program.
---

## State

目标：完成并验证 AI-native、destructive 的 Yotta 3.1 架构与发布计划。

结论：3.1 major upgrade engineering complete。全部 implementation Slice 与最终 contract/release acceptance 已完成；仓库内工程缺口已收口，唯一 runtime/application path、3.1 contract、AI authoring、Node Package/Process/Wasm host、GUI/headless seam 与冻结候选发布链均有最终证据。

当前可生成并 smoke 的是 unsigned engineering candidate，不是公开 stable。当前 LICENSE 仍是 source-available；许可证、签名证书/timestamp、canonical public repository/identity、真实多维护者权限、owner settings 与原生 Linux/macOS host/installer smoke 是公开 stable 的外部前置。

## Next

本 Topic 工程目标已完成，可归档。若继续公开 stable 发布，应在获得权利人、证书、公开仓库和维护者权限后新建独立 release/governance Topic，不再把外部前置伪装成 3.1 代码迁移。

## Read now

- work/major-upgrade-review/slices/final-contract-and-release-acceptance.md
- knowledge/agent/codex-working-agreement.md
- knowledge/build/build.md

## Read if

- work/major-upgrade-review/plan.md — 回查完整 major upgrade 定义
- work/major-upgrade-review/design.md — 回查目标架构
- work/major-upgrade-review/research/oss-governance.md — 准备公开 stable
- knowledge/architecture/node-package-manifest.md — 修改 package/plugin contract
- work/major-upgrade-review/slices/map.md — 回查全部 Slice frontier

## Progress

- Workflow、GUI/AI/MCP/schedule/debug 统一到 compiler/Program/Run Application 路径；旧 Container runtime、双执行路径和 release-number package/type name 已删除。
- stable nodeTypeId + SemVer version + semanticDigest、Catalog/Program/package locks 与 Go/TS/schema/reference/golden 由 drift gate 统一。
- AI prompt/tool provenance、bounded review state、offline eval、reviewed authoring、secure credential seam 已完成。
- Node Package manifest/archive/lifecycle/signing/trust、Process/Wasm host、SDK/example/conformance 已完成。
- dfdb8501 至 b5c0f893 完成最终审计修复：稳定 contract 名称、desktop composition 深模块、constructor-complete 装配、headless CLI、AI review TTL/capacity、明文 API key DTO 清除、文档/compatibility、冻结 candidate smoke 与发布链。
- 最终 `task package` 全绿：Go global 65.0%、根包 75.0%、CLI 65.3%，vet/staticcheck，frontend 28 files/106 tests，i18n 1269，Wails 14 services/94 methods/109 models，production bundle entry 262837/editor 97277 gzip bytes。
- 冻结 candidate manifest、Yotta.exe、Yotta.CLI.exe、ScriptWorker、WasmPluginRunner、capture DLL、ADB staging/archive 与 frozen-payload smoke 全绿；Process/Wasm 插件真实 Windows isolation chain 通过。
- Windows race group 与 4 组 10s fuzz 全绿；31 个 portable-core 包分别成功编译为 Linux amd64 与 Darwin arm64 测试二进制。
- WebView smoke 通过：100 catalog nodes、2 canvas nodes、AI review panel 可达，无 JS error/rejection/console.error；截图 `.task/workflow-editor-smoke/20260717-075351/workflow-editor.png` 已人工确认非黑屏且布局可用。
- 原生 Linux/macOS production GUI 与 portable-core 运行结果仍以 CI 原生 runner 为权威；Windows cross-compile 只证明可编译。

## Open questions

- 公开 stable：权利人选择并替换 OSI LICENSE。
- 发布身份：确定 canonical public repository/module/update/security URL，并配置真实多维护者权限与 owner-level rules。
- 发布凭据：提供 Authenticode 证书、timestamp 服务，并在原生 Windows/Linux/macOS host/installer 上执行公开候选 smoke。
