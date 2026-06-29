package services

import (
	"context"
	"sort"
	"time"

	"yotta/internal/services/llm"
)

// AIService 暴露给前端的 AI 连接相关 RPC。无状态: 只对传入的连接做测试, 不持久化、不读 settings。
type AIService struct{}

func NewAIService() *AIService { return &AIService{} }

type TestConnReq struct {
	Connection AIConnection `json:"connection"`
	TestModel  string       `json:"testModel"`
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
	p, err := llm.New(llm.ConnectionConfig{Protocol: c.Protocol, BaseURL: c.BaseURL, APIKey: c.APIKey})
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
