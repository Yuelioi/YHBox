# Yotta architecture

Yotta 把桌面壳、应用生命周期、节点引擎、automation contract 和平台 adapter 分开。核心原则是：平台中立层只依赖小而稳定的能力接口；装配、并发与持久化不变量集中在少数深模块中。

- [Runtime and lifecycle](runtime.md)
- [Node engine](node-engine.md)
- [Automation targets and controllers](automation-targets.md)
- [Storage consistency](storage.md)
- [Threat model](threat-model.md)

源码导航：`internal/appruntime` 管应用生命周期；3.1 内核由 `internal/datatype`、`internal/nodecontract`、`internal/nodecatalog`、`internal/capability`、`internal/workflow/*` 和 `internal/run` 组成；`internal/blob`、`internal/resource`、`internal/stream` 管 durable/ephemeral value carrier。旧 `internal/node` 与 `internal/services/container/runtime` 只服务待迁移生产路径，迁移完成后删除，不是 3.1 fallback。`internal/automation` 管 target/controller，`internal/services/*` 管应用服务，`pkg/*` 放可复用 adapter/helper。
