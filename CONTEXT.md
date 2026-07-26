# Yotta Workflow Automation

Yotta 将可编辑的自动化意图编译成不可变程序，再把程序绑定到本机能力执行。可编辑事实、可执行事实和运行事实必须使用不同身份。

## Language

**Workflow Source**:
用户或 AI 可编辑的版本化自动化文档，带文档身份与 revision，但本身不可执行；其合同版本独立于 Yotta 产品发布版本。
_Avoid_: Container, blueprint, flow

**Product Release Version**:
标识一次 Yotta 应用发布的 SemVer；它用于构建、安装包和展示，不表达 artifact、接口、协议或存储兼容性。
_Avoid_: Contract version, protocol generation

**Artifact Contract Version**:
某一种持久化 artifact 的独立格式代际；它与 `format` 共同选择严格 reader 和显式 migration chain。
_Avoid_: Product version, revision

**Host Interface Version**:
宿主与可执行扩展进行兼容判断的接口代际；支持范围由宿主接口 owner 定义，不跟随产品发版自动提升。
_Avoid_: Product version, plugin package version

**Wire Protocol Version**:
两个进程或传输端点解释消息时使用的协议代际；每种协议由自己的 module 拥有。
_Avoid_: Product version, transport name

**Storage Layout Version**:
某个本地 store 的目录、marker 与提交语义代际；不同 store 分别演进。
_Avoid_: Artifact contract version, product version

**Revision**:
同一实体内容变化的顺序计数；它不选择 parser，也不表示兼容范围。
_Avoid_: Version

**Workflow Release**:
外部发布工作流的内部不可变来源版本，精确绑定 Workflow Source、Workflow Resource 与 Node Package 依赖；它不是用户工作流类型或执行许可。
_Avoid_: Installed workflow, executable workflow, run authorization

**Workflow Installation**:
外部导入工作流的内部本机记录，关联来源版本、本机目标/凭据、计划与更新状态；产品界面统一把它呈现为 Workflow。
_Avoid_: User-visible workflow type, execution permission

**Installation Lifecycle**:
Workflow Installation 在本机的持久存在状态，例如 active 或 archived；它不表达依赖、目标或凭据是否齐全。
_Avoid_: Readiness, setup status

**Readiness Report**:
对一个工作流当前可执行性的投影，列出 dependency、target 与 credential blocker 及修复动作；它不推进 lifecycle。
_Avoid_: Installation status, setup step

**Platform Delisting**:
在线平台停止提供某个 Workflow Release 的新下载；它不是撤销、远程停用或本地信任策略变更。
_Avoid_: Release revocation, remote disable

**Publisher Namespace**:
工作流或节点包发布者的稳定命名空间，由 User 或 Organization owner 持有但不等于其可变用户名。
_Avoid_: Username, account ID

**Publisher Attestation**:
用户中心对“已认证发布者声明了某个精确 release digest”的长期可验证证明；它不同于平台审核后的上架证明。
_Avoid_: OIDC token, platform publication proof

**Platform Publication Proof**:
Registry 审核通过后对某个精确 release digest 曾获准上架的长期证明；后续下架不改写这项历史事实。
_Avoid_: Publisher attestation, remote authorization

**Installation Plan**:
一次在线或离线安装所需的精确 Workflow Release、Node Package Release、digest、大小与可用状态投影；它不自动配置本机目标或凭据。
_Avoid_: Offline pack, execution grant

**Compile Result**:
对某一份 Workflow Source 和 Catalog Snapshot 的完整诊断结果；没有 error diagnostics 时才包含 Program Snapshot。
_Avoid_: Validation result, build response

**Program Snapshot**:
Compiler 产生的不可变、内容寻址可执行事实；后续编辑 Workflow Source 不改变它。
_Avoid_: Compiled container, runtime config

**Catalog Snapshot**:
一次编译所绑定的不可变节点目录代际，包含节点 contract 与实现锁。
_Avoid_: Current registry, node list

**Node Type**:
具有稳定身份和版本化语义的一类节点定义；它描述端口、配置、执行特征、能力要求和展示注解。
_Avoid_: Node kind, Go node

**Node Instance**:
Workflow Source 中对某个 Node Type 的一次具名使用，包含实例配置和连线。
_Avoid_: Graph node, node config

**Node Contract**:
Node Type 的可序列化、版本化事实，可被 Compiler、运行时、编辑器、目录和文档共同消费。
_Avoid_: Backend spec, frontend node metadata

**Data Type**:
带命名空间和版本的连线值语义，定义其 schema、兼容关系、序列化形式和展示注解；Node Package 可以发布新的 Data Type。
_Avoid_: Pin color, Go type, TypeScript union

**Type Reference**:
对一个精确、版本化 Data Type 及其 semantic digest 的引用；持久化和跨执行边界不得只保存宽泛类型名。
_Avoid_: Type tag, PinType

**Resolved Type**:
运行值的完整具体类型，仅由精确 Type Reference 或递归 list 构成；它不得包含 union 或尚未解析的类型变量。
_Avoid_: Runtime TypeExpr, inferred value type

