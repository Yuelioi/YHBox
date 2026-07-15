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

`internal/workflow/compiler` 当前是 3.1 的 pure-data tracer：只 lower main graph、data edge、inline value/default binding 与 `pure-data` 节点。variables、secret refs、graph boundaries、disabled nodes 及 exec/error/status edge 会以稳定诊断 fail closed；在对应 Program/Run 决议实现前不得静默忽略。Program 保存 literal/default provenance、typed ValueEnvelope、effective ports、execution contract、完整 implementation lock 和按 graph/node/requirement attribution 的 sealed Capability Plan；严格 opener 使用可信 Catalog 与 compiler build 重验 hash、身份、拓扑、端口、类型、plan 和资源预算。

预览 interpreter 必须接收可信 Catalog 与已安装 adapter 的完整 implementation lock，并且只允许空 Capability Plan；effect Program 必须走后续唯一 Run admission，不能给 preview 塞字符串 grant。只按 entrypoint 字符串注册不足以授权执行。每个输出都按 pinned Data Type 复验，并受单值和整次运行 retained-value budget 限制。

Grant Authorizer 是 Capability Run Grant 到 Resource Broker 的唯一 adapter。Broker 的 authorizer 返回 sealed requirement 的 canonical scope 与 credential binding metadata，再由 Broker 注入 ProviderOpenRequest；workflow 的 config、prompt 或插件输入不能声明自己的授权范围。open、borrow 与每次 call 都校验 Program/plan/grant/policy/principal/plugin/session/graph/node/requirement 归属、operation、provider、target、kind、expiry 与 revoke 状态；通用 borrow 还要求 canonical capability scope 完全相同，不能把宽 scope 对象借给窄 scope requirement。

`internal/nodes31` 显式装配内建 Catalog。Concat 是首条 tracer：`a`、`b` 两个 string data input，`result` 一个 string data output，exec/error/status 均为空。生成器从同一 sealed contract 产出 machine Catalog、Vue presentation projection 与 Markdown 文档；MCP search/describe 返回 presentation digest 和 generator version。Vue 对 3.1 精确 Node Type ID 直接读取生成 projection，未知端口返回 unknown，不再猜成 exec。

旧 `internal/node` 与 `internal/services/container/runtime` 仍服务尚未迁移的生产编辑器/执行路径；3.1 interpreter 在 Program/Run 与 capability/resource 决议完成前不能作为它的 fallback。迁移完成后删除旧路径，不能长期保留双 runtime。
