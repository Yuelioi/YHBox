# Yotta architecture

Yotta 把桌面壳、应用生命周期、节点引擎、Target contract 和平台 adapter 分开。核心原则是：
Workflow 是唯一内容对象，Target 保存本机差异，GUI、CLI、MCP 与 Schedule 共用同一条运行路径。

- [Runtime and lifecycle](runtime.md)
- [Node engine](node-engine.md)
- [Automation targets and controllers](automation-targets.md)
- [Network targets](network-targets.md)
- [Application targets](application-targets.md)
- [Storage consistency](storage.md)
- [Threat model](threat-model.md)

源码导航：`main.go` 只保留进程入口与嵌入资源，`internal/desktopapp` 组合 Wails presentation，
`internal/localruntime` 打开 desktop 与 CLI 共用的本地运行环境，`internal/appruntime` 管后台资源生命周期；
`internal/workflow/*` 与 `internal/workflowstore` 保存和编译
Workflow，`internal/application` 是 GUI、headless CLI、AI、MCP 与 Schedule 共用的
`StartRun(workflowID)` command/worker seam。`internal/datatype`、`internal/nodecontract`、
`internal/nodecatalog` 和 `internal/nodeauthoring` 拥有节点与编辑契约，`internal/nodeadapter` 定义节点
host ABI，`internal/noderuntime` 安装具体 adapter；
`internal/automation` 管 Target/controller，`internal/blob`、`internal/resource`、`internal/stream`
管理数据载体，`internal/services/*` 提供用户动作，`pkg/*` 放可复用 adapter/helper。
