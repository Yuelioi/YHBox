---
topic: workflow-editor-capability-roadmap
title: 3.1 产品能力连续性审计与升级路线
summary: 审计旧产品能力与 3.1 现状，按唯一新架构、用户体验和必要性决定恢复、重做或删除，并完成发布前能力补齐。
---

## State

In progress。3.1 尚未发布；执行内核保留，产品外围按 Slices 29–37 纵向恢复。R1、R2 已完成并留下整仓门禁、Windows native、真实 UnrealWindow 与 WebView 旅程证据；开始 R3 编辑器与节点能力恢复。历史 Slices 1–26 仍不能直接作为发布证据。

## Next

执行 Slice 35：以 G01、G06、G09–G12 复核创作路径，补齐 typed State/输出候选、Stopwatch、观察、Switch、图像与状态便利能力，并在阶段末统一做 R3 editor/integration gate。

## Read now

- work/workflow-editor-capability-roadmap/slices/35-editor-capability-recovery.md
- work/workflow-editor-capability-roadmap/artifacts/capability-ledger.md
- work/workflow-editor-capability-roadmap/artifacts/golden-journeys.md
- work/workflow-editor-capability-roadmap/context/r0-worktree-ownership.md
- knowledge/architecture/feature-continuity-across-product-stack.md
- knowledge/subgraph/asset-subsystem.md
- knowledge/architecture/content-addressed-workflow-artifacts.md
- knowledge/nodes/typed-authoring-contract.md
- knowledge/build/build.md

## Read if

- work/workflow-editor-capability-roadmap/slices/map.md — 查询/调整完整 Slice registry
- work/workflow-editor-capability-roadmap/slices/27-architecture-recovery.md — 查看 R0–R5 恢复设计和架构理由
- work/workflow-editor-capability-roadmap/context/r0-worktree-ownership.md — 开始修改现有 dirty 路径前确认归属
- work/workflow-editor-capability-roadmap/slices/28-legacy-knowledge-reconciliation.md — 复查旧 Knowledge/docs 的范围和结果
- knowledge/architecture/go-multiplatform-boundary.md — 修改 target/profile/Adapter seam
- knowledge/architecture/installed-input-authority.md — 修改窗口选择、输入或 consent
- knowledge/nodes/node-system-architecture.md — 修改节点契约、Catalog、Compiler 或 adapter
- knowledge/nodes/recording-schema-cascade.md — 修改录制/finalize/codec/playback
- knowledge/build/build.md — 进入阶段验收、打包或真实宿主 smoke

## Progress

- 3.0 reference worktree、capability ledger、G01–G17、dirty ownership 与 18 条历史 Knowledge 退役 registry 已固定。
- Slice 29 已完成恢复事实基线；Slice 30 已完成 Typed RPC，Stage R1 全门禁统一延后到 Slices 30–33 完成后执行。
- Slice 31 已完成 adapter-owned profile intent/payload、versioned Installation Manifest 与 descriptor-driven Settings fallback；中央 Settings 和 composition root 不再理解平台 payload/capability switch。
- Host Profile、provider operations、Policy/consent、health 与 authoring 均消费同一 manifest/adapter registration；自定义 Adapter 的 verifier 不再回落默认 registry。
- AutomationTargetRuntime 已集中 prepare/publish/rollback/lease/reclaim/shutdown；同进程 target/consent mutation 与 old-Run exact generation lease 有聚合测试。
- Slice 31 聚合验收通过：相关 Go packages、vue-tsc、3 个 Vitest 文件 13 tests、oxfmt 与定向 diff check；Windows/ADB native 旅程留在 R2/R4。
- Slice 32 已完成单一 Recording Session：显式 simple/precise、exact generation lease、native clock canonicalization、monotonic snapshot/pending、InputClip v3 自校验与 asset reload/draft round-trip。
- Slice 32 定向实现检验通过：5 个 Go packages、vue-tsc、5 个前端文件 35 tests、Wails contract、i18n 与相关格式检查；其后已纳入 R1 整仓门禁。
- Slice 33 已完成共享 Asset Query/Picker：服务端分页搜索、recent/thumbnail budget、revision invalidation、exact BlobRef 反查与 stale binding；Inspector 不再全量加载资产或展开 variants。
- R1 阶段门禁 `task check` 已通过：Go 全仓、vet/staticcheck/coverage、43 个前端测试文件 183 tests、类型/i18n/Wails contract、production build 与 bundle budget 全绿。

- Slice 34 已完成 Run-owned held input lease、完整窗口操作族、deterministic resolver、原生 SendInput drag/failure propagation，以及 corrupt Source/stale Program/consent 的 workspace 恢复。
- R2 native smoke 全绿；真实 `HTGame.exe / UnrealWindow / 异环··`（末尾两个空格）Run `019f7556-279d-711a-9b98-db9bd616bf94` 成功投递 ESC，record `sha256:a3dfebe52e35404c3afa73e9f14633cbe06a24e2070fd39ae52e3d6541f289f6`。
- R2 WebView 旅程覆盖损坏 Source 隔离、launcher workflow 执行/复用、编辑/连线/调试/子图/AI/资源库；实图发现并修复 selection toolbar 窄画布压缩。
- R2 `task check`、`task build`、Windows native smoke、相关 Linux/Darwin cross-compile 与 requireAdministrator manifest 全绿；Yotta.exe SHA-256 `7652263517690B0A527DAE2F40810E456FB97AF60BA09B79A75E49536FAB136D`。

## Open questions

- specialized vision、EventTick 按旧真实 workflow 证据决定恢复、复合替代或删除；不阻塞已确定的 P0 恢复。
- Android emulator/device 尚无可信纵向验收；Asset Picker 的 1000×2 contract fixture 已通过，人工 UX/响应预算仍留在 R3。
- 现有 dirty business diff 的每一处实现仍需由对应 Slice 审查，不能按路径归属直接认定正确。

