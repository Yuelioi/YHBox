# 3.1 产品创作体验与运行工作台优化

## Goal

在稳定的 3.1 架构上恢复专业工作流创作能力，修正编辑器状态表达、节点发现、资源编辑和运行工作台体验。

## Status

Open

## Current

Stage N 的 N1–N2 已完成：定义管理、调用定位、稳定接口 ID/显示名称、显式单入口、typed data ports、
命名 exec/error exits、引用安全移除和带预览自动推导均已落到唯一 Source/authoring/compiler 链路。当前进入 N3，
实现复制调用、独立复制定义、展开调用和显式原子级联删除。

## Next

完成 [N3 子图生命周期](slices/stage-n3-subgraph-lifecycle.md)：先以 Source 转换测试锁定复制调用、复制定义、
展开调用和级联删除，再扩展单一 authoring patch seam 与无歧义 UI 动作。

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
- 2026-07-21 精准录制/回放 Windows 真机校验通过，Stage L 验收闭环。
- Stage M 已记录下载工作流的依赖预检与本机 target/credential 重绑定；现有 Bundle 携带 Source/Blob，
  但在 M1–M3 完成前不宣称下载工作流可无提示直接运行。
- 2026-07-21 确认在线试验市场使用独立兄弟仓库 `yotta-registry`、Nuxt UI 与 GoFrame；实现基线已对齐
  `yueli-official/platform/flightdeck/knowledge` 的 API、SSR/BFF、目录站、OIDC 与 staged import 约定。
- 2026-07-21 Stage M Grilling 已收敛 Workflow Resource/Global Asset、Release/Installation、Target/Credential
  Profile、Node Package trust/install/consent、更新/派生/回退、Registry 上架/下架、作者证明与在线/离线交付语义，
  并重写为 M1–M10 跨三仓执行计划。
- 2026-07-21 M1 完成首个正式可移植 Source/Bundle 合同：新增 Workflow Resource、Resource Binding、Target Profile
  Definition、Credential Requirement 与精确 Node Package Dependency；schema 统一负责严格校验、资源解析和 Blob
  inventory，compiler 校验 Catalog lock 并复用原 Blob value path，Bundle manifest 锁定 dependency 且包含所有资源字节。
- M1 `task check` 通过：Workflow/Wails 合同一致，20 个受影响 Go 包通过，frontend format/lint/typecheck/i18n 通过，
  70 个测试文件/294 项测试通过；未运行非增量的 `task check:full`、真实宿主或 production bundle。
- 2026-07-24 Stage N 启动：折叠开放边界修复已提交为 `26577e20`；调用节点和内部 boundary 的 handle gutter/
  长标签布局已由组件测试锁定；Unreal、Blender、Node-RED、Unity 官方资料已收敛为定义/调用分层、canonical
  interface 清单和安全定义删除方案。
- 2026-07-24 N1 完成：新增 Source-derived 子图管理器、同名 ID 消歧、调用引用定位、接口健康摘要和定义/
  调用无歧义删除；修复缺失 graph ID 误删最后一个定义。`task check` 通过 74 个测试文件/304 项测试。
- 2026-07-24 N2 完成：Graph interface 分离稳定 ID/显示名称，显式发布单入口、typed data ports 和命名
  exec/error exits；自动推导改为引用安全预览。`task check` 通过 21 个受影响 Go 包和 76 个测试文件/310 项测试。

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
- [Subgraph management research](references/subgraph-management-research.md) — 定义/调用、接口编辑和生命周期的一手资料与本地方案。
