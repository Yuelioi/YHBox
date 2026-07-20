# Recording Asset Lifecycle

## Goal

完成录制资产从开始、停止、命名、入库、引用、预览到安全清理的完整生命周期。

## Status

Finished

## Current

实现已完成：录制停止不再生成内存 pending，只有 Finalize 才入库，Discard/Cancel 不产生资产；
录制、蓝图和模板支持未引用项预览、勾选清理与删除前引用复检。后续 3.1 Work 已继续验证录制与
资源旅程，因此旧桌面 smoke 待办不再保持活动。

## Next

None.

## Progress

- 明确 Start、Stop、Finalize、Discard 和 Cancel 的资产状态转换。
- 只有命名并 Finalize 的录制进入 durable store。
- 录制、蓝图和模板共享未引用项预览与安全清理流程。
- 删除前重新检查引用，避免根据陈旧投影破坏资源。
- 当时前端 529 项测试、typecheck、i18n、build 和完整 `task check` 通过。

## References

- [Lifecycle design](design.md) — 录制、入库与清理语义。
- [Asset subsystem](../../knowledge/subgraph/asset-subsystem.md) — 当前资源 owner。
- [Recording schema cascade](../../knowledge/nodes/recording-schema-cascade.md) — 录制契约变化边界。
