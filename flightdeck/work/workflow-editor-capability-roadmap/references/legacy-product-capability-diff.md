# 旧产品栈与 3.1 能力机械对比

> 本文保留第一次机械盘点的 provenance，不再作为当前 verdict。用户指定基线、当前决策与验收状态以 [`capability-ledger.md`](capability-ledger.md) 为准。

## Scope and provenance

- 旧基线：用户指定提交 `8316d590dbc8429b783b99982ff30d15e650c59a`；独立 detached worktree 位于 `E:\projects\organizations\yottaapp\yotta-3.0-reference`。
- 新基线：当前工作树的 Workflow 3.1 产品栈。
- 取证入口：旧/新 router、主视图、store、编辑器 composables、Wails backend 调用、Settings schema、Node Catalog、compiler/provider composition。
- 判定层级：可见入口、管理流程、创作绑定、运行闭环。文件仍存在只算证据，不自动判定能力已恢复。
- 该表只保留原始能力清单；实现变化统一更新 capability ledger，不再在两张表维护平行 verdict。

## Capability matrix

| 能力域 | 旧版证据 | 3.1 现状 | Verdict / 路由 |
| --- | --- | --- | --- |
| 新建工作流与 Start | Container create/editor；执行入口 | 新 source 创建并带 run-root，Stage 1 GUI 已验证 | 已恢复 |
| 工作流运行/编辑 | ContainersView/store/execution | WorkflowsView Run/Edit | 已恢复基础 |
| 工作流删除/批量删除 | containers store delete/deleteMany | 无 DeleteSource；列表无选择 | 缺失，Slice 10 |
| 工作流搜索/排序/分页 | 旧 workspace/list 模型及选择 composable | ListSources 一次返回全部，无 query/total | 缺失，Slice 10 |
| 工作流 export/package | 旧版仅 ExportPackage zip，明确排除 installation；未发现 ImportPackage 闭环 | 3.1 已有 canonical portability 实现，待旅程复验 | 保留 Source portability；不恢复旧 Container zip |
| 节点目录搜索 | NodeLibraryPanel/NodeSearchModal | WorkflowEditorView 已有 catalog search | 已恢复基础 |
| 画布节点定位/命令面板 | NodeSearchModal/CommandPalette/useCommandPalette | Source-native Ctrl/⌘+F 定位已恢复；动作由工具栏/选择条/快捷键承载 | 定位已恢复；旧 palette 由现有入口替代 |
| 类型感知拖线推荐 | inlineNodeCandidates/useInlineMenu | 共用 compatibility + connection menu | 已恢复，Slice 2 |
| 多选/Delete/clipboard | selection/references/clipboard composables | EditorSession 原子命令 | 已恢复，Slice 3 |
| 对齐/分布/自动布局/吸附 | useGraphLayout/useElkLayout/useSnapEngine | 实测尺寸 ELK、对齐分布和吸附 | 已恢复，Slice 3 |
| 状态变量 | VarsPanel/useVarMutations | WorkflowStatePanel/EditorSession | 已恢复基础 |
| 动态 pin/复杂字段编辑 | DynamicInputs/Outputs、StructuredInput | GeneratedFieldEditor/contract projection | 部分替代，需按节点验证 |
| diagnostics/运行时间线 | ProblemsBar/debug panel | compiler diagnostics + journal timeline | 已恢复增强，Slice 4 |
| 暂停/单步/断点/停止 | legacy debug methods | 同 scheduler 真调试 | 已恢复增强，Slice 5 |
| comment/reroute | CommentBoxNode/图编辑工具 | 无 3.1 对等创作入口 | comment 延期为 annotation；旧语义 reroute node 不恢复 |
| subgraph 创作/折叠 | subgraph lifecycle/folding/props | 3.1 已实现 Source-native GraphCall/multigraph，待 G12 复验 | 保留新实现，不恢复 virtual marker |
| snippet/表达式/yt console | snippets、expression editor、yt console | 未接入 3.1；安全模型不同 | 旧任意宿主执行明确不恢复；未来仅 sandboxed Script Node |
| 模板复合节点 | Wait/Click template | exact target + BlobRef contracts | 已恢复，Slice 6 |
| 资源搜索/单项删除/variant | 旧 AssetDock/TemplateDetail | AssetsView 有搜索、单项删除、variant summary | 部分恢复 |
| 资源批量/分页/清理维护 | AssetPager/SelectionBar；Maintenance 实际清理 subgraph | 既有实现待 1000 assets 旅程复验 | 3.1 发布前完成；Blob GC 需完整 roots |
| 简易键鼠录制 | useRecording/RecordingSaveModal | Slice 7 steps draft WIP | 正在恢复 |
| 鼠标轨迹录制 | ClipTimeline/recording trajectory | Slice 7 trajectory draft WIP | 正在恢复 |
| 录制 HUD/暂停/恢复 | RecordingHUDView/store/service | 后端与 HUD 仍在，主编辑器正在接回 | 部分恢复，Slice 7 |
| 截图拾取/模板保存 | ScreenPicker/asset capture | ScreenPicker 和 asset service 仍在 | 需随 target seam 复验 |
| F9 捕获窗口 | NodeInspector 调 StartWin32WindowTargetCapture | 后端仍在，Settings 未调用 | 入口断线，Slice 9 |
| Win32 target 安装 | 旧 Win32WindowTarget + 新 exact profile | 当前唯一完整 installed target | 可用但架构写死，Slice 13 |
| Android/ADB | Android node/profile/controller/catalog | Controller 在，3.1 安装/Settings/runtime 未接 | 断线，Slices 13→8 |
| Browser CDP target | Browser controller/discovery/client | 3.1 产品路径曾实现但无可信真实 smoke | Slice 36 复验后决定支持声明 |
| macOS desktop target | 旧版无完整实现 | 当前 Profile/Settings/provider 均 Win32 专用 | 先做 Slice 13，之后只加 Adapter |
| 悬浮窗 launcher | FloatingLauncher/LauncherSurface/OpenLauncher | 页面与后端在，主入口无调用 | 入口断线，Slice 12 |
| 应用安装取消 | 应用安装流程 | 空 path return，但用户真机报告持续提示 | P0 复现，Slice 9 |
| AI endpoint | AI settings/connection | profile 无 endpoint，默认官方 URL | 缺失，Slice 11 |
| 浏览器 alert/成功 toast | 旧新反馈混用 | 已开始统一 Nuxt UI，但需全仓扫描 | 跨阶段 UI 约束 |

