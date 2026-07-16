# Yotta architecture

Yotta 把桌面壳、应用生命周期、节点引擎、automation contract 和平台 adapter 分开。核心原则是：平台中立层只依赖小而稳定的能力接口；装配、并发与持久化不变量集中在少数深模块中。

- [Runtime and lifecycle](runtime.md)
- [Node engine](node-engine.md)
- [Automation targets and controllers](automation-targets.md)
- [Storage consistency](storage.md)
- [Threat model](threat-model.md)
- [Installed network capabilities](network-capabilities.md)
- [Installed application lifecycle](installed-application-lifecycle.md)

源码导航：`main.go` 只保留进程入口与嵌入资源，`internal/desktopapp` 组合 Wails presentation，`internal/appruntime` 管应用生命周期；3.1 内核由 `internal/datatype`、`internal/nodecontract`、`internal/nodecatalog`、`internal/nodeauthoring`、`internal/capability`、`internal/workflow/*` 和 `internal/run` 组成；`internal/blob`、`internal/resource`、`internal/stream` 管 durable/ephemeral value carrier，`internal/nodes` 显式装配内建契约，`internal/noderuntime` 安装与 Catalog implementation lock 精确匹配的 adapter，`internal/application` 是 GUI、headless CLI、AI、MCP 与 schedule 共用的 Program Run command/worker seam。旧 Node registry/实现树与 Container 产品栈均已删除。`internal/automation` 管 target/controller，`internal/services/*` 管应用服务，`pkg/*` 放可复用 adapter/helper。
