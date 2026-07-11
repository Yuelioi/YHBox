# Application runtime 生命周期
SUMMARY: 后台资源必须由 internal/appruntime 单一 owner 在全部构造完成后顺序 Start，失败逆序 rollback，退出逆序 Close；当前首批顺序为 Worker → MCP HTTP → ScheduleDaemon，关闭反向执行后才 drain presentation/log。
READ WHEN: 新增后台 goroutine、HTTP server、cron、hotkey/hook、worker；修改 main 启动/退出；实现 Start/Close/Shutdown；排查端口占用、退出卡住、held input 或 goroutine 泄漏
RECHECK WHEN: application runtime 新增资源；调整 shutdown timeout；资源 Close 不再遵守 context；Wails lifecycle API 改变

---

生命周期契约：

- 构造函数只组装依赖，不启动 goroutine/listener；所有资源准备完成后统一 `Runtime.Start(ctx)`。
- Start 按声明顺序执行；配置先全量验证。任一 Start 失败时，用独立的有界 rollback context 逆序关闭已启动资源，启动失败与 rollback error 用 `errors.Join` 同时返回。
- Close 逆序尝试所有已启动资源，一个失败不能跳过其余；并发/重复 Close 观察同一最终结果。等待另一个 Start/Close 状态转换时必须响应调用方 context。
- Worker 的每个 run 从 worker lifetime context 派生。Stop 先取消 lifetime，再关 queue/stop channel；这样 queued→active 的过渡窗口也不可能产生未取消的新 run。
- HTTP server 必须先同步 `net.Listen`，端口占用直接使 Start 失败。请求使用独立 server lifetime，不能继承短期 Start deadline；Close cancel lifetime，再 graceful shutdown，超时则 force close。`Done` 是稳定广播，错误从 `Err` 读取。
- Schedule Stop 先拒绝新 fire，再注销全部 hotkey、移除 cron，并聚合 cleanup error；Reload 在未 Start 或已 Stop 时不动作。
- Wails Run 无论成功或失败，都先关闭 application runtime，再调用 `App.Shutdown` 排空 presentation/log，最后才返回/退出进程。

当前 owner 只覆盖 Worker、MCP HTTP、ScheduleDaemon。hotkey manager/registry、recording、calibration 与 tools secondary windows 仍须在后续批次纳入；在此之前不能宣称批次 E 完成。
