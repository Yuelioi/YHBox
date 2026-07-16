---
topic: major-upgrade-review
title: Yotta 3.1 major upgrade
summary: Implement and validate the AI-native destructive Yotta 3.1 architecture and release program.
---

## State

当前唯一实施主线是 Yotta 3.1 destructive upgrade。

Wave A“活跃入口切换”已完成并提交：

- `be1fc04b`：Launcher 设置模型从 Container 破坏性切换到 Workflow 3.1 Source/Run；悬浮启动器通过 Application presentation service 启动并跟踪 Run，包含终态事件/权威快照无缝交接。
- 删除活跃 Container 热键来源、批量清空 RPC、本地 MouseCalibration 向旧 Container 批量同步 RPC 与 UI。
- Wails contract 保持 18 services，methods 132→130，models 149→148；`LauncherBlock.containerId` 改为 `workflowId`，不保留旧字段或迁移 fallback。
- 产品 router 仍只暴露 `WorkflowsView` / `WorkflowEditorView`；剩余 `backend.containers` 调用集中在不可达旧产品树，将在 Wave C 整体删除。

当前进入 Wave B“平台 adapter 去 Container 化”。后端 `container.Store/Service` 仍被模板截图、录制目标、tools 和 composition wiring 使用；下一步先建立 installation/target identity 的深 interface，再删除按 containerID 解析 Win32WindowTarget 的路径，不给旧 Container 增加转发层。

旧独立 feature topics 已完成或被 3.1 收编；未完成的真实 Windows smoke 与迁移事项统一保留在本 topic。

## Work outline

### A. 活跃入口切换（✅ 已完成：be1fc04b）

1. 将悬浮启动器、设置中的启动器/热键、运行列表改为消费 Workflow 3.1 Source/Run。
2. GUI、Schedule、Hotkey、Debug、headless 统一调用 Application/Program runtime。
3. 删除旧 Container run/debug/validate/save 的活跃前端调用。

### B. 平台 adapter 去 Container 化

1. 模板截图和录制目标改为安装式 Automation Target / Application-owned adapter，不再按 containerID 取 Win32WindowTarget。
2. tools、校准、热键 binder 改用 3.1 installation/target identity。
3. 删除 main composition root 中仅为旧 Container 服务的后置 Configure/Set wiring。

### C. 删除旧产品树

1. 删除不可达的 `ContainersView`、`ContainerEditorView`、旧 Pin/Inspector/registry/composables/store 和对应 i18n/tests。
2. 删除旧 Container Wails RPC、Store、validator/package/export/subgraph compatibility 路径。
3. 重生成 Wails contract，确保没有双入口或 compatibility shim。

### D. Node Contract 单一事实源

1. 让剩余 UI、表达式/脚本 authoring 只消费 3.1 Authoring Projection/Catalog。
2. 删除旧 `NodeService.GetAllNodeSpecs`、前端 adapter registry、`CanonicalPinType`、`PinTypeCompat`、`CoerceInputValue` 和按 kind validator/dispatch。
3. 普通内置节点新增不再修改中央 switch；生成契约、文档和 fixtures 同源。

### E. AI 与插件

1. 删除旧 generic Chat/structured prompt fallback，收口到 provider-native installation/profile/eval/trace。
2. 实现 Node Package lifecycle、Wasm/Process host、SDK 和 conformance；禁止 Go plugin 和第三方前端代码。
3. Windows fail closed；Linux/macOS 只承诺平台中立 core 与 preview host 能力。

### F. 收尾验收

1. 完成 projections、reference docs、golden fixtures 与 breaking contract diff。
2. 只在最终阶段运行完整 `task check`；日常按受影响 package/spec 做定向验证。
3. 执行最终 Standards + Spec + architecture review。
4. 真实 Windows smoke 覆盖模板/录制、设置、悬浮启动器、工作流运行/调试和工具窗。

## Next

Wave B 当前切片：把模板截图与录制目标从 `containerID -> Win32WindowTarget` 改为安装式 Automation Target / Application-owned adapter。先画清 asset capture、recording、tools 与 main wiring 的调用链，再落一个窄 interface；定向验证并提交后，在本节和执行计划中标记 Wave B 完成。

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
- Launcher/Settings/Hotkey 活跃入口已切到 Workflow 3.1，旧 Container hotkey/calibration RPC 删除。

Latest Wave A verification:

- `go test ./internal/services ./internal/services/container ./internal/hotkey`
- Launcher Vitest：2 files / 14 tests
- `pnpm -C frontend typecheck`
- `pnpm -C frontend i18n:check`：3064 keys
- regenerated Wails bindings + `pnpm -C frontend bindings:check`
- `git diff --check`

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

无阻塞问题。桌面真实 smoke 统一留到最终验收，不再让已完成 feature topic 长期保持 active。
