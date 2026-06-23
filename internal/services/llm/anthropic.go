package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const anthropicDefaultMaxTokens = 1024

type anthropicProvider struct {
	client anthropic.Client
	hc     *http.Client
}

func newAnthropicProvider(c ConnectionConfig) Provider {
	hc := tunedHTTPClient()
	opts := []option.RequestOption{
		option.WithHTTPClient(hc),
		option.WithAPIKey(apiKeyOrPlaceholder(c.APIKey)),
	}
	if c.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(c.BaseURL))
	}
	return &anthropicProvider{client: anthropic.NewClient(opts...), hc: hc}
}

// CloseIdleConnections 释放该 Provider 连接池的空闲连接。缓存重建旧 Provider 时调,防泄漏。
func (p *anthropicProvider) CloseIdleConnections() { p.hc.CloseIdleConnections() }

func (p *anthropicProvider) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	maxTok := int64(req.MaxTokens)
	if maxTok <= 0 {
		// Anthropic Messages API 要求 max_tokens 必填; 0 视为未设, 补一个通用默认。
		maxTok = anthropicDefaultMaxTokens
	}
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(req.Model),
		MaxTokens: maxTok,
		Messages:  toAnthropicMessages(req.Messages),
	}
	if sys := anthropicSystemBlocks(req.Messages); len(sys) > 0 {
		params.System = sys
	}
	if req.Temperature > 0 {
		params.Temperature = anthropic.Float(req.Temperature)
	}
	msg, err := p.client.Messages.New(ctx, params)
	if err != nil {
		return ChatResponse{}, mapAnthropicErr(err)
	}
	var b strings.Builder
	for _, block := range msg.Content {
		b.WriteString(block.Text)
	}
	return ChatResponse{Text: b.String()}, nil
}

func (p *anthropicProvider) ChatStructured(ctx context.Context, req ChatRequest, schema JSONSchema, mode string) (ChatResponse, map[string]any, error) {
	// Anthropic tool-use 总可用, auto 即 native。
	if mode == ModePrompt {
		return structuredViaPrompt(ctx, p.Chat, req, schema)
	}
	maxTok := int64(req.MaxTokens)
	if maxTok <= 0 {
		maxTok = anthropicDefaultMaxTokens
	}
	props, required := schemaProperties(schema)
	const toolName = "result"
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(req.Model),
		MaxTokens: maxTok,
		Messages:  toAnthropicMessages(req.Messages),
		Tools: []anthropic.ToolUnionParam{{OfTool: &anthropic.ToolParam{
			Name:        toolName,
			InputSchema: anthropic.ToolInputSchemaParam{Properties: props, Required: required},
		}}},
		ToolChoice: anthropic.ToolChoiceParamOfTool(toolName),
	}
	if sys := anthropicSystemBlocks(req.Messages); len(sys) > 0 {
		params.System = sys
	}
	if req.Temperature > 0 {
		params.Temperature = anthropic.Float(req.Temperature)
	}
	msg, err := p.client.Messages.New(ctx, params)
	if err != nil {
		return ChatResponse{}, nil, asUnsupportedIfBadRequest(mapAnthropicErr(err))
	}
	for _, block := range msg.Content {
		if block.Type == "tool_use" {
			var m map[string]any
			if err := json.Unmarshal(block.Input, &m); err != nil {
				return ChatResponse{}, nil, &CodedError{Kind: KindBadRequest, Err: err}
			}
			return ChatResponse{}, m, nil
		}
	}
	// 无 tool_use(模型返纯文本/异常)→ 原文带回, 节点走 Fail。
	var b strings.Builder
	for _, block := range msg.Content {
		b.WriteString(block.Text)
	}
	return ChatResponse{Text: b.String()}, nil, &CodedError{Kind: KindBadRequest, Err: errors.New("model did not return a tool_use block")}
}

func (p *anthropicProvider) ListModels(ctx context.Context) ([]string, error) {
	page, err := p.client.Models.List(ctx, anthropic.ModelListParams{})
	if err != nil {
		return nil, mapAnthropicErr(err)
	}
	ids := make([]string, 0, len(page.Data))
	for _, m := range page.Data {
		ids = append(ids, m.ID)
	}
	return ids, nil
}

// toAnthropicMessages 映射 user/assistant; system 走顶层 System 参数(Anthropic 无 system role)。
func toAnthropicMessages(msgs []Message) []anthropic.MessageParam {
	out := make([]anthropic.MessageParam, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case RoleSystem:
			continue
		case RoleAssistant:
			out = append(out, anthropic.NewAssistantMessage(anthropic.NewTextBlock(m.Content)))
		default:
			out = append(out, anthropic.NewUserMessage(anthropic.NewTextBlock(m.Content)))
		}
	}
	return out
}

func anthropicSystemBlocks(msgs []Message) []anthropic.TextBlockParam {
	var out []anthropic.TextBlockParam
	for _, m := range msgs {
		if m.Role == RoleSystem {
			out = append(out, anthropic.TextBlockParam{Text: m.Content})
		}
	}
	return out
}

func mapAnthropicErr(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return &CodedError{Kind: KindTimeout, Err: err}
	}
	var apierr *anthropic.Error
	if errors.As(err, &apierr) {
		return &CodedError{Kind: statusToKind(apierr.StatusCode), Err: err}
	}
	return &CodedError{Kind: KindNetwork, Err: err}
}
