# Yotta 3.1 目标设计

## 结论

Yotta 已有可靠性基础：Go 全量测试、race-sensitive package、三平台 portable-core、原子持久化、节点 capability 校验和应用生命周期 owner 都已存在。升级不应推倒这些成果，而应删除围绕它们累积的重复协议、后置注入、旧数据分支和巨型 UI 协调层。

本次目标版本定义为 **Yotta 3.1**。它是一次格式、接口、装配、AI 和发布体系同时换代的 breaking release。产品层的 AI-native 设计见 `ai-native-design.md`，本文件聚焦共享核心：

- 不读、不迁移、不修复 Yotta 2.x 数据；旧版本用户须在升级前自行导出或保留旧安装。
- 不保留旧 RPC、旧节点 pin、旧环境变量、旧 schema、旧前端 re-export。
- 不做 dual-read、dual-write、feature flag 切换或 fallback shim。
- 失败必须显式、typed、可定位；不得用默认值、占位对象或 `nil error` 掩盖损坏状态。
- 只有明确成为产品策略的自动选择才允许存在，例如用户主动选择 `captureBackend=auto`；不得把静默降级伪装成成功。

## 审查证据

- 规模：703 个 Go 文件、290 个前端源码文件，约 12.45 万行 Go/TS/Vue；Go 329 个测试文件，前端 68 个测试文件。
- 装配：`main.go` 674 行，根目录 `wire_*.go` 约 1,700 行；大量 `Configure...`/`Set...` 在构造后补依赖。
- 核心依赖：`internal/node` 反向依赖 `internal/services/expr` 与 `internal/services/llm`，核心契约知道具体服务层。
- 运行时接口：`internal/node/interfaces.go` 的 `ServiceBundle` 聚合十余种能力；`VisionService` 是一个多算法宽接口；`runtime/node_services.go` 超过 1,100 行。
- 前端协议：`frontend/src/lib/backend.ts` 手写 DTO 与 100+ RPC 门面；Wails bindings 是 gitignored 生成物；pin type compatibility 还有前后端平行实现。
- 前端集中度：`ContainerEditorView.vue` 1,652 行，`NodeInspector.vue` 1,207 行，承担状态、协调、RPC、业务规则和渲染。
- 基线验证：Go test/vet/staticcheck、Vitest 529 tests、vue-tsc、i18n 和 production build 通过；`pnpm format:check` 对 188 个文件失败。
- CI 真实性：工作流没有运行 Vitest、typecheck、i18n、format check；Go coverage 只上传不设阈值。现有 coverage profile 总计 64.4%，root composition 与多个 platform adapter 覆盖偏低。
- 产物：主 chunk 977 KB、icons chunk 2.10 MB、ContainerEditor chunk 2.74 MB（gzip 859 KB），只有 warning，没有预算门禁。
- 可复现性：`@wailsio/runtime` 使用 `latest`，lock 当前解析到 `3.0.0-alpha.79`；Go/Wails CLI pin 为 `v3.1.0-alpha2.117`，验证脚本未检查前端 runtime。GitHub Actions、Rust `stable`、Task `3.x` 也未固定到不可变来源。
- 发布：当前 LICENSE 禁止商业使用，不是 OSI 开源许可证；release 只上传未签名的 `Yotta.exe`，没有 checksum、SBOM、provenance 或正式 smoke gate。
- 治理：缺少 CODEOWNERS、issue/PR templates、治理/维护者、支持、发布流程和 changelog 文件。

## 目标模块图

```text
main.go
  └─ internal/appbootstrap          构造完整应用，返回可 Start/Close 的 Application
       ├─ internal/workflow         工作流聚合
       │    ├─ schema               唯一 WorkflowSource v3 与生成 contract
       │    ├─ catalog              显式节点注册与生成 catalog
       │    ├─ compiler             draft → diagnostics + immutable ProgramSnapshot
       │    └─ runtime              只执行 Snapshot；拥有 adapter 和 dispatch state
       ├─ internal/workspace        container/subgraph/asset 的事务与引用完整性
       ├─ internal/execution        持久 Run/Attempt 账本、Worker 与时间线
       ├─ internal/automation       target、capability 与真实 controller adapter
       ├─ internal/recording        录制 session 与资产落库
       ├─ internal/scheduling       调度生命周期
       ├─ internal/ai               model/prompt/schema/tool/session/eval/trace/provider
       ├─ internal/extensions       manifest、制品锁与未来进程外 Runner
       ├─ internal/presentation     Wails RPC/event/window adapters
       └─ internal/security         Trust、MCP、credential 与 capability policy

frontend/src/app
  ├─ transport                     只包装生成 bindings、typed error 和 event
  ├─ contracts                     从 Go schema/catalog 生成，禁止手写镜像 DTO
  ├─ editor/EditorSession          draft/history/path/dirty/save/conflict/validation 状态机
  ├─ editor/InspectorRegistry      schema-driven 字段渲染与少量真实扩展点
  └─ views                         只负责布局、组合和交互委托
```

## 深模块与 seam

### Application

`appbootstrap.Build(Config) (*Application, error)` 一次完成依赖装配。`Application` 的外部 interface 只保留 `Start(context.Context)`、`Close(context.Context)` 与 presentation service 列表。所有依赖经 constructor 传入并在构造时校验；删除包级 `Configure...` 和构造后的 setter。

