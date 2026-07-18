# Slice 23 — Typed Authoring 体验

## Outcome / Question

强类型是否主动帮助用户完成图，而不是只拒绝错误？

## Completion criterion

用户能理解类型、发现候选、显式选择转换、提升输出为 State、定位状态引用并安全改型；Projection 解释与 Compiler 有固定 parity。

## Blocked by

Slice 22。

## Verification

Repeat.index、Number→Integer、String→Number、typed State、conversion bridge、Promote to State、update-state-variable 的 Go/TS 测试；最新完整 `task check` 含 160 frontend tests、65.5% Go coverage、Wails 167 models 和 production build。

## Out of scope

证明 List 协变安全；无法证明安全的引用状态改型不会自动断线或猜测迁移。

## Result

In progress。

已完成：

- EditorSession 图级固定点实例专化，State slot 是声明权威。
- exact/assignable/generic-bind 和 direct/conversion/incompatible 计划；ConversionSpec 候选按风险/成本排序。
- 有损/parser 由用户选择后插入真实转换节点与两条边，整个桥接一次 Undo。
- durable 精确输出支持 Promote to State：创建同类型状态、插入 State Write 并连线，一次 Undo。
- 状态搜索、Read/Write 拖放、引用计数、定位和无引用安全改型；UI 与后端共同阻止危险改型。
- 状态与连接候选列表有分段结果预算。

剩余：

- 引用状态改型的跨图影响预览和显式迁移。
- Projection 连接计划与 Compiler 的跨语言固定 fixture parity。
