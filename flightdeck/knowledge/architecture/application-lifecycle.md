# Application runtime 生命周期
生命周期契约：

- Windows production manifest 固定 `requireAdministrator`；UAC 在进程创建前决定完整性级别，代码不再维护 `asInvoker`、按需 `runas`、双运行级别或静默降级。这个产品决定扩大了主进程受损后的影响面，因此脚本/插件仍必须隔离，workflow effect 仍必须经过 capability/admission，前端和解析边界不能因主进程已管理员而获得 ambient authority。
- 构造函数只组装依赖，不启动 goroutine/listener；所有资源准备完成后统一 `Runtime.Start(ctx)`。
- Start 按声明顺序执行；配置先全量验证。任一 Start 失败时，用独立的有界 rollback context 逆序关闭已启动资源，启动失败与 rollback error 用 `errors.Join` 同时返回。
- Close 逆序尝试所有已启动资源，一个失败不能跳过其余；并发/重复 Close 观察同一最终结果。等待另一个 Start/Close 状态转换时必须响应调用方 context。
- Worker 的每个 run 从 worker lifetime context 派生。Stop 先取消 lifetime，再关 queue/stop channel；这样 queued→active 的过渡窗口也不可能产生未取消的新 run。
- HTTP server 必须先同步 `net.Listen`，端口占用直接使 Start 失败。请求使用独立 server lifetime，不能继承短期 Start deadline；Close cancel lifetime，再 graceful shutdown，超时则 force close。`Done` 是稳定广播，错误从 `Err` 读取。
- Schedule Stop 先拒绝新 fire，再注销全部 hotkey、移除 cron，并聚合 cleanup error；Reload 在未 Start 或已 Stop 时不动作。
- hotkey/recording/calibration/tools 是按需启动 native 资源的 service，因此 runtime 的 Start 仅声明 ownership，Close 才执行真正的 shutdown。关闭后必须永久拒绝新 binding、hook、录制和窗口。
- tools 先取消临时捕获热键，同时关闭现有窗口并让 in-flight open 观察 generation cancellation；两条清理路径不能互相阻塞。校准 HUD 的兜底回调先卸 LL hook/停 raw input，随后 calibration Shutdown 再做幂等确认。
- recording 退出使用 Cancel，不把未完成事件持久化成 clip/subgraph；若 shutdown 在 native Stop drain 期间到达，Stop 必须在持久化前丢弃结果。hotkey 的部分注销失败保留 binding ownership，Shutdown 必须重试所有残留 binding 并报告最终错误。
- debug session 使用同一 Application/Executor，但 session control 与 Run worker lifecycle 仍需独立关闭：拒绝新 session，取消 active Run，等待资源 lease/held input/capture 释放。所有外部 cleanup 均在 manager mutex 外执行，多个 Close 各自遵守 context 并观察同一 barrier。
- Wails Run 无论成功或失败，都先关闭 application runtime，再以独立 deadline 调 `App.ShutdownContext`。App 同步 detach presentation/停止 node-enter timer，LogMerger 不再向已退出 GUI emit；LogSink 先关闭文件再等待旧 delivery，callback 卡住时 caller 可超时且文件句柄已释放。

资源清单和顺序以 `internal/desktopapp/desktop.go` 的 `appruntime.Resource` 组装为准，不在 Knowledge 固定易漂移的完整列表。当前约束是依赖者先关闭：tools/calibration/recording/schedule 在 hotkey registry 与 Application worker 之前释放，presentation detach 后才有界 drain App/LogSink。
