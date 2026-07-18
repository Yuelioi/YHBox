---
kind: trap
summary: "历史 3.0 Container.Normalize 掩盖 MCP schema 漂移案例；3.1 只保留其‘严格验证前不可自愈’教训。"
activation: symptom
read_when: "仅在审查 3.0 MCP/Container 示例，或设计 3.1 schema fixture 是否被 normalize/default 掩盖时"
recheck_when: "Yotta v3 删除 Normalize self-heal、MCP 改用生成 JSON Schema 或 authoring tools 改版后"
---
# ⚠ Normalize 会掩盖 MCP schema 与示例的 contract 漂移

> 历史案例：具体 API/字段已删除。可复用结论是 fixture 必须先按 strict schema 验证，不能先 normalize/default 后再宣称 contract 正确。
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
