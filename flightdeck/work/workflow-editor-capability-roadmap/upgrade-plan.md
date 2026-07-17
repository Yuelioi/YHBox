# Yotta 3.1 产品升级执行计划

## 完成定义

“3.1 major upgrade 完成”必须同时满足：

- 旧产品能力已按入口、管理、创作、运行四层完成机械对比。
- Windows 与 Android 的安装、创作和运行闭环完整；未来 macOS 只新增 Adapter。
- 工作流可安全导入导出，资产库可随规模增长管理。
- Source 的多 graph 结构拥有真实 subgraph 调用/创作语义，comment/reroute 不依赖旧 Container 模型。
- Browser CDP 在 installation、Settings、policy、provider/admission、Catalog 和健康诊断完整后声明支持。
- 所有阶段按批量验收原则留下可信 build、test 与宿主 smoke 证据。

以上定义已满足；3.1 产品能力升级范围完成，但产品尚未对外发布。

## 执行原则

- 一个 Stage 包含多个相邻 Slice；Slice 与本地 commit 只作为实现、恢复和回滚边界。
- Slice 内仅运行支撑继续开发的定向测试、typecheck 或 compile。
- Stage 完成后统一执行相关聚合测试、`task check`、`task build` 和触发的真实宿主 smoke。
- 保存等原地动作成功不 toast；失败使用统一 Nuxt UI。
- 不恢复旧 Container runtime，不创建第二套 Workflow Source/compiler/runtime。
- 不恢复任意宿主 JS/Wails/yt console；脚本只能走 capability-admitted 隔离执行。

## 已完成阶段

- Stage 0：旧版能力审计与架构纠偏。
- Stage 1：交互正确性、类型感知连线、选择/布局。
- Stage 2：结构化诊断、运行轨迹、同 scheduler 真调试。
- Stage 3：模板复合节点、资源预览、键鼠/轨迹录制。
- Stage 4：基础 UX、桌面目标/F9、工作流库、AI endpoint、launcher。
- Stage 5：平台中立 automation installation Adapter seam。
- Stage 6：Android/ADB installation、创作、运行与 emulator smoke。
- Stage 7：高级能力恢复 umbrella（Slices 15–19 全部完成）。
- Stage 8：Workflow Source portability、资产规模化管理与安全 Blob cleanup。
- Stage 9：Browser CDP exact installation、Settings、provider/Catalog 与 Chrome/Edge smoke。
- Stage 10：Source-native 多图创作、graph-call、comment、reroute 和跨图 debug/authoring。

## Stage 10 交付结果

- typed graph interface、GraphCall、递归/深度预算及 compiler/program/scheduler/journal graph path 已完成。
- 编辑器可创建、进入、返回、重命名、删除 subgraph；可插入调用、推断接口、折叠选择、编辑 typed bindings。
- comment 是 authoring-only annotation；reroute 是 edge presentation metadata，均不进入 scheduler/capability。
- 调试断点、单步、诊断、节点定位、clipboard、布局、AI/MCP authoring 与 portability 支持多 graph。
- 阶段门禁已通过：`task check`、production `task build`、Windows WebView 多图 authoring/debug/assets smoke 和人工截图检查。

## 发布阈值

本计划定义的 3.1 major upgrade 发布阈值已经满足。后续发布准备不得重新阉割这些能力；签名、安装包、license 表述与最终真机矩阵属于发布工程，不改变本计划的完成状态。
