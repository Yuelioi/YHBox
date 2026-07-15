# Yotta architecture

Yotta 把桌面壳、应用生命周期、节点引擎、automation contract 和平台 adapter 分开。核心原则是：平台中立层只依赖小而稳定的能力接口；装配、并发与持久化不变量集中在少数深模块中。

- [Runtime and lifecycle](runtime.md)
- [Node engine](node-engine.md)
- [Automation targets and controllers](automation-targets.md)
- [Storage consistency](storage.md)
- [Threat model](threat-model.md)
- [Installed network capabilities](network-capabilities.md)

源码导航：`internal/appruntime` 管应用生命周期；3.1 内核由 `internal/datatype`、`internal/nodecontract`、`internal/nodecatalog`、`internal/capability`、`internal/workflow/*` 和 `internal/run` 组成；`internal/blob`、`internal/resource`、`internal/stream` 管 durable/ephemeral value carrier，`internal/nodes31runtime` 安装与 Catalog implementation lock 精确匹配的内建 adapter，`internal/application` 是所有 Program Run 的唯一 command/worker seam。旧 `internal/node` 与 `internal/services/container/*` 是待删除的 authoring/实现库存，不再被生产 composition 作为执行 fallback。`internal/automation` 管 target/controller，`internal/services/*` 管应用服务，`pkg/*` 放可复用 adapter/helper。
