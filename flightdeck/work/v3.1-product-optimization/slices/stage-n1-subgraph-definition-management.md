# N1 — 子图定义管理与对象语义

## Goal

让用户始终知道自己操作的是子图定义还是一次调用，并能从集中管理器查看定义、调用数、引用位置与接口摘要。

## Status

Complete

## Scope

- 复用 `WorkflowSource.graphs` 与各图的 `calls` 派生管理器投影，不创建平行 store。
- 主图置顶；子图可搜索，显示名称、短 ID、调用数、接口摘要与当前状态。
- 调用位置可跳转并选中；画布删除明确为删除调用，定义删除从管理器进入。
- 零引用定义允许确认删除；有引用定义默认阻止并列出位置。
- 调用节点与内部 boundary 共享端口行布局规则，长名称不覆盖 handle。

## Steps

1. 为管理器投影建立纯函数测试：名称消歧、调用计数、引用位置、搜索和接口摘要。
2. 实现子图管理 popover，替换当前同名图下拉菜单并接入打开、重命名、定位调用和删除定义。
3. 收敛删除文案和确认内容；保持 `remove-graph-call` 与 `remove-graph` 的后端引用保护。
4. 运行组件/会话定向测试与 `task check`，再进行一次 Windows WebView 旅程。

Windows WebView 的跨阶段完整旅程统一在 N4 执行，避免每个子切片重复构建宿主。

## Verification

- 两个同名子图仍可通过短 ID 区分。
- 一个定义被两个父图调用时显示调用数 2，并能分别定位。
- 删除一个调用后定义仍存在、调用数变为 1。
- 有引用时不能从管理器删除定义，并可查看引用；零引用删除后保存重开不再出现。
- 子图调用、入口、出口与数据边界的长标签均不与 handle 重叠。

## Result

- `WorkflowGraphManager` 直接从 `WorkflowSource.graphs/calls` 派生定义、调用数、引用位置和接口健康摘要，
  支持搜索、同名短 ID 消歧、打开、重命名、定位与零引用定义删除。
- 画布和 Inspector 明确删除的是 GraphCall；管理器明确删除的是 Graph 定义。有引用定义在 UI 和
  `EditorSession`/authoring engine 两层均拒绝删除。
- 修正不存在 graph ID 会触发 `splice(-1, 1)` 误删最后一个定义的问题。
- 调用节点与内部 boundary 的 handle gutter/长标签布局由渲染测试锁定。
- `task check` 通过：74 个测试文件、304 项测试。

## References

- [Subgraph management research](../references/subgraph-management-research.md)
- [Architecture](../../../../docs/architecture/runtime.md)
