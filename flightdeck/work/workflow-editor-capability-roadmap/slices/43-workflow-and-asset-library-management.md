---
slice: "43"
title: 工作流与资源库管理工作台
status: in_progress
---

# Slice 43：工作流与资源库管理工作台

## Outcome / Question

恢复工作流管理所需的耐久元数据，并把工作流首页和资源库统一升级为可管理数百到数千条记录的商业软件级列表工作台，而不是永久展开创建表单、稀疏卡片墙和只能上一页/下一页的临时页面。

## Completion criterion

- Workflow Source 耐久保存 name、description、category、tags、createdAt、updatedAt，旧 Source 继续合法；元数据修改仍走唯一 authoring patch / CAS 路径，时间戳由 Application 统一拥有。
- 新建工作流只保留单一主按钮；Modal 提供名称、起始模板、描述、分类和标签，导入源码包进入次级 overflow。
- 工作流首页仅提供 list，不恢复 grid 或容器快捷键；默认展示名称/描述、分类、标签、节点数和状态或 revision，并支持可选列。
- 工作流搜索覆盖名称、描述、分类、标签和 ID；分类与标签是独立 facet，标签支持多选；创建/修改时间支持可选列、最近日期筛选与排序；分页使用底部数字页码、结果范围和每页数量。
- 资源库的录制与视觉模板仍为互斥上下文，但统一使用高密度列表；名称、描述、分类、标签和类型特有信息分列展示。
- 资源查询提供明确 facet 选项、多维筛选、排序、底部数字分页、每页数量、整行选择和选择后 batch actions；500/1000 条数据时单页 DOM 有界。
- 创建、编辑、选择输入录制和选择视觉模板使用一致的复合字段与可搜索 picker；不以大卡片墙或不可搜索下拉承担大规模选择。
- 普通枚举 Select 使用共享的最长选项自适应宽度和最大宽度约束，短默认项不得压缩长选项；规模化实体选择仍必须使用搜索式 picker。
- 行选择进入统一上下文工具栏；批量分类/标签只修改用户明确选择的字段，普通动作和危险删除视觉分组。
- 关键行为由真实服务/组件测试覆盖，并通过阶段批量门禁与 WebView/UAC 真机视觉接受。

## Blocked by

Workflow Source / authoring contract、Asset query projection，以及用户对当前 production build 的真机接受。

## Verification

- Schema 与 authoring回归：旧 Source、完整元数据、trim/dedupe、长度边界、CAS 更新。
- Workflow service：多维搜索、category、all-tags、node count、facet、数字分页，至少 500 条 fixture。
- Asset service：kind 上下文下的 category/tag facet、组合筛选、数字分页，至少 1000 条 fixture。
- WorkflowsView 与 AssetsView 真实组件测试：创建/编辑 Modal、筛选、列选择、选择/batch、分页、类型上下文隔离。
- 批量元数据单元测试：分类保持/设置/清空，标签保持/添加/移除/替换/清空，大小写不敏感去重和空操作拒绝。
- 阶段末统一执行前端聚合测试、task check、task build、WebView smoke；人工检查 1920×1080 与较窄窗口截图。
- 启动 UAC production build，以真实 workspace 检查工作流与资源的创建、编辑、筛选、分页和选择器闭环。

## Out of scope

- 不恢复工作流或资源 grid 视图。
- 不恢复容器/工作流快捷键；快速启动承担该入口。
- 不在本阶段引入完整文件夹/Collection 后端或任意 dock 系统。
- 不用文件时间伪造旧 Source 无法可靠提供的历史创建日期；旧记录保持未知，新建和后续修改记录真实时间。
- 不在每个微改动后运行整仓门禁；按阶段批量验收。

## Result

Implementation complete; awaiting user acceptance。Workflow Source 已恢复 description/category/tags/createdAt/updatedAt 耐久契约和 CAS 更新；工作流首页、资源库均改为纯列表、多维 facet、底部数字分页与统一上下文选择工具栏，工作流增加日期列、最近日期筛选和时间排序，未恢复 grid 或工作流快捷键。工作流和资源均支持独立批量修改分类/标签；未明确选择的字段保持原值。新增 AdaptiveSelect 并迁移 48 个普通枚举选择器，避免短默认项压缩长选项；规模化资源 picker 仍保持搜索/分页路径。1000 条服务 fixture、真实 Nuxt UI 组件测试、task check（49 个前端测试文件、204 项测试）与 task build 均通过。浏览器自动视觉运行时当前无可用实例，需在 UAC production build 复核日期列密度和模板选择宽度；完整 smoke 的 Launcher 超时仍作为发布前独立问题保留。Slice 43 需经真实 workspace 接受后才能标记 completed。
