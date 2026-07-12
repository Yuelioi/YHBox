# ⚠ Normalize 会掩盖 MCP schema 与示例的 contract 漂移
SUMMARY: MCP `get_graph_schema` 曾把真实的 `graph.schemaVersion` 写成 `graph.version`；示例测试在校验前调用 `Container.Normalize()` 自动补 version/id，导致错误的 LLM-facing contract 仍然绿。
READ WHEN: 修改 MCP graph schema、LLM 示例、container Normalize、严格 JSON decode 或声称“生成示例已通过 validator”时
RECHECK WHEN: Yotta v3 删除 Normalize self-heal、MCP 改用生成 JSON Schema 或 authoring tools 改版后

---

`internal/services/mcpserver/schema.go` 的说明和两个示例曾使用：

```json
{"graph":{"version":1}}
```

真实 `container.Graph` 字段是：

```json
{"graph":{"schemaVersion":1}}
```

Go `encoding/json` 会静默丢弃未知的 `version`，随后 `Container.Normalize()` 又把零值 `Graph.SchemaVersion` 补成当前版本。`TestSchemaExamples_AllValid` 在 validation 前也调用 Normalize，因此只能证明“错误输入被修好后可运行”，不能证明对外示例符合 contract。

以后所有 LLM/tool-facing contract tests 必须：

1. 用与生产严格入口相同的 decoder（unknown field 拒绝）；
2. 在 schema conformance 之前不 normalize/migrate/default；
3. 从唯一 schema 生成示例或至少做 JSON Schema validation；
4. 把 normalization/defaulting 作为独立行为测试，不与 contract 测试混在一起。
