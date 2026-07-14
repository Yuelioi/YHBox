---
topic: asset-workbench-upgrade
title: "Asset workbench upgrade"
summary: "Research and implement a commercial-grade asset browser and management workbench for templates, blueprints, and clips."
---

## State

生产实现已完成，等待在匹配仓库 Node 版本的环境执行根门禁。模板、蓝图和 Clip 已共用浏览工具栏、分类、分页、网格/列表、键盘导航和展开式工作台；三类资产仍保留各自真实预览与主动作。

目标气质是 **Quiet Precision Asset Studio / 安静精密的资产台**：专业、可信、预览优先；不靠玻璃、霓虹、渐变或大阴影制造“商业感”。

## Product decision

资产体验拆成两个层级，共用同一套数据和浏览组件：

1. **编辑器资产浏览器**：留在当前 dock，负责搜索、快速预览、选择和拖入画布。普通用户的高频路径必须短。
2. **资产工作台**：从浏览器右上角展开，负责跨类型浏览、分类与标签、批量操作、引用与健康状态、变体和资源清理。

资源管理不是第四种资产类型。现有“资源管理”从平级 tab 移到工作台工具栏的“维护”入口；“本地 / 在线”未来成为全局来源过滤，而不是只属于蓝图库。

用户侧统一命名：

- 模板管理 -> **视觉模板**
- 子图库 Explorer -> **自动化蓝图**
- Clip 管理 -> **操作片段**
- 资源管理 -> **资产维护**

内部术语 `Template`、`Subgraph`、`Clip`、GUID、rev 和 blob 仅在高级信息中出现。

## Design model

统一的是浏览器 shell、选择、筛选、详情与批量语义；不能统一的是三类资产的预览、指标、主动作和生命周期。

| 资产 | 主预览 | 首屏指标 | 主动作 | 专属管理 |
| --- | --- | --- | --- | --- |
| 视觉模板 | 真实截图，保留原始比例 | 分辨率、变体数、当前目标匹配状态 | 用于当前节点 | 截图、重拍、变体 |
| 自动化蓝图 | 由真实 `graph.nodes/edges` 生成的简化拓扑 | 节点数、输入/输出、必需变量、检查状态 | 插入蓝图 | 新建、复制、编辑、检查 |
| 操作片段 | 真实录制摘要时间轴 | 时长、事件数、鼠标模式、基准分辨率 | 预览 / 插入片段 | 录制、裁剪、重录 |

Clip 当前前端只有总时长、事件数和录制元数据，没有逐段事件摘要。第一版只能展示真实时长轨道和已有指标；若要显示键鼠事件密度，后端需新增有界 `activityBuckets` 摘要，禁止用随机或仅按总数生成的假波形。

## Information architecture

### 编辑器资产浏览器

```text
[视觉模板 40] [自动化蓝图 12] [操作片段 8]       [展开]

[搜索当前类型………………………………] [筛选 2] [排序/视图]
[全部] [最近] [未分类] [战斗] ...

┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│ 真实预览内容 │ │ 真实预览内容 │ │ 真实预览内容 │
│ 名称         │ │ 名称         │ │ 名称         │
│ 关键指标     │ │ 关键指标     │ │ 关键指标     │
└──────────────┘ └──────────────┘ └──────────────┘

[已选 3 项  批量操作]                         [40 项]
```

- 搜索是唯一常驻的大控件。
- 分类使用单行可横向滚动的 scope chips；标签、排序和高级条件进入筛选 popover，启用后显示计数和可移除条件。
- dock 不再常驻两整行 select/input，也不显示页大小选择。保留分页时使用紧凑的上一页/下一页与总数；用户密度偏好放进视图菜单。
- 窄 dock 单击资产后进入原位详情 drill-in，保留返回路径；不再用 modal 覆盖浏览上下文。
- 选择与管理分离：普通模式单击预览，双击或 Enter 执行明确主动作；点击“选择”后进入显式批量模式，避免单击语义随场景改变。

### 完整资产工作台

```text
资产工作台                                      [+ 新建] [维护] [⋯]
[视觉模板 40] [自动化蓝图 12] [操作片段 8]
[跨资产搜索……………………………………] [筛选] [排序] [网格/列表]

分类 / 收藏        资产内容                            资产详情
全部               大预览卡 / 舒展列表                 大预览
最近               多选与拖拽                          名称、分类、标签
未分类             健康/兼容/引用状态                  核心指标与引用
自定义分类         空态、错误态、加载态                [主动作]
                                                        高级信息（折叠）
```

- 左侧 category rail 可折叠，展示计数、最近使用和未分类。
- 中间内容区支持网格、舒展列表和紧凑列表；默认视图按资产类型记忆。
- 右侧稳定 inspector 用于比较和连续整理；仅删除确认、冲突解决等阻断动作使用 modal。
- 资产健康信息包括：缺失 blob、失效引用、需要重录、当前目标不兼容、蓝图检查错误。

