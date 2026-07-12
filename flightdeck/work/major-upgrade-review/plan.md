# Yotta 3.0 破坏性升级实施方案

## 方案定位

这是一轮产品、协议、架构、AI、供应链和项目治理同时换代的 major release，不是兼容性重构。目标是把 Yotta 从“功能丰富的本地工作流编辑器”升级为：

> **本地优先、可审计、可扩展的 AI 自动化工作台；人类与 AI 共用强类型工作流编译器、权限模型和运行事实。**

本方案综合：

- 当前仓库的 Go/Wails/Vue、持久化、运行时、MCP、LLM、CI 与 release 审查；
- n8n、Node-RED、Windmill、Temporal、VS Code、ComfyUI 的官方机制；
- OpenAI、Anthropic、MCP 的当前官方 prompt/tool/eval/安全实践；
- OpenSSF、SLSA、GitHub、DCO、CNCF 等开源治理与供应链标准。

详细依据见 `design.md`、`ai-native-design.md` 和 `research/` 下三份报告。

## 不可妥协的 14 个决定

1. **只接受 v3 epoch。** 不读、不迁移、不修复 v2 数据；旧格式在 parse boundary 返回稳定错误。
2. **编译内存草稿。** `CompileDraft(source, catalog)` 不要求先保存，也不按 ID 偷读磁盘版本。
3. **运行锁定不可变快照。** `StartRun` 只接受 `ProgramSnapshot/programHash`；入队后编辑器变化不能影响它。
4. **不保留最低公分母 LLM。** OpenAI/Anthropic 各走原生 API；缺 capability 就拒绝，不用 prompt 模拟。
5. **删除 JSON prompt fallback。** structured output 必须由完整 schema 和 provider-native 能力保证。
6. **动态数据不进入 system/developer。** 所有不可信值只进入 typed user/context/tool-result。
7. **AI 不直接改整图 JSON。** UI、AI、CLI 都调用带 revision 的领域 patch 与同一个 Compiler/Workspace。
8. **副作用由 effect 建模。** input/process/file/network/LLM 默认不自动重试、不缓存、崩溃后不透明重放。
9. **保留单机串行 Worker。** 用持久 Run/Attempt 账本补可靠性，不引入 Redis/PostgreSQL/Temporal 集群。
10. **一个 NodeSpec 是事实源。** TS、Inspector、catalog、schema、docs、fixtures 全生成。
11. **第三方执行代码不上主进程。** 先做官方 Node SDK；插件必须等进程外 Runner、capability broker 和制品锁。
12. **默认 fail closed。** MCP、插件、危险 workflow 和遥测默认关闭或受限；权限由宿主执行，不靠 prompt。
13. **项目身份只有一个。** LICENSE、module、组织、仓库、更新源、漏洞入口和 provenance 必须一致。
14. **CI 与发布是合同。** 文档声明的 gate 必须真实运行；stable 产物必须完整、签名、带 SBOM/checksum/provenance。

## 总体里程碑

| 里程碑 | 对外含义 | 必须完成的 Wave |
| --- | --- | --- |
| M0 — Source Open | 可以诚实称为开源并接受贡献 | 0–2 的 source gate |
| M1 — Core Alpha | v3 workflow/compiler/runtime 可端到端运行 | 3–7 |
| M2 — AI Alpha | AI authoring 在权限与 eval 门禁内可用 | 8–10 |
| M3 — Extension Preview | 官方 Node SDK 完整；第三方声明式包可试用 | 11 的 A/B 门 |
| M4 — 3.0 Stable | 可信 Windows release 与公开治理成立 | 12；全量完成定义 |

任何里程碑都不允许用 feature flag 保留旧实现作为生产兜底。分支上的短期脚手架可以存在，但合并纵向切片时必须删掉对应旧路径。

## Wave 0 — 先解决“是否真是大型开源项目”

目标：在继续制造 release 之前统一法律、身份与控制面。

