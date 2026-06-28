# Index — mcp-node-exec

## State

③ MCP 对外暴露 topic，design.md 与 plan.md 已就绪但尚未开工。目标是在 GUI 进程内置 Streamable HTTP MCP server，让外部 AI 可以 list_nodes / find_window / run_node / save_container，闭合“AI 跑节点探测 -> 生成容器”的循环。

## Next

等 window-control 与 detect-click-config 真机 smoke 并归档后，按 plan.md 从 Task 1 开始实现。执行计划要求逐 task 做，并在每个 task 末尾提交。

## Read now

- design.md
- plan.md

## Read if

- knowledge/build/build.md — 开始 build / test / smoke 前。
- knowledge/wails/add-service.md — 如果实现需要新增或调整前端可调 Go service。
- knowledge/nodes/node-system-architecture.md — 如果需要确认节点注册、capability 或 dispatch 机制。
- knowledge/nodes/held-exec-outputs.md — 如果实现 run_node 输出收割。
- knowledge/nodes/ai-nodes.md — 如果需要对齐已完成的 AI 节点系统语义。

## Progress

Done:
- 需求定位、design、implementation plan。

Current:
- 未实现，排在 window-control 与 detect-click-config smoke / 归档之后。

## Open questions

- 计划内写着提交分支 `feat/v2-foundation`，但当前 repo 分支是 `migrate/flightdeck-new-form`；开工前需要以当前分支现实为准重新确认。
