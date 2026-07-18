---
kind: note
summary: "3.1 节点失败由 Contract ErrorSpec + 独立 error channel + NodeFailure 精确路由；未声明/未接错误终止 Run，journal 保留真实 attempt/action。"
activation: action
read_when: "给节点/adapter 加失败处理、错误码、error port、Retry，或排查失败未路由/被误报成功时"
recheck_when: "ErrorSpec、NodeFailure、scheduler routeFailure、Retry instruction、AdapterAction 或 frontend error presentation 改变后"
---
# 3.1 节点错误模型

Node Contract 显式声明允许的 `ErrorSpec{code, category, retryHint}` 和 error output。Adapter 对可路由业务/外部失败返回 `compiler.NodeFailure{Code, Output, Cause}`；Code 与 Output 必须同时匹配 Contract。

Scheduler 只在以下条件全部满足时路由失败：

1. 错误是唯一结构化 NodeFailure，而不是混入内部错误；
2. code 是该节点声明的 ErrorSpec；
3. output 是已声明 error port；
4. Program 存在从该 error port 出发的 route。

路由成功时 NodeAttempt 终态是 `routed`，下游 trigger 携带脱敏 RoutedFailure；它不伪装成成功 attempt。没有匹配 route、未声明错误、action/status journal 不一致或 adapter contract violation 会终止 Run。

effect adapter 返回前必须恰好记录真实 AdapterAction；failure code 要与 action outcome 一致。Executor 不从声明合成“看起来执行过”的日志。Retry 只消费显式 error edge 送入本 Retry region 的失败，不全局捕获，也不重试 `RetryNever` effect。

配置/类型/端口错误应在 Source/Compiler 阶段产生稳定 diagnostic；target/capability/consent 错误在 Admission 阶段；真实宿主失败在 adapter。前端 RPC transport 只解码并 rethrow，页面根据结构化 code 决定 inline recovery 或 failure toast，不能吞错后制造 `invalid result` 二次错误。
