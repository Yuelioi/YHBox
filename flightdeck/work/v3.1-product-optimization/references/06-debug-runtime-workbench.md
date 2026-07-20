# Debug runtime workbench

## Outcome / Question

用真实 3.1 Program、Admission、worker、scheduler 和 DebugController 证明调试可用；若无法通过完整真机硬闸门，则从 3.1 产品界面移除调试、断点和单步入口，不再展示半成品。

## Completion criterion

- Start 停在首个即将执行的可见节点之前，并向前端确认 paused 状态，而不是显示 running 后永久等待。
- Step 每次只执行一个可见节点且 effect 恰好一次；Run 开始、普通 effect、失败路由、Repeat/ForEach/Retry 和 GraphCall 都经过同一 scheduler 控制点。
- Continue 能运行到终态；Pause 在下一个可见控制点生效；Stop 不遗留 paused Run、held input 或旧 session。
- 前端对 RPC 与事件快照按 run identity 和 monotonic generation 合并，按钮可用性由单一 starting/paused/stepping/running/terminal 状态机决定。
- 日志、时间线、调试作为底部工作区的独立页签；普通运行不强制展开调试器。
- 连续启动与单步不复用旧 runId、generation 或当前节点。
- 若上述关键条件任一无法在 production UAC WebView 中闭环，隐藏所有用户可见调试入口并清楚记录保留的内部范围。

## Blocked by

- Slice 10 先恢复稳定的节点选择、运行态表达和工作区交互，避免视觉问题掩盖状态机问题。

## Verification

- 建立最小三节点 fixture 的应用层调试 harness，确定性断言 Start→paused、Step→下一个 paused、Continue→terminal、Stop→terminal。
- 制造 RPC 旧快照晚于更高 generation 事件返回，验证前端不回退。
- 覆盖失败路由、循环/重试和 GraphCall 控制点。
- production UAC WebView 中重复执行 Start、Step、Continue、Stop，并检查节点副作用次数与底部面板状态。
- 恢复阶段末与 Slice 10 一起运行聚合测试、`task check`、production build 和真机 smoke。

## Out of scope

- 新建第二套 debug runtime 或恢复 3.0 Container debug manager。
- 改变 Program、Admission、Grant、journal 或 effect adapter 的安全边界。
- 在调试闭环前增加 watches、条件断点等扩展能力。

## Result

已完成并保留产品入口。根因是 checkpoint 把 controller 的 generation 用新 scheduler 快照 generation 0 覆盖，导致 Step 的 running generation 2 后下一 paused 回退为 1，前端按正确的 monotonic 规则拒绝该旧快照。checkpoint 现在先继承 controller generation 再递增。确定性 Go 测试断言首个 paused < step running < 下一个 paused；增强 WebView smoke 断言 Run 开始暂停、Step 到 Delay、再次 Step 完成、重启与 Stop。完整 `task check` 和 production build 通过。