1. 权利人选择 OSI 许可证。默认建议 Apache-2.0；若要求网络服务修改公开，再单独法律评估 AGPL-3.0。
2. 统一 canonical identity 为一个 GitHub org/repo/module/update/security URL；清理 `Yuelioi/YHBox`、`Yuelioi/Yotta`、`yottaapp/yotta` 分裂。
3. 在公开本地领先历史前做 secret、license、large binary、author/ownership audit；确认公开主线包含实际开发历史。
4. 在 release 链重建前冻结新的 stable tag；当前 `push v*` 不得继续自动发布裸 exe。
5. 启用 main/tag rulesets、required checks、禁止 force push/delete、至少双人管理、2FA 和 protected release environment。
6. 增加最小治理合同：GOVERNANCE、MAINTAINERS、CODEOWNERS、CODE_OF_CONDUCT、SECURITY、SUPPORT、ROADMAP、RELEASING、DCO。
7. 启用并实测 private vulnerability reporting；指定 security responder、备份人和撤销/轮换流程。
8. 写明 v3 无迁移支持、Windows stable、Linux/macOS preview、Android 为 target 而非 host。

验收：LICENSE 是 OSI 文本；README/Go module/release/update/SECURITY 指向同一身份；main/tag 不能由单账号无审查改写；公开仓库就是实际开发主线。

## Wave 1 — 让工程门禁和 agent 指令说真话

目标：后续重构不能依赖 CI 漏项或模型猜仓库规则。

1. 一次性运行 oxfmt，提交纯格式化基线；随后 CI 强制 `format:check`。
2. frontend job 执行 frozen install、529+ Vitest、vue-tsc、i18n、lint check-only、format check、production build。
3. Go job执行 test、vet、staticcheck；coverage 从仓库 65% 起步并按关键包单设阈值，逐步提升到 75%。
4. 增加 race package group、parser/package/MCP fuzz smoke、Windows Wails smoke 和跨平台 compile gate。
5. 建立 bundle 预算：entry gzip ≤350 KB；editor 先 ≤650 KB、最终 ≤450 KB；禁止打包整套图标。
6. 用 generated contract manifest 取代“14 Services, 107 Methods”字符串断言。
7. 新建 tracked、简短的 `AGENTS.md`：架构入口、生成物边界、必跑命令、Flightdeck、危险操作；provider wrapper 只薄引用。
8. 删除/替换被 gitignore 且已漂移的本地 `CLAUDE.md` 规范，不再写旧名 `YHFish`、直接提交 main 或重复硬审批流程。
9. 把可机械验证的规则移进 lint/schema/test，不用超长 prompt 反复强调。

验收：当前 188 个格式错误归零；本地 `task check` 与 required CI 同源；新 agent/贡献者无需读 main.go 就能找到正确入口并跑全套 gate。`yotta dev ...` CLI 留在 Wave 7 随节点开发工具一起建立，不为尚不存在的入口增加 shim。

## Wave 2 — 固定工具链并重建供应链底座

目标：同一 commit 在干净环境得到同一依赖、bindings、catalog 和未签名产物。

1. 把 `@wailsio/runtime: latest` 改成与 Go library/CLI 匹配的精确版本；版本校验覆盖五处来源。
2. 固定 Rust toolchain、Task、pnpm/Node、Go/Wails；GitHub Actions 全部用完整 commit SHA，由 Dependabot 更新。
3. release build 禁止 `go mod tidy` 或非 frozen install；结束后必须 `git diff --exit-code`。
4. 生成 SPDX/CycloneDX SBOM、third-party notices、SHA-256 checksums 和 GitHub build attestation。
5. 增加 CodeQL、dependency review、secret scanning/push protection、Dependabot security updates、OpenSSF Scorecard。
6. 建立 Windows 完整 staging tree 和 artifact manifest；installer/portable 内容由同一 staging 产出。
7. 两次独立 clean build 对比未签名 payload，逐项消除时间、路径、排序和工具链差异。

