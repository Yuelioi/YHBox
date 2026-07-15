# Yotta Workflow Automation

Yotta 将可编辑的自动化意图编译成不可变程序，再把程序绑定到本机能力执行。可编辑事实、可执行事实和运行事实必须使用不同身份。

## Language

**Workflow Source**:
用户或 AI 可编辑的 3.1 自动化文档，带文档身份与 revision，但本身不可执行；它属于 Yotta 3.1 产品发布线。
_Avoid_: Container, blueprint, flow

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
由 Node Contract 和 Data Type schema 派生的编辑事实，包含画布端口、参数表单、校验提示、平台可用性和帮助内容。
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

**Capability Plan**:
Compiler 从每个 Effective Node Contract 汇总出的不可变最小权限清单，保留来源节点、所需操作、target slot、credential slot 和 scope。
_Avoid_: Permission list, service bundle

**Run Grant**:
Policy 在一次 Run 开始前签发的短期、不透明授权，精确绑定 Program、Capability Plan、principal、provider、target、operation、scope 与 expiry。
_Avoid_: Requested capability, access token, approval flag

**Credential Binding**:
宿主把 Program 中的 credential slot 绑定到安全存储中 credential 的运行期事实；secret material 不进入 Source、Program、Value Envelope 或 Run Grant projection。
_Avoid_: API key config, secret value

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
一次 Run 的不可变 generational durable state，绑定 Program、Catalog、Capability Plan、Run Grant、policy generation、状态、时间、稳定错误与 durable Run Value；只能通过 Run Store CAS 产生下一代。
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