## Platform seam finding

当前并非所有 Windows 代码都写死。pkg/input、pkg/capture、winutil 与 tools hotkey 使用 build-tagged Adapter；internal/automation/controller 也已有平台中立 Controller Interface。

真正的问题位于 3.1 production installation：

- Settings 公开 automation.win32Targets。
- ProfileDraft 直接含 Application executable、WindowTitle/Class、sendinput/postmessage、gdi/wgc。
- provider 的 TargetKind 固定为 win32-window，driver Interface 要求 ResolveWindow。
- non-Windows 的 installed module整体 PlatformSupported=false。
- 节点 capability、appbootstrap policy 和前端 Inspector 都引用单一 Win32 kind/集合。

因此现在添加 macOS 不是“写一个 darwin 文件”即可。Slice 31 必须把 profile/schema/editor/runtime 收敛到 adapter-owned manifest；Slice 36 用最小 macOS descriptor/compile proof 验收通用 Workflow/节点/runtime 不随 OS 增长而改动。

## Unknowns and invalidation

- 旧 ContainersView 只提供 ExportPackage，未发现 ImportPackage 产品闭环；该事实已确认。
- Source graphs 与旧 subgraph 的语义并非一一对应；compiler 当前明确只接受唯一 main entry graph。
- 资产规模化管理属于 3.1 发布门禁；任何 Blob GC 必须先证明 Source/Run/package 等完整引用根可枚举。
- automation Settings、Node capability target taxonomy、appbootstrap policy 或主导航变化后，必须重跑本表对应域的四层检查。
