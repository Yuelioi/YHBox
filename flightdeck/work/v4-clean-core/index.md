# V4 清爽高效工作台

## Goal

持续把 Yotta 做成一个打开就能找到、编辑和运行自动化的本地工作台。V4 以用户主路径的清晰度、
操作速度和现有数据的稳定性为完成标准；后续产品体验、功能与稳定性问题都在这一 Work 中继续优化。

## Status

Finished

## Current

V4 产品、编辑器、运行链和 Go 清扫已经闭环。产品层“编译”已改为“检查工作流”，检查当前内存草稿
且不隐式保存；composition、执行 seam、Application 职责与持久化兼容完成结构收敛。
[Go 生产代码瘦身](slices/v4-s-go-slimming.md) 的最终权威统计为相对 `2d2c2226` 新增 2546 行、
删除 2617 行，净减 71 行；删除来自浅对象、重复所有权/状态/锁、一次性转发和重复 provider 合并，
不是压缩排版。最终 `task check`、Windows production build、`fishing-v2` 和完整 WebView 旅程通过。

## Next

None.

## Progress

- 2026-07-28 完成 [Go 生产代码瘦身](slices/v4-s-go-slimming.md) 与 V4：最终生产 Go 相对
  `2d2c2226` 净减 71 行。第二轮把 automation generation 所有权收回唯一 Runtime，删除重复
  BlobStore 投影、provider digest/merge 小对象，以及 Application 的 command → private method
  转发。`task check`、正式 Windows build、`fishing-v2` 与完整 WebView 旅程通过；完整旅程目录为
  `.task/workflow-editor-smoke/20260728-022022`，启动 3213ms。
- 2026-07-28 [Go 生产代码瘦身](slices/v4-s-go-slimming.md) 首轮完成：生产 Go 从净增 246 行变为
  净减 80 行，累计减少 326 行净变化；测试仍单独记账为净增 573 行，smoke 工具净减 16 行。
  删除 source library 双层对象、重复 patch candidate、双 lifecycle 状态/锁和一次性 runtime
  转发；7 个相关区域的定向测试及 `git diff --check` 通过。
- 2026-07-28 启动 [Go 生产代码瘦身](slices/v4-s-go-slimming.md)：纠正“结构验收通过等于清扫完成”
  的判断。相对 `2d2c2226`，生产 Go 净增 246 行、测试净增 573 行、smoke 工具净减 16 行；
  轻量化必须让生产 Go 转为净减，不能以拆文件或测试增长代替。
- 2026-07-28 完成 [Go 清扫最终交付](slices/v4-r-go-cleanup-delivery.md)：拆分 2964 行 WebView
  smoke 入口，补齐 composition 架构约束并修复 Windows plugin ABI 与空 StorageRoot 接缝。最终
  `task check`、Windows production build、`fishing-v2` retention、黄金工作流和完整 WebView
  旅程全部通过；截图目检确认编辑器、子图、资源与计划页面图标和布局完整。
- 2026-07-27 完成 [Compatibility 清理](slices/v4-q-compatibility-deletion.md)：Settings 与 Node
  Package 的旧格式成功读取后原子写回当前格式；Run/Blob 的旧 authority 只进入显式 root migration；
  journal writer 强制输出 v2。`docs/compatibility.md` 记录 3.1.0 direct-upgrade floor 与逐项退役条件，
  四个相关包定向测试通过。
- 2026-07-27 启动 [Compatibility 清理](slices/v4-q-compatibility-deletion.md)：已删除
  `ReplaceAutomation`、`GetLogSink`、ApplicationService 未读字段、automation `TargetKind` alias
  以及 production `NewApp/Shutdown` convenience；相关定向测试通过。
- 2026-07-27 完成 [Application 深化](slices/v4-p-application-modules.md)：新增 Source transition、
  Run admission、Run coordinator 与 source library 四个 concrete Module，保留唯一 Application
  命令面和唯一 worker/Executor；Application 与 Workflow service 定向测试通过。
- 2026-07-27 启动 [Application 深化](slices/v4-p-application-modules.md)：`sourceTransitions` 集中
  patch/candidate/state migration/CAS，`runAdmission` 原子绑定 Program durability、admission、
  provider snapshot 与 generation lease；Application 定向测试通过。
- 2026-07-27 完成 [运行边界校正](slices/v4-o-runtime-boundaries.md)：测试 Interpreter 退出
  production，`nodeadapter` 接管 host ABI，adapter/plugin 不再依赖 Source compiler；保留
  `ProgramSnapshot` 与唯一 production Executor 的私有深 Module，拒绝公开 mutable execution view。
- 2026-07-27 完成 [本地运行环境单一装配](slices/v4-n-local-runtime.md)：desktop 与 CLI 共用
  `localruntime.Open/Close`，并直接启动其唯一 Application owner；首次启动和 automation 热替换共用
  sealed execution environment；
  Profile、Policy、Provider 与 generation lease 从同一安装快照一次封存。
- 2026-07-27 启动 [运行边界校正](slices/v4-o-runtime-boundaries.md)：测试 Interpreter 移出 production；
  `nodeadapter` 接管 host ABI，`noderuntime` 与 `pluginhost` 不再依赖 `workflow/compiler`。六个相关包
  定向测试通过。
- 2026-07-27 完成 [工作流检查反馈](slices/v4-m-workflow-check.md)：按钮、问题面板和提示统一使用
  “检查工作流”；新增不落盘的 `CheckDraft` RPC；断开草稿节点缺少必填输入会显示 warning，但不阻断
  可达 Program。`task check` 通过 18 个受影响 Go 包、Wails 16 服务 / 140 方法及 93 文件 /
  394 项前端测试；真实 Windows WebView 旅程通过，启动 5.897 秒。
- 2026-07-27 完成 V4 Go 全仓清扫审计：扫描 659 个 Go 文件、约 13.1 万行，深读
  `desktopapp → appbootstrap → application → compiler/executor → run` 及兼容读取。确认
  `appruntime`、`noderuntime.Installed` 和 durable migration 是应保留的深 Module；首要清扫对象是
  重复的本地 runtime/安装环境装配，其次是 compiler seam、Application/use case 和可证明无生产调用
  的表面。证据、反例和实施门禁见 [审计记录](references/go-cleanup-audit.md)。`task check` 通过
  12 个受影响 Go 包、Wails 16 服务 / 139 方法契约及 93 文件 / 392 项前端测试。
- 2026-07-27 编辑器两轮层级优化及窄窗口修复完成：工作区入口直接平铺，图路径和命令收敛到单层顶栏，
  添加节点位于画布左上，接口推导回到子图面板。900px WebView 旅程通过，启动 2.550 秒。
- 2026-07-26 V4 产品基线完成：Workflow/Schedule/Run 身份统一，首页、计划、支持工作区、运行就绪、
  一致取消、编辑器任务 Module 和稳定交付切片全部闭环。
- 2026-07-26 `fishing-v2` 验证 7 Graph、60 Node、18 Resource、36 Blob 引用；production Windows
  build、真实 profile 迁移、性能预算、`task check` 与完整 WebView smoke 通过。
