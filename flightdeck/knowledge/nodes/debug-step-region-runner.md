# Debug step and host-lowered regions

3.1 不再有 RegionRunner/第二 debug runtime。Repeat、ForEach、Retry 和 GraphCall 被 Compiler lower 为 Program instruction/展开图，但其中的可见节点仍由同一个 Executor scheduler 调用。

DebugController 必须在 scheduler 即将执行每个可见节点时检查 breakpoint/pause/step；嵌套 activation 不能整块调用普通执行器绕过控制点。StepOnce 的完成单位是下一个可见节点边界，不是“完成整个 loop body/GraphCall”。

测试至少覆盖：循环多次仍逐节点暂停、嵌套 region、error→Retry、同一 subgraph 多调用点 provenance、stop/cancel 和断点命中。Debug Run 继续使用正常 Program、Admission、Grant、journal、resource lease 和 cleanup。
