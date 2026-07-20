# Yotta 3.1：开源工作流与扩展平台一手资料研究

> 调研日期：2026-07-13
> 范围：n8n、Node-RED、Windmill；以 Temporal、VS Code、ComfyUI 补足可靠执行、插件治理和可视化 AI 工作流的参照。
> 取材规则：只采用项目官方文档、官方仓库和官方规范。本文不是功能抄录，而是回答“Yotta 3.1 应采用什么、明确不照搬什么、先建立哪些不可逆契约”。

## 1. 结论先行

Yotta 不应升级成一个缩小版 n8n、Windmill 或 Temporal。它是本地优先、强交互、会操作窗口/键鼠/ADB/截图/LLM 的桌面自动化系统；这些副作用使“任意重放、任意自动重试、每个节点远程排队”都既昂贵又危险。

真正值得组合的是六个平台各自最强的一层：

| 来源 | 最值得采用的机制 | 明确不照搬 |
| --- | --- | --- |
| n8n | 编辑器与执行器职责分开；运行历史与工作流历史分开；脏节点提示；节点脚手架、lint、测试与风险治理 | 为旧工作流在节点实现中长期保留多版本分支；未隔离的社区代码；把 Redis/数据库队列直接搬进单机产品 |
| Node-RED | Runtime、Editor、Admin API 的清晰接缝；流 revision 乐观并发；可插拔存储；简洁的节点包与测试入口 | 同一节点手写 JS Runtime + HTML Editor 两份契约；v1/v2 双流格式；进程内插件的共享故障域 |
| Windmill | 可序列化的 OpenFlow；不可变脚本哈希；父 Run/步骤 Job；明确的 retry/timeout/failure handler；全面的队列与 Worker 指标 | 每个本地节点都变成数据库 Job；任意多语言执行环境；把关键可观测性或恢复能力做成付费开关 |
| Temporal | 确定性编排与有副作用 Activity 的边界；事件历史；声明式重试；历史重放 CI | 为桌面输入、进程启动、文件写入、LLM 调用承诺透明重放；引入 Temporal 集群 |
| VS Code | 声明式贡献点；惰性激活；稳定/实验 API 分道；Workspace Trust 与能力降级 | 把“独立进程”误认为安全沙箱；允许扩展任意修改编辑器 DOM |
| ComfyUI | 入队时冻结整个工作流快照；只对纯节点做变更缓存；Registry 的不可变版本与工作流版本锁 | 进程内导入任意 Python；把任意插件 JS 全量注入页面；允许插件绕过宿主的类型与范围校验 |

这组组合导向一个明确的 3.1 主链路：

    EditorSession.draft
        -> CompileDraft(WorkflowSource, CatalogSnapshot)
        -> ProgramSnapshot（不可变、带内容哈希与能力清单）
        -> Durable RunRecord
        -> 单机 Worker
        -> NodeAttempt / AdapterAction 事件树

保存不是编译前置条件，编辑器后续改动不能改变已入队 Run，副作用节点默认不能自动重试，所有执行记录必须能用一个 run_id 串起来。

## 2. 对 Yotta 当前实现的针对性判断

本节结论来自仓库当前代码，不是外部平台推断。

1. internal/services/execution/queue.go 是内存 FIFO，Run ID 由进程内 atomic.Int64 递增；进程退出后队列、ID 连续性与运行状态均消失。
2. internal/services/execution/worker.go 是单 goroutine 串行消费，这一点符合键鼠独占的桌面约束；但事件主要是 Running、TargetIdx 和最后一个错误，缺少 Program 哈希、节点 attempt、终止原因和持久 Run 账本。
3. internal/services/container/service.go 的 ValidateContainerByID 明确校验持久化版本，frontend/src/i18n/zh.ts 也提示“请先保存再检查”。3.1 若仍把保存当校验入口，EditorSession 与 Compiler 的新分层就没有真正成立。
4. internal/services/log.go 已有 JSONL、500 行 GUI ring 和脱敏 action trace，这是可保留的基础；但 action trace 尚不是按 Run 查询的执行事实库，也没有统一 run_id / node_id / attempt / program_hash 关联。
5. internal/node/interfaces.go 已有 Node.Spec、Runnable、RegionRunner、Evaluator、Validator 与 ServiceBundle，足以演进为正式 Node SDK；当前缺少由宿主强制执行的副作用类别、确定性、缓存、重试安全、能力和敏感数据声明。

因此本轮大升级的核心不是“再加更多节点”，而是先让源代码、编译产物、运行快照、执行事实、节点契约成为五个彼此明确的对象。

## 3. n8n：编辑/执行分离、节点工程化与历史语义

### 3.1 官方机制

