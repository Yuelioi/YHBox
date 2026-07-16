---
topic: major-upgrade-review
title: Yotta 3.1 major upgrade
summary: Implement and validate the AI-native destructive Yotta 3.1 architecture and release program.
---

## State

3.1 视觉后端与契约驱动 authoring 已完成。显式 MatchTemplate（`d63431ab`）、typed image analysis primitives（`07cadde9`）和 port editor / immutable template binding（`64401a71`）均已提交。

Node Contract 的非语义 Authoring 现可为已声明端口提供严格 titleKey、descriptionKey 与 allowlisted editorAdapter；Projection、JSON Schema、生成 TypeScript 和文档同源。视觉端口都有中英文标题/提示，template Image 端口使用内置 `template-image` adapter，颜色范围使用 Data Type `color-range` adapter。AssetSummary 返回所有模板 variant 的 resolution + BlobRef，Workflow Source 保存精确 BlobRef，不保存 mutable asset GUID。

全仓 Go、staticcheck、contracts drift、Wails RPC contract、frontend format/lint/typecheck/i18n、101 files/650 tests、production build 与 bundle budgets 全绿。当前 frontier 是删除 legacy VisionService/detect/image/template matcher/GUID dependency 路径。

## Next

按独立 commit 连续推进，禁止 dual path、compatibility shim 和 runtime fallback：

1. 删除 legacy detect/image nodes、VisionService、template matcher adapter、template GUID validator/dependency 与旧前端特判；Wait/Click 由 Capture + Delay/Repeat + analysis + input 组合。
2. 删除剩余旧 Container RPC/UI/LLM/NodeSpec/coercion/dispatch，使 GUI、Schedule、Hotkey、Debug、headless 只进入 Application/Program runtime。
3. 实现 Node Package + Wasm/Process host、生命周期、SDK、conformance 与 Windows fail-closed 隔离；不加载 Go plugin 或第三方前端代码。
4. 完成 projections/docs/golden fixtures，运行 `task check` 和最终双轴 architecture review。

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
- port-level Authoring Projection + immutable template variant binding（`64401a71`）。

Verification for latest batch:
- `go test ./...`
- `staticcheck ./internal/nodecontract ./internal/nodeauthoring ./internal/datatype ./internal/nodes31 ./internal/services/asset`
- `task contracts:check`
- `pnpm -C frontend check`
- `pnpm -C frontend bindings:check`
- `git diff --check`

## Decisions

- Port titles, descriptions and built-in editor selection are non-semantic Node Authoring facts. They are strict and generated, but cannot change NodeRef semantic identity.
- A port editor adapter must be allowlisted by the host. Plugin JavaScript/DOM remains prohibited.
- Template asset GUID/name/category/tags are library metadata. A workflow binds one exact variant BlobRef; recapture does not silently mutate compiled behavior.
- Vision analysis is pull data evaluation with explicit Blob authority; no analysis node has exec/out.
- Capture, wait/repeat, analysis and input action remain separate. WaitTemplate and ClickTemplate are compositions, not primitive runtime capabilities.

## Open questions

None blocking. Continue autonomously with legacy vision deletion.
