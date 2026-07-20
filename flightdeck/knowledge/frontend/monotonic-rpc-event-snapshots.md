# RPC 返回不能覆盖更新的事件快照

Wails 命令常同时产生两条到达前端的路径：RPC promise 返回一个命令时快照，后端事件推送后续状态。二者没有固定到达顺序。后端可能已经完成动作并推送 generation 6 的 completed，随后 RPC 才返回 generation 5 的 running；若 RPC 路径直接赋值，UI 会永久回退且不会再收到纠正事件。

同一 session 内所有快照入口必须复用同一 monotonic merge 规则：

- 先确认 run/session identity 仍匹配。
- generation 小于当前值时拒绝覆盖；相等或更大才接收。
- RPC 命令返回当前被接受的快照，而不是强制返回 transport 的旧响应。
- event handler、control RPC、breakpoint RPC 和 refresh 路径不得各自维护不同合并逻辑。

回归测试要确定性制造乱序：让 control RPC 返回一个 deferred promise，先注入更高 generation 的 completed event，再 resolve 较低 generation 的 running 响应；最终 session 必须保持 completed。只跑真实异步 smoke 会让这个竞态时好时坏。

## 事件已到不等于控制请求已结算

即使 monotonic snapshot 已正确合并，事件也可能先于发起动作的 RPC promise 完成。典型例子是 Debug Step：页面已经收到下一个 paused snapshot，但按钮仍因前一条控制请求处于 busy/disabled。自动化验收若只等待 paused/node 改变就点击下一次 Step，浏览器会丢弃对 disabled button 的点击，随后表现为“调试器卡死”。

- 交互门禁必须同时等待领域状态推进和对应控制按钮恢复 enabled。
- 通用点击 helper 遇到 disabled 必须立即报错，不能静默调用 `.click()`。
- 组件是否允许 pipeline/queue 属于显式产品契约；没有该契约时，测试不得用事件到达推断命令已可再次提交。
