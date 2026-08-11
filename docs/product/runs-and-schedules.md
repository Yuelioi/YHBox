# Runs, debugging and schedules

## Run lifecycle

`PreviewRun` 是可选的无副作用检查：它只编译 Source 并返回冻结后的 capability requirements，不做
admission，也不创建 Run。真正启动 Run 时，Application 按下面的顺序完成持久化与入队：

```text
load Source → compile → Program persist → Configured Target snapshot → capability admission
                                                                          │
                                                                          ▼
                                                                durable queued Run
                                                                          │
                                                                          ▼
                                                              enqueue → running → terminal
```

Run 的状态集合由 `internal/run` 定义：`queued`、`running`、`succeeded`、`failed`、`cancelled`、`interrupted`。
启动时遗留的 running Run 会标记为 interrupted，未交付的 queued Run 会取消；Yotta 不透明重放可能已经产生
外部副作用的工作。

durable Run Record 封存 Program hash、Catalog hash、capability plan digest、Run Grant、policy generation、
principal、时间与 journal/value facts。Configured Target snapshot 本身不写进 Run Record；Application 的 queued
job 在进程内持有本次取得的 provider map 和 exact Target generation lease，直到该 Run 结束。修改 settings 只
影响之后取得环境的 Run；当前 Run 的 adapter 和资源 owner 会在成功、失败、取消和 teardown 时统一关闭。

## Timeline 与值

Run Ledger 保存 append-oriented timeline，当前事实分为：

- node attempt：started、succeeded、failed、cancelled 或 routed；
- adapter action：节点实际调用的外部操作及其 outcome/error code；
- node status：Contract 声明的 `progress`、`waiting` 或 `connection` 状态事件；
- durable produced value：成功 Run 的可持久 Value Envelope 与 graph/node/port/attempt provenance。

Stream、Resource session 和 HeldInput 是 Run-owned handle，不能成为 durable result。Timeline 可以分页读取并
导出 JSON；日志是补充诊断，不替代 Run Ledger。

## Debug Run

Debug 和普通 Run 共用 Compiler、Program、scheduler、adapter 与 journal。Debug controller 额外提供：

- 按 graph/node 或具体 subgraph call path 设置 breakpoint；
- pause、continue 和 step；
- 当前 graph path、node、attempt、queue、typed input/state/value 的 snapshot。

断点不会创建第二套模拟执行器。需要验证真实 Target 副作用时，Debug 仍会调用同一 adapter；停止调试要走
Run cancel，让 held input、worker 和 Target lease 正常释放。

## Schedule

Schedule 是本机触发配置，目前只以 Workflow 为 target。一个 Schedule 可以按顺序**提交启动**多个 Workflow，
并设置相邻 start 之间的间隔、dispatch timeout 和启动失败时 `stop` 或 `continue`。它不会等待前一个 Run 到达
terminal 后再启动下一个；`timeout`/`onError` 也不监控已经成功入队的 Run。

| Trigger | 精确行为 |
| --- | --- |
| manual | daemon 不注册自动触发；由 `ScheduleService.FireNow` 显式启动（当前 UI 的运行按钮使用它） |
| cron / daily | 每天在本地 `HH:MM` 触发 |
| cron / interval | 每 N 分钟触发 |
| hotkey | 注册全局热键后触发 |
| once | 每次 daemon 注册该 enabled Schedule 时触发一次；当前全量 `Reload` 也会重新注册并再次触发 |

触发器不会自行执行节点。Daemon 对每个 Workflow 调用和 GUI 相同的 Run command，因此仍经过 compile、
Program persist、Target snapshot、capability admission、durable Run、queue 和相同 runtime。Service 会把启动
结果分类为 readiness；Schedule 保存最近触发时间、queued/failed 状态和 readiness，便于区分“触发成功进入
队列”和“配置尚不可运行”。

`once` 没有 durable exactly-once 标记；它是当前 daemon 注册行为，应用启动和之后的全量 Reload 都可能触发。
Hotkey 是否可注册还受当前 host 和组合键冲突影响。

## 运行入口

- GUI：Workflow 编辑器中的 Run/Debug，以及 Schedule 页面和 floating launcher。
- global hotkey：Schedule daemon 或 launcher/系统 hotkey 注册表。
- CLI：`Yotta.CLI.exe run <workflow-id>`，使用与桌面相同的 local runtime；详见[CLI reference](../reference/cli.md)。
- MCP/AI authoring：MCP 当前提供创作、compile 和 run preview，AI 生成可审阅 proposal；两者都不暴露第二套
  执行器，实际 Run 仍只能通过 Application 的正式 command 创建。

Run/runtime 的模块所有权见[Workflow runtime](../architecture/runtime.md)。