- n8n 的 queue mode 将接收触发与 webhook 的 main、Redis 中的待执行 ID、从数据库读取工作流并执行的 worker 分开；所有组件要求版本一致，并提供 health、readiness 与 metrics 端点。[Queue mode 官方文档](https://docs.n8n.io/hosting/scaling/queue-mode/)
- task runner 进一步分成 broker、task requester 与 runner；官方建议生产环境使用 external mode sidecar，并要求 runner 与 n8n 版本匹配。runner 还有环境变量与内置/外部模块 allowlist。[Task runners 官方文档](https://docs.n8n.io/hosting/configuration/task-runners/)
- 已保存工作流会固定节点 type version，新建工作流使用最新版本；官方同时提供 light、feature、full 三种节点版本化方式，其中 feature versioning 会在一个实现里保留旧路径。[Node versioning 官方文档](https://docs.n8n.io/integrations/creating-nodes/build/reference/node-versioning/)
- workflow history 在每次保存、恢复或 Git pull 时产生版本，且官方明确它不同于 execution history；用户可以恢复、克隆或下载历史版本。[Workflow history 官方文档](https://docs.n8n.io/workflows/history/)
- 执行失败后可以用“当前已保存工作流”或“原执行所用工作流”重试。这一差异说明运行时必须知道自己绑定的是哪个工作流快照。[Executions 官方文档](https://docs.n8n.io/workflows/executions/all-executions/)
- 编辑器会根据节点配置和连线变化标记 dirty nodes，提示哪些已有输出已经陈旧；局部执行完成后才清除脏状态。[Dirty nodes 官方文档](https://docs.n8n.io/workflows/executions/dirty-nodes/)
- 对常规 REST 节点，官方优先推荐 declarative style，理由是实现更简单、缺陷更少且便于未来演进；复杂控制、触发或特殊依赖再用 programmatic style。[Choose your node building approach](https://docs.n8n.io/integrations/creating-nodes/plan/choose-node-method/)
- n8n-node 工具负责节点脚手架、构建和检查；linter 覆盖节点、凭据和包结构，并在发布前与贡献 PR 中运行。[Node development environment](https://docs.n8n.io/integrations/creating-nodes/build/node-development-environment/)、[Node linter](https://docs.n8n.io/integrations/creating-nodes/test/node-linter/)
- 官方仓库用 pnpm monorepo 分开 packages/cli 后端与 packages/frontend/editor-ui，提供 devcontainer、按包启动、Testcontainers、Playwright 和多数据库/queue mode 测试路径。[n8n CONTRIBUTING](https://github.com/n8n-io/n8n/blob/master/CONTRIBUTING.md)
- 官方直接警告未验证社区节点能访问运行机器和工作流数据、可能引入破坏性变更；管理员可以禁用社区节点，另有安全审计检查危险节点、文件访问和未保护 webhook。[Community node risks](https://docs.n8n.io/integrations/community-nodes/risks/)、[Security audit](https://docs.n8n.io/hosting/securing/security-audit/)
- 官方健康与监控面包含 /healthz、/healthz/readiness 与 /metrics；结构化日志建议携带 executionId、workflowId、sessionId 和 node type。[Monitoring](https://docs.n8n.io/hosting/logging-monitoring/monitoring/)、[Logging](https://docs.n8n.io/hosting/logging-monitoring/logging/)
- declarative 节点会自动保留输入输出 item lineage，而 programmatic 节点作者要手工维护 paired-item 关系。这展示了“宿主自动保证跨节点元数据”比要求节点作者自行遵守更可靠。[Item linking for node creators](https://docs.n8n.io/data/data-mapping/data-item-linking/item-linking-node-building/)

### 3.2 Yotta 应采用

1. **把 workflow history 与 run history 分开。** 保存历史回答“用户改了什么”，RunRecord 回答“当时究竟执行了什么”；两者通过 workflow_revision 与 program_hash 关联，但不能共用一套记录。
2. **Run 在入队时锁定 ProgramSnapshot。** 用户可以继续编辑，已排队运行不受影响；UI 可同时提供“用原快照重跑”和“用当前草稿重新编译后运行”，两者必须明确命名。
3. **给编辑器一等的 dirty 状态。** 不能只显示“未保存”；应基于 sourceHash、lastCompiledHash、lastRunHash 区分“未保存”“未重新编译”“预览结果已陈旧”。
4. **建立官方节点工具链。** 3.1 即使只有仓库内节点，也要有 node new、node lint、node test、catalog generate、docs generate，一套 Spec 同时生成 Go 校验材料、TypeScript 类型、Inspector schema 和文档。
5. **由宿主管理跨节点元数据。** run_id、graph_path、node_id、attempt、输入来源和敏感标记由 Runtime 注入，不让每个节点手填。
6. **把健康状态和运行指标作为 Community 版基线。** 桌面应用不需要暴露公网端口，但应有内部 health snapshot、队列深度、运行时长、错误分类和 adapter 失败计数，并允许用户显式开启本地端点或 OpenTelemetry 导出。

### 3.3 Yotta 不应照搬

1. **不保留 n8n 式节点旧版本分支。** 用户已授权破坏性 3.1：核心只接受 3.1 Workflow/Node contract；不得在节点 Run 中按旧 typeVersion 分叉。第三方制品版本锁定只服务可重复执行，不是兼容旧语义。
2. **不引入 Redis + 主数据库队列。** Yotta 的单机键鼠独占 Worker 是合理约束；所需的是本地持久 Run 账本与崩溃状态恢复，而不是分布式吞吐架构。
3. **不允许未隔离节点直接获得 ServiceBundle 全能力。** n8n 对社区代码的风险警告说明，“能安装 npm 包”不是成熟扩展系统；Yotta 应先有能力声明与进程边界，再开放第三方执行代码。

## 4. Node-RED：Runtime / Editor / Admin API 接缝与轻量节点模型

### 4.1 官方机制

- Node-RED 明确列出 Admin HTTP API、Runtime API、Storage API、Logging API、Context Store API、Editor UI API 等层；Admin API 被 Editor 和命令行共同使用。[Node-RED API reference](https://nodered.org/docs/api/)
- @node-red/runtime 是核心 flow engine 和 Runtime 入口，独立于 Editor。[Runtime package 官方仓库说明](https://github.com/node-red/node-red/blob/main/packages/node_modules/@node-red/runtime/README.md)
- 自定义节点由一份 JavaScript Runtime 文件和一份 HTML Editor 定义组成，两边以同一个 type 注册；package.json 再声明 node-red.nodes 映射。[Creating your first node](https://nodered.org/docs/creating-nodes/first-node)、[Node runtime](https://nodered.org/docs/creating-nodes/node-js)、[Node editor UI](https://nodered.org/docs/creating-nodes/node-html)
- 节点作为 npm 包发布，manifest 可声明 Node-RED engine 版本，官方要求 README、license、示例和稳定性，并为节点目录规定命名与关键字规则。[Packaging nodes](https://nodered.org/docs/creating-nodes/packaging)
- Admin API 的 v2 flow 结构支持可选 rev，用于防止基于旧 revision 的覆盖；Runtime 的 setFlows 还区分 full、nodes、flows、reload 等部署类型。[Admin API types](https://nodered.org/docs/api/admin/types)、[Runtime flows API](https://nodered.org/docs/api/modules/v/1.3/%40node-red_runtime_flows.html)
- Storage API 抽象了 flows、credentials、settings、sessions 和 library 的读写。[Storage API methods](https://nodered.org/docs/api/storage/methods/)
- Projects 将一组 flow 作为 Git 仓库管理，在 Editor 中展示 diff、commit 和 history，并把凭据独立加密保存。[Projects](https://nodered.org/docs/user-guide/projects/)
- Runtime 可限制外部模块 allow/deny、关闭 Editor 并设置日志级别。[Runtime configuration](https://nodered.org/docs/user-guide/runtime/configuration)
- Context 默认只驻内存；localfilesystem store 先缓存内存并默认每 30 秒落盘。它适合简单状态，不是强事务运行账本。[Context](https://nodered.org/docs/user-guide/context)
- 官方节点文档提醒：未捕获的节点错误可能停止整个 flow；Catch 和 Status 节点只处理被框架捕获并上报的错误。[Creating nodes](https://nodered.org/docs/creating-nodes/)、[Handling errors](https://nodered.org/docs/user-guide/handling-errors)
- 官方测试助手可以加载真实节点和测试 flow，而贡献指南要求功能/重构先讨论、补测试并保持 PR 聚焦。[Creating your first node](https://nodered.org/docs/creating-nodes/first-node)、[Node-RED CONTRIBUTING](https://github.com/node-red/node-red/blob/master/CONTRIBUTING.md)

### 4.2 Yotta 应采用

1. **把 Editor API 变成 Compiler/Runtime 的正式客户端。** Wails binding 不应把仓库 Service 原样暴露给 Vue；需要少量稳定 application commands，例如 CompileDraft、SaveSource、StartRun、CancelRun、GetRunTimeline、GetCatalog。
2. **引入 revision 乐观并发。** SaveSource 必须携带 baseRevision；如果文件已被另一个窗口、外部 Git 操作或同步器修改，返回结构化 conflict，而不是最后写入者静默覆盖。
3. **存储面按职责抽象。** WorkflowStore、RunStore、SecretStore、ArtifactStore 分开；这比一个“通用 Repository”更容易测试和演进。
4. **保留轻量节点包体验。** 一个节点应有聚焦的 Spec、执行函数、测试 flow 和文档示例；测试助手应能用 fake Vision/Input/LLM/Process capability 加载真实节点。
5. **提供有限的部署范围语义。** 对 Yotta 不必照搬 full/nodes/flows，但 Compiler 应精确给出受影响 graph/subgraph/node，供编辑器增量反馈和纯节点缓存失效。

### 4.3 Yotta 不应照搬

1. **不手写 Runtime 与 Editor 两份节点契约。** Node-RED 的 JS + HTML 配对简单但容易漂移。Yotta 应以 Go NodeSpec 为唯一事实源，生成 TS、JSON Schema、Inspector 控件描述和文档；CI 检查生成结果无 diff。
2. **不保留两代 flow API。** 3.1 只有一个规范化 WorkflowSource 格式；读到 v2/v3 立即返回明确的 UNSUPPORTED_WORKFLOW_FORMAT，不在 Runtime 内静默转换。
3. **不把 localfilesystem 的定时 flush 当执行可靠性。** Run 入队、状态转换和 attempt 事件必须事务写入；不能接受最长 30 秒丢失窗口。
4. **不让插件异常拖垮宿主。** 首批 3.1 官方节点可以同进程；第三方扩展必须先有 panic/异常边界、资源限制，并使用 Wasm 或独立 Process Host。执行隔离解决可用性边界，安全仍需能力代理。

## 5. Windmill：不可变程序、步骤 Job 与可观测运行

### 5.1 官方机制

- Windmill 的 flow 是可序列化 OpenFlow；每个步骤是独立排队 job，分支可并行，input transforms 形成数据依赖 DAG，并支持循环、分支、暂停等模块。[Flow architecture](https://www.windmill.dev/docs/flows/architecture)
- OpenFlow 以 OpenAPI 文档为事实源，根结构包含 summary、description、value、input schema，流程级与模块级包含 retry、timeout、suspend、failure module、continue-on-error 等字段。[OpenFlow 文档](https://www.windmill.dev/docs/openflow)、[openflow.openapi.yaml](https://github.com/windmill-labs/windmill/blob/main/openflow.openapi.yaml)
- 一个 job 有 UUID、队列/运行状态、日志、结果和元数据；脚本 job 固定到不可变 hash 和 lockfile。flow job 是父 job，每一步是子 job，进度可通过 SSE 推送。[Jobs](https://www.windmill.dev/docs/core_concepts/jobs)
- Worker 自主从队列原子取得一个 job，一次执行一个；tag 用于路由不同 worker group。日志、结果和状态由平台保留，进程隔离可叠加 nsjail。[Worker groups](https://www.windmill.dev/docs/core_concepts/worker_groups)
- 脚本、flow 和 app 都有版本历史；脚本版本由不可变 hash 标识，资源历史是线性全量快照，不提供分支合并。[Versioning](https://www.windmill.dev/docs/core_concepts/versioning)
- flow 可以给每个步骤配置常数或指数退避重试、timeout、failure handler 和 continue-on-error。[Error handling](https://www.windmill.dev/docs/flows/error_handling)、[Retries](https://www.windmill.dev/docs/flows/retries)
- Worker 崩溃或掉电后的 zombie job 可以自动重新启动，优雅终止则通过 grace period 降低误判。[Critical alerts and zombie jobs](https://www.windmill.dev/docs/core_concepts/critical_alerts)
- 官方仓库说明 API server 无状态、Worker 从 PostgreSQL 拉 job，前端使用 Svelte；同一 README 也给出约 50ms 的逐 job 队列开销与 ZOMBIE_JOB_TIMEOUT / RESTART_ZOMBIE_JOBS 配置。[Windmill 官方仓库](https://github.com/windmill-labs/windmill)
- 平台提供 Prometheus/OTLP 指标，覆盖 queue push/pull/count、zombie restart、Worker 时长/失败/忙碌度、数据库与健康；也支持 OpenTelemetry traces、logs 和 metrics。[OpenTelemetry metrics](https://www.windmill.dev/docs/misc/guides/otel)、[Instance settings](https://www.windmill.dev/docs/advanced/instance_settings)
- wmill CLI 的同步配置有 JSON Schema，可在同步时默认跳过 secrets 并做分支配置校验；官方也支持 workflows as code。[CLI sync](https://www.windmill.dev/docs/advanced/cli/sync)、[Workflows as code](https://www.windmill.dev/docs/core_concepts/workflows_as_code)

### 5.2 Yotta 应采用

1. **把编译产物做成不可变 ProgramSnapshot。** 至少记录 sourceHash、programHash、catalogHash、compilerBuild、pluginLocks、requiredCapabilities；哈希必须基于 canonical encoding，排除时间戳。
2. **Run 与 NodeAttempt 分层。** RunRecord 是父记录，graph/subgraph/node attempt 是子记录；每个 attempt 有 queuedAt、startedAt、endedAt、status、retryReason、errorCode、adapter 摘要。
3. **把控制策略编译进 Program。** timeout、failure edge、continue/stop、retry policy 不能在 Worker 中靠隐式默认猜测；Compiler 要验证策略与节点 effect 是否兼容。
4. **本地也要完整指标。** 队列深度、等待时间、运行时长、取消延迟、节点错误率、adapter 失败、被判定 interrupted 的 Run 数量都应免费、默认可见；导出只是可选通道。
5. **同步时默认排除 secrets。** WorkflowSource 只保存 SecretRef，不保存明文；导出、诊断包和 Git diff 都沿用这一约束。

### 5.3 Yotta 不应照搬

1. **不把每个节点都变成 PostgreSQL job。** Yotta 的大量节点处于同一交互控制循环，逐节点约几十毫秒的排队成本和数据库依赖不合适。持久化的是 Run 与 attempt 事实，不代表每个节点都必须跨进程排队。
2. **不在 3.1 承担任意语言 Runtime。** Python/TypeScript/Bun/Deno 等多语言执行会同时扩大安装体积、依赖冲突、沙箱、调试和供应链攻击面。先把 Go 官方节点 SDK 与受约束 Wasm/Process host 做深。
3. **不把线性历史当协作模型。** 保存快照适合本机恢复；开源协作仍由 Git 分支和 PR 完成。内部 revision 解决并发覆盖，不伪造一个弱 Git。
4. **不把核心恢复与指标做成版本差异。** 运行事实、崩溃标记和基础指标是可靠自动化的底座，不是高级附加功能。

## 6. Temporal：借用可靠性语义，不借用分布式平台

### 6.1 官方机制

- Temporal Workflow Execution 的状态由 Event History 保存；Worker 重放历史，在相同代码位置生成命令并与历史比对，从最后记录的状态继续。这要求 Workflow 编排具有确定性。[Workflow Execution](https://docs.temporal.io/workflow-execution)
- Retry Policy 默认应用于容易遭遇瞬态失败的 Activity；Workflow 编排自身通常不靠重试恢复。官方也区分 non-retryable error，并强调 Activity 幂等性。[Retry Policies](https://docs.temporal.io/encyclopedia/retry-policies)
- Worker Versioning 可以让 Pinned Workflow 留在同一版本完成，也可以让 Auto-Upgrade Workflow 迁移，但后者仍必须保持重放安全；旧版本需经过 draining 生命周期。[Worker Versioning](https://docs.temporal.io/worker-versioning)
- Go 测试指南提供跳时测试服务，并建议在 CI 中重放有代表性的 Event Histories，以发现不确定性或代码变更造成的不兼容。[Go testing suite](https://docs.temporal.io/develop/go/best-practices/testing-suite)

### 6.2 Yotta 应采用

1. **明确“确定性编排”与“有副作用动作”。** 条件、纯表达式、数据变换、静态图路由可以重算；Click、InputText、LaunchProcess、WriteFile、HTTP、LLM、ADB 等是 Activity 类副作用。
2. **重试是节点契约，不是全局万能开关。** 每个节点声明 effect、retrySafety 和可选 idempotencyKey；Compiler 拒绝给 unsafe 节点配置 automatic retry。
3. **把纯控制流历史纳入回归测试。** 保存代表性 ProgramSnapshot + deterministic event fixture，CI 重放路由选择和纯节点结果，防止 Compiler/Runtime 改动悄悄改变语义。
4. **历史 Run 固定到程序和节点制品。** 它保证“知道当时执行了什么”；这不等于 Runtime 为 v2 实现兼容解释器。

### 6.3 Yotta 不应照搬

1. **不承诺任意桌面流程透明恢复。** 崩溃后无法可靠知道外部窗口是否已点击、文件是否已写一半、进程是否已启动。默认行为应是把运行标为 INTERRUPTED，展示最后成功节点与副作用边界，由用户从显式 checkpoint 或全新 Run 继续。
2. **不照搬 Activity 默认自动重试。** 云服务 Activity 常可设计幂等；桌面键鼠动作通常不可幂等。Yotta 对 write/external/input/human 类节点默认 maxAttempts=1。
3. **不引入 Temporal Server。** 本地 SQLite/WAL RunStore、单 Worker 和严格状态机即可覆盖当前可靠性目标。

## 7. VS Code：插件契约、能力治理与实验通道

### 7.1 官方机制

- VS Code 在 local、web、remote 等不同 Extension Host 中运行扩展，并依据 extensionKind 选择位置；扩展与 UI/启动路径分离，但 Extension Host 本身不是通用 OS 安全沙箱。[Extension Host](https://code.visualstudio.com/api/advanced-topics/extension-host)
- 扩展通过 package.json contribution points 声明命令、视图、配置、语言等静态贡献；activation events 让扩展按需激活。[Contribution Points](https://code.visualstudio.com/api/references/contribution-points)、[Activation Events](https://code.visualstudio.com/api/references/activation-events)
- Proposed API 只在 Insiders 中使用，不能发布到 Marketplace；这把实验契约与稳定 API 的兼容承诺分开。[Using Proposed API](https://code.visualstudio.com/api/advanced-topics/using-proposed-api)
- Workspace Trust 允许扩展声明 unsupported、limited 或完整支持，并在 Restricted Mode 下禁用功能或限制配置；未声明时采用保守默认。[Workspace Trust extension guide](https://code.visualstudio.com/api/extension-guides/workspace-trust)
- 扩展解剖文档把 manifest、入口、贡献点和 API 使用方式分开。[Extension Anatomy](https://code.visualstudio.com/api/get-started/extension-anatomy)

### 7.2 Yotta 应采用

1. **先定义声明式 PluginManifest，再开放执行入口。** 建议字段包括 pluginId、version、apiVersion、engineRange、artifactDigest、signature、contributes.nodes、capabilities、entrypoint 和 publisher。
2. **3.1 Node API 先标记预览。** 达到契约测试、能力代理和 host 隔离门禁后再提升 stable。用户授权破坏性更新，正适合此时不作过早兼容承诺。
3. **按能力惰性装载。** 纯节点不应初始化 Vision/ADB/Win32/LLM；只有 Program 需要某能力时才启动对应 adapter/runner。
4. **引入 Workflow Trust。** 未信任导入流程只能查看与编译，不能执行 process、filesystem write、network、input、screen capture、secret read；用户授予的是具体能力，不是一个笼统“运行所有”按钮。
5. **插件不能直接改 Vue DOM。** Editor 只渲染宿主支持的声明式 Inspector schema、图标和文档；高级 UI 扩展另设极窄、版本化的 surface。

### 7.3 Yotta 不应照搬

1. **独立进程不等于安全。** Runner 仍必须通过 capability broker 请求屏幕、键鼠、文件、网络、Secret；宿主校验参数、记录审计并可撤销。
2. **不在 3.1 承诺 Go 插件 ABI。** Go 原生 plugin ABI 不适合作为跨版本边界；第三方扩展使用版本化 WIT/IPC protocol + Wasm/Process Host。
3. **不允许扩展把自己声明为可信。** Trust 是用户与宿主策略的决定，manifest 只能声明需求。

## 8. ComfyUI：运行快照、纯节点缓存与 Registry 锁定

### 8.1 官方机制

- ComfyUI 客户端在 Queue Prompt 时把整个 workflow snapshot 发送给服务端；入队后继续编辑不会改变该次执行。[Client-server communication](https://docs.comfy.org/development/comfyui-server/comms_overview)
- 执行从输出向后遍历，只运行必要节点并缓存未变化输出；IS_CHANGED 可参与失效判断。服务端有默认输入类型/范围校验，但自定义 VALIDATE_INPUTS 可以接管甚至绕过部分宿主校验。[Server overview](https://docs.comfy.org/custom-nodes/backend/server_overview)
- 后端启动时扫描并在同一 Python 进程导入 custom nodes，以 NODE_CLASS_MAPPINGS 注册；导入失败可以跳过单个模块。前端扩展则会把 custom node 提供的 JavaScript 文件加载进网页并允许多种 hook。[Custom node lifecycle](https://docs.comfy.org/custom-nodes/backend/lifecycle)、[JavaScript extensions](https://docs.comfy.org/custom-nodes/js/javascript_overview)
- Comfy Registry 使用全局唯一 node ID、语义版本和不可变已发布版本；可以 deprecate 版本，workflow JSON 保存节点版本。Registry 还做扫描与验证。[Registry overview](https://docs.comfy.org/registry/overview)
- Registry 标准禁止 eval/exec、运行时 pip subprocess 和混淆代码等行为；发布流程要求 pyproject 元数据、publisher 身份与版本，并提供 CLI scaffold / GitHub Action 路径。[Registry standards](https://docs.comfy.org/registry/standards)、[Publishing nodes](https://docs.comfy.org/registry/publishing)、[Custom node walkthrough](https://docs.comfy.org/custom-nodes/walkthrough)
- 官方安装文档明确提醒自定义节点可能恶意且依赖会污染环境。[Installing custom nodes](https://docs.comfy.org/installation/install_custom_node)

### 8.2 Yotta 应采用

1. **入队时冻结快照。** StartRun 接受的是成功编译的 ProgramSnapshot，不是 container ID；Worker 禁止在执行途中重新读取当前持久化 workflow。
2. **缓存仅限宿主证明为 pure 的节点。** cache key 至少包含 node kind/version、canonical config、输入值哈希、上游输出哈希、catalog/plugin digest；任何时间、随机、环境、文件、网络、窗口、屏幕、变量写入依赖都会让节点失去默认缓存资格。
3. **节点版本写入 ProgramSnapshot。** 精确版本/制品 digest 用于可追溯与可复现；核心 v3 不因此保留 v2 解释路径。
4. **未来 Registry 必须发布不可变制品。** 删除用 deprecate/yank，不允许同一版本覆盖上传；安装前做签名、静态策略扫描和能力审阅。

### 8.3 Yotta 不应照搬

1. **插件不能覆盖宿主硬校验。** 节点可以追加语义诊断，但不能绕过端口类型、范围、能力、Secret 流向和 effect/retry 兼容性校验。
2. **不在宿主进程导入任意第三方代码。** 单个模块“加载失败继续”只改善可用性，不解决数据与系统权限风险。
3. **不把任意插件 JS 注入编辑器。** 这会让 Editor API、DOM 和用户数据都成为非正式公共接口。
4. **不对有副作用节点做增量缓存。** “输入没变”不代表外部窗口、文件或网络状态没变。

## 9. 建议的 Yotta 3.1 目标契约

### 9.1 WorkflowSource、CompileResult 与 ProgramSnapshot

建议在代码层明确三个互不混用的类型：

    WorkflowSource {
      formatVersion: 3,
      workflowId,
      revision,
      metadata,
      graphs,
      secretRefs,
      requestedCapabilities
    }

    CompileResult {
      sourceHash,
      diagnostics[],
      program?: ProgramSnapshot
    }

    ProgramSnapshot {
      formatVersion: 3,
      sourceHash,
      programHash,
      catalogHash,
      compilerBuild,
      pluginLocks[],
      requiredCapabilities[],
      executableGraphs
    }

约束：

- CompileDraft 直接接收内存 WorkflowSource；不得先 Save 再 Validate。
- diagnostics 使用稳定 code、severity、graphPath、nodeId、fieldPath、params 和 optional fix，不把中文 message 当 API。
- 相同 Source + Catalog + Compiler 在 Windows/Linux 上生成相同 programHash；compiledAt 等非语义字段不得进入哈希。
- StartRun 只接收 programHash 或完整不可变 ProgramSnapshot；若只传 hash，必须从本地 ProgramStore 精确读取，绝不能按 workflowId 重新编译。
- formatVersion=2 在 parse boundary 立即失败；不提供 Runtime fallback、兼容分支或静默迁移。

灵感分别来自 Windmill 的不可变脚本 hash、ComfyUI 的入队快照、Node-RED 的 rev 与 n8n 的“原执行工作流”重试语义：[Windmill Jobs](https://www.windmill.dev/docs/core_concepts/jobs)、[ComfyUI communication](https://docs.comfy.org/development/comfyui-server/comms_overview)、[Node-RED Admin types](https://nodered.org/docs/api/admin/types)、[n8n Executions](https://docs.n8n.io/workflows/executions/all-executions/)。

### 9.2 单一 NodeSpec 与生成链

当前 node.Spec 应扩展但保持“单一事实源”：

    NodeSpec {
      kind,
      display,
      inputs,
      outputs,
      configSchema,
      execution: {
        effect: pure | read | write | external | input | human,
        deterministic,
        cachePolicy,
        retrySafety: safe | idempotent-with-key | unsafe,
        cancellation,
        defaultTimeout
      },
      capabilities[],
      secrets[],
      observability: {
        sensitiveFields[],
        outputRecording
      }
    }

生成链：

    Go NodeSpec
      -> catalog.json
      -> JSON Schema
      -> generated TypeScript types
      -> Inspector controls
      -> node reference docs
      -> lint/test fixtures

宿主硬规则不能由节点覆盖：

- 端口与 config 类型、required/range、图结构、能力、Secret 流向；
- pure 节点不得请求 input/process/network/filesystem-write 等能力；
- unsafe 节点不得配置自动 retry 或 cache；
- sensitiveFields 不进入普通日志、metrics attribute 或诊断包；
- 每个官方节点必须 exactly-one 执行 capability，并通过 panic/error/cancel 测试。

这吸收 n8n declarative node 与 lint 的工程优势，同时避免 Node-RED 双份 JS/HTML 定义和 ComfyUI 自定义校验绕过：[n8n node approach](https://docs.n8n.io/integrations/creating-nodes/plan/choose-node-method/)、[n8n linter](https://docs.n8n.io/integrations/creating-nodes/test/node-linter/)、[Node-RED first node](https://nodered.org/docs/creating-nodes/first-node)、[ComfyUI server validation](https://docs.comfy.org/custom-nodes/backend/server_overview)。

### 9.3 本地持久 Run 账本

不改变“单 Worker 串行占用输入设备”的合理设计，但用 SQLite/WAL 或等价事务存储替换“内存队列即事实源”：

    RunRecord {
      runId: UUIDv7,
      trigger,
      workflowId,
      workflowRevision,
      programHash,
      permissionGrantHash,
      status,
      queuedAt,
      startedAt,
      endedAt,
      termination,
      rootError
    }

    NodeAttempt {
      runId,
      graphPath,
      nodeId,
      nodeKind,
      attempt,
      effect,
      status,
      startedAt,
      endedAt,
      retryReason,
      errorCode,
      adapterSummary
    }

状态机至少包含 QUEUED、RUNNING、SUCCEEDED、FAILED、CANCELLED、INTERRUPTED。要求：

- 写入 QUEUED 与 ProgramSnapshot 引用后才通知 Worker；
- Worker 取得 Run 后以事务转换为 RUNNING；
- 应用启动时把遗留 RUNNING 转成 INTERRUPTED，绝不自动重放 unsafe 节点；
- CANCELLED 表示用户/系统明确取消，INTERRUPTED 表示进程或设备异常中断，两者不能混淆；
- 清理策略可以删除大 payload，但 Run 元数据、错误码和程序哈希保留；
- 工作流保存历史、ProgramStore、RunStore 分表/分接口。

Windmill 的父 job/子 job 与 zombie 检测证明这种事实模型的价值，但 Yotta 只取本地事务语义，不取 PostgreSQL 分布式队列：[Windmill Jobs](https://www.windmill.dev/docs/core_concepts/jobs)、[Critical alerts](https://www.windmill.dev/docs/core_concepts/critical_alerts)。

### 9.4 重试、缓存与重放矩阵

| effect | 示例 | 自动重试默认 | 缓存默认 | 崩溃后处理 |
| --- | --- | --- | --- | --- |
| pure | 数学、表达式、确定性数据转换 | 可，有限次数 | 可 | 可重算 |
| read | 读变量、读稳定本地元数据 | 仅明确瞬态错误 | 默认否 | 重新读取并提示状态可能变化 |
| write | 写文件、写变量、发消息 | 否；有 idempotency key 才可开启 | 否 | INTERRUPTED |
| external | HTTP、进程、ADB | 否 | 否 | INTERRUPTED |
| input | 点击、键盘、窗口操作 | 否 | 否 | INTERRUPTED |
| human | 等待人工、审批、验证码 | 否 | 否 | 显式 checkpoint 才可恢复 |

LLM 默认归 external：即使 prompt 相同，模型、服务状态和采样也可能变化；保存 request metadata 和 provider request ID，不把输出称为 deterministic replay。

Temporal 的 Activity/Workflow 边界和 ComfyUI 的纯节点缓存是参照，但 Yotta 的桌面副作用需要更保守默认：[Temporal Retry Policies](https://docs.temporal.io/encyclopedia/retry-policies)、[ComfyUI server execution](https://docs.comfy.org/custom-nodes/backend/server_overview)。

### 9.5 可观测性：Run 时间线是产品能力，不只是日志

建议统一事件树：

    workflow.run
      -> graph.execute
        -> node.attempt
          -> adapter.action

所有事件最少携带 run_id、program_hash、workflow_id、graph_path、node_id、node_kind、attempt、status 与 duration。实现要求：

1. RunStore 是状态与时间线事实源；JSONL 是诊断导出，不是唯一事实库。
2. UI 提供按 Run 查看：触发源、所用 revision/hash、能力授权、节点时间线、重试、取消/中断原因、结构化诊断。
3. metrics 只用低基数标签；run_id、node_id 不作为 Prometheus label，但可作 trace/log attribute。
4. OpenTelemetry exporter 默认关闭、本地优先；开启前清楚列出发送内容。
5. prompt、截图、OCR 文本、键入内容、文件路径、Secret 和原始节点输出默认不进入 spans/metrics。附件需用户显式选择并设置保留期。
6. 现有 action trace 脱敏保留，并补 run_id、node_id、attempt；不能把原始 payload 又镜像到 generic logs。

n8n 已建议 execution/workflow/session 关联字段，Windmill 展示了 queue/worker/zombie 指标面：[n8n Logging](https://docs.n8n.io/hosting/logging-monitoring/logging/)、[Windmill OpenTelemetry](https://www.windmill.dev/docs/misc/guides/otel)。

### 9.6 插件路线：先作者体验，再第三方代码

3.1 建议分三道门：

**门 A：官方 Node SDK（本轮必须）**

- 节点仍编译进主程序；
- 提供脚手架、lint、fake capability test kit、golden workflow、自动文档；
- 所有官方节点完成 effect/capability/sensitive 字段声明；
- Catalog 和 TS 全生成。

**门 B：只声明不执行的扩展（随后）**

- 允许主题、模板、Workflow 示例、节点文档包；
- manifest、签名、版本不可变、publisher 身份、deprecate/yank；
- 不加载第三方 Go/Python/JS。

**门 C：进程外可执行插件（满足安全门禁后）**

- 版本化 IPC，不用 Go plugin ABI；
- Runner 无直接 ServiceBundle，只能请求 capability broker；
- CPU/内存/时间/子进程/文件/网络策略；
- 精确制品 digest 写入 ProgramSnapshot；
- panic/crash 只失败当前 attempt，不拖垮 Editor；
- 未信任 workflow 的危险能力默认拒绝。

这一顺序同时吸收 VS Code 的贡献点/实验 API/Trust 和 Comfy Registry 的不可变发布，又避开两者各自的任意进程权限或网页脚本风险：[VS Code Contribution Points](https://code.visualstudio.com/api/references/contribution-points)、[VS Code Proposed API](https://code.visualstudio.com/api/advanced-topics/using-proposed-api)、[VS Code Workspace Trust](https://code.visualstudio.com/api/extension-guides/workspace-trust)、[Comfy Registry](https://docs.comfy.org/registry/overview)。

## 10. 贡献者体验升级

一个大型开源项目不能要求贡献者通过阅读 main.go、复制旧节点和手工同步 Vue 类型来学习系统。建议把“新增一个官方节点”收敛为可机械验证的路径：

    yotta dev node new <kind>
    yotta dev node lint ./...
    yotta dev node test <kind>
    yotta dev catalog generate
    yotta dev docs generate
    yotta dev check

node new 生成：

- NodeSpec 与执行骨架；
- 正常、校验失败、取消、panic/error、缺能力、敏感日志测试；
- fake Service capabilities；
- 最小 WorkflowSource fixture；
- 中英文文档占位和示例；
- 无手写前端 DTO。

CI 门禁：

1. Go unit/race/architecture tests；
2. Compiler golden + canonical hash 跨平台测试；
3. 所有 NodeSpec schema 校验与 effect/capability 规则；
4. 生成 catalog/TS/docs 后 git diff 必须为空；
5. 前端 typecheck、lint、component tests；
6. Editor -> CompileDraft -> RunStore -> Worker 的最小端到端测试；
7. representative deterministic history replay；
8. kill -9 / 强制进程退出后 Run 被标记 INTERRUPTED；
9. 诊断包与 logs 的 Secret/prompt/screenshot 泄漏测试；
10. PR 模板要求变更契约、测试、文档、breaking note。

n8n 的分包开发与 Testcontainers、Node-RED 的真实 flow test helper、ComfyUI 的 scaffold/publish flow 是直接参照：[n8n CONTRIBUTING](https://github.com/n8n-io/n8n/blob/master/CONTRIBUTING.md)、[Node-RED first node](https://nodered.org/docs/creating-nodes/first-node)、[ComfyUI publishing](https://docs.comfy.org/registry/publishing)。

## 11. 建议的落地顺序与硬门禁

### P0：先冻结不可逆契约

1. WorkflowSource v3、稳定 diagnostics 和 canonical encoding；
2. CompileDraft(source, catalog)；删除“保存后才能检查”的产品与代码路径；
3. ProgramSnapshot 与 programHash；
4. NodeSpec effect/capability/retry/cache/sensitive 声明；
5. formatVersion=2 明确拒绝，无 fallback。

**门禁：** Editor、保存、预览、运行全部只消费上述新契约；仓库内不存在通过 container ID 在 Worker 内临时读取并编译当前文件的路径。

### P1：执行可靠性与可观测性

1. 本地 RunStore + UUIDv7；
2. QUEUED/RUNNING/终态事务状态机；
3. NodeAttempt 与 adapter action 事件；
4. 安全重试矩阵、取消原因和 INTERRUPTED 恢复；
5. Run 时间线 UI、基础 metrics、可选 OTel。

**门禁：** 任意时刻强制终止应用，重启后没有“仍在运行”的幽灵 Run，也不会自动重复键鼠/进程/文件/HTTP/LLM 副作用。

### P2：编辑器历史与增量反馈

1. revision 乐观并发；
2. workflow history 与 execution history 分离；
3. sourceHash / compiledHash / runHash 三态 dirty UI；
4. 纯节点的宿主管理缓存；
5. 原快照重跑与当前草稿新 Run 明确分开。

**门禁：** Run 入队后继续编辑、保存甚至切换 workflow，都不能改变该 Run 的程序；用户能从 UI 看到它实际绑定的 hash/revision。

### P3：官方 Node SDK 与贡献者通道

1. node new/lint/test；
2. Catalog、TS、Inspector、docs 全生成；
3. fake capability 与官方节点契约测试；
4. 一条本地命令复现 CI。

**门禁：** 新增一个普通声明式节点无需修改中央 switch、手写前端 DTO 或复制 Inspector 组件。

### P4：第三方生态，满足条件才启动

1. PluginManifest v1alpha1；
2. workflow trust 与 capability broker；
3. 不可变签名制品与 lock；
4. 进程外 Runner、资源限制与崩溃隔离；
5. 实验 API 升 stable 的公开准则。

**门禁：** 在没有 Runner 隔离、能力代理、签名/制品锁和撤权 UI 前，不接受第三方可执行插件。这不是兼容兜底，而是供应链安全的发布条件。

## 12. 可验收的 3.1 结果

- 修改未保存草稿后点击“检查”，Compiler 直接校验内存 Source；磁盘内容不变。
- 相同 Source/Catalog/Compiler 在 CI 支持的平台上 programHash 一致。
- 一个 Run 入队后修改/保存当前 workflow，Run 仍执行原 ProgramSnapshot。
- 导入 formatVersion=2 得到单一、稳定、可定位的 unsupported format 诊断；代码中没有 v2 Runtime 分支。
- 应用在任意 input/process/file/HTTP/LLM 节点后被强制终止，重启后 Run 为 INTERRUPTED，且不会自动重试。
- pure 节点可以命中缓存；任何声明危险能力或非确定性的节点无法启用缓存。
- 给 unsafe 节点配置 automatic retry 时编译失败，而不是运行时静默忽略。
- Run 时间线能完整展示 graph/node/attempt/adapter 层级，所有关联字段一致。
- 默认日志、trace 和诊断包中搜索不到 Secret、原始 prompt、键入文本和截图内容。
- 所有官方节点均通过 Spec、生成契约、能力、取消、错误和敏感数据门禁。
- 新贡献者可用一条命令生成并验证节点；CI 与本地命令同源。
- 第三方插件能力未达到 P4 门禁前，产品中不存在“加载任意代码”的隐藏入口。

## 13. 最重要的五个决定

1. **编译草稿，不编译“某个 ID 当前保存的文件”。**
2. **Run 固定不可变 ProgramSnapshot；这叫执行正确性，不叫旧版本兼容。**
3. **以 effect 为中心设计 retry、cache、replay 和 permission；桌面副作用默认一次执行。**
4. **保留单机串行 Worker，但用持久 Run/Attempt 账本替换内存状态作为事实源。**
5. **3.1 把官方 Node SDK、生成契约和能力门禁做完整；第三方可执行插件只能经 Wasm/Process Host 与 capability broker。**

## 14. 一手来源索引

### n8n

- [Queue mode](https://docs.n8n.io/hosting/scaling/queue-mode/)
- [Task runners](https://docs.n8n.io/hosting/configuration/task-runners/)
- [Node versioning](https://docs.n8n.io/integrations/creating-nodes/build/reference/node-versioning/)
- [Workflow history](https://docs.n8n.io/workflows/history/)
- [Executions](https://docs.n8n.io/workflows/executions/all-executions/)
- [Dirty nodes](https://docs.n8n.io/workflows/executions/dirty-nodes/)
- [Node development environment](https://docs.n8n.io/integrations/creating-nodes/build/node-development-environment/)
- [Node linter](https://docs.n8n.io/integrations/creating-nodes/test/node-linter/)
- [Community node risks](https://docs.n8n.io/integrations/community-nodes/risks/)
- [Security audit](https://docs.n8n.io/hosting/securing/security-audit/)
- [Monitoring](https://docs.n8n.io/hosting/logging-monitoring/monitoring/)
- [Logging](https://docs.n8n.io/hosting/logging-monitoring/logging/)
- [n8n CONTRIBUTING](https://github.com/n8n-io/n8n/blob/master/CONTRIBUTING.md)

### Node-RED

- [API reference](https://nodered.org/docs/api/)
- [Runtime package](https://github.com/node-red/node-red/blob/main/packages/node_modules/@node-red/runtime/README.md)
- [Creating nodes](https://nodered.org/docs/creating-nodes/)
- [Creating your first node](https://nodered.org/docs/creating-nodes/first-node)
- [Node runtime](https://nodered.org/docs/creating-nodes/node-js)
- [Node editor UI](https://nodered.org/docs/creating-nodes/node-html)
- [Packaging nodes](https://nodered.org/docs/creating-nodes/packaging)
- [Admin API types](https://nodered.org/docs/api/admin/types)
- [Runtime flows API](https://nodered.org/docs/api/modules/v/1.3/%40node-red_runtime_flows.html)
- [Storage API](https://nodered.org/docs/api/storage/methods/)
- [Context](https://nodered.org/docs/user-guide/context)
- [Projects](https://nodered.org/docs/user-guide/projects/)
- [Runtime configuration](https://nodered.org/docs/user-guide/runtime/configuration)
- [Handling errors](https://nodered.org/docs/user-guide/handling-errors)
- [Node-RED CONTRIBUTING](https://github.com/node-red/node-red/blob/master/CONTRIBUTING.md)

### Windmill

- [Flow architecture](https://www.windmill.dev/docs/flows/architecture)
- [OpenFlow](https://www.windmill.dev/docs/openflow)
- [OpenFlow OpenAPI source](https://github.com/windmill-labs/windmill/blob/main/openflow.openapi.yaml)
- [Jobs](https://www.windmill.dev/docs/core_concepts/jobs)
- [Worker groups](https://www.windmill.dev/docs/core_concepts/worker_groups)
- [Versioning](https://www.windmill.dev/docs/core_concepts/versioning)
- [Error handling](https://www.windmill.dev/docs/flows/error_handling)
- [Retries](https://www.windmill.dev/docs/flows/retries)
- [Critical alerts and zombie jobs](https://www.windmill.dev/docs/core_concepts/critical_alerts)
- [OpenTelemetry](https://www.windmill.dev/docs/misc/guides/otel)
- [CLI sync](https://www.windmill.dev/docs/advanced/cli/sync)
- [Workflows as code](https://www.windmill.dev/docs/core_concepts/workflows_as_code)
- [Windmill official repository](https://github.com/windmill-labs/windmill)

### Temporal

- [Workflow Execution](https://docs.temporal.io/workflow-execution)
- [Retry Policies](https://docs.temporal.io/encyclopedia/retry-policies)
- [Worker Versioning](https://docs.temporal.io/worker-versioning)
- [Go testing suite](https://docs.temporal.io/develop/go/best-practices/testing-suite)

### VS Code

- [Extension Host](https://code.visualstudio.com/api/advanced-topics/extension-host)
- [Contribution Points](https://code.visualstudio.com/api/references/contribution-points)
- [Activation Events](https://code.visualstudio.com/api/references/activation-events)
- [Using Proposed API](https://code.visualstudio.com/api/advanced-topics/using-proposed-api)
- [Workspace Trust](https://code.visualstudio.com/api/extension-guides/workspace-trust)
- [Extension Anatomy](https://code.visualstudio.com/api/get-started/extension-anatomy)

### ComfyUI

- [Client-server communication](https://docs.comfy.org/development/comfyui-server/comms_overview)
- [Server execution and validation](https://docs.comfy.org/custom-nodes/backend/server_overview)
- [Custom node lifecycle](https://docs.comfy.org/custom-nodes/backend/lifecycle)
- [JavaScript extensions](https://docs.comfy.org/custom-nodes/js/javascript_overview)
- [Registry overview](https://docs.comfy.org/registry/overview)
- [Registry standards](https://docs.comfy.org/registry/standards)
- [Publishing nodes](https://docs.comfy.org/registry/publishing)
- [Custom node walkthrough](https://docs.comfy.org/custom-nodes/walkthrough)
- [Installing custom nodes](https://docs.comfy.org/installation/install_custom_node)