验收：依赖与 Action 不含可变 tag/`latest`；clean builds 的生成契约与未签名 payload hash 一致；release job 尚未签名也已能证明来源。

## Wave 3 — 建立 v3 Source → Compiler → ProgramSnapshot 主链

目标：先冻结最不可逆的协议，再让所有客户端迁入。

1. 定义唯一 `WorkflowSource v3`：workflow/revision/graphs/variables/secret refs/requested capabilities；生成 JSON Schema 与 TS。
2. 定义稳定 `Diagnostic`：code、severity、graphPath、nodeId、fieldPath、params、optional fix；message 仅用于展示。
3. `CompileDraft(Source, CatalogSnapshot) -> CompileResult` 合并 strict parse、reference closure、pin/type、config、effect/capability 与 permission compilation。
4. 产出 immutable `ProgramSnapshot`：sourceHash、programHash、catalogHash、compilerBuild、node/plugin locks、required capabilities、executable graphs。
5. canonical encoding 排除 timestamp/路径等非语义字段；Windows/Linux/macOS 对同一输入产生相同 hash。
6. `StartRun` 只接受 snapshot/hash；Worker 禁止按 workflow/container ID 重新读取、normalize 或编译当前文件。
7. 删除 schema=0 normalization、lock v1 upgrade、旧 pin/config/key/marker、旧环境变量、legacy rename 和 v2 fixture 的成功路径。
8. 旧格式只有 `UNSUPPORTED_WORKFLOW_FORMAT` 等拒绝测试；不提供 runtime fallback、dual-read/write 或 silent migration。
9. 修复 MCP 示例 `graph.version` 与真实 `graph.schemaVersion` 漂移；所有示例作为 strict contract fixture 编译。

验收：未保存草稿可直接检查；入队后继续编辑不影响 Run；v2 在边界失败；仓库不存在 Worker 运行时按 ID 读取并临时编译的路径。

## Wave 4 — 重建 Application 与 headless seam

目标：把约 2,500 行根装配收进可测试深模块，并让 GUI/CLI/MCP 共用 application commands。

1. `internal/appbootstrap.Build(Config)` constructor-complete 地构造 Application；删除生产 `Configure...`、`Set...Factory` 与 package global registry。
2. `Application` 统一拥有 Start/Close、goroutine、store、worker、scheduler、recording、MCP 和 presentation；失败回滚逆序可测试。
3. `main.go` 缩至约 100–150 行，只处理进程输入、Wails run 和退出码。
4. 建立 application command surface：CompileDraft、SaveSource、StartRun、CancelRun、GetRunTimeline、GetCatalog 等，不把 repository service 原样暴露给前端。
5. 增加 `yotta validate/compile/run/inspect` headless CLI；它使用相同 Application/Compiler/Policy，不复制业务逻辑。
6. `internal/node` 不再 import `services/*`；按消费者拥有 port，拆窄 Vision/LLM/Input/Window 等宽接口。

验收：除 main 外无 Fatal/os.Exit；所有后台资源有 owner；GUI/CLI/MCP 调用同一命令；bootstrap wiring statement coverage ≥70%。

## Wave 5 — Workspace 事务与本地 Run 事实库

目标：持久化源、资产和执行事实，保留合理的单 Worker 桌面约束。

