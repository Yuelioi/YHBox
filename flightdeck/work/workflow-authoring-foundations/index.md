# 工作流创作基础与旧能力连续性

## Goal

让 3.1 工作流从新建即可理解和操作，并恢复录制、Clip、模板等必要旧能力的产品入口。

## Status

Finished

## Current

目标已经完成：新建起点、目录搜索、键盘删除、工作流状态、exact target 与独立资源库均已落地；
录制、Clip 和视觉模板主入口已经恢复。后续复合模板节点与安全缩略图 adapter 已由其他 Work 承接。

## Next

None.

## Progress

- 新建 Workflow Source 由后端权威注入 RunStarted 根节点。
- 节点目录支持搜索与分类，Delete/Backspace 通过 EditorCommand 删除选择。
- Run 状态移出 Inspector，exact target 从已安装项中选择。
- 独立资源库恢复录制、Clip、模板搜索、元数据、删除和截图入口。
- 阶段末 `task check`、生产 bundle 和真实 Windows WebView smoke 通过。
- 旧 runtime 没有随用户体验一起恢复。

## References

- [Authoring basics](references/authoring-basics.md) — 新建、目录、选择和 target 创作证据。
- [Resource continuity](references/resource-continuity.md) — 录制、Clip 与模板入口证据。
- [Feature continuity](../../knowledge/architecture/feature-continuity-across-product-stack.md) — 跨层能力恢复规则。
- [Installed input authority](../../knowledge/architecture/installed-input-authority.md) — target 与授权边界。
