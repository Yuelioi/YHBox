# Yotta 3.0 破坏性升级实施方案

## 总原则

每一阶段必须落一个可验证的纵向切片；先建立会失败的门禁，再改实现。旧路径删除与新路径落地必须在同一阶段完成，不留长期双轨。

## Phase 0 — 产品与法律决策

目标：冻结 3.0 的外部承诺，避免工程完成后仍不能公开发布。

1. 权利人选择 OSI 许可证。默认建议 Apache-2.0（包含明确专利授权）；若必须要求网络服务修改公开，再单独法律评估 AGPL-3.0。
2. 确认 3.0 只支持新数据 epoch，发布说明明确“2.x 数据不可直接打开”；升级前备份/导出由旧版完成，新版不带迁移器。
3. 支持矩阵固定为 Windows stable、Linux/macOS preview；Android 是 target，不是 host。
4. MCP 默认关闭、Script/文件/网络/进程能力需要显式授权。
5. 版本升为 3.0.0；module identity 继续使用 `github.com/yottaapp/yotta`。

验收：LICENSE/README/SECURITY/platform-support/compatibility 文档表述一致，不再同时出现“开源”与非 OSI 限制。

## Phase 1 — 先让门禁说真话

目标：任何后续重构都不能靠 CI 漏项假绿。

1. 一次性运行 oxfmt，提交纯格式化基线；之后 CI 执行 `format:check`。
2. 新增独立 frontend job：frozen install、Vitest、vue-tsc、i18n、oxlint/eslint check-only、format check、production build。
3. 将 lint scripts 拆成 `lint`（只检查）与 `lint:fix`，禁止 CI 或 pre-commit 暗改源码。
4. Go job 保留 test/vet/staticcheck；coverage 设置仓库最低 65% 起步、关键包单独阈值，逐阶段提升到 75%。root bootstrap、recording、input/capture、MCP 不得下降。
5. race 列表改成可维护的 package group；为 parser/package/MCP fuzz 增加固定时长 smoke job。
6. production bundle 增加机器可读预算：entry gzip ≤350 KB、editor gzip 第一阶段 ≤650 KB，最终 ≤450 KB；单个图标资源不得打包全集。
7. 删除“14 Services, 107 Methods”日志字符串断言，改为生成 contract manifest 并用 git diff/校验器检测变更。

验收：本地与 CI 使用同一 task；当前 188 个格式失败归零；所有 documented gates 在 workflow 中可找到实际调用。

## Phase 2 — 工具链与供应链可复现

目标：同一 commit 在干净环境解析到同一工具和依赖。

1. 将 `@wailsio/runtime: latest` 改成与当前 Wails release 明确匹配的精确版本，并让 `verify-wails-version.ps1` 同时核对 Go library、CLI、frontend runtime、CI 和 README。
2. 固定 Rust toolchain 文件；固定 Task 版本；所有 GitHub Actions 使用完整 commit SHA，并由 Dependabot 更新。
3. release 生成 CycloneDX/SPDX SBOM、SHA-256 checksums、GitHub artifact attestation/provenance。
4. Windows exe/installer 在 release job 中 Authenticode 签名并 timestamp；签名失败必须阻断发布。
5. 增加 CodeQL、OpenSSF Scorecard 与 dependency review；保留 gitleaks、govulncheck、cargo-deny 和 license allowlist。

验收：两次 clean checkout 的 lock、bindings contract、hash 清单一致；release 不再上传裸的无签名单文件。

## Phase 3 — v3 contract 与数据 epoch

目标：先定义唯一新世界，再删除所有旧世界读取路径。

1. 定义 v3 container/package/graph/subgraph/installation/lock schema；生成 JSON Schema 与 TS contract。
2. decoder 使用严格字段与版本检查。未知字段、缺失 version、旧 version、hash 不一致均返回 typed error。
3. 删除 lock v1 自动升级、schema=0 normalization、旧 marker/config/pin/key fallback、legacy environment variable 和启动期数据目录 rename。
4. 删除对应迁移 fixture 和 back-compat tests，替换成“旧格式必须被拒绝”的 contract tests。
5. error code、event payload、RPC request/response 一并版本化；禁止前端 camel/Go 字段双读。
6. 提供清晰的启动阻断页：发现旧 data epoch 时显示版本、路径和“使用 2.x 处理或移走目录”的说明，但不读取或转换内容。