`internal/appruntime.Runtime` 保留并成为 Application 的内部生命周期实现。`main.go` 只处理进程级输入、调用 Build/Start、运行 Wails、调用 Close 和映射退出码。

### Workflow

`compiler.CompileDraft(Source, CatalogSnapshot) CompileResult` 是 strict decode、节点解析、子图闭包、pin 类型、effect/capability 和 permission assembly 的单一入口。它直接编译内存草稿，不要求先保存。成功时产生带 canonical `sourceHash/programHash/catalogHash/compilerBuild/nodeLocks` 的不可变 `ProgramSnapshot`；runtime 只执行 Snapshot，不再运行时按 ID 读当前文件、猜 schema 或修复输入。

节点 catalog 改为显式 builder；删除依赖 blank import + `init()` + global Freeze 的生产装配。测试构造局部 catalog，生产使用 generated `catalog.All()`。

`NodeSpec` 同时声明 effect、determinism、retry safety、cache policy、capabilities、secrets 与 sensitive fields。一套 Spec 生成 Go/TS/JSON Schema/Inspector/catalog/docs/test fixtures；宿主负责硬校验，节点不得覆盖能力、Secret 流向或 retry/cache 规则。

端口由消费者拥有。`internal/node` 不再 import `services/*`；LLM、Expr、Vision 等类型移动到中立 contract package。把宽 `VisionService` 按真实消费者拆为 template matching、frame analysis、QR decoding 等窄 interface；同一个 production adapter 可实现多个 interface，测试 adapter 是第二个真实 adapter。

### Workspace

Workspace 统一拥有 container、subgraph、asset 与 blob 的写事务、引用索引和 GC。外部 interface 以业务操作为单位，而不是暴露 Store 后再调用多个 setter。加载损坏数据返回显式 error/result，不把 incompatible placeholder 当成功对象。

Yotta 3.1 只接受 v3 epoch。所有 schema 必填、拒绝 unknown/zero version；删除 lock v1、旧单文件 container、旧 subgraph marker、顶层 config fallback 和启动期 legacy rename。

### Execution

保留符合桌面输入排他性的单机串行 Worker，但内存队列不再是事实源。`RunStore` 事务持久化 QUEUED/RUNNING/SUCCEEDED/FAILED/CANCELLED/INTERRUPTED 与每个 NodeAttempt；应用崩溃后遗留 RUNNING 只转 INTERRUPTED，不自动重放 input/process/file/network/LLM 等副作用。

Run 入队时锁定 `ProgramSnapshot` 与 `permissionGrantHash`。workflow 保存历史、program 制品与 run 历史是三个独立对象，通过 revision/hash 关联。pure 节点可由宿主管理缓存；unsafe effect 默认一次执行。

### EditorSession

编辑器的 interface 是意图：`load`、`apply(command)`、`undo/redo`、`validate`、`save`、`enter/leaveGraph`、`run/debug`。Vue view 不直接编排多个 store、RPC 和 watcher。该模块的测试面覆盖跨图层状态、dirty、并发 rev、录制插入和保存冲突。

Inspector 从 generated node schema 驱动。只有确实具有独特交互的 widget 才注册 adapter；删除 `pinSpec.ts` 的 legacy re-export、kind switch 和前后端平行 pin compatibility。

### Security policy

MCP 默认关闭；启用时显式配置 endpoint，仍只允许 loopback，并要求 session token。危险能力采用统一 permission manifest：文件 root、网络 host、进程执行、输入/截图和脚本绑定均在运行前展示并确认。Script/goja 仍不宣称为 sandbox，但不得默认获得所有可绑定节点。

导入 workflow 默认 untrusted，只允许查看和编译。API key/secret 由 OS credential store 保存，Source 只含 SecretRef；UI、日志、trace、诊断包和导出均不得接触明文。GUI、headless、AI、MCP 与未来 Runner 使用同一 CapabilityPolicy。

## 明确删除

- `backupLegacyDataIfNeeded` 与所有 2.x 数据识别/迁移测试。
- `YHFISH_ADB_PATH`、旧 pin/config key、旧 route/RPC、legacy re-export。
- schema version 为 0 时自动补值、旧 lock 自动升级、读取失败返回 incompatible + nil。
- 构造后 `Configure...` 注入、全局 mutable registry、未显式 owner 的 goroutine。
- WGC→GDI 等静默降级；若保留 `auto`，它必须是用户可见的明确策略并产生结构化选择结果。
- 手写 Go↔TS DTO、RPC 数量字符串断言、平行 pin-type 实现。
- 前端 raw `any`/Go `map[string]any` 在跨层 contract 中的无约束使用；动态 JSON 仅保留在明确的 JSON value 边界。

## 不做的事情

- 不承诺第三方 Go plugin ABI。3.1 先把 in-tree 官方 Node SDK 做稳定、可生成、可测试；第三方执行插件只使用版本化 Wasm/IPC host + capability broker。
- 不把 Linux/macOS compile gate 宣称为完整产品支持；没有宿主 smoke、签名和权限 UX 前继续标 preview。
- 不为“架构好看”创建只有一个 adapter 的 port。
- 不在同一 PR 中同时搬包、改行为、改 schema 和重写 UI。
