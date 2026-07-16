---
topic: major-upgrade-review
title: Yotta 3.1 major upgrade
summary: Implement and validate the AI-native destructive Yotta 3.1 architecture and release program.
---

## State

显式模板分析的第一块 3.1 纵切面已完成并提交 `d63431ab`。新增 `MatchTemplate` pull analysis：只接 nominal Image BlobRef、template Image BlobRef、Region 与 threshold，输出 matched/score/center/bounds 四个 typed data value；没有 exec/out/error pin，也不接 asset GUID、窗口、截图或输入权限。运行时通过唯一 `blob-read` capability 有界读取两张 PNG，实施压缩字节与解码像素预算、严格 ROI、TM_CCOEFF_NORMED best-match、typed output 与 bounded action journal。

Catalog、Authoring Projection 和生成节点文档已同步。全仓 `go test ./...`、`task contracts:check`、focused staticcheck 与 `git diff --check` 全绿。当前 frontier 是完成其余 image/color/QR/frame-diff 分析节点，并删除 legacy VisionService/detect/image 路径。

## Next

按独立 commit 连续推进，禁止 dual path、compatibility shim 和 runtime fallback：

1. 迁移 image comparison、color statistics、QR decode 等显式 Image analysis；等待/轮询通过 Capture + Delay/Repeat + analysis 组合，不再做环境式 WaitTemplate/ClickTemplate。
2. 切换 template asset authoring 为 immutable BlobRef 绑定，随后删除 legacy detect/image nodes、VisionService、template GUID validator/dependency 与前端特判。
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
- explicit Image + template Image analysis（`d63431ab`）。

Verification for latest batch:
- `go test ./...`
- `task contracts:check`
- `staticcheck ./internal/nodes31 ./internal/nodes31runtime`
- `git diff --check`

## Decisions

- Template library records are authoring metadata; executable matching binds immutable template Image BlobRef content, not mutable asset GUID.
- MatchTemplate is a data analysis node. Blob access makes it a recorded pull effect, but it has no exec/out pin; consumers pull typed results.
- Capture, wait/repeat, analysis and input action remain separate nodes. WaitTemplate and ClickTemplate are compositions, not primitive runtime capabilities.
- Runtime image inputs are strict bounded `image/png` blobs. Unsupported media, oversized images, invalid ROI and uniform templates fail explicitly.
- OpenCV's official template-matching model confirms explicit source image + template image + result localization; Unreal's official Blueprint model confirms data-only functions should not invent execution pins.
- Legacy UI still present under the pending Container deletion wave is not a runtime fallback and must be removed before completion.

## Open questions

None blocking. Continue autonomously with remaining explicit image analysis.
