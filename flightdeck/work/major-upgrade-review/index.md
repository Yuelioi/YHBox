---
topic: major-upgrade-review
title: Yotta 3.1 major upgrade
summary: Implement and validate the AI-native destructive Yotta 3.1 architecture and release program.
---

## State

当前唯一实施主线是 Yotta 3.1 destructive upgrade。

已完成的大 Wave：

- Wave A `be1fc04b`：Launcher/Settings/Hotkey 活跃入口切到 Workflow 3.1；旧 Container hotkey/calibration RPC 删除。
- Wave B `d060798c`：Asset、Tools、Recording 统一按已安装 `targetSlot` 使用 host-only AuthoringTargets projection；删除按 Container 节点解析窗口、重复创建 capture backend 与录制 target platform resolver。
- Wave C `9fce7870`：物理删除旧 Container/Subgraph/Node/CodeSnippet Wails RPC 与 Store、旧 Container editor 产品树、兼容迁移和 validate-fishing-v2；Wails contract 收敛为 14 services / 89 methods / 99 models。
- 同一份 sealed automation installation/provider 同时服务 Run capability broker 和可信本地制作工具；Workflow 仍无法获得 native handle 或绕过 Grant。

Wave D“Node Contract 单一事实源”当前切片已完成：旧 `internal/node`、`internal/nodes/*`、`internal/catalog`、旧表达式/script binding、节点 i18n registry 和对应前端/日志库存已物理删除。CLI 文档、节点文案、前端参数提示与测试只消费 3.1 Catalog/Authoring Projection；没有 adapter registry、kind switch、旧 coercion 或 compatibility type 回接。

Wave E 的 AI 收口首批已由 `99c3f5ff` 完成：零消费者的 `internal/services/llm`、通用 Chat/Mode、endpoint 猜测、structured prompt/fence fallback 与旧 provider SDK 依赖已物理删除。AI 单一路径是 `internal/ai` 的 provider-native installation/profile/resource contract，加 `internal/nodes31` / `internal/nodes31runtime` 的 capability session。

Wave E 的插件首批已由 `a8c0cfb5` 完成：`internal/nodepackage` 建立 canonical、内容寻址、可 reopen 的 Node Package manifest v1，冻结 publisher namespace、strict SemVer、半开 host API range、exact Type/Capability/Node semantics、WIT/Process ABI 和 portable payload identity；仍未开放发现、安装或执行入口。

## Work outline

### A. 活跃入口切换（✅ 已完成：be1fc04b）

1. 将悬浮启动器、设置中的启动器/热键、运行列表改为消费 Workflow 3.1 Source/Run。
2. GUI、Schedule、Hotkey、Debug、headless 统一调用 Application/Program runtime。
3. 删除旧 Container run/debug/validate/save 的活跃前端调用。

### B. 平台 adapter 去 Container 化（✅ 已完成：d060798c）

1. 模板截图和录制目标改为安装式 Automation Target / Application-owned adapter，不再按 containerID 取 Win32WindowTarget。
2. tools、校准、热键 binder 改用 3.1 installation/target identity。
3. 删除 main composition root 中仅为旧 Container 服务的后置 Configure/Set wiring。

### C. 删除旧产品树（✅ 已完成：9fce7870）

1. 删除不可达的 `ContainersView`、`ContainerEditorView`、旧 Pin/Inspector/registry/composables/store 和对应 tests。
2. 删除旧 Container Wails RPC、Store、validator/package/export/subgraph compatibility 路径，以及 CodeSnippet/NodeOptions 服务和旧数据迁移。
3. 重生成 Wails bindings/contract，确认 14 services / 89 methods / 99 models；不存在旧 RPC、DTO 或 compatibility shim。

### D. Node Contract 单一事实源（✅ 已完成：e29ff25d）

1. 让剩余 UI、表达式/脚本 authoring、CLI 文档只消费 3.1 Authoring Projection/Catalog。
2. 删除旧 `internal/node`、`internal/nodes/*`、`internal/catalog`、`CanonicalPinType`、`PinTypeCompat`、`CoerceInputValue` 和按 kind validator/dispatch。
3. 普通内置节点新增不再修改中央 switch；生成契约、参数提示、文档和 fixtures 同源。

### E. AI 与插件（进行中；AI generic fallback 已删除：99c3f5ff）

1. ✅ 删除旧 generic Chat/structured prompt fallback，收口到 provider-native installation/profile/eval/trace。
2. 🚧 Node Package immutable manifest 已完成；继续实现 lifecycle、Wasm/Process host、SDK 和 conformance。禁止 Go plugin 和第三方前端代码。
3. Windows fail closed；Linux/macOS 只承诺平台中立 core 与 preview host 能力。

### F. 收尾验收

1. 完成 projections、reference docs、golden fixtures 与 breaking contract diff。
2. 只在最终阶段运行完整 `task check`；日常按受影响 package/spec 做定向验证。
3. 执行最终 Standards + Spec + architecture review。
4. 真实 Windows smoke 覆盖模板/录制、设置、悬浮启动器、工作流运行/调试和工具窗。

## Next

Wave E 下一批从 immutable manifest 向 package lifecycle 前进：先定义并实现 archive payload verification + safe extraction 的纯核心，使 manifest 中 path/digest/size/media-type lock 能在任何信任或执行前 fail closed；随后再接 trust state 与 atomic install。执行 host 仍不提前开放。边界保持：不加载 Go plugin，不执行第三方前端 JavaScript/Vue/DOM；Windows fail closed，Linux/macOS 只承诺平台中立 core 与 preview host。完整 `task check` 仍只在最终 Wave F 运行。

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
- knowledge/architecture/installed-application-vs-plugin-process.md — 修改 Process/AE/UE/plugin host 时
- knowledge/architecture/node-package-manifest.md — 修改 package lifecycle、Catalog merge、Wasm/Process host 或插件 payload 时
- knowledge/architecture/go-multiplatform-boundary.md — 修改跨平台支持承诺时
- knowledge/build/wails-rpc-count-is-not-a-contract.md — 修改 Wails RPC/DTO/bindings 时

