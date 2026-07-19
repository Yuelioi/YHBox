---
slice: "39"
title: 录制与资产创作闭环
status: completed
---

# Slice 39：录制与资产创作闭环

## Outcome / Question

让资源库、录制、模板与节点资源绑定成为可管理数百至数千项资产的统一工作台；修复模板不可选择，并让选择器和 Inspector 遵守同一资源引用模型。

## Completion criterion

- recording completion 仍保持单一 store/session owner；资源库入口只保存，编辑器入口可保存后插入。
- 简易/精准录制与编辑能力不回退。
- Input Recording 与 Visual Template 是互斥的工作上下文；切到模板后录制控制和录制专属动作完全退出界面。
- 资源库默认使用高密度列表和服务端分页，提供搜索、筛选、排序、当前类型主动作、整行多选、选择后 batch bar、行内 overflow；模板可切换缩略图网格。
- 500+ 资产下单页 DOM 有界，管理动作不依赖浏览大卡片墙。
- Template/Clip picker 的整项可点击、可键盘选择；单击产生明确 selected state，双击或固定 footer 确认；多分辨率 variant 是选中资产后的次级选择。
- Inspector 使用单一复合资源字段显示缩略图/图标、名称、元数据、状态和更换/清除；不再出现“更换资源”按钮与重复资源行。
- 关键行为使用真实组件回归测试，不以 source-string test 替代。

## Verification

- AssetPickerModal 整卡点击、键盘、variant 与确认组件测试。
- AssetsView 类型上下文、分页/筛选、选择/batch、录制入口和视觉模板隔离测试。
- WorkflowInputBindingEditor 已绑定/空/stale 模板与 clip 组件测试。
- 阶段末统一跑前端聚合测试、task check、WebView 截图与 500+ fixture 视觉/交互 smoke。
- 提权真机复测录制、模板捕获、模板选择与 workflow 运行。

## Out of scope

- 不实现完整文件夹/Collection 后端或 UE 级可停靠多 Content Browser；当前先使用已有 kind/category/tag/search/sort/page 能力。
- 不把模板和录制拆成两套 Store。
- 不加载全部资产到前端后伪分页。
- 不在每个微改动后运行整仓门禁。

## Result

Completed。AssetPickerModal、Inspector 复合资源字段、互斥资源上下文、服务端分页与批量管理已完成；1000 模板规模测试、真实组件测试和编辑器资源工作区旅程通过。Windows native recorder 进一步验证 precise hook、canonical codec、asset/blob save/reload 与回放，宏/精准录制的 session owner 和安全释放由聚合门禁固定。
