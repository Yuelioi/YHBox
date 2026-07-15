# Yotta 3.1 内置脚本节点：安全执行隔离、typed ABI 与可重放 attempt journal

> 调研截点：2026-07-16。仅使用 Go、goja、WebAssembly/WASI、wazero、Microsoft Windows、Temporal 上游源码或官方规范/文档。文中的“Yotta 应当”是架构建议，不是这些项目或供应商的承诺。

## 决策摘要

Yotta 3.1 不应继续在桌面主进程里执行用户或 AI 生成的 JavaScript。推荐冻结为下面这条硬边界：

1. **内置 `Script` 节点保留 JavaScript 作者体验，但每个 execution attempt 都在一次性 `yotta-script-runner` 子进程中用 goja 执行。** 主进程只保留 admission、typed ABI、capability broker、journal 和生命周期控制。
2. **Windows 的 production launcher 必须同时使用 LPAC/AppContainer、Job Object 和独立进程。** AppContainer/LPAC 管权限，Job Object 管进程树、CPU/内存与强制终止，独立进程管崩溃和 Go heap 故障域；三者互补，不能互相替代。隔离创建失败就返回 `script.isolation_unavailable`，不得退回主进程执行。
3. **goja `Interrupt` 只作为 runner 内的快速协作式取消。** 它能掐断纯 JavaScript 死循环，却明确不能中断 Go 原生函数；最终强制取消由父进程 `TerminateJobObject` 完成。Go `context.Context` 只传递 deadline/cancel cause，不被当成 kill 或等待完成的承诺。
4. **WebAssembly 是第三方二进制插件和纯计算脚本的优选 guest 格式，不是对现有 JavaScript 作者体验的等价替换。** 即便使用 Wasm，仍建议置于 runner 进程中；默认不实例化通用 WASI，只导入 Yotta 明确批准的 typed host functions。
5. **脚本边界禁止 `any`、裸 `map[string]any`、Go 对象包装、任意路径和全 registry 枚举。** 输入、输出与 host call 都使用 Yotta 3.1 `TypeRef + Representation`，携带 type semantic digest，边界两侧都验证。`Script.Result: *` 应删除，输出类型必须在节点配置中声明。
6. **replay 只重算纯脚本，不重做副作用。** 时间、随机数、host-call 结果都来自父进程保存的 attempt history；每个 effect 必须先持久化 intent，再调用，再持久化 receipt/result。崩溃在 intent 与 result 之间时标记 `ambiguous`，不得静默重试非幂等操作。

这套方案适合 Yotta 的真实威胁模型：脚本多数并非恶意，但 AI 生成代码、死循环、内存爆炸、日志洪泛、错误的自动化调用和第三方脚本都必须按不可信输入处理。安全边界不应依赖作者是否“有意作恶”。

## 当前实现暴露出的边界缺口

当前仓库状态可从 `internal/nodes/script/script.go`、`internal/services/script/binding.go` 和 `internal/services/script/compile.go` 直接观察到：

- `goja.New()` 与 `RunProgram` 在 Yotta 主进程内运行；goja panic、宿主 binding bug、Go heap 压力与桌面应用共享故障域。
- watchdog 在 `ctx.Done()` 后调用 `vm.Interrupt`，能处理纯 JS 死循环，但同步绑定函数正在运行时不构成强制终止。
- `Install` 枚举所有 `ScriptBindable` 节点并注入全局函数，另注入 `Subgraph`、变量、参数、sleep 和 log；脚本的真实权限是动态 registry 的宽集合，不是节点 contract 中可审查、可签名的最小 capability 集。
- 动态输入通过 `vm.Set(k, in.Raw(k))` 进入 VM，调用参数和结果经 `Export()` / `map[string]any` / `NormalizeJS` 往返；最终结果是 wildcard `*`。这绕开了 Yotta 3.1 已经建立的 content-addressed datatype contract。
- `CompileCached` 是按完整源码字符串索引、无容量边界的进程级 map；即使单次执行可取消，反复提交不同源码仍可让主进程缓存持续增长。