验收：仓库内搜索 `legacy|back-compat|backward-compat` 只允许出现在 release note/拒绝测试；新 runtime 没有 dual-read/dual-write。

## Phase 4 — 重建 composition root

目标：把当前约 2,500 行根装配收进可测试的 Application 深模块。

1. 新建 `internal/appbootstrap`，用 Config + constructors 构造所有 store、runner、daemon、MCP、hotkey、recording、presentation。
2. 各模块 constructor 一次接收完整必需依赖；删除 `ConfigureRunner`、`ConfigureEmitter`、`Configure...Scanner`、`Set...Factory` 等后置注入。
3. `Application` 持有 `appruntime.Runtime`，统一 Start/Close；构造失败不启动 goroutine，不调用 `log.Fatal/os.Exit`。
4. `main.go` 缩到约 100 行，只做进程入口、Wails run 和退出码映射。
5. bootstrap 用 fake presentation/platform adapters 做启动失败 rollback、关闭逆序和 wiring completeness 测试。

验收：除 main 外的 production package 不调用 `os.Exit`/Fatal；所有 background resource 都由 Application 生命周期拥有；root wiring statement coverage ≥70%。

## Phase 5 — Workflow 深模块

目标：让节点作者只理解 node contract，让执行复杂度集中在 compiler/runtime。

1. 将 schema、catalog、compiler、runtime 明确分层；`internal/node` 不再 import `internal/services/*`。
2. 用显式 `catalog.All()` 替代 blank import + `init()` global registry；生成 catalog snapshot 给前端与文档。
3. 把 validate、dependency closure、pin/capability assembly 合并到 `compiler.Compile`；runtime 只接 immutable Program。
4. 清理 `ServiceBundle`：端口由消费者定义，按 template/frame/QR/input/window/LLM 等真实能力拆窄；production 与 test adapter 都通过相同 seam。
5. 将 `runtime/node_services.go` 按 adapter 归属移入内部目录；对外只暴露构造 runtime 所需的少量 interface。
6. 删除旧 shallow module 的细粒度实现测试，用 compiler/runtime interface 上的行为测试替代；保留算法纯函数测试。
7. 给 compile/dispatch/large graph 添加 benchmark 与 allocation budget。

验收：删除一个 adapter 模块时复杂度不会散回节点；节点 package 不依赖 services/container/pkg platform 实现；Program 可并发复用且 runtime state 每次执行隔离。

## Phase 6 — Workspace 与资产一致性

目标：让多文件、多 store 操作成为一个业务事务，而不是靠回调拼接。

1. `workspace` 统一拥有 container、global subgraph、asset/blob 索引和引用查询。
2. Save/Delete/Import/RecordingFinalize 以单个 command 执行并返回 committed generation；事件只在 durable commit 后发布。
3. GC 从启动期/删除后 callback 改为 workspace transaction 后台任务，使用 generation snapshot，具备 dry-run/metrics。
4. `incompatible Container + nil` 改为显式 `LoadResult`/error；列表 UI 单独展示 corrupt entry，不把它当正常 Container。
5. 对 crash point、并发 save/export/delete、磁盘满、fsync/rename failure 做 fault-injection tests。

验收：没有跨 store 直接写另一个 owner 的磁盘目录；任何事件都对应已提交 generation；损坏数据不会进入 runner。

## Phase 7 — 前端 contract 与编辑器重构

目标：删除协议镜像，把两个巨型 Vue 文件变成深模块调用者。

1. 从 Go v3 schema/Wails contract 生成 TS DTO、RPC 和 event types；`backend.ts` 只保留 transport/error adapter。
2. 删除 `as any` 驱动的 RPC 参数、Go/camel 双字段 normalize、手写 Graph/Container/Schedule DTO。
3. 建立 `EditorSession` 状态机，统一 draft、history、dirty、graph path、subgraph rev、validation、save/run/debug。
4. `ContainerEditorView` 只做布局和 command 绑定；目标 <500 行。业务 watcher 移入 EditorSession 并用确定性测试覆盖。
5. Inspector 改 schema-driven renderer + `InspectorRegistry`；目标 <400 行。仅保留有真实第二实现/测试 adapter 的扩展 seam。
6. node catalog、defaults、pin types、widgets、labels key 全从 backend Spec 生成；删除 `pinSpec.ts` legacy re-export 和 TS parity implementation。
7. bundle：图标改按需集合；ELK、CodeMirror、Prettier、脚本补全、复杂 inspector 按功能 lazy-load；达成 editor gzip ≤450 KB。
8. 增加 Playwright web harness 覆盖容器列表、编辑、校验、保存冲突、录制落点；Windows Wails smoke 覆盖实际 bindings/event/window。

