# V4-G 计划首页

## Goal

让 Schedule 保持为 Workflow 的可选触发方式，而不是另一个常驻运营控制台；默认页服务查找、启停、
试运行和编辑，完整统计与策略信息按需进入管理态。

## User outcome

- 打开计划页只看到新建、搜索和自己的计划。
- 计划行直接显示触发方式、目标 Workflow、上次状态，并可一键启停或运行。
- 只有需要整理时才打开管理态查看统计、状态筛选和失败策略。
- 运行失败时在原行看到统一缺项原因，并能直达对应 Workflow 修复。

## Preserve

- Cron、Hotkey、启动一次和仅手动四类触发。
- 多 Workflow 顺序、超时、失败停止/继续、启停和删除。
- 统计、状态筛选、失败策略、创建/编辑完整表单。
- Schedule v3 数据迁移、最后触发状态和统一 Run readiness。

## Current

默认 browse 态已经移除英文眉题、四张统计卡、状态筛选和失败策略列，只保留计划主路径。显式“管理”
切换恢复全部统计和策略信息，搜索状态和功能保持不变。完整 Windows WebView 旅程验证 browse/manage
往返、创建、绑定 Workflow、行内运行、queued 状态和重新打开。

新建/编辑默认只展开目标 Workflow 与触发方式，自动生成的名称、启用、超时和失败策略完整保留在“高级
设置”；名称校验失败会自动展开高级区。弹窗从 4xl 收窄到 3xl。状态栏不再暴露 `Program queued` 和
Run ID，改为本地化的“等待运行 / 正在运行”。最新完整验收目录为
`.task/workflow-editor-smoke/20260726-202512`。

旅程同时暴露并修复了悬浮启动器的短 Run 交接竞态：终态事件可能早于前端记录 `runId`，紧接着读取的
timeline 仍可能短暂为 running。启动器现在会短时轮询权威记录，随后继续依赖事件，不会永远卡在运行中。

## Next

1. 为首次启动、已有 Schedule 和 40+ Schedule 建立同一 browse/manage 验收。
2. 统一 Schedule 与其他入口的 Run 取消反馈。
3. 评估大量 Schedule 下的虚拟化或分页阈值。