1. Workspace 统一拥有 workflow/subgraph/asset/blob 的 revision、引用完整性、事务、GC 与 durable event。
2. Save/Delete/Import/RecordingFinalize 原子提交并返回 generation；事件只在 durable commit 后发布。
3. WorkflowStore、ProgramStore、RunStore、SecretStore、ArtifactStore 分责；不得用一个通用 repository 隐藏不同一致性要求。
4. 用 SQLite/WAL 或等价事务存储实现 RunRecord，ID 改 UUIDv7；状态至少 QUEUED/RUNNING/SUCCEEDED/FAILED/CANCELLED/INTERRUPTED。
5. NodeAttempt/adapter action 记录 graphPath、node、effect、attempt、时间、error code、脱敏摘要与 programHash。
6. 写入 QUEUED 与 snapshot 引用成功后才通知 Worker；启动时遗留 RUNNING 转 INTERRUPTED，绝不自动重放 unsafe effect。
7. 定义 effect 矩阵：pure/read/write/external/input/human；Compiler 阻止 unsafe retry/cache。
8. 增加 crash point、磁盘满、rename/fsync、并发 save/delete/export、kill/restart 的 fault-injection tests。

验收：没有幽灵 Running；强杀应用不会重复点击/输入/进程/文件/HTTP/LLM；损坏数据不进入 runner；每个 UI 事件对应已提交 generation。

## Wave 6 — EditorSession 与生成 contract

目标：将巨型 Vue 协调层变成明确的编辑状态机客户端。

1. 从 Go v3 schema/Wails contract 生成 DTO、RPC/event、catalog 与 Inspector schema；`backend.ts` 只做 transport/typed error。
2. `EditorSession` 统一 draft、revision、sourceHash、compiledHash、lastRunHash、history、dirty、graph path、save conflict、validate、run/debug。
3. SaveSource 必须带 baseRevision；外部 Git/第二窗口修改时返回结构化 conflict。
4. 区分“未保存”“未重新编译”“预览结果已陈旧”；workflow history 与 run history 分开。
5. `ContainerEditorView` 目标 <500 行，`NodeInspector` <400 行；view 不直接拼 patch JSON 或编排多个 store/RPC。
6. Inspector = generated renderer + 少量真实 adapter；删除 legacy re-export、kind switch、Go/TS 平行 pin compatibility。
7. 图标、ELK、CodeMirror、Prettier、复杂 inspector 按功能 lazy-load，达成 editor gzip ≤450 KB。
8. Playwright web harness 覆盖列表、草稿编译、保存冲突、录制插入、AI patch diff、run timeline；Windows smoke 覆盖真实 binding/event/window。

验收：跨层契约只有一个来源；Run 入队后的编辑不改变快照；529 个既有测试保持或提升；编辑器核心状态可无 Vue 渲染测试。

## Wave 7 — NodeSpec、官方 Node SDK 与运行语义

目标：节点贡献成为可生成、可验证的产品接口。

1. 扩展单一 NodeSpec：ports/config，以及 effect、deterministic、cachePolicy、retrySafety、cancellation、timeout、capabilities、secrets、sensitive fields。
2. 宿主硬校验端口、range、能力、secret flow 与 effect/retry/cache 兼容；节点只能追加语义 diagnostics，不能绕过。
3. 用显式 `catalog.All()`/generated catalog 替换 blank import + `init()` mutable registry。
4. 建立 `yotta dev node new/lint/test`、fake capability kit、golden workflow、panic/error/cancel/secret-leak tests。
5. Go NodeSpec 生成 catalog JSON、JSON Schema、TS、Inspector controls、i18n/doc keys、node reference 和 fixtures；CI 生成后 diff 必须为空。
6. Compiler/Runtime interface 行为测试替代 shallow implementation tests；为 large graph/dispatch/canonical hash 加 benchmark/allocation budget。
7. pure 节点才允许宿主管理 cache；key 包含 kind/version/config/input/upstream/catalog/plugin digest。

验收：新增普通节点无需改中央 switch、手写 DTO/Inspector；unsafe 节点无法配置自动 retry/cache；敏感字段不会进入日志/trace/诊断包。

## Wave 8 — Capability policy、Workflow Trust 与 credential

目标：GUI、headless、AI、MCP 和未来插件共用 fail-closed 权限模型。