验收：跨层 DTO 只有一个来源；views 不直接拼 patch JSON；529 个现有测试迁移后保持或提升，新增 EditorSession contract tests。

## Phase 8 — 安全能力模型

目标：大型开源桌面自动化项目默认 fail closed。

1. MCP 默认不启动；设置中显式启用、端口可配、loopback 强制、session token 必需、armed 仍保留。
2. workflow 编译生成 permission manifest：filesystem roots、network hosts、process、input、capture、LLM、script bindings。
3. 首次运行或权限扩大时确认；headless/MCP 执行必须显式传允许的 policy，不继承 UI 隐式状态。
4. Fetch 加 URL policy、redirect 后重检、DNS rebinding 防护与 response budget；file nodes 用 canonical root policy；日志统一 secret redaction。
5. Script 默认只绑定 pure/data nodes；危险 binding 按 manifest 明确授权。文档继续声明 goja 不是安全 sandbox。
6. 对 MCP auth、path traversal、SSRF、zip bomb、permission escalation 增加 negative/fuzz tests。

验收：默认启动没有监听端口；无授权 workflow 在执行前失败；错误精确指出 capability 与来源节点。

## Phase 9 — 开源协作与发布

目标：技术能力、治理承诺和发布产物一致。

1. 增加 CODE_OF_CONDUCT、GOVERNANCE/MAINTAINERS、CODEOWNERS、SUPPORT、ROADMAP、CHANGELOG、RELEASING。
2. 增加 issue forms、security redirect、PR template；PR 必填设计、平台、权限、数据格式和验证。
3. 推荐 DCO + sign-off，避免早期 CLA 摩擦；是否要求 CLA 由权利人另决。
4. 文档站提供 architecture decision records、node author guide、security model、data format、release support matrix。
5. release 只从通过 required checks 的 protected tag/environment 发布；生成 Windows signed installer/exe、checksums、SBOM、provenance 和 release notes。
6. Linux/macOS 只发布明确标记 preview 的 artifact；完成真实宿主 smoke、签名与权限 UX 后再升 stable。
7. 启用 GitHub private vulnerability reporting，验证入口后才公开仓库。

验收：新贡献者能只凭仓库文档完成环境搭建、添加节点、运行全套门禁和提交 PR；发布可验证来源与签名。

## 推荐提交/PR 顺序

1. `chore(frontend): establish formatting baseline`
2. `ci: enforce frontend and coverage gates`
3. `build: pin complete toolchain and dependencies`
4. `docs!: define Yotta 3.0 contracts and license`
5. `feat(schema)!: replace persisted formats with v3 epoch`
6. `refactor(app): introduce constructor-complete bootstrap`
7. `refactor(workflow)!: add explicit catalog and compiler`
8. `refactor(runtime): narrow execution ports`
9. `refactor(workspace)!: unify durable asset transactions`
10. `refactor(frontend)!: generate transport contracts`
11. `refactor(editor): introduce EditorSession`
12. `perf(frontend): enforce editor bundle budgets`
13. `feat(security)!: require explicit capability policy`
14. `chore(release): add signed reproducible releases`
15. `docs: complete open-source governance`

每个 PR 都必须删除对应旧路径；不得以“下一阶段再清理”为理由保留 shim。

## 完成定义

- LICENSE 为已选 OSI 许可证，公开定位准确。
- 所有本地门禁与 required CI checks 一致且全绿。
- v2 数据、RPC、pin/config、环境变量和 migration code 不存在。
- main <150 行；Application constructor-complete；无 production Configure injection。
- node core 不依赖 services；前端 contract 由 Go 生成；无平行 pin compatibility。
- ContainerEditorView <500 行，NodeInspector <400 行，editor gzip ≤450 KB。
- MCP 默认关闭并带 token；危险能力由 permission manifest 阻断。
- Windows release 已签名，附 checksum、SBOM、provenance；安全报告入口可用。
- architecture、contributor、release、security、support 文档可由新维护者独立执行。