goja 官方说明一个 `Runtime` 同一时刻只能由一个 goroutine 使用，而且不同 runtime 之间不能传递 object values。[goja README: goroutine safety](https://github.com/dop251/goja#is-it-goroutine-safe) 官方 `ToValue` 文档还说明 struct、map、slice 会被包装，修改会反映到原 Go 值；这正是脚本 ABI 不应把宿主对象直接暴露给 VM 的原因。[goja `Runtime.ToValue`](https://pkg.go.dev/github.com/dop251/goja#Runtime.ToValue)

## goja：可以取消执行，但不是内存或权限沙箱

### `Interrupt` 的精确语义

goja `Runtime.Interrupt(v)` 会让对应 Go 调用返回携带 `v` 的 `*InterruptedError`；若 interrupt 一直传播到空栈，已排队的 Promise resolve/reject jobs 会被清空。官方同时明确写明：它只在 JavaScript code 中生效，**不能中断 native Go functions，包括 built-ins**；如果 runtime 当时未运行，下一次 `Run*` 会立即被中断，复用前必须 `ClearInterrupt`。[goja `Runtime.Interrupt`](https://pkg.go.dev/github.com/dop251/goja#Runtime.Interrupt) [Yotta 当前锁定版本的 upstream source](https://github.com/dop251/goja/blob/348e6bea910dc4acc4df9de8942a24411265c0b0/runtime.go#L1510-L1528)

因此当前 binding 里的任一同步 Go 调用都决定了最坏取消延迟：

- binding 若正确轮询/传播 context，可以协作返回；
- binding 若阻塞在不支持 context 的库、系统调用或 bug 中，`vm.Interrupt` 不能把它抢占回来；
- binding 若在 Go 侧分配大量内存，interrupt 也不是 heap quota；
- interrupt 与 runtime 结束有竞态，复用 VM 需要额外同步；一次性 runner/VM 可以直接删除这类复用状态。

goja 暴露 `SetMaxCallStackSize` 来限制函数调用深度，默认值为 `math.MaxInt32`，官方将其描述为防止无限递归导致内存耗尽的措施；但其公开 Runtime API 没有对应的 JavaScript heap/allocation byte quota。[goja `Runtime.SetMaxCallStackSize`](https://pkg.go.dev/github.com/dop251/goja#Runtime.SetMaxCallStackSize) 所以 call-stack limit 必须显式设置，但它不能替代进程内存上限。

### Go context 的边界

Go 官方定义 `Context` 为跨 API 传递 deadline、cancel signal 和 request-scoped values 的值；调用 `CancelFunc` 会关闭派生 context 的信号并释放关联资源，但 `CancelFunc` **不等待工作真正停止**。[Go `context` package](https://pkg.go.dev/context) 特别是 `CancelFunc` 的文档直接说明它只是告诉 operation 放弃工作，且不会等待停止。[Go `context.CancelFunc`](https://pkg.go.dev/context#CancelFunc)

Yotta 因而应把一次脚本取消拆成可观察的阶段，而不是一个模糊的 `cancelled=true`：

```text
cancel requested
  -> context cause propagated
  -> runner cancel frame delivered
  -> engine interrupt requested
  -> cooperative exit acknowledged
  -> grace expired -> job force-terminated
  -> process reaped and accounting frozen
```

`context.Cause(ctx)` 用于保留 `user_cancelled`、`deadline_exceeded`、`run_shutdown`、`budget_exceeded` 等语义；最终 outcome 另记 `terminationStrength = cooperative | engine_interrupt | job_terminate | process_crash`。

Go 的 `exec.CommandContext` 默认只把 `Cancel` 设为对该 `Process` 调用 `Kill`，`WaitDelay` 默认未设置；文档还警告孤儿子进程可能继续持有 I/O pipe，使等待 EOF 卡住。[Go `os/exec.CommandContext`](https://pkg.go.dev/os/exec#CommandContext) [Go `Cmd.Cancel` / `WaitDelay`](https://pkg.go.dev/os/exec#Cmd) 因此 Windows runner 不能只靠 `CommandContext`，必须由 Job Object 收拢整个 process tree，并对 pipe/frame 另设 byte limit 与 wait deadline。

## WebAssembly/WASI：默认无 ambient authority，但资源预算仍由 embedder 负责

WebAssembly Core 规范保证模块在 memory-safe sandbox 中执行，并明确说明核心 Wasm 对计算环境没有 ambient access；I/O、资源和系统调用只能来自 embedder 提供的 imports。规范也明确把具体 capability policy 归为 embedder 的责任，并提醒直接在硬件上执行仍可能受 side-channel 影响。[WebAssembly 3.0 security considerations](https://webassembly.github.io/spec/core/intro/introduction.html#security-considerations)

WASI 官方将应用描述为从无 ambient authority 开始，只能使用 host 明确授予的 capability；同时 WASI 的 filesystem、sockets、CLI、HTTP 等是不同接口集合，不能把“使用 WASI”误解成自动获得全部系统能力。[WASI introduction](https://wasi.dev/) [WASI releases and proposal inventory](https://wasi.dev/releases)

对 Go embedder，wazero 当前稳定 API 提供几个很有价值的硬开关：

- `WithMemoryLimitPages` 限制每个 memory 的最大 page；默认上限是 65,536 pages，即模块未声明 max 时可达 4 GiB，所以 Yotta 必须显式降低，不能依赖默认值。[wazero `RuntimeConfig.WithMemoryLimitPages`](https://pkg.go.dev/github.com/tetratelabs/wazero#RuntimeConfig)
- `WithCloseOnContextDone(true)` 会在 function call 的 context 取消、deadline 到达或 module 被关闭时终止执行并关闭 module；该能力对不可信 Wasm 特别重要，且默认关闭。[wazero `RuntimeConfig.WithCloseOnContextDone`](https://pkg.go.dev/github.com/tetratelabs/wazero#RuntimeConfig)
- `ModuleConfig` 默认不继承 host 环境变量，不允许文件访问，stdin 返回 EOF，stdout/stderr 丢弃；wall/monotonic clock 和随机源也不是 host 的真实来源，除非 embedder 显式启用。[wazero `ModuleConfig`](https://pkg.go.dev/github.com/tetratelabs/wazero#ModuleConfig)

需要特别避免一个陷阱：wazero 的 `WithDirMount` 文档明确说 guest 可用 `../../` 逃出给定目录范围；read-only mount 也仍允许这种相对路径逃逸。`WithFSMount` 也不是 chroot。[wazero `FSConfig` isolation notes](https://pkg.go.dev/github.com/tetratelabs/wazero#FSConfig) 所以 Yotta 不应把宿主项目目录直接 mount 给脚本；文件访问继续走 resource broker，以 opaque token 表示已授权资源。

### 对 Yotta 的使用结论

- **内置文本脚本：**保留 goja，但只在一次性 runner 中运行。
- **第三方插件/预编译脚本：**采用 Wasm/component contract；默认只导入 `yotta:script/host`，不实例化 `wasi_snapshot_preview1`。确需 WASI 的能力逐项声明、逐项 admission，不能启用整包后再靠文档约束。
- **Wasm 也放进 runner：**线性内存安全不能隔离 runtime 自身 bug、host function bug、Go heap、JIT/AOT native code 或整个桌面进程 OOM。Wasm memory limit 是 guest memory 上限，Windows Job memory limit才是 runner 进程总 committed-memory 边界。
- **WIT 用作长期 typed contract 参考：**WIT 是 Component Model 的 IDL，`interface` 描述 typed functions，`world` 完整描述 imports/exports，并支持 record、variant、enum、list、option、result 和 resource handle。[WebAssembly WIT specification](https://github.com/WebAssembly/component-model/blob/main/design/mvp/WIT.md) Yotta 现有 datatype schema 可先作为唯一事实源生成 JS/Wasm bindings，不必等 Component Model runtime 才开始清理 `any`。

## Windows production isolation：三层同时存在

### 1. 独立进程：故障域与可回收 heap

一次 attempt 一个 runner process，完成后立即退出：

- goja runtime、编译产物、JS heap、计时器、日志 buffer 随进程一次性回收；
- runner panic、fatal error 或 OOM 不直接终止 Wails 主进程；
- 不池化 interrupted VM，不需要 `ClearInterrupt`，也不会跨 run 泄漏 globals/Promise jobs；
- runner 永远不持有数据库、secret store、registry、窗口 controller 或文件系统 service，只持有一条受限 IPC channel。

独立进程本身不是权限沙箱：普通子进程仍以当前用户身份拥有大量 ambient authority。因此 Windows production 还必须有 AppContainer/LPAC。

### 2. LPAC/AppContainer：权限边界

Microsoft 官方把 AppContainer 描述为基于 SID、token 和 DACL 的沙箱环境；进程只能访问显式允许的系统、其他应用和用户数据。没有 network capability 就不能访问网络，资源权限是 user/group SID 与 AppContainer SID 授权的交集；LPAC 比普通 AppContainer 更严格，连普通 AppContainer 默认能访问的 registry/COM 等也需要额外 capability。[Microsoft: Launch an AppContainer](https://learn.microsoft.com/en-us/windows/win32/secauthz/implementing-an-appcontainer) AppContainer 隔离覆盖 credentials、device、file/registry、network、process 和 window 等维度。[Microsoft: AppContainer isolation](https://learn.microsoft.com/en-us/windows/win32/secauthz/appcontainer-isolation)

Yotta runner profile 应采用 LPAC，并从 **零 capability** 开始：

- 不授予 `internetClient`、private network、broadFileSystemAccess、COM、camera/microphone 等能力；
- runner executable 目录只授予 AppContainer SID 的 read/execute；
- IPC named pipe/handle 只授予该 attempt 的 SID/进程所需访问；若采用 `PROC_THREAD_ATTRIBUTE_HANDLE_LIST`，仅把明确列出的句柄标成 inheritable，并按 Windows API 要求令 `bInheritHandles=TRUE`，不让其他句柄进入 list。[Microsoft: `PROC_THREAD_ATTRIBUTE_HANDLE_LIST`](https://learn.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-updateprocthreadattribute)
- source、input 与 result 经 IPC 传输，不经命令行、环境变量或临时明文文件；
- 如脚本要读写文件、网络、窗口或自动化设备，必须向主进程发 typed host call，由主进程 capability broker 代办；guest 永远拿不到 path、credential 或 native handle。

AppContainer 启动通过 `STARTUPINFOEX` 的 `PROC_THREAD_ATTRIBUTE_SECURITY_CAPABILITIES` 注入 package SID 与 capability SIDs；Microsoft 的官方示例给出了完整创建流程。[Microsoft: AppContainer process creation](https://learn.microsoft.com/en-us/windows/win32/secauthz/implementing-an-appcontainer#launching-the-appcontainer-or-lpac)

### 3. Job Object：资源、进程树与强终止

Windows Job Object 能把一组进程作为单元管理，默认情况下 job 内进程用 `CreateProcess` 创建的 children 也进入同一 job；`TerminateJobObject` 终止当前 job 及嵌套 child jobs 中的全部进程。[Microsoft: Job Objects](https://learn.microsoft.com/en-us/windows/win32/procthread/job-objects) `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE` 使最后一个 job handle 关闭时终止所有关联进程，适合主进程崩溃/退出时 fail closed。[Microsoft: `JOBOBJECT_BASIC_LIMIT_INFORMATION`](https://learn.microsoft.com/en-us/windows/win32/api/winnt/ns-winnt-jobobject_basic_limit_information)

每个 attempt 建一个 job，至少设置：

```text
JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
JOB_OBJECT_LIMIT_DIE_ON_UNHANDLED_EXCEPTION
JOB_OBJECT_LIMIT_ACTIVE_PROCESS = 1
JOB_OBJECT_LIMIT_PROCESS_MEMORY = configured runner committed-memory cap
JOB_OBJECT_LIMIT_JOB_MEMORY     = same or slightly higher aggregate cap
no BREAKAWAY_OK / no SILENT_BREAKAWAY_OK
CPU rate or per-job user-time budget
```

Job memory limit的精确语义是：超过 committed-memory limit 时新的 commit 失败，并可发 completion-port message；它不保证脚本收到一个可恢复的语言异常。因此 Yotta 要把 runner crash/exit、job notification 和最终 accounting 合并判定为 `budget.memory_exceeded`，不能只等一条通知。[Microsoft: job basic limits](https://learn.microsoft.com/en-us/windows/win32/api/winnt/ns-winnt-jobobject_basic_limit_information) `TerminateJobObject` 类似对每个关联进程调用 `TerminateProcess`，进程无法推迟或处理，正好提供 Go/goja 无法提供的最终 kill 语义。[Microsoft: `TerminateJobObject`](https://learn.microsoft.com/en-us/windows/win32/api/jobapi2/nf-jobapi2-terminatejobobject)

为避免“进程先运行、之后才 Assign 到 job”的窗口，Windows 10+/Server 2016+ 可在 `CreateProcess` 的 attribute list 中使用 `PROC_THREAD_ATTRIBUTE_JOB_LIST`，让新进程创建时就被加入指定 jobs；同一个 `STARTUPINFOEX` 同时携带 security capabilities。[Microsoft: `UpdateProcThreadAttribute`](https://learn.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-updateprocthreadattribute) 这是 Yotta Windows 3.1 launcher 应采用的唯一 production 路径。

## 建议冻结的脚本 execution contract

### 生命周期与组件

```text
workflow scheduler
  -> admission: script contract + declared grants + budgets
  -> append AttemptStarted
  -> platform launcher: LPAC + Job + one-shot runner
  -> runner validates ABI/source/input digests
  -> engine executes
       -> pure computation stays in guest
       -> effectful call -> typed IPC -> host capability broker
  -> terminal outcome validated and journaled
  -> job accounting queried, runner reaped, handles closed
```

主进程是 authoritative journal writer。runner 发送的日志和状态都只是待验证的 input；runner 不能直接写 run database，也不能决定自己拥有哪些能力。

### Wire envelope

Yotta 现有 datatype subsystem 已定义 `TypeRef{typeId, semanticDigest}`、`inline-json`、`blob-ref`、`stream-ref`、`handle-ref` 以及 `yotta.jcs/v1` 等 representation。脚本 ABI 应直接复用这些定义，不再建立第二套弱类型 JSON 约定。

```go
type ExecutionRequest struct {
    Format         string // "yotta.script-execution"
    Version        string // "3.1"
    AttemptID      string
    RunID          string
    InstructionID  string
    Engine         EngineRef       // kind + exact version/semantic digest
    ContractDigest Digest          // exact node/script ABI contract
    Source         SourceEnvelope  // utf-8 bytes + digest, never command line
    Limits         BudgetSnapshot
    Determinism    DeterminismSeed // virtual epoch, PRNG seed/version
    Inputs         map[string]TypedValue
    Grants         []GrantRef      // opaque, exact scope/digest/expiry
}

type TypedValue struct {
    TypeRef        datatype.TypeRef
    Representation datatype.RepresentationSpec
    Payload        json.RawMessage // canonical inline value or opaque resource ref
}

type HostCallRequest struct {
    AttemptID string
    Sequence  uint64
    CallID    string
    Grant     GrantRef
    Operation OperationRef // provider/ABI/operation semantic digests
    Inputs    map[string]TypedValue
}

type HostCallResult struct {
    AttemptID string
    Sequence  uint64
    CallID    string
    Outcome   Result[map[string]TypedValue, ScriptFailure]
    Receipt   *EffectReceipt
}

type ExecutionOutcome = Completed | Failed | Cancelled | Killed
```

每一帧必须有 byte length 上限、结构深度/节点数上限、严格 unknown-field rejection 和单调 sequence；frame 本身用长度前缀，内容使用 canonical JCS。RFC 8785 的 JCS 通过受限 JSON primitive serialization 与确定性 property sorting 生成可 hash 的表示，适合 Yotta 现有 digest/receipt 模型。[RFC 8785 JSON Canonicalization Scheme](https://www.rfc-editor.org/rfc/rfc8785.html)

### 类型规则

1. `Script` 节点的动态 input 和 output 都必须选择一个精确 `TypeRef`；output 允许 record/variant，但不允许 `*`。
2. 父进程在发送前验证 input；runner 解码后再次验证。runner 产出先在 runner 内验证，父进程接收后再次验证，任何 mismatch 都是 `script.contract_violation`。
3. JS 侧只看到由 canonical bytes 解码得到的 plain value snapshot；不得把 Go map/slice/struct、`goja.Value`、service pointer 或 native handle 直接 `vm.Set` 进去。
4. JS 输出拒绝 `undefined`、function、symbol、cyclic graph、Proxy/resource object、非有限 number，以及 contract 未声明的字段；`BigInt`、bytes、datetime、file/image 等必须按对应 Yotta datatype representation 编码。
5. blob/stream/handle 只以 opaque resource token 跨边界，token 绑定 attempt、grant、type digest、operation 和 expiry；guest 不得看到实际 path/URL/OS handle。
6. host functions 不再按 registry 自动注入为同名 globals。节点 contract 必须声明 `ScriptImports`，编译/admission 产生 exact operation digests；JS bindings 生成在单一 `yotta` namespace 下，文档、autocomplete、UI 参数提示也从同一 contract 投影。

JSON Schema 2020-12 明确把 schema 定义为描述 instance 的 JSON 文档，并用 assertions 约束 instance；Yotta 可继续用当前 schema bundle 做边界验证。[JSON Schema 2020-12 Core](https://json-schema.org/draft/2020-12/json-schema-core) WIT 的 interface/world、record/variant/result/resource 则提供了跨语言 binding generation 的成熟语义参照。[WIT interfaces, worlds and types](https://github.com/WebAssembly/component-model/blob/main/design/mvp/WIT.md)

## Resource budget：每一种资源都要有独立上限

仅设置 timeout 会留下 memory、log、host-call 和 result-size DoS。建议 `BudgetSnapshot` 至少冻结以下字段，并把实际用量写入 terminal journal：

| 资源 | goja runner enforcement | Wasm runner enforcement | 父进程 enforcement |
|---|---|---|---|
| source/module bytes | IPC frame cap、parser 前检查 | binary byte cap、compile 前检查 | contract/admission cap |
| wall time | context deadline + `Interrupt` | `WithCloseOnContextDone(true)` | hard timer 后 `TerminateJobObject` |
| CPU | cooperative checks不是硬限 | context close不是确定 instruction fuel | Job CPU rate/per-job user time |
| total process memory | goja 无 heap quota | guest page limit不覆盖 runtime heap | Job process/job committed memory |
| JS call depth | `SetMaxCallStackSize` | Wasm validation/runtime stack policy | runner process最终兜底 |
| Wasm linear memory | 不适用 | `WithMemoryLimitPages` | Job aggregate兜底 |
| input/result bytes | typed codec budgets | typed codec budgets | IPC/frame/artifact budgets |
| structure depth/nodes | decode + schema budgets | lift/lower + schema budgets | authoritative revalidation |
| host calls | local counter | import-call counter | broker authoritative counter |
| call concurrency | JS built-in先固定 1 | contract 显式声明 | broker grant/policy |
| per-call duration | child context | child context | provider/controller deadline |
| log count/bytes | capped ring + dropped count | capped stdout/log import | journal redaction + cap |
| subgraph recursion | host-call counter | host-call counter | scheduler depth/attempt budget |
| open resources | 不给 native handle | opaque handle table cap | broker lease count/expiry |

不要把 wall-clock timeout描述成可重放的“指令数”预算：相同脚本在不同机器/负载下消耗时间不同。journal 要保存配置预算与实际 CPU/wall/accounting；需要跨机器确定性 compute quota 时，应新增明确的 engine instruction/fuel contract，不能从 elapsed time 推导。

对 wazero，默认 4 GiB memory ceiling 对桌面脚本明显过宽，必须显式设置 page limit；`WithCloseOnContextDone` 默认关闭也必须显式开启。[wazero RuntimeConfig](https://pkg.go.dev/github.com/tetratelabs/wazero#RuntimeConfig) 对 goja，必须同时设置 call-stack depth 和 Job memory cap；只设置其一都不完整。

## Capability 模型：脚本不再等于“所有可绑节点”

脚本的 capability manifest 应是节点 config 的 semantic content，而不是运行时扫描结果：

```text
ScriptImports:
  - operation: yotta.automation.click@<semantic-digest>
    targetSlot: current-window
    bounds: {maxCalls: 20}
  - operation: yotta.asset.read@<semantic-digest>
    resource: asset:<content-digest>
    bounds: {maxBytes: 1048576}
  - operation: yotta.log.info@<semantic-digest>
    bounds: {maxCalls: 100, maxBytes: 32768}
```

编译时确认 operation、input/output TypeRef、provider ABI 和 target slot；admission 时把安装状态、用户 consent、resource scope、运行目标与 budget 绑定成 immutable grants。broker 每次调用重新检查 `attempt + call sequence + exact grant + operation + resource`，不信任 runner 传来的显示名。

纯运算函数应以 guest library/builtin 实现，不经过 authority broker；会观察时间、随机、文件、网络、设备、窗口、变量状态、subgraph 或任何外部系统的调用都是 effectful host call。尤其 `Subgraph` 不是普通函数：它可递归执行整个工作流，必须有独立 capability、最大深度和 call budget。

## Attempt journal 与 deterministic replay

Temporal 的上游架构把 durable workflow 建立在 append-only event history 上，并要求 workflow code deterministic/no side effects，effectful activity 则必须 idempotent 或 non-retryable。[Temporal architecture](https://github.com/temporalio/temporal/blob/main/docs/architecture/README.md) 它的 lifecycle 文档也展示了 Activity schedule/start/completion/failure 与 retry attempt 如何进入持久化 history。[Temporal workflow lifecycle](https://github.com/temporalio/temporal/blob/main/docs/architecture/workflow-lifecycle.md) Yotta 不必复制 Temporal，但应采用同一条不可绕过的原则：**replay 消费历史结果，不重新执行外部 effect。**

### 建议事件

```text
ScriptAttemptStarted
  attempt/run/instruction IDs
  source + engine + contract + input digests
  isolation profile + platform launcher version
  exact grants and budget snapshot digests
  virtual clock epoch + PRNG algorithm/seed digest

ScriptHostCallPlanned
  sequence/call ID
  operation/grant/typed-args digests
  idempotency key if the operation supports it

ScriptHostCallCompleted | ScriptHostCallFailed | ScriptHostCallAmbiguous
  typed outcome artifact digest
  redacted summary
  provider/controller receipt and stable external ID
  retry disposition

ScriptAttemptCompleted | Failed | Cancelled | Killed
  typed result/error digest
  termination cause + strength
  wall/CPU/peak committed memory/host calls/log bytes/output bytes
  runner exit code and job accounting snapshot
```

需要 replay 的完整 typed input/result 作为受访问控制、content-addressed artifact 保存；普通 run journal 只存 digest、size、type、redacted summary 和 artifact reference。secret、credential、raw OS handle、AppContainer token 和未脱敏日志永远不进入 replay artifact。

### Effect commit protocol

```text
1. validate HostCallRequest
2. append + fsync ScriptHostCallPlanned
3. invoke provider/controller with stable call ID/idempotency key
4. append + fsync Completed/Failed receipt
5. return HostCallResult to runner
```

若父进程在 2 与 4 之间崩溃，恢复时该 call 是 `ambiguous`。只有 provider 能按 stable idempotency key 查询/去重时才能自动 reconcile；否则停止 run，要求显式恢复决策。绝不能因为“runner 没收到 result”就假定 effect 未发生。

### 纯脚本 replay

goja 提供 `SetTimeSource` 与 `SetRandSource`，默认分别使用真实 `time.Now` 与 math/rand；Yotta runner 必须安装 versioned virtual clock 与 deterministic PRNG，而不是把宿主时间/随机直接暴露给脚本。[goja `SetTimeSource`](https://pkg.go.dev/github.com/dop251/goja#Runtime.SetTimeSource) [goja `SetRandSource`](https://pkg.go.dev/github.com/dop251/goja#Runtime.SetRandSource)

replay runner 只允许：

- 从 attempt start event 恢复 source、inputs、engine、budget、clock 和 seed；
- 对每个 host call 校验 sequence、operation 与 args digest；
- 从 history 注入已记录的 outcome，不接触真实 broker/provider；
- 终态结果 digest 必须与历史一致，否则报告 deterministic mismatch，包含首个偏离 call/sequence。

“相同 wall time”“相同日志时间戳”不是 deterministic contract；虚拟时间只在脚本显式读取 clock 或完成记录的 sleep/timer event 时推进。

## 取消、超时与退出分类

建议 terminal error taxonomy：

```text
script.user_cancelled
script.deadline_exceeded
script.budget.source_exceeded
script.budget.memory_exceeded
script.budget.cpu_exceeded
script.budget.host_calls_exceeded
script.budget.output_exceeded
script.contract_violation
script.capability_denied
script.guest_thrown
script.engine_fault
script.runner_crashed
script.runner_protocol_violation
script.isolation_unavailable
script.effect_ambiguous
```

父进程收到 cancel/deadline 后立即拒绝新的 host calls，并给所有进行中的 broker calls 传播 child context；随后发 cancel frame。runner 内 goja 调 `Interrupt(cause)`，Wasm call 的 context 取消。grace 到期仍未退出就 `TerminateJobObject`。job kill 不能执行 guest finally/defer，也不保证发送 terminal frame，所以 authoritative `Killed` event 必须由父进程在 wait/job accounting 后生成。

如果 cancel 与正常 completion 竞态，按父进程 journal 中先成功提交的 terminal transition 决定；后到的 runner frame 或 cancel request 只作为 diagnostic，不改写终态。

## 破坏性 cutover 建议

1. 冻结 `yotta.script-execution/3.1`、typed value envelope、host-call protocol、error taxonomy 与 budget schema，并从 contract 生成 Go/TypeScript/JS declarations 和编辑器提示。
2. 新增父进程 `ScriptExecutor` deep module：唯一入口接收 sealed execution request，内部封装 launcher、IPC、broker、journal、deadline、reaping 与 accounting；scheduler 不接触 goja/Windows handle。
3. 实现 Windows launcher：LPAC profile/SID、zero capabilities、explicit handle list、`PROC_THREAD_ATTRIBUTE_JOB_LIST`、Job limits、completion port/wait/accounting。任一 hardening 步骤失败即 fail closed。
4. 新增一次性 `cmd/yotta-script-runner`：只支持 exact ABI；goja 设置 stack/time/random/cancel，输入先 canonical decode + validate，输出先 validate + canonical encode。
5. 删除主进程 `goja.New/RunProgram`、全 registry `ScriptBindable` 注入、wildcard Result、raw `any`/Go object bridge 和无界源码 cache；不保留 legacy execution path 或 runtime feature fallback。
6. 把现有脚本调用节点的方式改成显式 `ScriptImports` 与 broker calls；为每个 effect operation 定义 idempotency/retry/receipt semantics。
7. 加 replay runner 和 conformance fixtures；同一 history 在不同进程重复 replay，必须得到相同 result/call sequence/digests。
8. Wasm/plugin 在同一 executor/ABI 上新增独立 engine kind；不做源内容自动探测，不把 JS 与 Wasm 的取消/资源能力伪装成完全相同，只统一 terminal contract。

跨平台规则也应 fail closed：Windows production 使用 LPAC+Job；Linux/macOS 在各自 launcher 达到项目承诺的隔离 profile 前，GUI 可以编译、核心 contract/replay 可以测试，但 `Script` 执行应明确返回 `script.isolation_unavailable`，不得悄悄切回 in-process。

## 必须通过的安全与稳定性测试

- 纯 JS `while(true){}`：先 engine interrupt，grace 内退出；journal 记录实际 termination strength。
- native Go binding 故意忽略 context 永久阻塞：`Interrupt` 无效，Job 强制结束且主进程存活。
- `SetMaxCallStackSize` 下无限递归；错误稳定分类，不拖垮主进程。
- JS 连续分配 array/string 到 Job memory cap；runner 失败/被终止，主进程 committed memory 回落。
- 不同源码洪泛：主进程不存在无界 goja Program cache。
- output、log、IPC frame、object depth、host-call count 洪泛：各自在自己的 budget 处 fail closed。
- runner 尝试直接读项目目录、用户目录、registry、network、window、credential：LPAC zero-capability 下全部拒绝。
- runner 尝试生成 child process/break away：active-process/no-breakaway/job tree 策略拒绝或收拢，关闭 job handle 后没有残留进程。
- 主进程在 runner 运行中崩溃：`KILL_ON_JOB_CLOSE` 清理 runner tree。
- cancel-before-start、cancel-during-host-call、cancel-after-terminal、runner-frame-after-kill 四种竞态只产生一个 authoritative terminal event。
- host effect 在 planned 后、completed 前模拟崩溃：恢复为 ambiguous，不重复非幂等调用。
- replay 时真实 file/network/window broker 全部不可达；host-call args/sequence 任一变化立即 deterministic mismatch。
- typed ABI 拒绝 unknown field、错误 semantic digest、非法 representation、非 canonical JCS、循环对象、`undefined`、NaN/Infinity、超大 blob/result。
- 同一 source/history 在 Windows runner 与平台中立 replay test harness 中产生相同 typed result digest。

## 最终判断

`goja.Interrupt + context` 是有用的执行控制，但不是安全隔离；`Wasm memory safety` 是强 guest memory boundary，但不是完整进程资源/权限边界；`Job Object` 是 Windows 的进程树和资源生命周期控制，但不是文件/网络权限沙箱；`AppContainer/LPAC` 是权限边界，但不提供 Yotta 所需的 typed workflow semantics、journal 或 deterministic replay。

Yotta 3.1 应把它们组合成一个深模块：**一次性 runner process 承载 engine，平台 launcher 提供 OS hardening，主进程 capability broker 承载所有 effect，content-addressed typed ABI 承载所有数据，append-only attempt journal 承载恢复与 replay。** 这样内置脚本仍然好写，但它不再拥有主进程、全 registry、宿主对象和未经声明的环境权限。
