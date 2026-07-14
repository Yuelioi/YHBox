package services

import (
	"context"
	"sort"
	"time"

	"github.com/yottaapp/yotta/internal/services/llm"
)

// AIService exposes transient credential operations without returning stored
// secret values to the presentation layer.
type AIService struct {
	secrets *AISecrets
}

func NewAIService(secrets ...*AISecrets) *AIService {
	service := &AIService{}
	if len(secrets) > 0 {
		service.secrets = secrets[0]
	}
	return service
}

type TestConnReq struct {
	Connection AIConnection `json:"connection"`
	TestModel  string       `json:"testModel"`
	APIKey     string       `json:"apiKey"`
}

type TestResult struct {
	Ok     bool     `json:"ok"`
	Models []string `json:"models"`
	Error  string   `json:"error"`
	Kind   string   `json:"kind"`
}

// TestConnection 测试一个连接(可为未保存的表单值): 先拉模型列表验 endpoint+key,
// 拉不到再用可选 TestModel 发一次最小 chat 兜底。
func (s *AIService) TestConnection(req TestConnReq) TestResult {
	c := req.Connection
	apiKey := req.APIKey
	if apiKey == "" && c.ID != "" && s.secrets != nil {
		stored, err := s.secrets.Get(c.ID)
		if err == nil {
			apiKey = stored
		}
	}
	p, err := llm.New(llm.ConnectionConfig{Protocol: c.Protocol, BaseURL: c.BaseURL, APIKey: apiKey})
	if err != nil {
		return TestResult{Error: err.Error(), Kind: string(llm.KindOf(err))}
	}

	lmCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if models, lerr := p.ListModels(lmCtx); lerr == nil {
		sort.Strings(models)
		return TestResult{Ok: true, Models: models}
	}

	if req.TestModel == "" {
		return TestResult{Error: "无法列出模型;填一个测试模型名再试", Kind: string(llm.KindNotFound)}
	}
	chatCtx, cancel2 := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel2()
	if _, cerr := p.Chat(chatCtx, llm.ChatRequest{
		Model:     req.TestModel,
		MaxTokens: 1,
		Messages:  []llm.Message{{Role: llm.RoleUser, Content: "ping"}},
	}); cerr != nil {
		return TestResult{Error: cerr.Error(), Kind: string(llm.KindOf(cerr))}
	}
	return TestResult{Ok: true}
}

// SecretStatus returns only presence metadata, never credential values.
func (s *AIService) SecretStatus(connectionIDs []string) map[string]bool {
	status := make(map[string]bool, len(connectionIDs))
	for _, id := range connectionIDs {
		has, err := s.secrets.Has(id)
		status[id] = err == nil && has
	}
	return status
}

func (s *AIService) SetAPIKey(connectionID, apiKey string) error {
	return s.secrets.Set(connectionID, apiKey)
}

func (s *AIService) DeleteAPIKey(connectionID string) error {
	return s.secrets.Delete(connectionID)
}
