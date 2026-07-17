---
kind: trap
summary: "同一异步状态同时由 RPC 返回与事件推送更新时，两个入口都必须按 monotonic generation 合并；否则旧 RPC 响应会覆盖先到的新事件。"
activation: symptom
read_when: "Wails RPC 与事件共同更新运行/调试状态；completed 偶发回退到 running；界面永久卡在运行中；实现带 generation 的前端 snapshot"
recheck_when: "删除或重定义 snapshot generation；改变 Workflow debug/run 事件与 RPC 协议；替换 EditorSession 状态所有权"
---
# RPC 返回不能覆盖更新的事件快照

Wails 命令常同时产生两条到达前端的路径：RPC promise 返回一个命令时快照，后端事件推送后续状态。二者没有固定到达顺序。后端可能已经完成动作并推送 generation 6 的 completed，随后 RPC 才返回 generation 5 的 running；若 RPC 路径直接赋值，UI 会永久回退且不会再收到纠正事件。

同一 session 内所有快照入口必须复用同一 monotonic merge 规则：

- 先确认 run/session identity 仍匹配。
- generation 小于当前值时拒绝覆盖；相等或更大才接收。
- RPC 命令返回当前被接受的快照，而不是强制返回 transport 的旧响应。
- event handler、control RPC、breakpoint RPC 和 refresh 路径不得各自维护不同合并逻辑。

回归测试要确定性制造乱序：让 control RPC 返回一个 deferred promise，先注入更高 generation 的 completed event，再 resolve 较低 generation 的 running 响应；最终 session 必须保持 completed。只跑真实异步 smoke 会让这个竞态时好时坏。
