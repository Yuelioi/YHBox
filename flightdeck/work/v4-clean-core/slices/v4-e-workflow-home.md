# V4-E 工作流首页

## Goal

把应用默认入口从“工作流管理后台”改成“快速打开和运行工作流的首页”，完整保留现有管理能力。

## User outcome

- 启动后第一眼看到自己的 Workflow，而不是筛选器和表头配置。
- 点击整行打开 Workflow；运行按钮独立且不会弹确认。
- 新建只要求名称，模板可选。
- 需要整理大量 Workflow 时，进入明确的管理模式继续使用现有筛选、批量和文件操作。

## Preserve

- 搜索、分类、标签、创建/修改日期、排序、列选择和分页。
- 分类/标签批量编辑、批量导出和批量删除。
- 导入、导出、替换和 Source 恢复。
- 新建模板、编辑元数据、运行反馈和引用删除保护。

## Steps

1. 将默认浏览态和管理态拆成明确状态，默认态只渲染主操作、搜索和 Workflow 内容。
2. 让 Workflow 行本身成为打开入口，并保留独立运行按钮和 overflow 菜单。
3. 将高级筛选、列配置、多选和批量工具移入管理态，不复制查询状态。
4. 将导入、导出和恢复动作重组到管理入口；有损坏项时保留可见但安静的通知。
5. 提取页面内高内聚任务，缩小 `WorkflowsView` 协调面，行为通过组件和 WebView 旅程验证。

## Acceptance

- 默认截图只有一个主按钮、一个搜索框和 Workflow 内容；无常驻高级筛选或批量工具。
- 一次点击 Workflow 行进入编辑器，一次点击运行按钮启动。
- 切换管理态后，现有筛选、列、分页、批量、导入导出、替换和删除仍可用。
- 空 profile、损坏 Source profile 和 40+ Workflow profile 均可用。
- `fishing-v2` 仍可打开、编译和运行到 readiness 阶段。

## Current

默认浏览态与显式管理态已经落地。浏览态只显示新建、搜索和 Workflow 内容，主内容区域打开编辑器，
运行和 overflow 为独立同级操作；管理态继续提供原有筛选、列、多选和批量能力。Source 恢复通知默认
折叠。组件测试、类型检查、Go smoke 单测与完整 WebView 旅程已通过；`task check` 最终通过 82 个
测试文件 / 351 项测试。

真实 WebView 截图位于本地 `.task/workflow-editor-smoke/20260726-190741/workflows.png` 和
`.task/workflow-editor-smoke/20260726-190741/workflow-management.png`。V4 技术基线已用迁移后的
`fishing-v2` 通过编译；本轮复核时真实 profile 被仍存活的 `Yotta.exe` 写锁占用，因此未强制结束进程。
原始 V2 JSON 不能替代迁移后 Source 做当前编译验收。

## Next

1. 在 profile 写锁自然释放后复核迁移后的 `fishing-v2`，不操作用户进程。
2. 以相同能力保留规则设计 V4-2 Run Readiness 切片。
3. 审计全部运行入口当前的缺项判断和反馈，建立统一用户结果。
