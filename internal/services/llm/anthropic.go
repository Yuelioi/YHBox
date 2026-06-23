package llm

import (
	"context"
	"errors"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const anthropicDefaultMaxTokens = 1024

type anthropicProvider struct {
	client anthropic.Client
}

func newAnthropicProvider(c ConnectionConfig) Provider {
	opts := []option.RequestOption{
		option.WithHTTPClient(tunedHTTPClient()),
		option.WithAPIKey(apiKeyOrPlaceholder(c.APIKey)),
	}
	if c.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(c.BaseURL))
	}
	return &anthropicProvider{client: anthropic.NewClient(opts...)}
}

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
