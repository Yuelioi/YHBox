# Yotta 3.1 破坏性升级实施方案

## 方案定位

这是一轮产品、协议、架构、AI、供应链和项目治理同时换代的 major release，不是兼容性重构。目标是把 Yotta 从“功能丰富的本地工作流编辑器”升级为：

> **本地优先、可审计、可扩展的 AI 自动化工作台；人类与 AI 共用强类型工作流编译器、权限模型和运行事实。**

本方案综合：

- 当前仓库的 Go/Wails/Vue、持久化、运行时、MCP、LLM、CI 与 release 审查；
- n8n、Node-RED、Windmill、Temporal、VS Code、ComfyUI 的官方机制；
- OpenAI、Anthropic、MCP 的当前官方 prompt/tool/eval/安全实践；
- OpenSSF、SLSA、GitHub、DCO、CNCF 等开源治理与供应链标准。

详细依据见 `design.md`、`ai-native-design.md`、`research/` 下三份报告，以及仓库中的 [Node System 3.1 Wayfinder map](../../../docs/wayfinder/node-system-3.1/map.md)。

## 版本关系与合并原则

- **产品、节点系统和持久协议统一使用 3.1。** Yotta 3.1 是同一个升级项目，不再维护另一套节点版本号。
- 已删除未发布的 Workflow/Catalog/Program v3 wire contract；3.1 复用 strict parse、JCS、内容寻址和诊断预算原则，但使用新的 DTO/hash domain 并显式拒绝 v3，不保留双 runtime。
- Node System 3.1 替换此前方案里尚未进入生产 runtime 的 NodeSpec、类型、Program lowering、Authoring Projection 与扩展生态部分；旧 ContainerRunner 不得作为第二执行事实保留。
- 设计决策由 Wayfinder tickets 管理，实施波次与发布门只在本文维护。未关闭的 meta-schema、capability、resource、Program、authoring 与 plugin tickets 是实现 gate，不允许调用方自行猜协议。

## Node System 3.1 实施主脊

| 阶段 | 纵向结果 | 进入条件 | 删除/退出条件 |
| --- | --- | --- | --- |
| N0 — 契约冻结 | Data Type、Node Contract、Capability、Resource、Program、Plugin 与 Authoring 决策闭合 | Wayfinder frontier 逐票关闭 | golden corpus 能表达全部 contract，不再靠 Go/TS switch 补语义 |
| N1 — Contract Kernel | schema registry、canonical digest、validator、generator 与 Catalog 3.1 | Data Type + Node Contract 决议 | Go/TS/WIT/Proto/docs 投影均可追溯同一 generation |
| N2 — Concat tracer bullet | “拼接”端到端拥有 2 个 data input、1 个 data output、0 个 exec pin | Contract Kernel 可用 | Source 3.1 → Compiler → Program 3.1 → interpreter → Vue → docs 全链通过，前端不再伪造 `out` |
| N3 — Authoring Projection | schema 表单、类型/约束/生命周期/capability 提示与内置 Editor Adapter | authoring 原型通过 | Inspector/画布/MCP 不再维护平行端口或参数语义 |
| N4 — 内置节点迁移 | pure-data → effect → control/event 分批迁移 137 个节点 | Program/Run 与 capability 决议闭合 | 每批合并即删除对应 legacy spec/coercion/dispatch 分支 |
| N5 — Resource 与 Plugin Host | Blob Store、Stream、Resource Broker、Wasm/Process host 与包生命周期 | resource/plugin/lifecycle 决议闭合 | 两种示例插件通过相同 conformance、权限和崩溃清理测试 |
| N6 — 唯一运行链 | 所有 GUI/CLI/MCP/debug/schedule 只运行 Program Snapshot | 全目录 3.1 可编译 | 删除 ContainerRunner、Normalize、自定义 kind switch 与旧 Source/Catalog/Program reader |
| N7 — SDK/Docs/Release | SDK、示例包、生成文档、breaking diff、Windows stable 与跨平台 preview | 全部迁移门通过 | task check、race/fuzz、Windows smoke、跨平台 build、文档 drift gate 全绿 |

