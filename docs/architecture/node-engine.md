# Node engine

Yotta 3.1 把节点系统拆成一条单向、可验证的契约链：

```text
Data Type 3.1 → Node Contract 3.1 → Catalog 3.1
                                      ↓
Workflow Source 3.1 → Compiler → Program 3.1 → authorized host adapter
```

`internal/datatype` 定义版本化 `TypeRef`、`ResolvedType` 与不可变 `ValueEnvelope`。跨 Program/host 边界的值必须携带完整 resolved type、representation 与 codec；四个封闭分支是 inline JSON、durable Blob Reference、runtime-only Stream Reference 与 Resource Reference。打开信封会严格解码、验证并 reseal，不存在 `any`/string fallback。

`internal/blob` 独占 immutable content-addressed bytes、quota、range read、integrity 与 Sweep。`internal/resource` 独占 Run/invocation-scoped opaque lease、authorization、operation narrowing、expiry 与 cleanup；`internal/stream` 只作为 Broker Provider 提供 bounded backpressure、finish/EOF 和 cancel。原始路径、channel、pointer、HWND、fd 与 process object 不得进入 Value Envelope。

`internal/runid` 生成并验证 canonical UUIDv7 Run ID。`internal/run` 把 durable RunRecord/Store 与 ephemeral Owner 分开：Record 使用固定状态机和 generation/digest CAS，Store 原子替换 canonical artifact，并在启动恢复时把遗留 RUNNING 转为 INTERRUPTED；Owner 独占 Run context、Grant Authorizer 与 Broker，关闭后 authority 永久失效。Run Value 的 graph/node/port/attempt provenance 位于 Value Envelope 外，只有 inline/blob 能进入 durable Record。

`internal/capability` 是 Capability Definition、attributed Requirement 与 sealed Plan 的唯一事实源，统一 operation/scope/target/credential slot 的规范化、schema 验证、排序和摘要。`internal/nodecontract` 是节点 machine contract 的唯一事实源。端口按 data、exec、error、status 分频道声明；空频道就是空，不允许 UI 或 runtime 猜测一个通用 `out`。展示字段不参与 semantic digest。`internal/nodecatalog` 把精确 NodeRef、Data Type definition、Capability Definition 与 implementation lock 封成不可变 machine snapshot。

`internal/admission` 是 Host Profile、Target Planner 与 Policy admission 的深模块。Host Profile 封存 OS、architecture、host API generation、provider artifact/ABI/capability inventory、Automation Target 与 non-secret credential binding metadata；Source、config、prompt 和插件结果都不能构造它。`Admitter.Admit` 对每个 target slot 求 provider/target 候选交集，要求零歧义或显式 selection，再调用统一 Policy；只有 approved decision 才能 seal 短期 Run Grant 并先持久创建 QUEUED RunRecord。ConsentOnce/ConsentEveryRun capability 没有 durable consent lineage 时不能 seal Grant。Run Store create 显式区分未发布、已发布但目录持久性未确认、已确认持久三种结果；中间态会返回原 Run identity 与 `persistence_unconfirmed`，调用方不得生成第二个 Run 或通知 Worker。provider OS/architecture/host API、capability ref、ABI、target kind 和 artifact lock 任一不匹配都在 Policy/provider effect 前以稳定 admission code 失败。

`internal/workflow/schema` 只接受 `yotta.workflow` / `3.1`。节点固定精确 NodeRef，边固定显式 channel 和 `{nodeId, portId}` endpoint。Source 不接受自由 `requestedCapabilities`；权限完全由 Effective Node Contract 推导。解析边界在递归处理前执行 byte/depth/node budget，并验证嵌套 TypeExpression。

`internal/workflow/compiler` 当前 lower 单个 main graph、data/exec/error edge、inline value/default/blob binding，以及可执行的 `pure-data`/`effect`/`control`/`event` 节点。Data edge 只固化为目标节点的 typed input plan 与独立 `dataOrder`；Exec/Error edge 只固化为保留 Source 顺序的 `signalRoutes`，控制环不会被误判成 data cycle。variables、secret refs、graph boundaries、disabled nodes 及尚无 Program 指令的 region/marker/visual 会以稳定诊断 fail closed。Program 还保存 literal/default/blob provenance、effective ports、execution contract、完整 implementation lock 和按 graph/node/requirement attribution 的 sealed Capability Plan；严格 opener 使用可信 Catalog 与 compiler build 重验 hash、身份、data 拓扑、signal route、端口、类型、plan 和资源预算。

