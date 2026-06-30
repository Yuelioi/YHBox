# fishing-v2 新节点系统重建

SUMMARY: 按当前节点系统和容器 package 四件套重建自动钓鱼容器；旧 `container.json` 只作业务流程参考。

READ WHEN: 改 `bin/data/containers/fishing-v2/`、钓鱼状态机、买鱼饵/卖鱼闭环、钓鱼模板/clip 依赖、容器数据清理前。

## 状态

- 旧容器路径: `bin/data/containers/fishing-v2/container.json`。
- 旧容器是 v1 单文件布局，使用旧 `WindowTarget` 和 `graph.version`。
- 当前目标是重建为 `package.json` + `graph.json` + `installation.json` + `yotta-lock.json`。
- 其它 `bin/data/containers/*` 旧测试容器可删除；资产库和子图库先保留。
- 关键用户确认: 没钱买鱼饵时默认自动卖鱼；新增 `autoSellWhenNoCurrency=true`，关闭后没钱直接结束流程。

## 文件

- [design.md](design.md) — 重建方案。
