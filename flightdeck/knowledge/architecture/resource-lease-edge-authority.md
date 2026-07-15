---
kind: note
summary: "Yotta 3.1 leased data edge 的 authority、carrier 与 operation narrowing 规则"
activation: action
read_when: "新增或修改 runtime stream/handle data port、ResourceLeaseBinding、Compiler/Program edge validation 或 Executor borrow 时"
recheck_when: "Node Contract、Catalog、Compiler、Program opener、Executor 或 ValueEnvelope carrier 规则变化时"
---
# Resource lease edge authority

## 核心规则

`ResourceLeaseBinding` 是 data port 的 runtime authority 契约，不是隐藏的执行顺序。stream/handle bearer 只能沿显式 leased data edge 传递；durable blob/inline 值不得借此获得 runtime authority。

一条 data edge 必须满足：

- 两端 carrier class 一致：要么双方都是 runtime lease，要么双方都不是；禁止 runtime/durable 混接。
- leased port 的 pinned Data Type 必须具有 stream 或 handle representation。
- 下游 operations 必须是上游 operations 的子集；借用只能缩窄，不能增加能力。
- 上下游各自引用本节点的 exact capability requirement。Executor 用上游 session 出借、下游 session 接收，不把裸 token 当普通数据扩散。

## 多层复验

同一规则必须在 Catalog、Compiler、trusted Program opener 与 Executor 四个边界复验。Catalog/Program 都是可持久化输入，不能因为编译阶段验证过就跳过执行前重验。

Executor 通过 Run Owner/Broker 建立 narrow borrow，并把租约生命周期绑定到 Run。公共 `ExecutionResult` 只返回 durable ValueEnvelope；stream/handle runtime artifact 在 Owner 收口时失效，不得序列化进 Run Value 或文档示例。

需要在 durable 与 runtime carrier 之间转换时，使用显式 conversion node。不要增加隐式 coercion、通用 fallback 或看似存在但运行时未实现的 signal port。