每个阶段都以可运行纵向切片合并；允许分支内短期脚手架，但主线不接受 dual-read、dual-write、隐藏 conversion 或两个生产 runtime。

## 不可妥协的 14 个决定

1. **3.1 stable 只接受 3.1 workflow/node/program epoch。** 当前 v3 Compiler artifact 是未发布脚手架；3.1 cutover 后不读、不迁移、不修复 v3/v2/legacy 数据，旧格式在 strict parse boundary 返回稳定错误。
2. **编译内存草稿。** `CompileDraft(source, catalog)` 不要求先保存，也不按 ID 偷读磁盘版本。
3. **运行锁定不可变快照。** `StartRun` 只接受 `ProgramSnapshot/programHash`；入队后编辑器变化不能影响它。
4. **不保留最低公分母 LLM。** OpenAI/Anthropic 各走原生 API；缺 capability 就拒绝，不用 prompt 模拟。
5. **删除 JSON prompt fallback。** structured output 必须由完整 schema 和 provider-native 能力保证。
6. **动态数据不进入 system/developer。** 所有不可信值只进入 typed user/context/tool-result。
7. **AI 不直接改整图 JSON。** UI、AI、CLI 都调用带 revision 的领域 patch 与同一个 Compiler/Workspace。
8. **副作用由 effect 建模。** input/process/file/network/LLM 默认不自动重试、不缓存、崩溃后不透明重放。
9. **保留单机串行 Worker。** 用持久 Run/Attempt 账本补可靠性，不引入 Redis/PostgreSQL/Temporal 集群。
10. **一个 NodeSpec 是事实源。** TS、Inspector、catalog、schema、docs、fixtures 全生成。
11. **第三方执行代码不上主进程，但 3.1 纳入最小 Wasm + Process Plugin Host。** 两者共用 capability broker、Value Envelope、制品锁和 conformance；不承诺 Go plugin ABI，也不加载插件前端 JavaScript。
12. **默认 fail closed。** MCP、插件、危险 workflow 和遥测默认关闭或受限；权限由宿主执行，不靠 prompt。
13. **项目身份只有一个。** LICENSE、module、组织、仓库、更新源、漏洞入口和 provenance 必须一致。
14. **CI 与发布是合同。** 文档声明的 gate 必须真实运行；stable 产物必须完整、签名、带 SBOM/checksum/provenance。

## 总体里程碑

| 里程碑 | 对外含义 | 必须完成的 Wave |
| --- | --- | --- |
| M0 — Source Open | 可以诚实称为开源并接受贡献 | 0–2 的 source gate |
| M1 — Core Alpha | 3.1 workflow/node/compiler/runtime 可端到端运行 | 3–8 |
| M2 — AI Alpha | AI authoring 在权限与 eval 门禁内可用 | 8–10 |
| M3 — Extension Preview | 官方 Node SDK 完整；第三方声明式包可试用 | 11 的 A/B 门 |
| M4 — 3.1 Stable | 可信 Windows release 与公开治理成立 | 12；全量完成定义 |

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

## Wave 3 — 冻结 Node System 3.1 契约并打通首个 tracer bullet

目标：复用已经验证的严格解析、JCS、Compiler 与 Program seal 基础，替换其未发布的 v3 wire identity，先让一个真实 pure-data 节点完整穿过 3.1 唯一事实链。

