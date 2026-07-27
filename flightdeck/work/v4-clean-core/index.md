# V4 清爽高效工作台

## Goal

持续把 Yotta 做成一个打开就能找到、编辑和运行自动化的本地工作台。V4 以用户主路径的清晰度、
操作速度和现有数据的稳定性为完成标准；后续产品体验、功能与稳定性问题都在这一 Work 中继续优化。

## Status

Open

## Current

V4 产品与编辑器基线已经闭环，当前只保留 Go 全仓清扫这一条活动路线。审计确认首要问题是启动装配
重复和安装事实多次投影，其次是 `workflow/compiler` 边界过宽、Application 与 Wails use case 混杂。
用户可见的“编译”实际只检查并构造临时 Program，运行时仍会重新生成，因此产品术语应改为“检查工作流”。

## Next

1. 按 [Go 清扫审计](slices/v4-l-go-cleanup.md) 收敛 local runtime 与 execution environment 装配。
2. 之后校正 Program / Adapter ABI / Compiler / Executor 边界，并同步清除产品层“编译”术语。

## Progress

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
