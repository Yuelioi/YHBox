---
kind: note
summary: "后台资源必须由 internal/appruntime 单一 owner 在全部构造完成后顺序 Start，失败逆序 rollback，退出逆序 Close；当前顺序为 Worker → debug manager → hotkey → MCP HTTP → ScheduleDaemon → recording → calibration → tools，关闭反向执行后才有界 drain application/log。"
activation: action
read_when: "新增后台 goroutine、HTTP server、cron、hotkey/hook、worker；修改 main 启动/退出；实现 Start/Close/Shutdown；排查端口占用、退出卡住、held input 或 goroutine 泄漏"
recheck_when: "application runtime 新增资源；调整 shutdown timeout；资源 Close 不再遵守 context；Wails lifecycle API 改变"
---
# Application runtime 生命周期
生命周期契约：

- 构造函数只组装依赖，不启动 goroutine/listener；所有资源准备完成后统一 `Runtime.Start(ctx)`。
- Start 按声明顺序执行；配置先全量验证。任一 Start 失败时，用独立的有界 rollback context 逆序关闭已启动资源，启动失败与 rollback error 用 `errors.Join` 同时返回。
- Close 逆序尝试所有已启动资源，一个失败不能跳过其余；并发/重复 Close 观察同一最终结果。等待另一个 Start/Close 状态转换时必须响应调用方 context。
- Worker 的每个 run 从 worker lifetime context 派生。Stop 先取消 lifetime，再关 queue/stop channel；这样 queued→active 的过渡窗口也不可能产生未取消的新 run。
- HTTP server 必须先同步 `net.Listen`，端口占用直接使 Start 失败。请求使用独立 server lifetime，不能继承短期 Start deadline；Close cancel lifetime，再 graceful shutdown，超时则 force close。`Done` 是稳定广播，错误从 `Err` 读取。
- Schedule Stop 先拒绝新 fire，再注销全部 hotkey、移除 cron，并聚合 cleanup error；Reload 在未 Start 或已 Stop 时不动作。
- hotkey/recording/calibration/tools 是按需启动 native 资源的 service，因此 runtime 的 Start 仅声明 ownership，Close 才执行真正的 shutdown。关闭后必须永久拒绝新 binding、hook、录制和窗口。
- tools 先取消临时捕获热键，同时关闭现有窗口并让 in-flight open 观察 generation cancellation；两条清理路径不能互相阻塞。校准 HUD 的兜底回调先卸 LL hook/停 raw input，随后 calibration Shutdown 再做幂等确认。
- recording 退出使用 Cancel，不把未完成事件持久化成 clip/subgraph；若 shutdown 在 native Stop drain 期间到达，Stop 必须在持久化前丢弃结果。hotkey 的部分注销失败保留 binding ownership，Shutdown 必须重试所有残留 binding 并报告最终错误。
- debug session 不属于 Worker，必须作为独立 resource 关闭：拒绝新 session，cancel starting/active step，等待 `StopRuntime` 完成后才算释放 held input/capture。所有外部 cleanup 均在 manager mutex 外执行，多个 Close 各自遵守 context 并观察同一 barrier。
- Wails Run 无论成功或失败，都先关闭 application runtime，再以独立 deadline 调 `App.ShutdownContext`。App 同步 detach presentation/停止 node-enter timer，LogMerger 不再向已退出 GUI emit；LogSink 先关闭文件再等待旧 delivery，callback 卡住时 caller 可超时且文件句柄已释放。

声明顺序决定依赖，实际关闭顺序为 tools → calibration → recording → ScheduleDaemon → MCP HTTP → hotkey → debug manager → Worker。Schedule 注销时 registry 仍然存活；hotkey 先拒绝新触发，debug/Worker 随后释放各自 runtime。`wailsToolsPresenter` 在 tools 清理后 detach；App/LogSink 由 executable application owner 在 runtime 全部关闭后统一有界 drain。