1. 按 Wayfinder 依赖顺序关闭 Node Contract、Capability、Resource、Program/Run、Authoring 与 Plugin tickets；Data Type 决议中的 TypeRef/Resolved Type/Value Envelope 先形成 golden corpus。
2. 建立 versioned Node Contract meta-schema 和 Data Type registry；semantic digest preimage 排除自身与 presentation，所有 bundled schema 离线解析并受预算限制。
3. 由同一 generator 产生 Go validator/model、Catalog 3.1、Workflow Source 3.1 JSON Schema、TypeScript authoring contract、WIT/Protobuf adapter surface、文档模型和 conformance vectors。
4. CompileDraft(Source 3.1, CatalogSnapshot 3.1) 统一 strict parse、instance contract resolution、edge assignability、config、effect/capability 与 permission compilation；所有 diagnostics 保持稳定 code 和预算。
5. Program 3.1 冻结完整 Resolved Type、effective ports、implementation locks、capability plan 和 executable plan；OpenProgram 用可信 catalog/build 重验，runtime 不再解析节点 config 猜端口。
6. 以 Concat 作为第一条 tracer bullet：两个 string data inputs、一个 string data output、零 exec input/output；覆盖创建、连线、default/absent、编译、解释执行、前端 render/Inspector、MCP describe 和生成文档。
7. tracer bullet 期间新 interpreter 只能由 3.1 Program 测试/预览入口调用；不得让生产调用方在 3.1 失败后回退 ContainerRunner。切换生产入口前必须完成 Program/Run 决议。
8. 3.1 cutover 同时更新 compatibility policy、breaking-change 说明、v3/legacy 拒绝 fixtures 和 hash domains；禁止在 v3 domain 下悄悄改变 canonical DTO。

验收：同一 contract generation 能重建所有投影；Concat 在 UI 中与 UE pure function 一致，完全没有伪造 out；Go/TS digest 与 assignability golden tests 一致；未保存 Source 可编译，入队 Program 不受后续编辑影响。
## Wave 4 — 重建 Application 与 headless seam

目标：把约 2,500 行根装配收进可测试深模块，并让 GUI/CLI/MCP 共用 application commands。

1. `internal/appbootstrap.Build(Config)` constructor-complete 地构造 Application；删除生产 `Configure...`、`Set...Factory` 与 package global registry。
2. `Application` 统一拥有 Start/Close、goroutine、store、worker、scheduler、recording、MCP 和 presentation；失败回滚逆序可测试。
3. `main.go` 缩至约 100–150 行，只处理进程输入、Wails run 和退出码。
4. 建立 application command surface：CompileDraft、SaveSource、StartRun、CancelRun、GetRunTimeline、GetCatalog 等，不把 repository service 原样暴露给前端。
5. 增加 `yotta validate/compile/run/inspect` headless CLI；它使用相同 Application/Compiler/Policy，不复制业务逻辑。
6. `internal/node` 不再 import `services/*`；按消费者拥有 port，拆窄 Vision/LLM/Input/Window 等宽接口。

验收：除 main 外无 Fatal/os.Exit；所有后台资源有 owner；GUI/CLI/MCP 调用同一命令；bootstrap wiring statement coverage ≥70%。

## Wave 5 — Workspace、Blob/Stream/Resource 与本地 Run 事实库

目标：把 durable value、ephemeral capability 和执行事实分开持久化，同时保留合理的单 Worker 桌面约束。

1. Workspace 统一拥有 workflow/asset/blob 的 revision、引用完整性、事务、GC 与 durable event；Save/Delete/Import/RecordingFinalize 原子提交并在 durable commit 后发事件。
2. WorkflowStore、ProgramStore、RunStore、SecretStore 与 BlobStore 分责；blob 以 media type + digest + size 定位，写入先校验后发布，支持 quota、range read、引用/租约与 crash cleanup。
3. Resource Broker 只发放绑定 authority/plugin/session/run/type/owner/operation 的高熵 token；原始 HWND、指针、fd、Blob URL 和进程内对象不得进入 Source、Program 或持久 Run value。
4. Stream 定义 bounded chunk、背压、取消/deadline、half-close、terminal error 与 producer/consumer cleanup；stream/handle 到 blob 的 materialize/export 是显式 effect 节点。
5. 用 SQLite/WAL 或等价事务存储实现 RunRecord，ID 使用 UUIDv7；状态至少 QUEUED/RUNNING/SUCCEEDED/FAILED/CANCELLED/INTERRUPTED。
6. NodeAttempt/adapter action 记录 graphPath、node、effect、attempt、时间、error code、脱敏摘要、programHash、catalogHash 与 grant hash；Value Envelope 的 provenance 放在 Run Value wrapper，不污染值摘要。
7. 写入 QUEUED 与 Program 引用成功后才通知 Worker；启动时遗留 RUNNING 转 INTERRUPTED，绝不自动重放 unsafe effect。
8. 增加 blob digest/size mismatch、quota、GC race、stream backpressure/cancel、handle 越权/过期、磁盘满、kill/restart 的 fault-injection tests。

