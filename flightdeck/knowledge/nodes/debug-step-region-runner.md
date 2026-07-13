---
kind: trap
summary: "Loop/ForEach/Subgraph 等 RegionRunner 在普通运行里会整块执行 body；调试 StepOnce 若直接复用会越过用户期望的下一步暂停点。"
activation: symptom
read_when: "改 DebugStep/StepOnce/Loop/ForEach/Subgraph 调试语义；排查“单步遇到循环直接跑完”“禁用控制节点后队列断掉”“强停停不掉调试 session”。"
---
# ⚠ 调试单步不能直接执行 RegionRunner 整块语义
调试单步的暂停单位是用户看到的节点，不是普通 runtime 的执行单元。

普通运行中 `Loop` 是 RegionRunner：一次执行 Loop 节点会调用 body 回调 N 次或 forever，直到整个 region 完成后才从 `Loop.Done` 继续。这对 `Run()` 正确，但对 `StepOnce()` 会让用户点击一次单步后直接跑完整个循环体。

调试路径需要单独展开 region 队列：

- `Loop` 单步应先进入 `Body`，把 body 下游 token 入队并带上 `LoopFrame`。
- body 内节点每次 `StepOnce` 后，如果同一 loop frame 的队列已耗尽，再推进下一轮或 `Loop.Done`。
- `Break` / `Continue` 在 loop frame 内不是顶层 sentinel leak，要消耗当前 frame 剩余 token 后跳到 Done 或下一轮。
- 禁用控制节点必须按真实 canonical pin 路由；`Loop.Done` 是大写，旧小写 fallback 只能作为兼容。
- 全局强停必须走统一 `ContainerService.StopAll()` / runner adapter，否则只会取消普通 worker，调试 session 仍然活着。
