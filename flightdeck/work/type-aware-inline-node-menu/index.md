# Type-aware Inline Node Menu Plan

## Goal

保存按端口类型过滤候选节点并自动连线的历史实现方案。

## Status

Finished

## Current

该目录只有旧实现计划，没有完成证据。当前 3.1 已有 Catalog Projection、精确 Data Type、转换与
Tab 快速添加能力；未来若需要 inline candidate menu，应基于当前接口重新审计，不能直接执行旧计划。

## Next

None.

## Progress

- 完整保留类型感知候选和自动连线的旧方案。
- 明确计划没有证明对应实现已经交付。
- 将其保留为普通 Reference，而不是活动产品状态。

## References

- [Preserved plan](references/historical-implementation-plan.md) — 历史候选过滤与自动连线设计。
- [Typed authoring contract](../../knowledge/nodes/typed-authoring-contract.md) — 当前类型化创作边界。
- [Node data flow](../../knowledge/nodes/node-data-flow.md) — 当前连线语义。
