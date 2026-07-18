# Slice 22 — 类型与节点能力闭包

## Outcome

任何公开 durable 值都有最小可操作能力；类型不能只被节点产生却无法比较、观察、保存、转换或拆字段。

## Delivered

- Node Contract ConversionSpec：input/output ports、lossless/lossy/parser、total、cost、autoInsert；seal 校验 deterministic pure-data、端口、错误和自动插入安全。
- Integer 保持类型的 add/subtract/multiply/modulo/negate/absolute/minimum/maximum/clamp，超出 JSON safe integer 失败。
- Integer 可直接进入 Number 比较/运算；Number→Integer 按 trunc/floor/ceil/round 显式选择；String parsers 可失败。
- Log<T: Observable>、Equal<T: Equatable>、ListContains<T: Equatable>、State<T: Durable>。
- Data Type semantic 的 StructureSpec 显式声明稳定字段 ID、JSON key 与具体 TypeExpression；对象 schema 必须与字段契约完全一致。
- Point、Region、TemplateMatch、QRCode、ColorBlob、FileMetadata 结构字段已建模；Catalog 自动生成 6 个确定性 pure-data Break 节点并由 runtime 真实执行字段投影。
- Catalog 构建时生成并验证 Type × Capability matrix：producer/consumer、literal、durable/observable/equatable、numeric/ordered、structure/break。新增公开 TypeRef 缺能力或明确 waiver 时构建失败。
- InputClip 的非节点生产路径以窄 waiver 记录：它由录制/资源库创建和选择，不把 waiver 泛化为绕过机制。
- 转换、结构与能力元数据进入 Authoring Projection，AI 固定评测和生成节点文档同步更新。

## Acceptance evidence

- Repeat.index 可比较、保持 Integer 计算、Log 观察和写 Integer state。
- List<Integer> 保持名义元素类型；集合 constraint、转换 contract/runtime/projection 来自同一 sealed Catalog。
- 六类结构化领域结果的所有公开字段均可通过精确类型 Break 输出消费。
- 删除任一必要 Break 或全部数值消费节点会使 closure 测试失败。
- 2026-07-18 完整 task check 通过：157 frontend tests、Go 全量与覆盖率、vet/staticcheck、契约、AI eval、生产构建和 bundle budget 全绿。