## Progress

Completed foundations:

- Workflow/Program/Run/Grant/Resource Broker、typed patch、MCP、EditorSession 和 Application runtime；
- installed application lifecycle、HTTP、workspace file/log、AI、Script isolation、input/window/capture 3.1；
- single shared Blob Store、exact window capture、nominal InputClip/exact playback；
- explicit MatchTemplate 与 typed multi-match/frame-diff/color/QR analysis；
- port-level Authoring Projection + immutable template variant binding；
- legacy Container executor、PlayClip、detect/image/VisionService/template GUID dependency 和 unsafe asset GC 删除；
- Launcher/Settings/Hotkey 活跃入口已切到 Workflow 3.1，旧 Container hotkey/calibration RPC 删除；
- Asset/Tools/Recording 已按 installed target slot 运行，Container window resolver 与重复 capture adapter 删除；
- 旧 Container/Subgraph/Node/CodeSnippet RPC 与 Store、Container editor 产品树和兼容迁移已物理删除。
- 旧 `internal/node`、`internal/nodes/*`、`internal/catalog`、表达式/script binding、旧节点 i18n registry 与 Container 专用日志/UI helper 已物理删除；`cmd/node-catalog` 只导出当前构建的 3.1 catalog/authoring/docs artifacts。
- Wails 日志 DTO 已收口为 SYS/WF 与 graph/node/invocation/attempt provenance；contract 保持 14 services / 89 methods / 99 models。
- frontend ESLint `no-explicit-any` debt 随旧产品树删除从 258 收紧到 24；生产 bundle 为 entry 259,767 bytes、editor 94,274 bytes gzip。
- 旧 `internal/services/llm` 和 OpenAI/Anthropic Go SDK 依赖已删除；provider-native OpenAI Responses / Anthropic Messages adapter、typed outcome/failure、profile installation、resource session、credential binding 与 workflow consent 成为 AI 单一实现路径。
- `internal/nodepackage` manifest v1 已锁定 package/publisher/host API、exact contract semantics、WIT/Process implementation 和 payload identity；当前没有第三方代码发现、安装或执行面。

Latest Wave E AI verification:

- `go list ./...`
- `go test -count=1 -timeout=60s ./...`
- affected `staticcheck ./internal/ai ./internal/nodes31 ./internal/nodes31runtime ./internal/services ./internal/appbootstrap`
- `go mod verify`
- generic LLM/API SDK/residue scan + `git diff --check`
- 全库 `staticcheck ./...` 唯一失败为未改动的 `internal/automation/installed/platform_windows.go:64` S1016

Latest Wave E Node Package verification:

- `go test -count=1 ./internal/nodepackage`
- `go vet ./internal/nodepackage`
- `staticcheck ./internal/nodepackage`
- `go test -count=1 -timeout=60s ./...`
- `git diff --check`

Latest Wave D verification:

- `go list ./...`
- `go test -count=1 -timeout=60s ./...`（全绿；并行首次运行 LPAC worker 曾抖动一次，单测复跑两次和全仓单独复跑均通过）
- updated CI race package group：`services/application/nodes31runtime/run/workflowstore/schedule/tools/inputclip/hotkey/winutil/capture` 均通过 race
- `go vet ./...` 唯一失败为未改动的 `pkg/winutil/window_windows.go:385` `unsafe.Pointer` 警告；本切片 affected `staticcheck` 全绿
- Vitest：26 files / 100 tests
- `pnpm -C frontend lint`：0 warning / 0 error，tracked debt 24
- `pnpm -C frontend format:check`
- `pnpm -C frontend typecheck`
- `pnpm -C frontend i18n:check`：1214 keys
- `task contracts:check`
- regenerated Wails bindings + `pnpm -C frontend bindings:check`：14 services / 89 methods / 99 models
- `pnpm -C frontend build:dev`
- `pnpm -C frontend build` + bundle budget
- legacy path/import residue scan + `git diff --check`

上一完整阶段门禁仍为 `go test ./...`、affected `staticcheck`、`task contracts:check`、`pnpm -C frontend check`（100 files / 641 tests、production build、bundle budgets）。全量 `task check` 只在最终阶段运行。

## Decisions

- 3.1 没有 legacy compatibility path、dual read/write 或 runtime fallback。
- Application/Program runtime 是执行深模块；Wails、MCP、Schedule、Hotkey、Debug 和 headless 都是 adapter。
- Workflow 持久化 immutable BlobRef，不持久化 mutable asset GUID。
- Arbitrary UUID text 永远不是 dependency authority；只有结构化字段可形成依赖。
- Shared Blob GC 必须拥有覆盖全部 durable owner 的 global live set；在此之前不提供 asset-local GC。
- Vision analysis 是显式 pull-data evaluation；Capture、wait/repeat、analysis 和 input action 分离。
- 插件只允许 Node Package + Wasm/Process host，不加载 Go ABI 或插件 JavaScript/Vue/DOM。

## Open questions

无阻塞问题。当前非本切片 gate 风险是 `pkg/winutil/window_windows.go:385` 的 `go vet` unsafe-pointer warning，以及 `internal/automation/installed/platform_windows.go:64` 的 `staticcheck` S1016；进入最终全量门禁前需单独修复或确认工具链行为。桌面真实 smoke 统一留到最终验收，不再让已完成 feature topic 长期保持 active。