**Value Envelope**:
携带完整 Resolved Type，并以 inline、blob、stream 或 handle 之一承载运行值的封闭联合；它不包含输入是否提供或运行血缘。
_Avoid_: Any value, output data map

**Binding State**:
输入绑定是否 absent 或 present 的独立状态；present-null、present-default 和 absent 必须保持可区分。
_Avoid_: Nullable input, defaulted value

**Workflow Resource**:
归属于一个工作流并随其完整导出、导入的图片、Macro 或 InputClip；它不自动成为用户的可复用资产。
_Avoid_: Local asset, embedded file

**Global Asset**:
独立于任何单个工作流、由用户管理并可被多个工作流选用的资源；资产库中创建的资源默认属于此类。
_Avoid_: Shared file, common blob

**Resource Promotion**:
用户将 Workflow Resource 显式提升为 Global Asset 的领域动作；提升不改变原工作流的资源归属。
_Avoid_: Automatic import, asset copy

**Blob Reference**:
通过 media type、内容摘要和字节大小引用不可变大对象的可持久化值表示；存储位置不属于其身份。
_Avoid_: Base64 payload, image bytes

**Blob Store**:
拥有 immutable content-addressed object、quota、integrity、range read 与 Sweep 生命周期的唯一模块；上层不能读取其路径或直接删除对象。
_Avoid_: Asset blob folder, file cache

**Stream Reference**:
对带类型、背压、取消和明确终态的运行期增量通道的引用；它不是可持久化 list。
_Avoid_: List, byte slice

**Resource Reference**:
由宿主 Resource Broker 解析、限定 authority、scope、owner 和 operation 的临时 capability token；原始 HWND、指针和文件描述符不得跨边界传递。
_Avoid_: HWND, pointer, handle object

**Resource Broker**:
在授权后签发 Run-scoped opaque lease、解析 provider object，并统一执行 operation check、borrow/drop、expiry、cancel 与 crash cleanup 的宿主边界。
_Avoid_: Handle registry, service locator

**Resource Lease**:
Resource Broker 中绑定 Run、invocation、kind、operation set 与 expiry 的临时 authority；borrow 只能收窄，最后一个 lease drop 才释放对象。
_Avoid_: Resource ID, reusable token

**Resource Lease Binding**:
Node Contract 数据端口对某个 Capability Requirement 及其 operation 子集的显式绑定；只有带该绑定的 runtime-only Stream/Resource Value 才能跨 data edge 借用 authority。
_Avoid_: Implicit handle passing, ambient resource access

**Conversion**:
Workflow Source 中显式可见的 Data Type 转换；类型系统不得通过隐式 coercion 改变连线值的语义。
_Avoid_: Auto-cast, runtime coercion

**Authoring Projection**:
由精确 Catalog、Node Contract、Data Type 与 Capability Definition 派生并可严格重开的非语义编辑事实，包含画布端口、参数表单、默认值提示、校验约束、值生命周期、target/credential/risk、平台可用性和帮助内容；编辑器与文档不得重新解释 raw contract/schema。
_Avoid_: Frontend registry, pin spec

**Editor Adapter**:
Yotta 为无法由通用 schema 表单充分表达的复杂交互提供的内置编辑能力；它不拥有或改写 Node Contract。
_Avoid_: Custom node UI, plugin JavaScript

**Data Channel**:
在节点之间传递带 Data Type、值身份和来源关系的业务数据。
_Avoid_: Value pin, data wire

**Exec Channel**:
表达 effect、control 和 event 节点之间执行顺序与分支选择的控制关系，不携带业务值。
_Avoid_: Out pin, flow wire

**Error Channel**:
传递带稳定 code、来源、重试属性和修复建议的结构化失败。
_Avoid_: Fail output, error string

**Status Event**:
Run 中的进度、等待和连接状态事实，不是 Workflow Source 中的普通连线值。
_Avoid_: Status output, progress pin

**Execution Class**:
Node Type 的执行类别，取 pure-data、effect、control 或 event；类别决定其可拥有的通道与调度方式。
_Avoid_: Runnable kind, pure flag

**Recorded Value**:
Run 首次产生后以运行事实保存、后续重放直接复用的非确定性值。
_Avoid_: Random cache, nondeterministic result

**Capability Requirement**:
Node Contract 声明的精确、版本化外部能力、操作、目标种类与 scope 需求；它只表达需求，不代表 Run 已获得授权。
_Avoid_: Runtime service, platform target

**Capability Definition**:
Catalog 中内容寻址的能力语义，封闭定义 operation set、target kind、scope schema、credential mode、risk/consent class 与 provider ABI；改变授权语义必须发布新 `/vN`。
_Avoid_: Capability name, permission string

**Host Profile**:
一次运行宿主不可由 workflow 伪造的平台事实与已安装 provider inventory，例如 OS、architecture 和可提供的 capability。
_Avoid_: Environment variables, platform capability

**Automation Target**:
可被 capability provider 操作的具名对象，例如 host desktop、Android device 或 After Effects instance；它不是授权本身。
_Avoid_: Platform, credential, capability