验收：没有幽灵 Running；强杀应用不会重复外部 effect；大对象不再经 JSON/Wails/Protobuf 多次复制；handle/stream 不可持久化；每个 UI 事件对应已提交 generation。
## Wave 6 — Authoring Projection、EditorSession 与生成 contract

目标：让画布、Inspector、AI/MCP 和文档只消费 Node Contract/Data Type 的生成投影，将巨型 Vue 协调层变成明确的编辑状态机客户端。

状态：Authoring Projection、生产 Inspector、MCP/docs 同源投影已由提交 `06481351` 完成并通过完整门禁；EditorSession 与巨型组件拆分留在后续独立阶段。

1. Authoring Projection 统一生成 effective ports、参数控件、类型/约束/单位/default/optional/null 提示、representation/lifecycle/security 文案、capability/platform badge、examples、errors 与 conversions。
2. 第三方 package 不得注入 JavaScript/Vue；复杂交互只能引用 Yotta 内置 allowlist Editor Adapter。区域/坐标/颜色、代码、AE/UE 对象选择器不得拥有或改写节点语义。
3. EditorSession 统一 draft、revision、sourceHash、compiledHash、lastRunHash、history、dirty、graph path、save conflict、validate、run/debug；SaveSource 必须带 baseRevision。
4. 画布端口、拖线检查、加载诊断与 Compiler 使用同一 generated assignability；空 exec 集合是有效事实，任何层不得猜测或补 out。
5. ContainerEditorView 目标 <500 行，NodeInspector <400 行；view 不直接拼 patch JSON、解析 config 生成动态端口或编排多个 store/RPC。
6. machine Catalog 与 Authoring Projection 分 generation；展示注解变化不改变 Program identity，但 UI/文档能追溯 projection digest/generator version。
7. Playwright web harness 覆盖 Concat、类型不兼容、default/absent/null、动态端口、Editor Adapter、保存冲突、AI patch diff 与 run timeline；Windows smoke 覆盖真实 binding/event/window。

验收：普通节点无需手写 Vue/TS pin 映射；UI 参数提示可直接由 schema 解释；前端删除 PinType/backendTypeToPinType/legacy registry 后仍能渲染并编辑全部已迁移节点。
## Wave 7 — Node Contract 3.1、Program interpreter 与内置节点迁移

目标：把 137 个内置节点迁入可生成、可验证的产品接口，并让所有执行语义 lower 到 Program，而不是散落在 runtime kind switch。

1. Node Contract 完整声明 identity/version、config schema、static/instance ports、Execution Class、determinism、effects、capabilities、errors、retry/cache/cancel/timeout、implementation lock、authoring 与文档注解。
2. instance contract resolver 给定 contract + config 产生 immutable effective ports/config/dependencies/capabilities；Compiler、authoring、runtime、MCP 和 docs 只消费结果，不再各写 dynamic parser。
3. Program interpreter 只认识通用 plan instruction 与窄 capability call；region、subgraph、listener、disabled、retry、error routing、recorded value 和 lineage 全由 Compiler lower。
4. 按 pure-data → conversion/collection → effect → control/region → event/listener 顺序迁移。每批先生成 catalog/docs/golden fixtures，再切调用方并删除该批 legacy Spec/coercion/validator/dispatch 分支。
5. pure-data 仅在 deterministic 且无 effect 时允许 memoize/cache；cache key 包含 contract、implementation、config、Resolved Type、input digests 与 catalog generation。
6. 官方 SDK 提供 yotta dev node new/lint/test、fake capability kit、golden workflow、panic/error/cancel/secret-leak tests；CI 生成后 diff 必须为空。
7. 最终删除 CanonicalPinType、PinTypeCompat、CoerceInputValue/coerceToType、前端 parallel compatibility、blank-import mutable registry 与按 kind 的中央 dispatch。

