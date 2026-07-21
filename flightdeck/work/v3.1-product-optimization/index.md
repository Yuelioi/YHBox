# 3.1 产品创作体验与运行工作台优化

## Goal

在稳定的 3.1 架构上恢复专业工作流创作能力，修正编辑器状态表达、节点发现、资源编辑和运行工作台体验。

## Status

Open

## Current

Stage A–L 已完成实现与必要增量门禁。精准回放使用独立单调时钟和 `start + event.TUs` 绝对 deadline；
playback session 在 Open 时固定已验证目标，Windows 不再逐事件解析窗口或置前。InputClip 回放节点只读
展示录制源与本机目标 counts/360，runtime 自动按 `target/source` 换算，不提供人为倍率。节点契约已恢复
稳定摘要 `5c353fb…`；开发期 `ff7ea9…` 节点会移除临时 binding 后迁回稳定契约。

## Next

重新构建 Windows 应用并重新打开工作流，先确认 `NODE_CONTRACT_MISMATCH` 与派生 `UNKNOWN_PORT` 消失；
再用原 1504/3526 事件样本复测：节点实际时长应接近 Clip nominal duration，绕花坛应回到接近录制终点，
半径不再由约 20 放大到约 25；核对 Inspector 的录制源/本机目标 counts/360。

## Progress

- Stage A–I 完成专业画布、Source-native 子图、Macro/InputClip、资源工作区、typed Authoring Surface、
  durable Snippets、Tab 快速添加、Macro 原地编辑、共享计划 Modal 与三路安全退出。
- Stage J 闭环录制/模板连续性、运行停滞可解释性、节点与大选项发现、参数密度、Run State 初值和动态
  Switch；阶段完整门禁与 WebView smoke 通过。
- Stage K 修正 Catalog 图标、运行工作台、可缩放/折叠侧栏、durable 状态类型、动态端口标题和悬空草稿
  执行语义，并把日常门禁改为按 Git 变更路由。
- Stage L 统一资源列表与拖拽，修正 binding 清除、日志 message、Inspector 总开关、工作流设置和子图
  空态/接口推断，所有事实继续写入 Source/Contract/runtime。
- 录制链已统一为后端权威 `armed → countdown → recording`，默认 F10/F11/F12；校准按“目标自定义 >
  活动档案 > 未校准”解析，保存页、任意时间线裁剪和可创建资源元数据均已闭环。
- 精准回放恢复绝对单调 deadline 和 session 级目标固定，按录制源/本机目标 counts/360 自动换算；撤回
  `turn-scale` 并把开发期契约安全迁回稳定摘要，production bundle 为 218155/220000 bytes。
- 2026-07-21 收尾增量门禁通过：Wails/节点/工作流契约、25 个相关 Go 包及 vet、frontend
  format/lint/typecheck/i18n、70 个测试文件/294 项测试；ESLint 债务从 24 收紧到 23。未运行无关 Rust、
  全仓 coverage/staticcheck；本轮 production bundle 已在精准回放修复后单独通过。
- Stage M 已记录下载工作流的依赖预检与本机 target/credential 重绑定；现有 Bundle 携带 Source/Blob，
  但在 M1–M3 完成前不宣称下载工作流可无提示直接运行。

## References

- [Real-device feedback](references/real-device-feedback-2026-07-20.md) — 正在收集的真机问题、证据与待验证项。
- [Real-device feedback follow-up](references/real-device-feedback-2026-07-21.md) — 第二批八项反馈、根因与复测点。
- [Real-device feedback batch 3](references/real-device-feedback-2026-07-21-batch-3.md) — 第三批九项反馈与验收点。
- [Real-device feedback research](references/real-device-feedback-research-2026-07-20.md) — 十项反馈的本仓事实映射与一手 UI/交互资料。
- [3.0/3.1 editor audit](references/current-vs-3.0-editor-audit.md) — 能力连续性和不恢复旧 runtime 的依据。
- [Editor discovery and modal decisions](references/editor-discovery-and-modal-decisions.md) — Stage I 研究与取舍。
- [Node density and optional pins](references/14-node-density-and-optional-pins.md) — 紧凑节点投影证据。
- [Workflow resource editing](references/15-workflow-resource-edit-and-safe-exit.md) — Macro 编辑与退出语义。
- [Tab and Snippet flow](references/16-tab-menu-and-snippet-shortcuts.md) — 快速添加与快捷键实现边界。
- [Schedule modal flow](references/17-schedule-modal-flow.md) — 计划 Modal 的验收记录。
- [Canvas authoring boundary](../../knowledge/frontend/canvas-node-authoring-boundary.md) — 当前画布创作规则。
- [Build and acceptance](../../knowledge/build/build.md) — 完整门禁的触发条件。
