# Slice 22 — 类型与节点能力闭包

## Outcome / Question

每个公开类型是否都有适用的创建、消费、观察、比较、状态、转换和结构字段能力，或理由明确的窄 waiver？

## Completion criterion

Catalog 构建时验证 Type × Capability matrix；结构化对象有显式字段契约和可执行 Break 节点；新增 TypeRef 缺能力时门禁失败。

## Blocked by

Slice 21。

## Verification

Repeat.index、List<Integer>、6 类结构值字段旅程、删除 Break/数值消费者的负向测试，以及 2026-07-18 完整 `task check`。

## Out of scope

转换选择、Promote to State、状态改型和大列表交互，由 Slice 23 承担。

## Result

Completed。

- ConversionSpec 固定 input/output、lossless/lossy/parser、total、cost 与 autoInsert。
- Integer 有保持类型的运算族；Number→Integer 与 String parser 均为显式策略。
- Log<T:Observable>、Equal<T:Equatable>、ListContains<T:Equatable>、State<T:Durable> 执行真实约束。
- StructureSpec 显式声明字段 ID、JSON key 与 TypeExpression；对象 schema 必须一致。
- Point、Region、TemplateMatch、QRCode、ColorBlob、FileMetadata 自动生成并执行 Break 节点。
- Type × Capability matrix 覆盖 producer/consumer、literal、traits、numeric/ordered、structure/break；InputClip 仅保留录制/资源库生产路径的窄 waiver。
