# Yotta architecture

Yotta 把桌面壳、应用生命周期、节点引擎、automation contract 和平台 adapter 分开。核心原则是：平台中立层只依赖小而稳定的能力接口；装配、并发与持久化不变量集中在少数深模块中。

- [Runtime and lifecycle](runtime.md)
- [Node engine](node-engine.md)
- [Automation targets and controllers](automation-targets.md)
- [Storage consistency](storage.md)
- [Threat model](threat-model.md)

源码导航：`internal/appruntime` 管生命周期，`internal/node` 管节点契约，`internal/services/container/runtime` 管图执行，`internal/automation` 管 target/controller，`internal/services/*` 管应用服务，`pkg/*` 放可复用 adapter/helper。

