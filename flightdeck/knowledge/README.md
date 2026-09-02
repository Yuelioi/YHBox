# Yotta task knowledge

Knowledge 只保存可跨 Work 复用的项目操作指南。先按要修改的对象选择一篇；系统事实与代码地图回到
[docs](../../docs/README.md)。

| 任务 | 指南 |
| --- | --- |
| 选择检查、构建、打包或 smoke | [构建与验证](build/build.md) |
| 修改页面、组件、样式或视觉反馈 | [前端 UI](frontend/ui.md) |
| 修改 Workflow 画布、选择、拖拽或 EditorSession | [Workflow 编辑器](frontend/workflow-editor.md) |
| 新增/修改 Data Type、Node Contract 或 adapter | [节点开发](nodes/development.md) |
| 修改 Target、输入、捕获、录制或鼠标/键盘语义 | [自动化输入与捕获](automation/input-and-capture.md) |
| 新增 Go service、RPC、event 或 frontend transport | [Wails services](wails/services.md) |
| 新增或修改错误 ID、params、RPC/Run/异步失败反馈 | [错误契约](errors/error-contract.md) |
| 修改 Settings installation、运行环境代或后台生命周期 | [配置与生命周期](runtime/configuration-and-lifecycle.md) |
| 修改 profile、durable store、schema、backup 或 migration | [存储与迁移](storage/migrations.md) |
| 新增或更新 README、docs、Knowledge 与链接 | [文档维护](documentation/maintenance.md) |

这里不保存某次故障的时间线、旧版本方案、测试数量快照或生成清单。Knowledge 本身也不是事实源；使用
一篇指南前先核对其中的路径、命令和不变量仍受当前生产代码/Task/测试支持。新的结论只有在当前实现验证、
能服务另一项独立工作、并可改写为正向规则时才进入 Knowledge。