验收：新增普通内置节点不修改中央 switch；全部节点可由 Catalog 3.1 strict compile；debug 与 normal run 共用 interpreter 语义；旧 runtime 特判数量只能下降且最终为零。
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

## Wave 10 — AI authoring 与 MCP 3.1

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

## Wave 11 — Node Package、Wasm/Process Plugin Host 与生成生态

目标：把用户已要求的第三方节点纳入 Yotta 3.1，同时保持 Go 主进程 ABI、UI 与 capability 边界封闭；不做 marketplace。

1. Node Package manifest 固定 publisher namespace、package/version、host API range、Node/Data Contracts、implementation kind、artifact digest、capability requirements 与 bundled schemas/docs；Program Snapshot 锁定实际 artifact。
2. Wasm Node 使用 WebAssembly Component Model/WIT 与 host resource imports；默认无 filesystem/network/process/GUI 能力，只能使用 Program + Run policy 授权的 imports。
3. Process Node 使用 length-delimited binary Protobuf frame，包含 protocol/message/request/run/attempt/deadline；大型值只传 blob/stream/resource reference，崩溃只失败当前 attempt。
4. 两种 host 共用 Value Envelope、Binding State、structured Error/Status、取消/超时、日志脱敏、Resource Broker 与 golden conformance corpus；不得各自实现兼容算法。
5. package lifecycle 覆盖发现目录、开发包、签名/本地信任、namespace ownership、原子 install/update/disable/uninstall、quarantine、撤销、rollback 与 crash cleanup。Windows 是完整支持；Linux/macOS 保持 package validation/core host 可测试和 GUI/plugin preview。
6. Process sandbox 与 quota 明确 CPU、memory、wall time、child process、filesystem roots、network hosts、open handles/streams；安全能力缺失时 package 不能被标记为可运行。
7. 插件 UI 仅使用 schema-generated Authoring Projection 和内置 Editor Adapter；禁止插件 JavaScript/Vue/DOM 注入，也不加载 Go plugin/shared library ABI。
8. SDK/文档工具生成 Go/TS/WIT/Proto bindings、示例包、node reference、conformance vectors、breaking-contract diff 与 CI drift gate；提供一个 Wasm 示例和一个 Process 示例作为 release fixture。

验收：Windows 上两种示例插件完成 install → compile → run → cancel/crash → disable/uninstall 全链；越权、摘要不符、协议错版、schema bomb、stream/handle 泄漏均 fail closed；不安装任何第三方包时主程序仍无任意代码加载面。
## Wave 12 — 可信 release 与社区成熟度

目标：让 3.1 的二进制、治理承诺与真实控制面一致。

1. Windows staging 通过 Wails 启动、bindings、DLL、ADB、安装/卸载/升级 smoke；发布 installer + portable archive，不是裸 exe。
2. Authenticode 签名并 timestamp；签名失败阻断。附 checksums、SBOM、notices、attestation/provenance 和验证命令。
3. protected tag/environment 经非 self reviewer 审批后发布 immutable release；asset/tag 与 source commit 可验证。
4. Linux/macOS artifact 明确 preview；完成真实宿主 smoke、签名和权限 UX 后再升 stable。
5. 新贡献者只凭仓库文档完成 setup、添加节点、运行全 gate、签 DCO、提交聚焦 PR。
6. 建立 contributor → reviewer → maintainer ladder、ownership/offboarding、release shadow rotation、事故演练和密钥轮换。
7. 成熟度按 OpenSSF Passing → Silver 和 SLSA L2 → L3 推进；未满足多维护者/公开开发/可信发布前，不宣称“大型成熟开源项目”。

