---
topic: workflow-editor-capability-roadmap
title: 3.1 产品能力连续性审计与升级路线
summary: 审计旧产品能力与 3.1 现状，按唯一新架构、用户体验和必要性决定恢复、重做或删除，并完成发布前能力补齐。
---

## State

In progress。3.1 尚未发布；R1–R4 已完成，执行内核与 Windows、Android、Browser、编辑器外围已经通过纵向旅程和阶段门禁。当前进入 R5 发布门禁与历史 Knowledge 退役；历史 Slices 1–26 仍只作为实现记录，不能替代最终发布证据。

## Next

执行 Slice 37：按 capability ledger 与 G01–G17 完成最终发布矩阵，随后退役已登记的 3.0 主动 Knowledge，并形成 3.1 用户验收候选。

## Read now

- work/workflow-editor-capability-roadmap/slices/37-release-gate-knowledge-retirement.md
- work/workflow-editor-capability-roadmap/artifacts/capability-ledger.md
- work/workflow-editor-capability-roadmap/artifacts/golden-journeys.md
- work/workflow-editor-capability-roadmap/context/legacy-knowledge-retirement.md
- knowledge/build/build.md

## Read if

- work/workflow-editor-capability-roadmap/slices/map.md — 查询/调整完整 Slice registry
- work/workflow-editor-capability-roadmap/slices/27-architecture-recovery.md — 查看 R0–R5 恢复设计和架构理由
- work/workflow-editor-capability-roadmap/context/knowledge-and-docs-review.md — 执行历史 Knowledge/docs 退役
- work/workflow-editor-capability-roadmap/context/r0-worktree-ownership.md — 修改既有 dirty 路径前确认归属
- knowledge/architecture/feature-continuity-across-product-stack.md — 判断 ledger 能力是否真的完成
- knowledge/build/build.md — 执行最终门禁、打包或真实宿主 smoke
- knowledge/git/commits.md — 形成阶段或最终提交

## Progress

- 3.0 reference worktree、capability ledger、G01–G17、dirty ownership 与历史 Knowledge 退役 registry 已固定。
- R1 完成 Typed RPC、Installation Manifest/Target Runtime、单一 Recording Session 与 Asset Query/Picker；整仓门禁全绿。
- R2 完成 Windows UAC、真实 UnrealWindow、键鼠/窗口/截图/模板/录制、workspace 恢复与 launcher；native、WebView、build 和 cross-compile 全绿。
- R3 完成 145-node typed authoring、State/Target 搜索、调试/多图、1000 workflows/assets/states 规模证据；整仓门禁与人工截图检查全绿。
- R4 完成 Android 应用发现与 InputClip playback、Chrome/Edge CDP 纵向旅程和 macOS Adapter seam proof；Android emulator 全链路、cross-compile、`task check` 与 `task build` 全绿。
- ADB effect 子进程现由 Adapter 强制 10 秒上限并继承更短调用截止时间，输入副作用不自动重试；本地 emulator 与 controlled browser 已精确清理。

## Open questions

- R5 最终 Windows 高完整性用户旅程可能需要用户以 UAC 重新打开 Yotta；仅在矩阵确实触发时再请求操作。
- 代码签名、公开仓库、OSI 许可证替换仍属独立发布工程，不在本 Topic 内擅自推进。