1. Compiler 从 NodeSpec 生成 permission manifest：filesystem roots、network hosts、process、input、capture、window enumeration、LLM、secret、script binding。
2. 导入 workflow 默认 untrusted：可查看/编译，不可运行危险能力；首次运行或权限扩大显示精确 delta 并确认。
3. headless/MCP 必须显式传 Policy/Grant，不继承 UI 隐式状态；grant hash 写入 RunRecord。
4. API key/secret 从 settings JSON 迁出到 OS credential store；前端只接触 SecretRef/masked metadata；导出默认排除 secret。
5. Fetch 实施 redirect 后复检、DNS rebinding、防内网、response budget；file 使用 canonical root；script 默认只绑定 pure/data capability。
6. 日志、trace、metrics、诊断包统一 secret/prompt/screenshot/路径脱敏和 retention policy。
7. 增加 SSRF、path traversal、zip bomb、prompt injection、permission escalation、secret exfiltration negative/fuzz tests。

验收：无授权 workflow 在执行前失败并指出来源节点；不存在明文 key 回传前端；权限不能被 prompt/tool result 或插件 manifest 自行提升。

## Wave 9 — AI provider、prompt、schema 与 eval 基座

目标：删除旧模型时代的启发式，把模型变化变成可测、可追踪的依赖升级。

1. 新建 `internal/ai/{modelpolicy,prompt,schema,tools,session,eval,trace,safety,provider}`。
2. 删除 `Provider.Chat -> Text` 最低公分母、`ModeAuto/Native/Prompt`、endpoint substring guessing 与 `structuredViaPrompt`。
3. OpenAI 官方 adapter 只走 Responses API；Anthropic 走原生 Messages；第三方 adapter 显式声明 capabilities，不冒充官方语义。
4. 用完整 JSON Schema 取代 flat `SchemaField`；strict required/enum/nested/nullable/constraints，默认拒绝 unknown fields。
5. 节点引用 `AISlot`/intent；installation 绑定 ModelProfile、固定 snapshot、capabilities、limits、budget 和 eval status。
6. PromptManifest 包含 goal/rules/tool policy/input/output schema/eval suite/version/hash；dynamic data 只能作为 typed low-trust block。
7. AI 节点拆为 Generate、Extract/Classify、Agent；structured result 整体校验、原子提交，缺字段不保留旧变量。
8. trace 记录 model/prompt/schema/toolset hash、usage/latency/cache/finish/request ID/approval lineage，并默认脱敏。
9. 建立真实任务 eval corpus 与确定性 grader；model/prompt/schema/toolset 任一升级都要比较 pass rate、安全失败率、p95 成本/延迟。

验收：仓库搜索不到 JSON fence/brace 截取与 prompt fallback；模型升级不能绕过 eval gate；trace 能复原一个 AI 决策使用的全部版本和授权而不泄露 secret。

## Wave 10 — AI authoring 与 MCP 3.0

目标：让更聪明的模型可靠地创作工作流，而不是更大胆地覆盖整份 JSON。

1. 实现 `catalog.search/describe`、`workflow.inspect/apply_patch/compile/explain_diagnostic/run_preview` typed authoring protocol。
2. patch 是 add/remove/configure/connect 等领域 command tagged union，带 baseRevision；服务端生成 ID/default 并保证 reference integrity。
3. authoring loop 固定为 goal/success criteria → discovery → minimal patch → compile → bounded repair → permission approval → diff/result。
4. Agent 必须有 iteration/tool/token/time/cost budget；外部副作用或权限扩大前进入宿主 approval gate。
5. machine catalog 与 UI 帮助文案分离；不把约 13.3 万字符、137 个节点一次送入上下文。
6. MCP 默认关闭、优先 stdio；HTTP 仅 authenticated loopback + 随机 token + origin/host 校验 + 生命周期 owner。
7. MCP tools 声明严格 input/output schema 与 structuredContent；大 catalog/schema/trace 用分页 resource。
8. 删除 `save_container(container_json)`、`validate_container(container_json)`、整 blob `get_graph_schema`；`list_windows` 默认不注册。
9. MCP 示例、prompt examples 和 eval fixtures 全部 strict compile，禁止 Normalize 掩盖漂移。