验收：干净机器可验证 release signature/checksum/attestation；至少两名有真实权限的维护者；安全入口、支持窗口、发布与撤销流程均实际演练。

## 推荐 PR / issue 编排

不要用一个“3.1 重写”大分支。每个条目是可独立评审的纵向切片，合并即删除对应旧路径：

1. `legal!: choose OSI license and canonical project identity`
2. `governance: enforce rulesets, ownership and vulnerability intake`
3. `ci: make documented frontend/go gates real`
4. `chore(frontend): establish one-time formatting baseline`
5. `build: pin complete toolchain and action digests`
6. `contract!: freeze Data Type and Node Contract 3.1 with conformance corpus`
7. `schema!: introduce WorkflowSource/Catalog/Program 3.1 and stable diagnostics`
8. `runtime!: land Concat 3.1 tracer bullet and execute snapshots only`
9. `refactor(app): constructor-complete Application and headless commands`
10. `storage!: add workspace transactions and revision conflicts`
11. `runtime: persist RunRecord and NodeAttempt state machine`
12. `refactor(frontend)!: generate contracts and add EditorSession`
13. `node!: migrate catalog to Node Contract 3.1 and delete implicit coercion`
14. `plugin!: ship Wasm and Process hosts with package lifecycle`
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
| 插件扩大攻击面 | 3.1 只交付最小 Wasm/Process host；不做 marketplace、Go ABI 或前端代码注入，任何隔离/信任/conformance gate 未满足都阻断 stable |
| 开源名义先于公开事实 | Source open gate 阻断宣传与 stable release |
| 单维护者造成发布/密钥风险 | 双人规则、第二管理员、shadow rotation、offboarding 演练 |
| 文档再次与实现漂移 | 示例可编译、契约生成后 diff gate、tracked AGENTS、RECHECK 触发器 |

## Yotta 3.1 完成定义

- 法律/身份：OSI LICENSE；公开主线完整；canonical org/repo/module/security/update 一致。
- 工程：所有本地/required gates 同源全绿；format 0 failures；coverage/bundle budget 强制。
- 数据：只接受 Workflow/Catalog/Program 3.1；v3/v2/legacy 明确拒绝；无迁移器、旧 key、dual-read/write 或 Normalize 修复边界输入。
- 编译：内存草稿可编译；ProgramSnapshot canonical、不可变、跨平台 hash 一致。
- 执行：Run/Attempt 持久；副作用不透明重试/重放；强杀后准确 INTERRUPTED。
- 架构：main <150 行；constructor-complete Application；node core 不依赖 concrete services。
- 前端：contracts 生成；EditorSession 成立；两个巨型组件达到行数目标；editor gzip ≤450 KB。
- 节点：NodeSpec 是唯一事实源；effect/capability/retry/cache/secret 由宿主强制；官方 SDK 可一命令使用。
- 安全：Workflow Trust、permission manifest、OS credential store；默认无 MCP listener；危险能力 fail closed。
- AI：provider-native API；无 JSON prompt fallback；prompt/schema/tool/model 可版本化、eval、trace；dynamic data 不进入高权限指令。
- Authoring：AI/UI/CLI 共用 typed patch/Compiler；catalog 按需发现；用户可见 diff、diagnostics、权限 delta。
- 扩展：最小 Wasm/Process Plugin Host、包生命周期、制品锁、capability broker、SDK/docs/conformance 完成；不加载 Go plugin 或插件前端代码。
- 发布：完整 Windows installer/portable、签名/timestamp、checksum、SBOM、provenance、immutable release、smoke 全绿。
- 社区：DCO、治理、CODEOWNERS、支持/安全/发布合同真实可执行；至少两名有实际权限的维护者。

只有这些条件同时成立，才发布 `v3.1.0`；不能用“模型已经足够聪明”替代任何 schema、权限、eval 或供应链门禁。
