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

// Provider 协议无关的模型调用面, 绑定一个连接(endpoint+key)。
type Provider interface {
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
	ListModels(ctx context.Context) ([]string, error)
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
