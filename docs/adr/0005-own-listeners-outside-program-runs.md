# ADR 0005: Own listeners outside Program Runs

Status: accepted

Yotta 3.1 把 cron、hotkey、once、manual 和外部 listener 视为 Run trigger。lifecycle-owned trigger adapter 在每次事件到达时通过同一 Application admission seam 提交一个独立 Run；Program 从 host-lowered `RunStarted` instruction 开始。

不在 Program 内提供长寿命 ambient listener 节点，也不共享 listener 子流程的 frame、queue、Run state、Grant 或 capability session。这使每个事件都有自己的 Program hash、admission、取消、journal 与 terminal status，并让 trigger 生命周期由 `internal/appruntime` 显式关闭。

旧 Container `EventTick` 在一个运行中 spawn 子图并共享 container-global state，该语义不进入 3.1 Catalog，也不保留兼容执行路径。
