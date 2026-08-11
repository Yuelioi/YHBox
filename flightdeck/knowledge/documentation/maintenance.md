# Documentation maintenance

## Authority order

更新文档时按以下顺序取证：

1. 生产调用路径和持久/运行时类型；
2. schema、Taskfile、CI、测试和正式 generator；
3. 生成合同，用于核对投影而不是反向定义实现；
4. 现有 `docs/`、Knowledge、Finished Work 和 Git 历史，只作为待验证线索。

文档之间不能互相证明。注释也可能过期；遇到冲突要追到实际 composition、writer/reader 或 adapter 注册路径。

## Where a fact belongs

- `README.md`：产品定位、最短上手、平台/许可摘要与导航。
- `CONTEXT.md`：稳定领域语言，必须和当前 public model/DTO 一致。
- `docs/product/`：用户对象及其当前行为。
- `docs/architecture/`：模块 ownership、数据流、合同、存储和信任边界。
- `docs/reference/`：代码已有的稳定操作接口。
- `flightdeck/knowledge/`：另一项工作可直接执行的修改方法。
- `flightdeck/work/`：当前/历史 Work 的目标、决策、证据和 handoff；不作为系统权威。

不要把一次故障时间线、完成报告、旧方案、临时计数或“最近审计日期”写进稳定知识。节点数、RPC 数、测试
数、bundle bytes 和独立版本域等易变事实应指向 generator/Task，而不是手抄快照。

## Editing loop

1. 列出要陈述的 claim，并为每项找到代码/schema/Task/test 证据。
2. 删除无法证实或已经失去用户价值的内容；不要为了维持旧标题而留下空泛段落。
3. 使用项目术语，区分 Workflow Source/Program/Run、Configured Target/capability、Workflow Resource/Global
   Asset/Snippet。
4. 命令优先写正式 Task 入口；底层脚本只在没有 Task 或需要解释副作用时出现。
5. 链接到稳定入口，不复制生成清单。新页面同时加入 `docs/README.md` 或 Knowledge 路由。
6. 运行 `task check:docs`，再按实际代码改动运行 `task check`。视觉文档若带截图，必须检查当前 UI 后再更新；
   不保留已经不代表产品的旧品牌截图。

`task check:docs` 只机械验证本地 Markdown link、文档中引用的 Task 名和禁止的旧公开名称。它不能证明语义
正确；review 仍需逐条对照当前代码。
