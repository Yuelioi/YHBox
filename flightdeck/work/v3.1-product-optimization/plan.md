# 3.1 产品优化计划

## Outcome

在唯一 3.1 Source/compiler/runtime 上持续改善专业创作与运行体验，同时保持每个 Stage 都能由真实用户旅程、定向验证和阶段门禁闭环。

## Current stage

Stage A–I 已完成，当前没有已批准的开放 Stage。不要重新执行历史清单，也不要把等待中的想法当作当前任务。

## Starting the next stage

只有出现新的真机反馈或用户明确扩展产品范围时才创建下一 Stage：

1. 先复现一条具体用户旅程，记录当前行为、期望行为和可核验差异。
2. 核对 [context](context.md) 中的架构边界，确认修复不会恢复 3.0 Container、第二套 store 或第二套 runtime。
3. 把同一旅程上的相邻问题组成一个可交付 Stage，并在这里写明范围、非目标与验收门槛。
4. 实施中运行最小定向检查；Stage 完成后统一运行 `task check` 和被改动触发的真实宿主 smoke。
5. 验收后重写 `index.md` 的 Current、Next 与 Progress，让下一会话只看到仍然成立的状态。

## Stable constraints

- Selection、execution、debug 和 validation 保持独立状态。
- 复杂节点由 Authoring Projection 与类型级 Editor Adapter 承载，画布只显示高频摘要。
- Macro 与 InputClip 分轨；脏资源退出必须保留取消、放弃、保存并退出三路语义。
- 单对象短流程优先 Modal；长生命周期、多页面任务才使用独立路由。

## Historical evidence

- [Approved plan through Stage G](references/approved-plan-through-stage-g.md)
- [Stage H node menu and template flow](references/13-node-context-menu-and-template-flow.md)
- [Stage I node density](references/14-node-density-and-optional-pins.md)
- [Stage I resource editing](references/15-workflow-resource-edit-and-safe-exit.md)
- [Stage I Tab and Snippet flow](references/16-tab-menu-and-snippet-shortcuts.md)
- [Stage I schedule modal flow](references/17-schedule-modal-flow.md)
