# LogSink 序列号必须按 FIFO 实际交付
`seq.Add(1)` 只保证编号生成顺序，不保证 goroutine 调度顺序。旧实现每次 `flushLocked` 都启动一个 callback goroutine；第一个 callback 阻塞时，第二个可以先进入 Wails emit，前端会把它判为 gap。固定 `time.Sleep` 的测试还会把 timer 延迟、batch 合并和 callback 调度混为一谈。

当前约束：

- `flushLocked` 只复制 batch、分配 seq、入 FIFO queue；单一 delivery pump 在不持 `LogSink.mu` 时串行调用 emitter。
- `Flush` 只强制入队且不等待，因此 emit callback 写新日志或调用 `Flush` 不会自锁。
- 包内 `drain` barrier 等待调用时已排在队尾的 delivery 完成，只用于 App shutdown；它不导出，避免 emit callback 自调用后等待自身完成。`Close` 仍保持非等待可重入。
- 慢 callback 后面同 emitter 代际的 batch 会合并，最多保留 `4 * ringCapacity` 条最新日志；溢出数写入 `LogBatchEvent.Dropped`，前端累计并显示，避免无界内存和静默丢弃。
- presentation transport 统一为 `log:batch`；system/runtime/dump/action 都先归一化成 `LogEntry`，前端每批只做一次浅数组提交。
- 顺序回归测试必须阻塞 seq=1 callback 并证明 seq=2 不可越过；不要用“睡 120ms 期望正好三个 debounce batch”验证单调性。
