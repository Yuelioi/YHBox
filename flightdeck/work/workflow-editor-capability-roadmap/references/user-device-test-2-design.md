# 用户真机测试 2：根因与工作台重构方案

## Source first

本轮先检查用户真实 workflow、前端 session/event 合并、picker 交互、资源库布局和 Inspector 字段，再做外部模式调研。

- `EditorSession.start()` 等待 `StartDebugRun` RPC 返回后才设置 `activeRun/debugSnapshot`；`acceptDebugSnapshot()` 在此之前拒绝 event。后端 worker 可先发 paused event，因此较新的 generation 被丢弃，随后旧的 running RPC snapshot 成为永久状态。
- `AssetPickerModal` 的 `article` 没有 click、role、tabindex 或 keyboard handler。模板只能点击分辨率 chip 形成 candidate；普通用户点击卡片没有任何结果。
- `WorkflowDebuggerPanel` 将 graph path、current/previous、inputs/state/queue/outputs 直接平铺；所有空组都占列；completed 仍沿用 will-execute 标签。
- `AssetsView` 将 recording section 放在 kind tabs 之外，导致视觉模板上下文仍显示录制；默认每页 24 个大卡片、搜索/分类/标签/排序/创建/维护动作挤在同一行。
- `WorkflowInputBindingEditor` 先渲染“更换资源”按钮，再渲染第二个已选资源容器和第三行清除动作，形成重复控件。

## Confirmed red loop

- EditorSession：paused generation 2 event 在 StartDebugRun 返回 running generation 1 前到达，最终错误地保留 running。
- AssetPickerModal：点击模板 `article` 后没有 `aria-selected`，confirm 保持不可用。

## Reference patterns

- [Unreal Engine Content Browser](https://dev.epicgames.com/documentation/en-us/unreal-engine/content-browser-interface-in-unreal-engine) 将来源、Collection、Filters、Search、Asset View 分区，并允许 List/Tiles/Columns；不同资产类型可以有独立浏览上下文。
- [Unreal filters and collections](https://dev.epicgames.com/documentation/unreal-engine/filters-and-collections-in-unreal-engine?lang=en-US) 将文本搜索与可组合筛选分开，最近使用也是正式筛选。
- [Carbon Data Table](https://carbondesignsystem.com/components/data-table/usage/) 将 search/filter/settings/primary action 放在 toolbar；行支持紧凑密度、排序、分页和选择；选择后才出现 batch action bar。
- [VS Code debugging](https://code.visualstudio.com/docs/editor/debugging) 将调试控制与当前暂停位置放在首层，Variables/Watch/Call Stack 等按任务分组，不要求用户先解释内部 event snapshot。

## Local decisions

### Asset workbench

- 页面使用全宽工作台，不再使用营销式 eyebrow/说明 + 大卡片展厅。
- 左侧窄类型导航显示 Input recordings / Visual templates 与数量；切换即替换主动作和内容，不只改变查询。
- 主区使用紧凑 toolbar：搜索为主，筛选收进 popover，排序/视图切换为次级，创建动作随类型变化，清理进入 overflow。
- 默认密集列表，行包含 selection、预览、名称/说明、kind-specific metadata、category/tags、更新时间、row menu；模板允许 thumbnail grid，但列表仍可用。
- 服务端分页保持 DOM 有界；默认 50 行，footer 提供页信息和翻页。批量动作只在选择后替换 toolbar。
- 录制不是永久独立卡片：clips 上下文右上角提供 mode + Start；录制中显示紧凑 session bar；templates 上下文只提供 Capture template。

### Asset picker

- 资产整项是单选控件；click/Enter/Space 选中，double click 可确认。
- 模板卡片选中时自动选择首个/当前 variant；variant 选择是次级控件，不是唯一入口。
- selected state 使用边框、背景、check 与 `aria-selected`；固定 footer 显示选择摘要和 confirm。
- 查询仍服务端分页；默认 40 项。模板使用缩略图网格，clip 使用密集列表；搜索保持首要，category/tag 收进筛选区域。

### Inspector resource field

- 空值：单个“选择输入录制/视觉模板”复合按钮。
- 已绑定：同一容器显示 preview/icon、资源名、resolution/size/status，以及 Change/Clear；不重复展示。
- stale/unavailable 属于字段内错误状态，不额外制造 toast。

### Debug workbench

- header 只显示产品状态、主句和 Step/Continue/Pause/Stop。
- paused 主句为“已暂停，将在执行 X 前继续”；running 为“正在运行，等待下一个暂停点”；terminal 为“运行已结束/已停止”，绝不再显示 will-execute。
- 执行位置使用 Previous → Current/Stop position 的紧凑序列，可点击定位。
- 值区只显示非空分组及 count，以 tabs/折叠展开；无值时显示一个解释性 empty state，不出现四列“无”。
- run ID、graph path、digest 放在可复制的技术详情层。

## Stage acceptance

1. 两个红测转绿，再补 keyboard/terminal/monotonic cases。
2. 组件级验证工作台上下文隔离、整项选择、Inspector 单字段和 Debug 状态。
3. 500+ fixture 下服务端分页、DOM 有界、搜索/筛选/批量动作有效。
4. 阶段末统一运行 `task check`、production build、WebView smoke，并实际检查 Assets/Picker/Inspector/Debug 截图。
5. 启动提权 production build，真机完成模板选择和三节点连续单步；用户接受前不关闭 Slice 37。
