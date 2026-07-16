---
topic: major-upgrade-review
title: Yotta 3.1 major upgrade
summary: Implement and validate the AI-native destructive Yotta 3.1 architecture and release program.
---

## State

InputClip / playback 的 3.1 破坏性纵切面已完成并提交 `fbb1712c`。InputClip v2 carrier 只包含不可变回放内容，asset GUID、名称、分类和标签不再参与 Blob identity；工作流持久化 nominal InputClip BlobRef，并通过独立 blob-read + exact playback capability 执行。installed target 新增独占 playback session、逐事件严格协议、取消/失败释放 held state 与 MouseCounts360 相对位移校准。旧 PlayClip 节点、ClipPlayer ServiceBundle、hybrid backend、旧 runtime scheduler、ClipID validator/dependency/package/RPC 路径已删除，不保留兼容或 fallback。

3.1 Editor 已补齐 bind-blob domain command，并能从 InputClip 资产选择 BlobRef；Catalog/Authoring Projection/Wails contract/Settings 与中英文提示同源更新。全仓 Go、完整 frontend check（101 files / 650 tests）、production build、focused race 与 staticcheck 全绿。当前 frontier 是 detect/template/color/image 能力迁移。

## Next

按独立 commit 连续推进，禁止 dual path、compatibility shim 和 runtime fallback：

1. 设计并实现 nominal Template/Image observation 类型、exact installed target capture/read authority 与 detect/template/color 节点的 typed capability/runtime/journal。
2. 切换每批调用方后删除相应 legacy detect/image node、VisionService、validator、container dependency 与前端特判。
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
- exact window capture（`fe3e647d`）及 Flightdeck checkpoint（`34d56550`）；
- nominal InputClip/exact playback 与旧 PlayClip 删除（`fbb1712c`）。

Verification for latest batch:
- `go test ./...`
- `pnpm -C frontend check`
- `go test -race ./internal/automation/installed ./internal/nodes31runtime ./internal/services/inputclip`
- `staticcheck ./internal/automation/installed ./internal/nodes31runtime ./internal/services/inputclip ./internal/nodes31 ./pkg/input`
- `git diff --check`

## Decisions

- InputClip asset metadata is mutable authoring state; carrier bytes are immutable playback content. Metadata updates must preserve BlobRef identity.
- Workflow playback binds content, not asset GUID. Deleting or renaming a library record cannot mutate an already-authored Program input.
- Playback authority is a separate dangerous ConsentOnce capability and is mutually exclusive with atomic input sessions for the same provider target.
- Relative motion requires both source and target MouseCounts360; missing calibration fails rather than replaying unscaled.
- Playback provider receives one validated event at a time and owns exact target revalidation/scaling/input release; workflow runtime owns bounded carrier read, decode and event timing.
- Legacy UI still present under the pending Container deletion wave is not a runtime fallback and must be removed with that wave before completion.

## Open questions

None blocking. Continue autonomously with detect/template migration.
