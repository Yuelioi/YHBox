# Slice 3：多选、对齐、分布与自动布局

## Outcome / Question

用户能快速整理中大型工作流，批量动作可预测、可撤销且不改变业务语义。

## Completion criterion

- 多选、框选、批量移动/Delete、复制/剪切/粘贴/复制节点。
- 两节点可六向对齐，三节点可水平/垂直等距，使用实测宽高。
- ELK LR/TB 保持选择、视口和中心，异步结果带 graph/revision token。
- 吸附线不与位置同步冲突，Alt 临时反转。
- action registry 统一上下文工具条、右键和快捷键。
- 每个批量动作一个 history 条目。

## Blocked by

Slice 1。

## Verification

定向验证几何、clipboard id/edge、undo 和过期布局；Slice 1–3 后统一 task check、build、GUI smoke。

## Out of scope

subgraph、reroute、注释框和跨 graph 布局。

## Result

Planned。不复活旧 Container 虚拟节点协议。