**Target Profile Definition**:
工作流对逻辑自动化目标的设置形状、首次安装默认值与发现提示的可移植定义；它不是用户在某台宿主上的实际配置。
_Avoid_: Target installation, user target settings

**Automation Target Profile**:
用户在一台宿主上为 Target Profile Definition 选定的应用、目标选项与校准配置；它可具有工作流或全局归属。
_Avoid_: Target definition, device handle

**Workflow Target Profile**:
归属于某个工作流安装的本机 Automation Target Profile；它不参与 Workflow Source 身份且在工作流升级时保留。
_Avoid_: Embedded target, workflow default

**Global Target Profile**:
独立于单个工作流、由用户管理并可显式用来初始化或重绑定 Workflow Target Profile 的 Automation Target Profile。
_Avoid_: Default target, shared handle

**Target Installation**:
在一台宿主上将 Automation Target Profile 解析为精确本机身份、provider 与授权的事实；它不随工作流分发。
_Avoid_: Target profile, workflow target

**Target Planner**:
Run admission 内把 Capability Plan 的 target/credential slot 与可信 Host Profile 中的 provider、Automation Target 和 non-secret credential metadata 做精确消歧的模块；零候选、歧义或 host/ABI/digest 不兼容都在 Policy 与 provider effect 前失败。
_Avoid_: Provider lookup, default target, service discovery

**Policy Admission**:
对已经完成 target planning 的精确 permission request 做统一 allow/deny/consent-required 决策，批准后 seal 短期 Run Grant 并先持久创建 QUEUED RunRecord；GUI、headless、AI、MCP 与插件不得绕过。
_Avoid_: Local trust, allowlist, run options

**Capability Plan**:
Compiler 从每个 Effective Node Contract 汇总出的不可变最小权限清单，保留来源节点、所需操作、target slot、credential slot 和 scope。
_Avoid_: Permission list, service bundle

**Run Grant**:
Policy 在一次 Run 开始前签发的短期、不透明授权，精确绑定 Program、Capability Plan、principal、provider、target、operation、scope 与 expiry。
_Avoid_: Requested capability, access token, approval flag

**Credential Binding**:
宿主把 Program 中的 credential slot 绑定到安全存储中 credential 的运行期事实；secret material 不进入 Source、Program、Value Envelope 或 Run Grant projection。
_Avoid_: API key config, secret value

**Workflow Credential Profile**:
归属于某个工作流安装的本机 credential 配置与安全存储绑定；它不随工作流分发或升级。
_Avoid_: Embedded secret, workflow API key

**Global Credential Profile**:
由用户在本机管理并可显式绑定到多个工作流的 credential 配置；其 secret 仍只存在安全存储中。
_Avoid_: Shared secret value, exported credential

**Node Package**:
可独立安装和验证的节点发布制品，包含一个或多个 Node Contract 及其可执行实现。
_Avoid_: Go plugin, node bundle

**Plugin Host**:
Yotta 用于发现、验证并隔离执行第三方 Node Package 的宿主能力。
_Avoid_: Dynamic Go loader, extension registry

**Wasm Node**:
由 Plugin Host 在受限 WebAssembly 环境中执行的 Node Type 实现，默认不能直接访问外部能力。
_Avoid_: Script node, embedded plugin

**Process Node**:
由 Plugin Host 在独立本机进程中执行、通过版本化协议交换值与运行事件的 Node Type 实现。
_Avoid_: Go plugin, child command

**Diagnostic**:
Compiler 对 Source 的稳定机器可读判断，以 code、位置、params 和 optional fix 表达；message 只用于展示。
_Avoid_: ValidationError, error message

**Run**:
对一个 Program Snapshot 的一次有身份执行，记录 program hash 与调用绑定。
_Avoid_: Container run, workflow execution

**Run Record**:
一次 Run 的不可变 generational durable state，绑定 Program、Catalog、Capability Plan、完整 non-secret Run Grant artifact、policy generation、状态、时间、稳定错误与 durable Run Value；只能通过 Run Store CAS 产生下一代。
_Avoid_: Runtime state map, container status

**Run Value**:
RunRecord 中包裹 Value Envelope 的 provenance，记录 value ID、graph/node/port、attempt 与 envelope digest，但不改变值本身的摘要。
_Avoid_: Output map, traced envelope

**Run Owner**:
一次 admitted Run 的临时 composition owner，独占 cancellable context、Grant Authorizer、Resource Broker 与 provider object 生命周期；终止后不可重开。
_Avoid_: Global runtime, service bundle

**Node Attempt**:
某个 Node Instance 在一次 Run 中的一次具名执行事实；以 started 和 terminal 事实包围本次实现调用，记录时间、稳定错误与脱敏 lineage，重试会产生新的 attempt 而不是改写旧事实。
_Avoid_: Retry counter, debug log

**Adapter Action**:
Node Attempt 内由 adapter 主动记录的真实 effect 动作；必须匹配 Node Contract 声明的 Effect，使用稳定 action/error code 与只含 code、数值 counters 的脱敏摘要。
_Avoid_: Synthesized effect log, raw error, debug message