验收：AI/UI/CLI 对同一 patch 得到同一 revision/diagnostics；模型无法直接覆盖整图；MCP 默认无监听端口；恶意 tool result 无法修改 policy。

## Wave 11 — 扩展生态按三道门开放

目标：先改善作者体验，再决定是否承诺第三方执行平台。

**门 A — 3.0 必须：官方 Node SDK**

- 官方节点仍编进主程序；Node API 标 `v1alpha1`，不承诺 Go plugin ABI。
- 完成脚手架、lint/test、生成 contract、capability/effect/sensitive gate 和 node author guide。

**门 B — 3.x 可开放：声明式扩展**

- 主题、模板、workflow 示例、节点文档包；manifest、publisher、签名、不可变版本、deprecate/yank。
- 不加载第三方 Go/Python/JS，不允许插件修改 Vue DOM。

**门 C — 条件成熟后：进程外执行插件**

- PluginManifest + versioned IPC + out-of-process Runner；不用 Go plugin ABI。
- Runner 无直接 ServiceBundle，只向 capability broker 请求能力；宿主校验参数、审计、限时并可撤销。
- CPU/内存/时间/子进程/文件/网络限制；artifact digest 写入 ProgramSnapshot；崩溃只失败当前 attempt。
- 未达到隔离、签名、制品锁、撤权 UI、漏洞响应与 registry moderation 前，不发布门 C。

验收：3.0 不存在隐藏任意代码加载；声明式包无法越过宿主 UI/能力边界；可执行插件只有在全部安全 gate 后另行进入 preview。

## Wave 12 — 可信 release 与社区成熟度

目标：让 3.0 的二进制、治理承诺与真实控制面一致。

1. Windows staging 通过 Wails 启动、bindings、DLL、ADB、安装/卸载/升级 smoke；发布 installer + portable archive，不是裸 exe。
2. Authenticode 签名并 timestamp；签名失败阻断。附 checksums、SBOM、notices、attestation/provenance 和验证命令。
3. protected tag/environment 经非 self reviewer 审批后发布 immutable release；asset/tag 与 source commit 可验证。
4. Linux/macOS artifact 明确 preview；完成真实宿主 smoke、签名和权限 UX 后再升 stable。
5. 新贡献者只凭仓库文档完成 setup、添加节点、运行全 gate、签 DCO、提交聚焦 PR。
6. 建立 contributor → reviewer → maintainer ladder、ownership/offboarding、release shadow rotation、事故演练和密钥轮换。
7. 成熟度按 OpenSSF Passing → Silver 和 SLSA L2 → L3 推进；未满足多维护者/公开开发/可信发布前，不宣称“大型成熟开源项目”。

验收：干净机器可验证 release signature/checksum/attestation；至少两名有真实权限的维护者；安全入口、支持窗口、发布与撤销流程均实际演练。

## 推荐 PR / issue 编排

不要用一个“3.0 重写”大分支。每个条目是可独立评审的纵向切片，合并即删除对应旧路径：

1. `legal!: choose OSI license and canonical project identity`
2. `governance: enforce rulesets, ownership and vulnerability intake`
3. `ci: make documented frontend/go gates real`
4. `chore(frontend): establish one-time formatting baseline`
5. `build: pin complete toolchain and action digests`
6. `schema!: introduce WorkflowSource v3 and stable diagnostics`
7. `compiler!: compile drafts into canonical ProgramSnapshot`
8. `runtime!: execute snapshots only and reject v2`
9. `refactor(app): constructor-complete Application and headless commands`
10. `storage!: add workspace transactions and revision conflicts`
11. `runtime: persist RunRecord and NodeAttempt state machine`
12. `refactor(frontend)!: generate contracts and add EditorSession`
13. `node!: expand NodeSpec with effect and capability semantics`
14. `devx: ship generated official Node SDK workflow`
15. `security!: require Workflow Trust and capability grants`
16. `security!: move credentials to OS secret storage`
17. `ai!: replace generic chat with provider-native adapters`
18. `ai!: version prompts, schemas, tools and eval suites`
19. `ai: add typed workflow authoring loop`
20. `mcp!: replace whole-JSON tools with typed authoring protocol`
21. `perf(frontend): enforce editor and icon bundle budgets`
22. `release: ship signed reproducible Windows artifacts`
23. `docs: complete architecture, node authoring and operations guides`

