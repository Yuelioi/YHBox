---
title: 定义 Blob Store、Stream 与 Resource Broker
label: wayfinder:grilling
parent: ../map.md
status: open
assignee:
blocked_by:
  - define-data-types-and-value-envelope.md
  - define-node-contract-metaschema.md
---

# 定义 Blob Store、Stream 与 Resource Broker

## Question

内容寻址 Blob Store 的 identity、storage、quota、lifecycle、cleanup、range read 与完整性校验，Stream 的背压、取消、终态和 producer/consumer cleanup，以及 Resource Broker token 的 authority、ownership、borrow/drop、expiry 与 crash cleanup 应如何统一定义，才能让 runtime、Wasm Node 和 Process Node 使用相同语义，同时不把路径、指针、平台句柄或进程内对象泄漏到 Value Envelope？

还需明确 blob、stream 与 handle 之间哪些转换必须成为画布可见、带 effect/capability 声明的节点，并规定其失败、配额和重放行为。

## Resolution

已接受，详见 [ADR-0003](../../../adr/0003-separate-durable-blobs-from-run-resources.md)。Yotta 采用 durable Blob Reference 与 ephemeral Resource Reference 两层模型；Stream 不是第三套句柄系统，而是 Resource Broker Provider。

### Blob Store

- Blob Reference 固定为 canonical parameter-free media type、`sha256:<64 lowercase hex>` 原始字节摘要和 exact raw byte size；路径、URL、文件名与 provider 不属于身份。
- 对象不可变；写入受单对象/总量 quota 约束，使用唯一 staging 文件、sync、原子 rename。读取和同摘要复用前同时验证 size 与 digest，等长篡改也必须失败。
- Blob Store 独占对象目录、range read、staging cleanup 与 Sweep。资产层不得扫描或删除内部文件；`CommitRecordBlob` / `CommitVariantBlob` 把对象写入和 durable reference commit 放在排斥 GC 的同一生命周期临界区。
- asset record schema 破坏性升到 v2，旧 string SHA、旧 BlobStore、skip-corrupt preload 与兼容读取全部删除。

### Resource Broker

- Provider inventory 在 Broker composition 时冻结；每次 open 必须先经过 Authorizer，provider side effect 不能早于授权。
- handle 是 256-bit random opaque token，lease 精确绑定 Run、invocation、resource kind、operation set 与 UTC expiry。调用方不能从 token 获得 provider ID、路径、指针、HWND、fd 或进程对象。
- borrow 只能在同 Run 内显式发生，并缩小 operation set 与 expiry；不能跨 Run、扩大 authority 或重新解释 kind。最后一个 lease drop 才 close 底层对象。
- Drop、idle expiry sweep 与 Run revoke 使用同一 cleanup；先取消 active provider context，再等待调用退出并 exactly-once close。provider object 永不离开 Broker。

### Stream

- Stream 只通过 Broker 的 `stream/send`、`stream/receive`、`stream/finish`、`stream/cancel` operation 使用；producer/consumer 通过 narrowed leases 分权。
- capacity 与 max chunk bytes 必须显式配置且受 host ceiling 限制。队列满时 send 阻塞并响应 context cancellation；不允许无界缓冲或静默丢包。
- finish 禁止后续 send，已排队 chunk 继续可读，排空后稳定返回 EOF。cancel 立即丢弃队列并唤醒所有阻塞方；最后 lease cleanup 也会 cancel session。

### Value、conversion 与 replay

- Value Envelope 的四个封闭分支均进入摘要 preimage：inline JSON、Blob Reference、Stream Reference、Resource Reference。打开时按 branch/codec 严格解码、验证并 reseal；不存在 `any` 或字符串猜测分支。
- inline/blob 可以持久化；stream/resource token 只允许 Run 内传输，禁止进入 Program literal、durable trace、日志、clipboard 或 cache。重放只能复用仍被保留的 Blob；stream/resource 必须在新 Run Grant 下重新 open。
- inline↔blob、blob→stream、stream→blob/inline 都是画布可见的 effect conversion，声明 storage/resource capability、quota、cancel 与 error contract。Resource Reference 没有通用持久化转换，只能由 capability-specific operation 产生新的 inline/blob 值。

### Conformance

- 覆盖 path forgery、same-size tamper、per-object/total quota、range、GC/reference race、cross-Run/cross-invocation、operation widening、borrow/drop、expiry、Run crash cleanup、exactly-once close、backpressure、cancel wakeup、finish drain/EOF 和 envelope branch forgery。
- builtin、Wasm 与 Process Provider 使用同一 Broker lease 语义；禁止为本地 builtin 或平台 adapter 建立旁路。

## Remaining implementation gate

Kernel、资产 schema v2、strict Value Envelope carriers 与 admitted blob→stream→blob tracer 已实现。该 tracer 使用显式 effect Node Contract、Capability Plan/Run Grant、端口 Resource Lease Binding、Run Session、Blob/Stream Provider 和独立可信的 built-in implementation manifest lock；adapter 不接触 Store、Broker 或原始 channel。Blob writer 在 Run Owner 存活期间 pin 已提交对象，阻止 Sweep 在 Run Value/reference commit 前制造悬空 BlobRef；Owner 关闭后 pin 与 lease 一起释放。本票保持 open，直到以下纵向结果全部进入唯一运行链：

- inline↔blob 与 stream→inline conversion Node Contract、capability/effect 声明和实现；
- `internal/run.Owner` 已成为每个 admitted Run 的 Broker composition owner，并在终止时按 grant revoke → context cancel → Run lease revoke → permanent Broker close 收口；后续 production interpreter 只能消费该 owner，不能另建 Broker；
- builtin、Wasm、Process 三类 provider 共用同一 conformance corpus；
- capture/preview 的整图 Data URL Wails transport 被 bounded blob/stream adapter 替换。

这些缺口不得用旧 BlobStore、string SHA、base64 thumbnail RPC、ambient service 或 runtime fallback 填补。
