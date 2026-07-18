# 工作流首页与资源库管理阶段：源码审计和设计决策

## Source first

本阶段先检查当前 durable contract、服务查询、两个 View 和 3.0 旧界面，再查成熟管理表格模式。

- 当前 `schema.Workflow` 只有 `id/name`；`SourceView` 只有 workflow ID、name、revision、hash、json；工作流描述、分类和标签不是“页面没显示”，而是 3.1 Source 契约确实没有保存。
- `ParseSource` 使用严格 JSON Schema，不能把未知字段塞进 JSON 侥幸保存；恢复元数据必须扩展 schema、authoring command、application/service projection 和 tracked contracts。
- 外部修改通过 `Application.ApplyPatch` 与 authoring Engine 的 CAS 路径完成。列表编辑不得旁路直接改文件；新建由 Application composition root 负责生成 ID 与初始 Source。
- Asset Store 已保存 `Name/Description/Category/Tags/CreatedAt`，`AssetQuery` 已支持全文、精确 category、all-tags、sort 与 paging。资源页的主要问题是把信息挤进卡片/组织列、facet 输入粗糙、分页只有前后按钮。
- 3.0 `ContainersTab.vue` 已有搜索、分类、标签、排序、列选择、列表/卡片和底部分页；列包含状态、分类、标签、节点数、创建/修改时间与快捷键。用户明确只保留 list，并删除快捷键，因为快速启动已经覆盖。
- 当前首页把导入、名称、模板和创建永久放在页头，挤压常用查询；工作流行暴露过多动作，缺少描述/分类/标签。当前资源库虽已进入类型上下文，仍保留 grid 和缺少真正数字分页。

## Product direction

目标用户是长期高频使用桌面自动化的创作者。设计方差 3、动效 2、信息密度 8，沿用项目 semantic dark、Nuxt UI 与 Iconify Tabler。

1. 标题区只表达当前位置、结果总量和一个主动作。工作流用“新建工作流”，资源上下文用“开始录制”或“截图新模板”；导入、清理、刷新进入次级菜单。
2. 创建动作打开 Modal。工作流创建一次收集 name、template、description、category、tags；编辑复用同一元数据表单。
3. 搜索独占首行或主要宽度；category、tags、sort、columns 位于清晰的第二行。category 是单选 facet，tags 是可搜索多选 facet，不再要求用户手输逗号字符串。
4. 结果固定为密集 list/table。名称单元格同时承载描述摘要；分类、标签、节点数/类型信息分列；低频动作进入 row overflow。
5. 行选择后出现 batch bar；未选择时不展示批量操作噪音。
6. 分页器必须与列表底边相连，显示结果范围、数字页码、边缘页和每页数量。初始 20 行，支持 20/50/100，避免 500–1000 条数据时大 DOM。
7. column selector 参考旧版，但不包含快捷键。现阶段只提供真实可靠的列；创建/修改日期待 durable 来源明确后再恢复，不从文件时间伪造产品历史。
8. 资源录制与视觉模板是互斥内容上下文。切换 visual template 后不渲染 recording 控制；无论类型都不恢复 grid。
9. Picker 与 Inspector 保持上一阶段的整项选择和复合资源字段，搜索/筛选/分页适配大规模库。

## Reference patterns

- [Nuxt UI Pagination](https://ui.nuxt.com/docs/components/pagination) 提供受控 page、total、items-per-page、edge 与 sibling page，适合直接替换前后翻页按钮。
- [Nuxt UI Table](https://ui.nuxt.com/docs/components/table) 支持 selection、sorting、loading 与 sticky header；本项目可按现有 list 样式实现同一信息架构，避免为了库替换产生额外风险。
- [Carbon Data Table](https://carbondesignsystem.com/components/data-table/usage/) 将 search/filter/settings/primary action 放在 toolbar，并在选择后切换为 batch actions。
- [Primer Data Table](https://primer.style/product/components/data-table/) 强调密集扫描、明确列头、行级操作和合理默认行数。
- [Atlassian Dynamic Table](https://atlassian.design/components/dynamic-table/) 将排序、分页、加载与空状态视为同一数据表体验，而不是页面外零散控件。

## Acceptance shape

使用 500 个工作流、1000 个资产的 fixture 检查搜索/筛选/分页仍为服务端有界查询；1920×1080 下无需大面积空白或卡片墙，较窄窗口下关键操作不被挤掉。功能和视觉在一个阶段末批量验收，不对每个小字段重复跑完整门禁。


## Implemented batch

- Durable contract：Workflow Source 增加 description/category/tags；authoring 增加 tagged metadata command、Unicode 长度校验、trim 与大小写不敏感标签去重；列表编辑不旁路文件。
- Query projection：工作流全文搜索覆盖名称、描述、分类、标签和 ID，支持精确分类、all-tags、节点数排序、category/tag facets 与有界分页；资产 facets 按当前 kind 统计。
- Workflows UI：新建/编辑复用 Modal，分类和标签可搜索并可创建；列表支持列控制、本地列偏好、批量导出/删除和数字分页，导入/刷新收进次级动作。
- Assets UI：录制与视觉模板使用互斥左侧上下文；去掉 grid，复合元数据分列，筛选与创建/编辑字段统一为可搜索组件。
- Scale proof：服务测试覆盖 1000 条工作流或资产，前端真实组件测试验证第 2 页服务查询，因此大数据量不会把全部记录渲染进 DOM。

## Verification result

- `task check` passed：Go 全包、契约、静态检查、47 个前端测试文件与 195 项测试、production frontend build 和 bundle budget 全部通过。
- `task build` passed：生成 `bin/Yotta.exe` 及伴随 worker/CLI。
- `smoke-workflow-editor.ps1 -SkipLauncher` passed：新建工作流、编辑器主旅程和资源库导航完成，截图显示纯列表筛选工作台与互斥资源上下文。
- 默认完整 smoke 尚未通过：悬浮启动器执行新建的仅 Run 开始工作流时等待 success 超时。本阶段页面路径没有 WebView 错误，但该问题仍属于发布前开放项。