每个 issue 必须写：行为变化、删除清单、schema/API diff、安全影响、测试/benchmark、文档、观测指标、rollback 方式。这里的 rollback 是回滚整个 commit/release，不是保留旧生产路径。

## 风险与控制

| 风险 | 控制 |
| --- | --- |
| 全面 breaking 导致长期不可发布 | 每 Wave 保持一个可运行纵向切片；v3 分支早期持续产出 nightly，不维护 v2 shim |
| schema/Compiler 一次改太多 | 先冻结 Source/Diagnostic/Snapshot contract，再分别迁移 runtime、UI、AI |
| AI 升级靠主观体验 | 固定 snapshot + PromptManifest + corpus + regression threshold |
| 能力模型拖慢功能开发 | effect/capability 先作为 NodeSpec 必填；由生成器和 fake kit 降低作者成本 |
| 插件野心扩大攻击面 | 3.0 只承诺官方 SDK；门 C 是独立 roadmap，安全条件不满足就不开放 |
| 开源名义先于公开事实 | Source open gate 阻断宣传与 stable release |
| 单维护者造成发布/密钥风险 | 双人规则、第二管理员、shadow rotation、offboarding 演练 |
| 文档再次与实现漂移 | 示例可编译、契约生成后 diff gate、tracked AGENTS、RECHECK 触发器 |

## Yotta 3.0 完成定义

- 法律/身份：OSI LICENSE；公开主线完整；canonical org/repo/module/security/update 一致。
- 工程：所有本地/required gates 同源全绿；format 0 failures；coverage/bundle budget 强制。
- 数据：只接受 v3；无迁移器、旧 key、dual-read/write、Normalize 修复边界输入。
- 编译：内存草稿可编译；ProgramSnapshot canonical、不可变、跨平台 hash 一致。
- 执行：Run/Attempt 持久；副作用不透明重试/重放；强杀后准确 INTERRUPTED。
- 架构：main <150 行；constructor-complete Application；node core 不依赖 concrete services。
- 前端：contracts 生成；EditorSession 成立；两个巨型组件达到行数目标；editor gzip ≤450 KB。
- 节点：NodeSpec 是唯一事实源；effect/capability/retry/cache/secret 由宿主强制；官方 SDK 可一命令使用。
- 安全：Workflow Trust、permission manifest、OS credential store；默认无 MCP listener；危险能力 fail closed。
- AI：provider-native API；无 JSON prompt fallback；prompt/schema/tool/model 可版本化、eval、trace；dynamic data 不进入高权限指令。
- Authoring：AI/UI/CLI 共用 typed patch/Compiler；catalog 按需发现；用户可见 diff、diagnostics、权限 delta。
- 扩展：3.0 不加载任意第三方代码；未来 Runner 必须进程外、制品锁定、经 capability broker。
- 发布：完整 Windows installer/portable、签名/timestamp、checksum、SBOM、provenance、immutable release、smoke 全绿。
- 社区：DCO、治理、CODEOWNERS、支持/安全/发布合同真实可执行；至少两名有实际权限的维护者。

只有这些条件同时成立，才发布 `v3.0.0`；不能用“模型已经足够聪明”替代任何 schema、权限、eval 或供应链门禁。
