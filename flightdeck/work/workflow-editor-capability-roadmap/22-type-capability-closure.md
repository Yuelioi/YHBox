# Slice 22 — 类型与节点能力闭包

## Outcome

任何公开 durable 值都有最小可操作能力；类型不能只被节点产生却无法比较、观察、保存、转换或拆字段。

## Delivered

- Node Contract ConversionSpec：input/output ports、lossless/lossy/parser、total、cost、autoInsert；seal 校验 deterministic pure-data、端口、错误和自动插入安全。
- Integer 保持类型的 add/subtract/multiply/modulo/negate/absolute/minimum/maximum/clamp，超出 JSON safe integer 失败。
- Integer 可直接进入 Number 比较/运算；Number→Integer 按 trunc/floor/ceil/round 显式选择；String parsers 可失败。
- Log<T: Observable>、Equal<T: Equatable>、ListContains<T: Equatable>、State<T: Durable>。
- 转换元数据进入 Authoring Projection，AI 固定评测和生成节点文档已更新。

## Remaining

1. 在 Data Type semantic 显式声明结构字段 TypeExpression 与稳定字段 ID；JSON Schema 继续只负责值校验。
2. 从 Catalog 生成 Break/field projection 节点并接入 runtime/compiler；模板匹配、二维码、颜色块、Point/Region、文件元数据字段必须可消费。
3. 生成 Type × Capability matrix；每个公开类型必须满足适用的 literal/state/observe/equality/operation/collection/conversion/serialization/debug 或提交有理由 waiver。
4. 补齐 List<Integer> 的完整 journey 与 conversion contract/runtime/projection 自动一致性矩阵。

## Acceptance

Repeat.index 可比较、保持 Integer 计算、观察和写 Integer state；结构化领域结果所有公开字段可消费；新增 TypeRef 缺能力或 waiver 时门禁失败。
