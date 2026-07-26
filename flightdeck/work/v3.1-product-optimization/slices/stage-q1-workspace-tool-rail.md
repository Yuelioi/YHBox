# Q1 — 编辑器工作区工具栏重组

## Journey

用户在主图与子图之间频繁切换时，子图管理原先藏在顶部弹出层；节点目录又长期占据左栏，和已经成熟的
Tab 快速添加重复。Macro、精准录制和视觉模板被合并在同一个“工作区资源”入口中，也增加了一次二级选择。

期望编辑器左侧承担定义与资源导航：

- 子图管理是默认停靠工具，持续展示定义、调用数、引用定位和生命周期动作。
- 节点目录不再常驻；Tab 与画布“添加节点”按钮共同提供 Catalog 快速添加。
- Macro、精准录制、视觉模板分别拥有一级图标；Snippet 保持独立入口。

## Implementation

- `WorkflowGraphManager` 从 Popover 改为全高停靠面板，定义创建、打开、复制、重命名、引用定位、
  普通删除和级联删除语义不变。
- 左侧 rail 固定为子图、Macro、精准录制、视觉模板和 Snippet 五个工具；面板可折叠并保留 resize。
- `WorkflowResourceDock` 改为接收单一资源 kind，不再在内部重复显示三栏切换。
- 画布工具条新增显式“添加节点”，与 Tab 共用 `WorkflowQuickAddMenu`；快速添加项暴露稳定 node identity，
  供 WebView smoke 精确选择。
- WebView 纵向旅程删除旧目录点击/拖放假设，改为验证显式快速添加、五个工具、停靠式子图管理和三类资源入口。

## Regression findings

- 第一次真实 WebView 失败不是 WebSocket 并发，而是 Analyze Color 烟测仍点击已删除的节点目录；节点数一直
  不变，15 秒后底层才显示写锁/上下文超时。新增 Go 回归断言，要求该节点必须经过显式快速添加入口。
- 子图管理搬迁时新建按钮丢失稳定 `workflow-graph-new` 标识；真实旅程捕获后先新增红灯组件测试，再恢复标识。

## Acceptance

- `task check`：退出码 0；受影响 Go 包通过，前端 79 个测试文件、340 项测试通过。
- `task webview:smoke`：退出码 0；真实 DEV host 创建工作流、快速添加节点、创建/进入子图、推导接口、
  保存重开并轮流打开三类资源工具。
- 已目检 `20260725-065209` 的 workflow、subgraph、resource-tools、workflows、assets 和 schedules PNG：
  左侧停靠面板、画布和检查器无重叠，子图 boundary 与接口面板可见。

## Non-goals

- 不删除 Catalog/Authoring Projection；它继续服务 Tab、画布添加、连线候选和其他发现入口。
- 不改变 Graph/GraphCall、资源归属、录制或运行语义。
- 不在本阶段实现 Stage M2 的 Workflow Resource/Global Asset 快照与提升合同。
