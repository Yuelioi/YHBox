# V4 execution plan

## Outcome

用户把 Yotta 理解成一个本地工作流工作台：打开即看到自己的工作流，进入即可编辑，配置缺项时就地修复，
随后直接运行。复杂能力仍然完整，但不会同时占据默认界面和主运行路径。

## Completed baseline

- [x] 技术基线：`645e0bad` 保留 V3 恢复点；`e330f47b` 统一 Workflow、Schedule 与 Run 路径。
- [x] 产品骨架：[工作流首页](slices/v4-e-workflow-home.md)、
  [计划首页](slices/v4-g-schedule-home.md)与[支持工作区](slices/v4-i-support-workspaces.md)完成。
- [x] 运行闭环：[运行就绪反馈](slices/v4-f-run-readiness.md)与
  [一致运行控制](slices/v4-j-run-control.md)完成。
- [x] 聚焦编辑器：[编辑器任务 Module](slices/v4-h-focused-editor.md)、单层命令顶栏、
  画布创建入口和窄窗口回归完成。
- [x] [稳定交付](slices/v4-k-stability-delivery.md)：`fishing-v2`、Windows build、性能预算、
  `task check` 与完整 WebView 旅程通过。

## Active: Go 全仓清扫

- [x] [Go 清扫审计](slices/v4-l-go-cleanup.md)：完成规模、依赖、执行链、组合根、兼容读取和
  可删除表面的证据化审计。
- [ ] 收敛启动与执行环境装配：桌面和 CLI 共享本地 runtime 打开路径；安装事实只派生一次
  Host Profile、Policy、Provider 集合与 generation lease。
- [ ] 校正运行边界：分离 Program、Adapter ABI、Compiler 与 Executor，但只保留一个 production
  Executor；将产品层“编译”改为符合实际行为的“检查工作流”。
- [ ] 深化 Application 与 Workflow use case：Source authoring、Run coordination、library
  query/batch decision 进入各自的具体 Module。
- [ ] 删除已证明无生产调用的兼容别名、测试便利 API 和浅转发；为 settings、node package、
  Run 与 Blob 的持久化兼容入口建立退役证据。
- [ ] 清理 `cmd/workflow-editor-smoke` 与架构文档，并以 `task check`、Windows build、
  `fishing-v2` 和完整 WebView 旅程验收。
