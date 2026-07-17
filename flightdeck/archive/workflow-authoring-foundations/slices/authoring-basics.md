# 工作流创作基础

## Outcome

新建工作流立即可运行，节点可搜索、可键盘删除，空画布可自助恢复；工作流级状态与节点级配置不再混放。

## Completion criterion

- CreateSource 权威创建 RunStarted。
- 节点目录支持本地化搜索与分类。
- Delete/Backspace 删除选中节点且不抢输入框。
- Run 状态拥有独立面板。
- exact target 字段从已安装设置中选择。
- 空画布提供明确起点动作。

## Blocked by

无。

## Verification

`task check` 全绿；真实 Windows Wails WebView smoke 验证目录搜索、RunStarted、节点增删、状态面板与 AI 面板，无前端运行时错误；截图人工检查通过。

## Out of scope

命令面板、自动布局、完整快捷键自定义和旧 Container 高级编辑能力复刻。

## Result

完成。新工作流可直接编译与 PreviewRun；相关 Application、bootstrap、workflow service 时间线断言已纳入 RunStarted 执行记录。