## Visual system

- 继续使用深色中性 surface 与 emerald 品牌色。emerald 只表达当前选择、主动作和成功，不给每张卡铺绿色。
- 模板图像承担主要色彩；蓝图和 Clip 通过真实拓扑/时间轴建立身份，不靠高饱和类型色。
- 默认卡片从当前最小 112px 提升到约 156–180px；模板预览占卡片约 70%，名称与一行关键指标放在 footer。
- 卡片使用 8px 左右统一圆角、细微 surface 层级和边界变化；取消通用大阴影、发光与玻璃拟态。
- 正文不低于 12px，资产名称 13–14px；9–10px 决策信息升级到 11–12px。
- 选中态同时使用边框、check 状态和可读文字/ARIA，不只依赖绿色。
- 加载、缩略图失败、损坏、空结果和过滤无结果均使用不同的可恢复状态。

## Interaction contract

- 单击：选中并预览。
- 双击 / Enter：执行该资产的主动作（用于节点或插入画布）。
- Space：在显式批量模式中切换选择。
- 右键：上下文动作；所有关键动作同时存在可见或键盘路径。
- Arrow / Home / End：完整 grid/listbox 导航；使用 roving tabindex 或 `aria-activedescendant`，不让每个 option 都进入 Tab 序列。
- Ctrl/Cmd 多选、Shift 范围选择、Esc 退出批量/详情、Delete 进入受保护删除流程。
- 过滤、排序、密度、上次分类与 dock 宽度按资产类型持久化。
- 拖拽仍是高级用户快捷路径，但不能是唯一清晰的插入路径。

## Component boundaries

第一层抽取共同浏览能力，不把三类资产强塞进一个万能卡片：

- `AssetBrowserShell`：类型、搜索、过滤、视图、内容、状态和 footer slot。
- `AssetFilterToolbar`：查询、filter chips、sort、view density。
- `AssetCategoryRail`：分类、最近、未分类和计数。
- `AssetPager`：分页与结果状态。
- `AssetBatchBar` / `AssetBatchDialogs`：显式 selection mode 与批量编辑。
- `AssetInspectorShell` / `AssetMetadataEditor`：共同元数据布局；保存策略仍由各类型 adapter 持有。
- `TemplateAssetCard`、`BlueprintAssetCard`、`ClipAssetCard`：各自真实 preview 与指标。
- `AssetListRow`：仅供自动化蓝图与操作片段的紧凑列表复用。
- `useAssetBrowserState(kind)`：查询、过滤、排序、分类、选择、键盘 active item 与持久化。

修复现有漂移：Library/Clip 注释声明“双击插入”，实际却打开详情；新 contract 必须由共享 interaction controller 和测试锁定。

## Responsive contract

- dock `< 520px`：单列舒展列表或模板双列；筛选收进 popover；详情原位 drill-in。
- dock `520–760px`：模板 3 列，蓝图/Clip 舒展列表；可显示一行分类 chips。
- 工作台 `>= 1040px`：category rail + content + inspector 三栏。
- inspector 宽度不足时折叠为右侧 slideover；不能挤压卡片到失去预览价值。
- 使用 container query 适应 dock，而不是只依赖 app viewport；1640 最小应用宽度仍保留，但不是资产区布局的前提。

## Delivery plan

### Slice 1 — 浏览器骨架与模板纵切

- 建立 `AssetBrowserShell`、统一 toolbar、分类 chips、视图/密度持久化和可访问键盘模型。
- 视觉模板升级为 156–180px media-first 卡片，完成新空态、失败态、选择态和原位详情。
- 新建截图延续自动/上次分类，彻底消除截图后补分类。
- 移走“资源管理”平级 tab，增加“展开工作台”和“资产维护”入口。

### Slice 2 — 自动化蓝图

- 用现有 `Subgraph.graph` 生成有界、缓存的静态拓扑 preview；只保留主要节点块与边，不渲染完整编辑器。
- 展示节点数、输入/输出、必需变量、引用与检查状态。
- 统一双击/Enter 插入、拖拽、详情和批量语义。

### Slice 3 — 操作片段

- 第一版使用真实总时长轨道、事件数、mouse mode 与 base resolution。
- 评估并实现有界 `activityBuckets` 后端摘要，再升级为真实事件密度时间轴。
- 增加片段预览、插入和重录入口，避免只有通用 movie icon。

### Slice 4 — 完整资产工作台

- 三栏工作台、跨类型搜索、稳定 inspector、引用/健康状态与资产维护整合。
- 收藏、最近使用、保存的筛选视图作为后续效率能力；不阻塞第一轮升级。

### Slice 5 — 质量与商业化收尾

