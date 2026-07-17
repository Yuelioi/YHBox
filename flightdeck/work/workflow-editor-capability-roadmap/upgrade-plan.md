# Yotta 3.1 产品升级执行计划

## 完成定义

“3.1 major upgrade 完成”必须同时满足：

- 旧产品能力已按入口、管理、创作、运行四层完成机械对比。
- Windows 与 Android 的安装、创作和运行闭环完整；未来 macOS 只新增 Adapter。
- 工作流可安全导入导出，资产库可随规模增长管理。
- Source 的多 graph 结构拥有真实 subgraph 调用/创作语义，comment/reroute 不再依赖旧 Container 模型。
- Browser CDP 只有在 installation、Settings、policy、provider/admission、Catalog 和健康诊断完整后才声明支持。
- 所有阶段按批量验收原则留下可信 build、test 与宿主 smoke 证据。

## 执行原则

- 一个 Stage 包含多个相邻 Slice；Slice 只作为实现、恢复和提交边界。
- Slice 内仅运行支撑继续开发的定向测试、typecheck 或 compile。
- Stage 完成后统一执行相关聚合测试、task check、task build 和触发的真实宿主 smoke。
- 保存等原地动作成功不 toast；失败才使用统一 Nuxt UI。
- 不恢复旧 Container runtime，不创建第二套 Workflow Source/compiler/runtime。
- 不恢复任意宿主 JS/Wails/yt console；需要脚本时只能走 capability-admitted 隔离执行。

## 已完成阶段

- Stage 0：旧版能力审计与架构纠偏。
- Stage 1：交互正确性、类型感知连线、选择/布局。
- Stage 2：结构化诊断、运行轨迹、同 scheduler 真调试。
- Stage 3：模板复合节点、资源预览、键鼠/轨迹录制。
- Stage 4：基础 UX、桌面目标/F9、工作流库、AI endpoint、launcher。
- Stage 5：平台中立 automation installation Adapter seam。
- Stage 6：Android/ADB installation、创作、运行与 emulator smoke。

## Stage 7：高级能力恢复（重新打开）

Slice 15 已完成能力审计和 Source-native 节点定位，但不能以决策或延期替代产品交付。Stage 7 作为 umbrella 保持进行中，直到 Slices 16–19 完成。

## Stage 8：可迁移数据与规模化资源（当前）

### Slice 16：Workflow Source portability

- canonical bundle，只包含 Source manifest、Source bytes 与 exact referenced Blob payload。
- 导出不携带 installation、secret、credential、endpoint、PID/path/HWND 或 executable payload。
- 导入执行限额、路径、hash、schema、NodeRef/BlobRef 完整性校验，私有 staging 后原子发布。
- 默认生成新 workflow identity；显式覆盖才允许 revision/sourceHash CAS。
- 工作流首页提供导入与单项/批量导出，取消文件对话框为无副作用。

### Slice 17：资产规模化管理

- QueryAssets 后端分页、搜索、分类/标签/类型筛选与稳定排序。
- 当前页/跨页选择、批量 metadata 与批量删除，逐项结果和失败原地反馈。
- variant 增删/重拍保持 exact BlobRef；详情和列表都可维护。
- Blob 清理必须基于完整 root inventory 和 preview/confirm，不能把删 metadata 当 GC。

阶段门禁：Slices 16–17 定向聚合测试 → task check → task build → Windows WebView import/export/asset management smoke → 人工截图检查。

## Stage 9：Browser CDP 产品闭环

### Slice 19：Browser CDP installation

- 通过既有 target-kind Adapter seam 接入 exact loopback endpoint + page identity。
- Settings 完成发现、安装、consent、健康/漂移诊断。
- provider/admission/Catalog 暴露真实支持的截图、点击、移动、拖拽、滚动、组合键与文本。
- 不负责静默启动外部浏览器；用户必须显式启用受控调试端口。
- Windows 真 Chrome/Edge CDP smoke 后才声明产品支持。

阶段门禁：Browser Adapter conformance/定向测试 → task check → task build → Windows WebView + 真 Browser CDP smoke。

## Stage 10：Source-native 多图创作

### Slice 18：subgraph、comment 与 reroute

- 定义 graph-call 深模块：动态 typed graph ports、调用节点、递归/深度预算、compiler/program/scheduler/journal graph path。
- 编辑器可创建、进入、返回、重命名和删除 subgraph，并正确处理引用。
- comment 是 authoring-only annotation，不进入 scheduler/capability。
- reroute 是 edge presentation metadata，不伪装为可执行 Node；布局与选择行为明确。
- 调试断点、单步、诊断、节点定位与自动布局在跨 graph 场景成立。

阶段门禁：schema/compiler/scheduler/editor 聚合测试 → task check → task build → Windows WebView 多图 authoring/debug smoke → 人工截图检查。

## 发布阈值

Stage 8–10 全部完成并通过阶段门禁前，不再称 3.1 major upgrade 完成。发布说明只用于解释明确不恢复的旧双运行时和危险宿主脚本，不得用来掩盖仍在本版本范围内的能力缺口。
