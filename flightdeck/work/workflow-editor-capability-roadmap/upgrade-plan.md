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
- Stage 8：Workflow Source portability、资产规模化管理与安全 Blob cleanup。
- Stage 9：Browser CDP exact installation、Settings、provider/Catalog 与 Chrome/Edge smoke。

## Stage 7：高级能力恢复（umbrella）

Slice 15 已完成能力审计和 Source-native 节点定位；Slices 16、17、19 已完成。Stage 7 保持进行中，直到 Slice 18 完成，不能以决策或延期替代产品交付。

## Stage 10：Source-native 多图创作（当前）

### Slice 18：subgraph、comment 与 reroute

- 定义 graph-call 深模块：动态 typed graph ports、调用节点、递归/深度预算、compiler/program/scheduler/journal graph path。
- 编辑器可创建、进入、返回、重命名和删除 subgraph，并正确处理引用。
- comment 是 authoring-only annotation，不进入 scheduler/capability。
- reroute 是 edge presentation metadata，不伪装为可执行 Node；布局与选择行为明确。
- 调试断点、单步、诊断、节点定位与自动布局在跨 graph 场景成立。
- clipboard、AI authoring patch、import/export 对多 graph 与 annotation/presentation metadata 保持完整。

阶段门禁：schema/compiler/scheduler/editor 聚合测试 → task check → task build → Windows WebView 多图 authoring/debug smoke → 人工截图检查。

## 发布阈值

Stage 10 完成并通过阶段门禁前，不称 3.1 major upgrade 完成。发布说明只用于解释明确不恢复的旧双运行时和危险宿主脚本，不得用来掩盖仍在本版本范围内的能力缺口。
