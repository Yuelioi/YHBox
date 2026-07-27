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
- [x] [工作流检查反馈](slices/v4-m-workflow-check.md)：产品层统一为“检查工作流”，检查当前内存草稿，
  缺少必填输入时展示可定位问题，断开草稿警告不阻断可达运行。

## Completed: Go 清扫与瘦身

- [x] [Go 清扫审计](slices/v4-l-go-cleanup.md)：完成规模、依赖、执行链、组合根、兼容读取和
  可删除表面的证据化审计。
- [x] [收敛启动与执行环境装配](slices/v4-n-local-runtime.md)：桌面和 CLI 共享本地 runtime 打开
  路径；安装事实只派生一次 Host Profile、Policy、Provider 集合与 generation lease。
- [x] [校正运行边界](slices/v4-o-runtime-boundaries.md)：分离 Program、Adapter ABI、Compiler 与
  Executor，但只保留一个 production Executor。
- [x] [深化 Application 与 Workflow use case](slices/v4-p-application-modules.md)：Source
  authoring、Run coordination、library query/batch decision 进入各自的具体 Module。
- [x] [清理 Compatibility 与无调用表面](slices/v4-q-compatibility-deletion.md)：删除已证明无
  production 调用的兼容别名、测试便利接口和浅转发；为 settings、node package、Run 与 Blob 的
  持久化兼容入口建立退役证据。
- [x] [Go 清扫最终交付](slices/v4-r-go-cleanup-delivery.md)：清理 `cmd/workflow-editor-smoke`
  与架构文档，并以 `task check`、Windows build、
  `fishing-v2` 和完整 WebView 旅程验收。
- [x] [Go 生产代码瘦身](slices/v4-s-go-slimming.md)：以 `2d2c2226` 为基线单独核算生产代码，
  删除浅 Module、重复决策和过度防御，使生产 Go 从净增转为净减。
