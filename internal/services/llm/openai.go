package llm

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"
)

type openaiProvider struct {
	client   openai.Client
	hc       *http.Client
	official bool // 官方端点(api.openai.com / 空 BaseURL): auto 模式走 native json_schema
}

func newOpenAIProvider(c ConnectionConfig) Provider {
	hc := tunedHTTPClient()
	opts := []option.RequestOption{
		option.WithHTTPClient(hc),
		option.WithAPIKey(apiKeyOrPlaceholder(c.APIKey)),
	}
	if c.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(c.BaseURL))
	}
	return &openaiProvider{client: openai.NewClient(opts...), hc: hc, official: isOfficialEndpoint(c.Protocol, c.BaseURL)}
}

// CloseIdleConnections 释放该 Provider 连接池的空闲连接。缓存重建旧 Provider 时调,防泄漏。
func (p *openaiProvider) CloseIdleConnections() { p.hc.CloseIdleConnections() }

func (p *openaiProvider) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(req.Model),
		Messages: toOpenAIMessages(req.Messages),
	}
	if req.MaxTokens > 0 {
		params.MaxTokens = openai.Int(int64(req.MaxTokens))
	}
	if req.Temperature > 0 {
		params.Temperature = openai.Float(req.Temperature)
	}
	resp, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return ChatResponse{}, mapOpenAIErr(err)
	}
	if len(resp.Choices) == 0 {
		return ChatResponse{}, &CodedError{Kind: KindUpstream, Err: errors.New("response had no choices")}
	}
	return ChatResponse{Text: resp.Choices[0].Message.Content}, nil
}

func (p *openaiProvider) ChatStructured(ctx context.Context, req ChatRequest, schema JSONSchema, mode string) (ChatResponse, map[string]any, error) {
	if mode == ModeAuto {
		mode = ModePrompt
		if p.official {
			mode = ModeNative
		}
	}
	if mode == ModePrompt {
		return structuredViaPrompt(ctx, p.Chat, req, schema)
	}
	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(req.Model),
		Messages: toOpenAIMessages(req.Messages),
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &shared.ResponseFormatJSONSchemaParam{
				JSONSchema: shared.ResponseFormatJSONSchemaJSONSchemaParam{
					Name:   "result",
					Strict: openai.Bool(true),
					Schema: openaiSchemaObject(schema),
				},
			},
		},
	}
	if req.MaxTokens > 0 {
		params.MaxTokens = openai.Int(int64(req.MaxTokens))
	}
	if req.Temperature > 0 {
		params.Temperature = openai.Float(req.Temperature)
	}
	resp, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return ChatResponse{}, nil, asUnsupportedIfBadRequest(mapOpenAIErr(err))
	}
	if len(resp.Choices) == 0 {
		return ChatResponse{}, nil, &CodedError{Kind: KindUpstream, Err: errors.New("response had no choices")}
	}
	text := resp.Choices[0].Message.Content
	fields, perr := parseStructuredText(text)
	if perr != nil {
		return ChatResponse{Text: text}, nil, &CodedError{Kind: KindBadRequest, Err: perr}
	}
	return ChatResponse{Text: text}, fields, nil
}

func (p *openaiProvider) ListModels(ctx context.Context) ([]string, error) {
	page, err := p.client.Models.List(ctx)
	if err != nil {
		return nil, mapOpenAIErr(err)
	}
	ids := make([]string, 0, len(page.Data))
	for _, m := range page.Data {
		ids = append(ids, m.ID)
	}
	return ids, nil
}

func toOpenAIMessages(msgs []Message) []openai.ChatCompletionMessageParamUnion {
	out := make([]openai.ChatCompletionMessageParamUnion, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case RoleSystem:
			out = append(out, openai.SystemMessage(m.Content))
		case RoleAssistant:
			out = append(out, openai.AssistantMessage(m.Content))
		default:
			if len(m.Images) == 0 {
				out = append(out, openai.UserMessage(m.Content))
				continue
			}
			parts := []openai.ChatCompletionContentPartUnionParam{openai.TextContentPart(m.Content)}
			for _, img := range m.Images {
				parts = append(parts, openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
					URL: dataURI(img),
				}))
			}
			out = append(out, openai.UserMessage(parts))
		}
	}
	return out
}

// dataURI 把图像编成 data:image/<fmt>;base64,<...> 供 OpenAI image_url。
func dataURI(img ImageData) string {
	return "data:image/" + img.Format + ";base64," + base64.StdEncoding.EncodeToString(img.Data)
}

func mapOpenAIErr(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return &CodedError{Kind: KindTimeout, Err: err}
	}
	var apierr *openai.Error
	if errors.As(err, &apierr) {
		return &CodedError{Kind: statusToKind(apierr.StatusCode), Err: err}
	}
	return &CodedError{Kind: KindNetwork, Err: err}
}

// apiKeyOrPlaceholder: 本地代理常不校验 key, 给非空占位免 SDK 空 key 报错。
func apiKeyOrPlaceholder(key string) string {
	if key != "" {
		return key
	}
	return "noauth"
}
