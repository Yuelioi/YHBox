---
topic: major-upgrade-review
title: Yotta 3.1 major upgrade
summary: Implement and validate the AI-native destructive Yotta 3.1 architecture and release program.
---

## State

legacy Vision/GUID 纵切面已经完全删除并以 `4c6b9512` 提交。旧 `internal/nodes/detect`、`internal/nodes/image`、`VisionService`、template matcher wiring、模板 GUID dependency/validator、template picker 特判、录制/模板 GUID 引用推断与 asset-local Blob GC 均不再存在。

Workflow 3.1 现在只持久化精确 BlobRef。模板和录制 GUID 只属于素材库 authoring metadata；删除素材记录不会改写或破坏已保存 Workflow。共享 Blob Store 没有生产 `Sweep` 调用，避免仅凭资产记录集合误删 Workflow 仍引用的内容。自动清理仅保留具有结构化 SubgraphID 边的蓝图路径。

Wails contract 已同步为 18 services / 132 methods / 149 models。全仓 Go、affected staticcheck、Workflow/Node 3.1 contracts、frontend format/lint/typecheck/i18n/bindings、100 files/641 tests、production build 与 bundle budgets 全绿。

## Next

按独立 commit 连续推进，禁止 dual path、compatibility shim 和 runtime fallback：

1. 删除剩余旧 Container RPC/UI/LLM/NodeSpec/coercion/dispatch，使 GUI、Schedule、Hotkey、Debug、headless 只进入 Application/Program runtime。
2. 实现 Node Package + Wasm/Process host、生命周期、SDK、conformance 与 Windows fail-closed 隔离；不加载 Go plugin 或第三方前端代码。
3. 完成 projections/docs/golden fixtures，运行 `task check` 和最终双轴 architecture review。

## Read now

- knowledge/agent/codex-working-agreement.md
- work/major-upgrade-review/context.md
- work/major-upgrade-review/review.md
- work/major-upgrade-review/design.md
- work/major-upgrade-review/ai-native-design.md
- work/major-upgrade-review/plan.md
- knowledge/build/ci-documented-gates-can-be-absent.md
- knowledge/build/wails-rpc-count-is-not-a-contract.md
- knowledge/mcp/normalize-masks-schema-prompt-drift.md

## Read if

- knowledge/build/build.md — 开始运行构建、测试或产物验证前
- knowledge/architecture/content-addressed-workflow-artifacts.md — 修改 Source/Catalog/Compiler/Program/Blob identity 时
- knowledge/architecture/resource-lease-edge-authority.md — 修改 stream/handle carrier 或 Executor borrow 时
- knowledge/architecture/resource-broker-open-revocation.md — 修改 Broker/Owner/session cleanup 时
- knowledge/architecture/installed-application-vs-plugin-process.md — 修改 Process/AE/UE/plugin host 时
- knowledge/architecture/go-multiplatform-boundary.md — 修改跨平台支持承诺时
- knowledge/build/wails-rpc-count-is-not-a-contract.md — 修改 Wails RPC/DTO/bindings 时

## Progress

Done:
- exact installed application lifecycle、HTTP、workspace file/log、AI、Script isolation、input/window/capture 3.1 纵切面；
- Workflow/Program/Run/Grant/Resource Broker、typed patch、MCP、EditorSession、state/control/pure-data/collection 等 3.1 核心；
- legacy Container executor 删除；
- 单一共享 Blob Store（`75ef8d9b`）；
- exact window capture（`fe3e647d`）；
- nominal InputClip/exact playback 与旧 PlayClip 删除（`fbb1712c`）；
- explicit MatchTemplate（`d63431ab`）；
- typed multi-match/frame-diff/color/QR analysis（`07cadde9`）；
- port-level Authoring Projection + immutable template variant binding（`64401a71`）；
- legacy detect/image/VisionService/template GUID dependency 与不安全资产 GC 删除（`4c6b9512`）。

Verification for latest batch:
- `go test ./...`
- `staticcheck . ./internal/node ./internal/catalog ./internal/nodes/all ./internal/nodes/script ./internal/nodes/stopwatch ./internal/services/asset ./internal/services/recording ./internal/services/container/... ./internal/services/script ./pkg/vision`
- `task contracts:check`
- `pnpm -C frontend check`
- `git diff --check`

## Decisions

- Port titles, descriptions and built-in editor selection are non-semantic Node Authoring facts. They are strict and generated, but cannot change NodeRef semantic identity.
- A port editor adapter must be allowlisted by the host. Plugin JavaScript/DOM remains prohibited.
- Template asset GUID/name/category/tags are library metadata. A workflow binds one exact variant BlobRef; recapture does not silently mutate compiled behavior.
- Vision analysis is pull data evaluation with explicit Blob authority; no analysis node has exec/out.
- Capture, wait/repeat, analysis and input action remain separate. WaitTemplate and ClickTemplate are compositions, not primitive runtime capabilities.
- Arbitrary UUID text is never dependency authority. Subgraph dependencies come only from structured SubgraphID fields.
- Shared Blob GC requires a global live-set across every durable owner. Asset-local GC and automatic template/recording cleanup are removed until such an authority exists.
- Deleting template or clip metadata does not delete immutable Blob content or invalidate a Workflow 3.1 source. Only subgraphs retain reference-aware cleanup because their edges are explicit and enumerable.

## Open questions

None blocking. Continue autonomously with remaining Container RPC/UI/LLM/NodeSpec removal.
