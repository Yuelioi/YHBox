---
topic: major-upgrade-review
title: Yotta 3.1 major upgrade
summary: Implement and validate the AI-native destructive Yotta 3.1 architecture and release program.
---

## State

显式视觉分析的 3.1 backend 纵切面已完成两批：`MatchTemplate`（`d63431ab`）以及 typed image analysis primitives（`07cadde9`）。Catalog 现有 MatchTemplate、FindTemplateMatches、CompareImages、DecodeQR、AnalyzeColor、FindColorBlobs；全部是 recorded pull analysis，只接 nominal Image BlobRef 与 typed 参数，只有最小 `blob-read` authority，没有 exec/out/error pin、窗口、隐式截屏、等待或输入动作。

新增 TemplateMatch、QRCode、ColorRange、ColorBlob 四个 nominal inline type。运行时统一实施严格 image/png、32 MiB compressed budget、16M pixel budget、严格 ratio/px ROI、bounded blob read、确定性排序和 4096 result budget。Catalog、Authoring Projection 与生成节点文档同步。全仓 Go、focused race/staticcheck、contract drift 与 diff checks 全绿。

当前 frontier 是切换 template asset authoring 为 immutable Image BlobRef 绑定，并删除 legacy VisionService/detect/image/Template GUID 路径。

## Next

按独立 commit 连续推进，禁止 dual path、compatibility shim 和 runtime fallback：

1. 切换 template asset authoring 为精确 variant BlobRef 绑定；工作流不保存 mutable asset GUID。
2. 删除 legacy detect/image nodes、VisionService、template matcher adapter、template GUID validator/dependency 与旧前端特判；Wait/Click 由 Capture + Delay/Repeat + analysis + input 组合。
3. 删除剩余旧 Container RPC/UI/LLM/NodeSpec/coercion/dispatch，使 GUI、Schedule、Hotkey、Debug、headless 只进入 Application/Program runtime。
4. 实现 Node Package + Wasm/Process host、生命周期、SDK、conformance 与 Windows fail-closed 隔离；不加载 Go plugin 或第三方前端代码。
5. 完成 projections/docs/golden fixtures，运行 `task check` 和最终双轴 architecture review。

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
- typed multi-match/frame-diff/color/QR analysis（`07cadde9`）。

Verification for latest batch:
- `go test ./...`
- `go test -race ./internal/nodes31runtime ./pkg/vision`
- `staticcheck ./internal/nodes31 ./internal/nodes31runtime`
- `task contracts:check`
- `git diff --check`

## Decisions

- Template library records are authoring metadata; executable matching binds immutable template Image BlobRef content, not mutable asset GUID.
- Vision analysis is pull data evaluation with explicit Blob authority; consumers pull typed results and no analysis node invents an exec/out pin.
- Capture, wait/repeat, analysis and input action remain separate. WaitTemplate, ClickTemplate and specialized polling nodes are compositions, not primitive runtime capabilities.
- Multi-match uses local maxima + NMS; frame comparison exposes grid-size and cell-delta; color analysis exposes explicit RGB/HSV inclusive range; connected components use 4-neighbor topology and stable area/y/x ordering.
- Runtime image inputs are strict bounded `image/png` blobs. Unsupported media, oversized images, invalid ROI/range and uniform templates fail explicitly.
- OpenCV's official template-matching model confirms explicit source image + template image + result localization; Unreal's official Blueprint model confirms data-only functions should not invent execution pins.

## Open questions

None blocking. Continue autonomously with asset binding and legacy vision deletion.
