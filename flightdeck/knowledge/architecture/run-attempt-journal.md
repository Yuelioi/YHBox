# Run attempt journal 是 RunRecord 本身，不是旁路日志

NodeAttempt 与 AdapterAction 进入同一个 immutable RunRecord generation，由每个 Run 唯一的 JournalWriter 做 previous-digest CAS。不要建立 sidecar event log、内存 fallback 或与旧 runtime dual-write；否则 terminal 与 effect 事实会失去原子顺序。

每次 node invocation 必须先追加 AttemptStarted，再由 adapter 对 Node Contract 的每个 declared effect 主动记录恰好一个真实 AdapterAction，最后追加 AttemptSucceeded、AttemptFailed 或 AttemptCancelled。宿主不得根据 effect 声明合成成功 action。adapter 返回值不能覆盖其自报的 failed/cancelled；多 effect 同时出现 failed 与 cancelled 时确定性以 failed 为优先，保证 append-only attempt 永远存在合法 terminal。

这些不变量必须同时在 Executor recorder 和 durable RunRecord validator 强制。只放在 Executor 会允许其他 JournalWriter 调用者构造矛盾事实；只在落盘后检查会把 active attempt 写成无法终止的状态。SUCCEEDED Run 要求每个已执行 node 的最新 attempt 为 succeeded，但允许早先 failed attempt 后重试成功。

journal projection 只允许稳定 code、UTC 时间和非负数值 counters，不保存 raw error、路径、prompt、secret 或任意文本。Workflow Source 与 Run graph/node attribution 使用受限的 128 字符稳定 ID；durable Run Value ID 使用包含 run/graph/node/port/attempt 的 domain-separated 固定长度 digest，不能直接拼接用户控制的 ID。

attempt terminal 与 Run terminal 写入使用 non-cancellable context：调用方取消仍影响业务执行，但不能阻断已经确定结果的 durable 收口。