- 修复全部控件 accessible name、详情 label 绑定、模板变体键盘删除和 listbox 导航。
- 视觉回归覆盖 520/600/760px dock 与 1640/1920 app；覆盖 125%/150% Windows scaling。
- 真实 0、40、200、1000 项数据压测；图片懒加载，拓扑 preview 按 graph rev memoize。
- 验证拖拽、批量、删除保护、缩略图失败、损坏资产、长名称、中英文和键盘-only 路径。

## Acceptance criteria

- 普通用户进入任一类型后 5 秒内能识别“这是什么”和“如何使用”。
- 搜索之外最多保留 3 个同层常驻操作，顶部不再出现两排等权控件。
- 三类资产在不读名称时仍能通过真实内容结构被区分。
- 单击、双击、Enter、Space 和拖拽语义在三类资产中一致并有测试。
- dock 520–760px 无控件挤压、遮挡或文本小于 12px 的决策信息。
- 完整工作台可以连续检查多个资产，不需要反复开关 modal。
- 所有筛选和详情表单具备明确 accessible name；grid/list 支持 Arrow/Home/End。
- 不伪造 Clip 波形、蓝图图形或健康状态。

## Research basis

- Unreal Content Browser 将 sources、collections、filters、search、asset view 和 settings 分层，并允许 tiles/lists/columns 与直接拖入场景：<https://dev.epicgames.com/documentation/en-us/unreal-engine/content-browser-interface-in-unreal-engine>
- Unity Search 提供 list/grid/table、缩略图尺寸、Inspector、保存查询和快捷键，说明“结果视图 + 稳定详情”比 modal 浏览更适合专业工具：<https://docs.unity3d.com/6000.0/Documentation/Manual/search-window-reference.html>
- Blender Asset Browser 使用 catalogs、真实 preview、tags 与右侧 asset details，支持把分类浏览和资产元数据分开：<https://docs.blender.org/manual/en/4.1/editors/asset_browser.html>
- Figma Assets 将资产选择留在可调宽 sidebar，支持搜索、grid/list 和拖入 canvas，同时把更重的 library 管理放进独立 modal：<https://help.figma.com/hc/en-us/articles/360039831974-Explore-the-navigation-bar-and-left-sidebar>

## Review record

- Nielsen：21/40，处于“可用但需要显著升级”。
- Cognitive load：5/8 项失败，主要由窄 dock 同时承担选择、管理、详情和维护造成。
- 确定性 UI 扫描未发现典型 AI-slop 规则命中；问题属于信息架构、行为漂移与规则未覆盖的可访问性细节。
- 已确认的可访问性缺口：15 个筛选/分页控件缺显式 accessible label；详情分类/标签 label 未绑定；模板变体删除为 pointer-only；listbox 没有完整 Arrow/Home/End 模型。

## Next

- 使用仓库固定的 Node 22.23.1 执行 `task check`；当前机器 Node 24.18.0 会被严格工具链检查提前拒绝。
- Windows 桌面真机复核 520/600/760px dock、1640/1920 工作台以及 125%/150% 缩放；当前会话没有可连接的应用内浏览器实例。
- 后续若增加 Clip `activityBuckets`、引用/健康状态或跨类型搜索，必须使用真实后端摘要，不在前端推测或伪造。

## Implemented

- `AssetBrowserToolbar`、`AssetCategoryRail`、`AssetPager` 与 `useRovingAssetList` 统一了筛选、视图、分页及 Arrow/Home/End 键盘模型。
- 视觉模板使用真实截图和变体元数据；自动化蓝图从真实 nodes/edges 生成有界拓扑；操作片段仅展示真实时长、事件数、鼠标模式和基准分辨率。
- dock 单击选择、显式详情 drill-in；蓝图/Clip 双击或 Enter 插入。完整工作台提供分类、内容、常驻 inspector 三栏。
- 搜索、分类、标签、排序方向和视图按资产类型持久化；工作台低于 1040px 时 inspector 转为原位覆盖详情，低于 760px 折叠分类 rail。
- 资产维护从第四个产品 tab 移为独立工具入口；父宿主统一预载三类资产，避免父子重复请求。
- 详情表单补齐 accessible name，模板变体删除改为独立可聚焦按钮，列表使用 roving tabindex。

## Verified

- `pnpm -C frontend typecheck`
- `pnpm -C frontend test`：90 files / 598 tests
- `pnpm -C frontend build`：bundle budget passed
- `pnpm -C frontend i18n:check`：中英文 2859 keys parity / compile / refs passed
- `pnpm -C frontend lint`：0 warnings / 0 errors；`no-explicit-any` 技术债由 270 降至 267
- `task check`：供应链 pin 检查通过，随后因本机 Node 24.18.0 与仓库固定 22.23.1 不匹配而提前停止
