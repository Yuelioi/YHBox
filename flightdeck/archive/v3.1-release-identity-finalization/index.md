---
topic: v3.1-release-identity-finalization
title: Yotta v3.1 release identity finalization
summary: Align the product release version to 3.1.0 and correct final acceptance evidence.
---

## State

完成。产品 release identity 已从 2.0.0 统一提升为 3.1.0，版本仅存在于 version 属性、binary metadata、manifest、artifact 与 tag；未重新引入 release-number package/type/runtime 名称。

## Next

本恢复 Topic 可归档。后续公开 stable 发布应新建 release/governance Topic，并等待许可证、签名凭据、canonical public identity、owner settings 与原生非 Windows host/installer smoke 条件具备。

## Read now

- knowledge/agent/codex-working-agreement.md
- knowledge/build/build.md

## Read if

- archive/major-upgrade-review/slices/final-contract-and-release-acceptance.md — 回查完整 3.1 工程验收

## Progress

- 在最终交付前发现 `task package` 与 WebView 截图仍显示 2.0.0，撤回过早的完成结论并建立本恢复任务。
- 通过正式 `scripts/bump-version.ps1 -Version 3.1.0` 同步 `pkg/version`、Wails config/manifest/info、NSIS 与 frontend package；提交 c29feb38，并创建本地 `v3.1.0` tag。
- `scripts/verify-version.ps1 -ExpectedVersion 3.1.0` 通过，7 个权威消费者一致。
- 阶段级 `task package` 通过：完整 `task check`、production build、stage/archive、manifest、frozen candidate smoke 全绿；产物为 `artifacts/Yotta-3.1.0-windows-amd64.zip`。
- candidate manifest 记录 version 3.1.0、sourceCommit c29feb38，并覆盖 Yotta.exe、Yotta.CLI.exe、ScriptWorker、WasmPluginRunner、capture DLL、ADB 与 licenses 的 exact file set/size/SHA-256。
- WebView smoke 功能断言通过。第一次 PNG 捕获到 WebView2 偶发黑帧，按验收原则未接受；一次定向复现生成 `.task/workflow-editor-smoke/20260717-080403/workflow-editor.png`，人工确认完整布局且左上角显示 `Yotta v3.1.0`。
- 本阶段仅改产品元数据，没有重复上一工程阶段已经通过的 race/fuzz/cross-platform 逻辑验收。

## Open questions

- 公开 stable 仍受 source-available LICENSE、签名凭据、canonical public identity、owner settings 与原生非 Windows host/installer smoke 外部条件阻塞。
