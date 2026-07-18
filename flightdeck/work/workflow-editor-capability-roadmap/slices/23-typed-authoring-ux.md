# Slice 23 — Typed Authoring 体验

## Outcome / Question

强类型是否主动帮助用户完成图，而不是只拒绝错误？

## Completion criterion

用户能理解类型、发现候选、显式选择转换、提升输出为 State、定位状态引用，并在不静默断线的前提下安全改型。

## Blocked by

Slice 22。

## Verification

Repeat.index、Number→Integer、String→Number、typed State、conversion bridge、Promote to State、update-state-variable、引用影响预览和 Application 原子迁移门禁的 Go/TS 测试；2026-07-18 完整 `task check` 通过。

## Out of scope

证明 List 协变安全；Projection/Compiler 固定 fixture parity 独立进入 Slice 25。

## Result

Completed。

- EditorSession 图级固定点实例专化，State slot 是声明权威。
- exact/assignable/generic-bind 和 direct/conversion/incompatible 计划；ConversionSpec 候选按风险/成本排序。
- 有损/parser 由用户选择后插入真实转换节点与两条边，整个桥接一次 Undo。
- durable 精确输出支持 Promote to State：创建同类型状态、插入 State Write 并连线，一次 Undo。
- 状态搜索、Read/Write 拖放、跨图引用计数与精确定位均已落地。
- 改型前模拟目标类型并重解相关图；每条会退化为 conversion/incompatible 的现有数据边都会被列出并阻止确认。
- 可直接兼容的引用改型允许原子提交；Application 用正式 Compiler 比较基线与候选诊断，新增任何错误都拒绝落盘。
- PreparedPatch 保存同一安全证明，不能通过“预览后提交”路径绕过迁移门禁。
- 状态与连接候选列表有分段结果预算。
