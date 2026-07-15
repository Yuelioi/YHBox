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

`internal/workflow/schema` 只接受 `yotta.workflow` / `3.1`。节点固定精确 NodeRef，边固定显式 channel 和 `{nodeId, portId}` endpoint。Source 不接受自由 `requestedCapabilities`；权限完全由 Effective Node Contract 推导。解析边界在递归处理前执行 byte/depth/node budget，并验证嵌套 TypeExpression。

`internal/workflow/compiler` 当前 lower 单个 main graph、data edge、inline value/default/blob binding，以及 `pure-data`/`effect` 节点。variables、secret refs、graph boundaries、disabled nodes 及 exec/error/status edge 会以稳定诊断 fail closed；在对应调度语义实现前不得静默忽略。Program 保存 literal/default/blob provenance、typed ValueEnvelope、effective ports、execution contract、完整 implementation lock 和按 graph/node/requirement attribution 的 sealed Capability Plan；严格 opener 使用可信 Catalog 与 compiler build 重验 hash、身份、拓扑、端口、类型、plan 和资源预算。

预览 interpreter 必须接收可信 Catalog 与已安装 adapter 的完整 implementation lock，并且只允许 pure-data 与空 Capability Plan；effect Program 不能进入 preview。admitted execution 由 `Executor` 校验 Program/Catalog/Run Grant/implementation lock，为每个 requirement 创建窄 `Run Session`，并只在端口声明 `Resource Lease Binding` 时跨 data edge borrow runtime authority；source/target carrier class 必须一致，target operation 必须是 source operation 的子集。adapter 只能看到 typed inputs、克隆配置、该 invocation 的窄 session 与由 Run Owner 管理的 task spawn；不能取得 Broker、provider inventory 或 ambient service。每个输出都按 pinned Data Type 和 runtime lease binding 复验，整次执行 retained envelope 受 16 MiB 上限约束；runtime-only 输出在 lease 回收前只供执行器内部连线，公开 `ExecutionResult` 只含 durable envelope。

Grant Authorizer 是 Capability Run Grant 到 Resource Broker 的唯一 adapter。Broker 的 authorizer 返回 sealed requirement 的 canonical scope 与 credential binding metadata，再由 Broker 注入 ProviderOpenRequest；workflow 的 config、prompt 或插件输入不能声明自己的授权范围。open、borrow 与每次 call 都校验 Program/plan/grant/policy/principal/plugin/session/graph/node/requirement 归属、operation、provider、target、kind、expiry 与 revoke 状态；通用 borrow 还要求 canonical capability scope 完全相同，不能把宽 scope 对象借给窄 scope requirement。

`internal/nodes31` 显式装配内建 Catalog。Concat 保持 `a`、`b` 两个 string data input 与 `result` 一个 string data output，exec/error/status 均为空。Blob→Stream 与 Stream→Blob 是画布可见的 effect conversion：Blob Reference 可持久化，Stream Reference 只在 Run 内通过带 operation 子集的 Resource Lease Binding 传递；两者必须经 Blob/Stream Provider 和 admitted capability 执行，不允许 adapter 直读 Store 或 channel。当前 Compiler 尚未 lower error/status channel，因此 conversion contract 也不发布不可连线的虚假 signal port；稳定错误仍由 ErrorSpec 定义，待 NodeAttempt/channel slice 一次性接入。生成器从同一 sealed contract 产出 machine Catalog、Vue presentation projection 与 Markdown 文档；内建 adapter 的锁从本机构建所信任的 manifest 独立计算，不采信 Catalog 自报 digest。MCP search/describe 返回 presentation digest 和 generator version。Vue 对 3.1 精确 Node Type ID 直接读取生成 projection，未知端口返回 unknown，不再猜成 exec。

旧 `internal/node` 与 `internal/services/container/runtime` 仍服务尚未迁移的生产编辑器/执行路径；3.1 interpreter 在 Program/Run 与 capability/resource 决议完成前不能作为它的 fallback。迁移完成后删除旧路径，不能长期保留双 runtime。