预览 interpreter 必须接收可信 Catalog 与已安装 adapter 的完整 implementation lock，并且只允许 pure-data 与空 Capability Plan；effect Program 不能进入 preview。admitted execution 只接受 `Admitter` 已持久化的 Record/Grant，由 `Executor` 校验 Program/Catalog/Run Grant/implementation lock 和 running RunRecord admission，为每个 requirement 创建窄 `Run Session`，并只在端口声明 `Resource Lease Binding` 时跨 data edge borrow runtime authority；source/target carrier class 必须一致，target operation 必须是 source operation 的子集。RunRecord 内嵌完整 non-secret Grant artifact，重启后的 Worker 必须用 strict-open Program Plan 与 Catalog 再次 `OpenRunGrant`，不能依赖内存 grant。Run Owner 还会把 Grant 中的 provider artifact digest/ABI 与实际安装 provider 逐项比较，同名替换不能获得 authority。adapter 只能看到 typed inputs、克隆配置、该 invocation 的窄 session、由 Run Owner 管理的 task spawn，以及绑定 graph/node/attempt 的 AdapterAction recorder；不能取得 Broker、Run Store、provider inventory 或 ambient service。每个声明 effect 必须由 adapter 在返回前恰好记录一次真实 action，Executor 不从声明合成日志。每个输出都按 pinned Data Type 和 runtime lease binding 复验，整次执行 retained envelope 受 16 MiB 上限约束；runtime-only 输出在 lease 回收前只供执行器内部连线，公开 `ExecutionResult` 只含 durable envelope。NodeAttempt/AdapterAction 通过同一 RunRecord generation/digest CAS 追加，Executor 返回前将 Record 原子推进到 SUCCEEDED、FAILED 或 CANCELLED。

Grant Authorizer 是 Capability Run Grant 到 Resource Broker 的唯一 adapter。Broker 的 authorizer 返回 sealed requirement 的 canonical scope 与 credential binding metadata，再由 Broker 注入 ProviderOpenRequest；workflow 的 config、prompt 或插件输入不能声明自己的授权范围。open、borrow 与每次 call 都校验 Program/plan/grant/policy/principal/plugin/session/graph/node/requirement 归属、operation、provider、target、kind、expiry 与 revoke 状态；通用 borrow 还要求 canonical capability scope 完全相同，不能把宽 scope 对象借给窄 scope requirement。

`internal/nodeauthoring` 从精确 Data Type、Node Contract、Capability Definition 与 Catalog Snapshot 派生完整 Authoring Projection。它固定参数控件、默认值提示、约束、端口 binding/carrier/type lifecycle、信号频道、availability、target/credential/consent/risk 和 Editor Adapter；投影不拥有语义，严格 reopen 会从可信输入重新生成并要求 canonical bytes 完全一致。JSON Schema `default` 只作为提示，UI 不自动写入；复杂类型只进入显式 JSON 控件或具名内置 Editor Adapter。前端与 Markdown 文档共同消费该投影，不再各自解释 raw contract/schema。

`internal/nodes31` 显式装配内建 Catalog。Concat 保持 `a`、`b` 两个 string data input 与 `result` 一个 string data output，exec/error 均为空。Blob→Stream 与 Stream→Blob 是画布可见的 effect conversion：Blob Reference 可持久化，Stream Reference 只在 Run 内通过带 operation 子集的 Resource Lease Binding 传递；两者必须经 Blob/Stream Provider 和 admitted capability 执行，不允许 adapter 直读 Store 或 channel。Status 是 Node Contract 声明、Run journal 承载的观察事实，不是 Workflow Source 连线或画布 handle；Authoring Projection 将 `statusEvents` 与可连线 `signals` 分离。当前 Compiler 尚未 lower error channel，因此 conversion contract 也不发布不可执行的虚假 error port；稳定失败写入 NodeAttempt/AdapterAction 和 terminal RunError，待 channel lowering 后再开放可连线错误分支。生成器从同一 sealed contract 产出 machine Catalog、Authoring Projection 与 Markdown 文档；内建 adapter 的锁从本机构建所信任的 manifest 独立计算，不采信 Catalog 自报 digest。MCP search/describe 返回 projection digest 和 generator version。Vue 对 3.1 精确 Node Type ID 直接读取生成 projection，未知端口返回 unknown，不再猜成 exec。

旧 `internal/node` 与 `internal/services/container/runtime` 仍服务尚未迁移的生产编辑器/执行路径；3.1 interpreter 在 Program/Run 与 capability/resource 决议完成前不能作为它的 fallback。迁移完成后删除旧路径，不能长期保留双 runtime。
