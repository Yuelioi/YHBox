package llm

import (
	"context"
	"fmt"
	"strings"
)

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type Message struct {
	Role    Role
	Content string
}

type ChatRequest struct {
	Model       string
	Messages    []Message
	Temperature float64
	MaxTokens   int
}

type ChatResponse struct {
	Text string
}

// 结构化输出模式。AI 节点把 "auto" 按端点解析成下面之一再调 ChatStructured。
const (
	ModeNative = "native" // OpenAI json_schema / Anthropic 强制 tool-use
	ModePrompt = "prompt" // schema 描述注入 + 容错解析(兼容端点保底)
)

// SchemaField — 一个结构化输出字段(名 + JSON 类型)。
type SchemaField struct {
	Name     string
	JSONType string // string | number | integer | boolean | object | array
}

// JSONSchema — 由 config.Outputs[] 构造的扁平对象 schema。
type JSONSchema struct {
	Fields []SchemaField
}

// Provider 协议无关的模型调用面, 绑定一个连接(endpoint+key)。
type Provider interface {
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
	ListModels(ctx context.Context) ([]string, error)
	// ChatStructured 要模型按 schema 出 JSON。mode = ModeNative | ModePrompt。
	// 失败时 error≠nil 且返回的 ChatResponse.Text 仍填原文(供节点 Fail.Text 供下游消费)。
	ChatStructured(ctx context.Context, req ChatRequest, schema JSONSchema, mode string) (ChatResponse, map[string]any, error)
}

type ConnectionConfig struct {
	Protocol string
	BaseURL  string
	APIKey   string
}

// New 按协议建 Provider。每调用新建(含连接池 Transport), 不缓存。
// 官方端点缺 key 早报错, 免等到真调用才炸。
func New(c ConnectionConfig) (Provider, error) {
	if c.APIKey == "" && isOfficialEndpoint(c.Protocol, c.BaseURL) {
		return nil, ErrAPIKeyRequired
	}
	switch c.Protocol {
	case "openai":
		return newOpenAIProvider(c), nil
	case "anthropic":
		return newAnthropicProvider(c), nil
	default:
		return nil, &CodedError{Kind: KindBadRequest, Err: fmt.Errorf("unknown protocol %q", c.Protocol)}
	}
}

// isOfficialEndpoint 判断是否打官方付费端点(空 BaseURL = SDK 默认官方域名)。
// 官方端点空 key 应早报错; 本地代理(非官方域名)允许空 key。
func isOfficialEndpoint(protocol, baseURL string) bool {
	if baseURL == "" {
		return true
	}
	switch protocol {
	case "openai":
		return strings.Contains(baseURL, "api.openai.com")
	case "anthropic":
		return strings.Contains(baseURL, "api.anthropic.com")
	}
	return false
}
