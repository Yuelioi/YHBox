package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// schemaProperties 把 JSONSchema 摊成 JSON-schema properties map + required 名单(全字段必填)。
func schemaProperties(schema JSONSchema) (props map[string]any, required []string) {
	props = make(map[string]any, len(schema.Fields))
	for _, f := range schema.Fields {
		props[f.Name] = map[string]any{"type": f.JSONType}
		required = append(required, f.Name)
	}
	return props, required
}

// openaiSchemaObject 构造 OpenAI strict json_schema 的完整 object schema
// (additionalProperties:false + 全字段 required, strict 模式硬要求)。
func openaiSchemaObject(schema JSONSchema) map[string]any {
	props, required := schemaProperties(schema)
	return map[string]any{
		"type":                 "object",
		"properties":           props,
		"required":             required,
		"additionalProperties": false,
	}
}

// structuredViaPrompt — ModePrompt 路径: schema 描述注入 system + 调普通 Chat + 容错解析。
// 兼容端点(不支持 native)保底。失败时 ChatResponse.Text 仍填原文供节点 Fail。
func structuredViaPrompt(
	ctx context.Context,
	chat func(context.Context, ChatRequest) (ChatResponse, error),
	req ChatRequest,
	schema JSONSchema,
) (ChatResponse, map[string]any, error) {
	aug := req
	aug.Messages = append([]Message{{Role: RoleSystem, Content: schemaInstruction(schema)}}, req.Messages...)
	resp, err := chat(ctx, aug)
	if err != nil {
		return ChatResponse{}, nil, err
	}
	fields, perr := parseStructuredText(resp.Text)
	if perr != nil {
		// 原文带回(resp.Text), 文案不泄注入的内部细节。
		return resp, nil, &CodedError{Kind: KindBadRequest, Err: perr}
	}
	return resp, fields, nil
}

// schemaInstruction 生成"只回 JSON"指令(注入 system)。
func schemaInstruction(schema JSONSchema) string {
	var b strings.Builder
	b.WriteString("Respond with ONLY a single JSON object, no prose and no markdown fences. Fields:")
	for _, f := range schema.Fields {
		fmt.Fprintf(&b, " %q (%s)", f.Name, f.JSONType)
	}
	b.WriteString(".")
	return b.String()
}

// parseStructuredText 容错解析: 剥 ``` 围栏 + 取首个完整 {...} + Unmarshal。
// 错误文案只说"未返合法 JSON", 不回灌注入的 system 内部。
func parseStructuredText(raw string) (map[string]any, error) {
	s := strings.TrimSpace(raw)
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), "```"))
	}
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start < 0 || end < start {
		return nil, errors.New("model did not return a JSON object")
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(s[start:end+1]), &m); err != nil {
		return nil, errors.New("model did not return valid JSON")
	}
	return m, nil
}

// asUnsupportedIfBadRequest — native 路径下 badRequest 多半是端点不支持该能力 → 改 KindUnsupported
// 并提示切提示词注入模式。其余错误原样透传。
func asUnsupportedIfBadRequest(err error) error {
	if KindOf(err) == KindBadRequest {
		return &CodedError{Kind: KindUnsupported, Err: fmt.Errorf("endpoint may not support native structured output, try prompt mode: %w", err)}
	}
	return err
}
