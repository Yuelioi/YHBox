# V4-F 运行就绪反馈

## Goal

用户从任何入口运行同一个 Workflow，都得到同一种结果：能运行就立即启动；不能运行就说明当前缺什么；
磁盘、配置或内部故障保持为真正错误，不伪装成用户缺项。

## User outcome

- 工作流列表、编辑器和悬浮启动器不再显示互相矛盾的错误。
- 无需授权或发布步骤；正常 Workflow 一次点击直接运行。
- 缺 Target、Credential、节点配置或当前环境能力时，看到可理解的单一原因。
- 系统故障不会被吞掉，仍可进入日志和错误诊断。

## Preserve

- 所有入口继续调用唯一的 `Application.StartRun` / `StartDebugRun` 执行路径。
- Target 和 Credential 仍是本机用户配置，不写回可迁移的 Workflow Source。
- 编译诊断保留代码、Graph 和 Node 定位。
- Run 记录、Debug snapshot、Schedule daemon 和现有 admission/runtime 完整保留。

## Contract

`StartRunView.readiness` 是 UI 的唯一运行启动结果：

- `started`：已有有效 Run。
- `workflow-invalid`：编译诊断阻止启动。
- `target-required`：Target 缺失或不唯一。
- `credential-required`：Credential 缺失或不唯一。
- `environment-unavailable`：当前 provider 或 host 不支持。
- `failed`：持久化、策略配置或其他系统故障；Wails 调用继续返回错误。

前五类中除 `started` 外均为用户可理解结果，不通过异常驱动页面；`failed` 不得被投影层吞掉。

## Current

Application 已成为唯一 readiness 分类器；workflow service、工作流列表、编辑器、悬浮启动器和
Schedule 均复用它。编辑器不再出现无诊断的静默失败，启动器会显示短期可读原因。Schedule 行内可直接
试运行，失败后保存缺项状态并提供“去修复”入口。此前 Schedule 把“有编译诊断但 error 为 nil”误标为
queued 的缺陷已经修复。

Schedule 持久化 schema 从 v3 升到 v4，只新增可选 `lastReadiness`；旧文件迁移时不编造缺项，也不会
把设备路径、窗口或 Credential 写进 Workflow Source。

完整 Windows WebView 全旅程已通过，最新目录为
`.task/workflow-editor-smoke/20260726-200049`。旅程真实创建 Schedule、绑定 Workflow、行内运行并验证
queued 状态，再重新打开验证引用。此前启动器超时经精确复现确认不是产品故障：命令点击后
50ms 内进入成功态，失败源于整段 smoke 共用 180 秒预算。测试现使用 5 分钟总保险，并给启动器阶段
单独设置 30 秒上限。

## Next

1. 统一 Run 取消在四个入口的可见行为。
2. 验证设备路径、窗口绑定和 Credential 更新后无需修改 Workflow Source 即可再次运行。
3. 将 Schedule 默认页面按 V4 首页规则减负，完整高级策略进入管理态。
