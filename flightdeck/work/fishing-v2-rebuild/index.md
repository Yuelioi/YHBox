# fishing-v2 新节点系统重建

SUMMARY: 按当前节点系统和容器 package 四件套重建自动钓鱼容器；旧 `container.json` 只作业务流程参考。

READ WHEN: 改 `bin/data/containers/fishing-v2/`、钓鱼状态机、买鱼饵/卖鱼闭环、钓鱼模板/clip 依赖、容器数据清理前。

## 状态

- 已重建 `bin/data/containers/fishing-v2/` 为 `package.json` + `graph.json` + `installation.json` + `yotta-lock.json`。
- 旧容器已备份到 `bin/data/_backups/fishing-v2-container.json`。
- 旧测试容器目录已从 `bin/data/containers/` 清理，只保留 `fishing-v2`。
- 资产库和子图库保留；`state_BUYBAIT` 子图已补 `no_currency` 分支。
- 关键用户确认: 没钱买鱼饵时默认自动卖鱼；新增 `autoSellWhenNoCurrency=true`，关闭后没钱直接结束流程。
- `bin/` 被 gitignore 忽略，本地容器数据不进入 git commit；已用脚本和结构化 JSON 检查确认落盘。
- 下一步: 启动应用并做游戏内 smoke。

## 文件

- [design.md](design.md) — 重建方案。
- [plan.md](plan.md) — 已执行的实施计划。
