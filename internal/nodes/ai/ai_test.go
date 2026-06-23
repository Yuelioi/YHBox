package ai

import (
	"context"
	"errors"
	"testing"

	"yotta/internal/node"
	"yotta/internal/services/llm"
)

type fakeProvider struct {
	gotReq llm.ChatRequest
	resp   llm.ChatResponse
	err    error
}

func (f *fakeProvider) Chat(_ context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
	f.gotReq = req
	return f.resp, f.err
}
func (f *fakeProvider) ListModels(_ context.Context) ([]string, error) { return nil, nil }

type fakeAIService struct {
	provider llm.Provider
	err      error
}

func (f fakeAIService) Provider(string) (llm.Provider, error) { return f.provider, f.err }

func setupAI(t *testing.T, svc node.ServiceBundle, cfg map[string]any, dataWire map[string]any) node.RunResult {
	t.Helper()
	node.ResetRegistryForTest()
	node.Register(&AI{})
	rn, _ := node.Get("AI")
	return node.RunNode(context.Background(), rn, dataWire, cfg, nil, svc, false)
}

func TestAI_TextChat_RendersAndOutputs(t *testing.T) {
	fp := &fakeProvider{resp: llm.ChatResponse{Text: "hello world"}}
	svc := node.StubServices()
	svc.AI = fakeAIService{provider: fp}

	cfg := map[string]any{
		"User":   "hi {{name}}",
		"Inputs": []any{map[string]any{"Name": "name", "Type": "String"}},
	}
	res := setupAI(t, svc, cfg, map[string]any{"name": "bob"})

	if res.Error != nil {
		t.Fatalf("unexpected error: %v", res.Error)
	}
	if res.ExitName != "Done" {
		t.Fatalf("exit = %q, want Done", res.ExitName)
	}
	if res.OutputData[outText] != "hello world" {
		t.Errorf("Text = %v, want %q", res.OutputData[outText], "hello world")
	}
	last := fp.gotReq.Messages[len(fp.gotReq.Messages)-1]
	if last.Content != "hi bob" {
		t.Errorf("user content = %q, want %q", last.Content, "hi bob")
	}
}

func TestAI_ProviderError_FiresFail(t *testing.T) {
	svc := node.StubServices()
	svc.AI = fakeAIService{provider: &fakeProvider{
		err: &llm.CodedError{Kind: llm.KindAuth, Err: errors.New("bad key")},
	}}
	res := setupAI(t, svc, map[string]any{"User": "hi"}, nil)

	if res.ExitName != "Fail" {
		t.Fatalf("exit = %q (err=%v), want Fail", res.ExitName, res.Error)
	}
	if res.OutputData[outCode] != "auth" {
		t.Errorf("Code = %v, want auth", res.OutputData[outCode])
	}
}

func TestAI_ConnectionNotFound_FiresFail(t *testing.T) {
	svc := node.StubServices()
	svc.AI = fakeAIService{err: &llm.CodedError{Kind: llm.KindNotFound, Err: errors.New("no connection")}}
	res := setupAI(t, svc, map[string]any{"User": "hi"}, nil)

	if res.ExitName != "Fail" {
		t.Fatalf("exit = %q (err=%v), want Fail", res.ExitName, res.Error)
	}
	if res.OutputData[outCode] != "notFound" {
		t.Errorf("Code = %v, want notFound", res.OutputData[outCode])
	}
}

func TestAI_SystemMessagePrepended(t *testing.T) {
	fp := &fakeProvider{resp: llm.ChatResponse{Text: "ok"}}
	svc := node.StubServices()
	svc.AI = fakeAIService{provider: fp}

	res := setupAI(t, svc, map[string]any{"System": "be terse", "User": "go"}, nil)
	if res.Error != nil {
		t.Fatalf("err: %v", res.Error)
	}
	if len(fp.gotReq.Messages) != 2 || fp.gotReq.Messages[0].Role != llm.RoleSystem {
		t.Fatalf("expected [system,user], got %+v", fp.gotReq.Messages)
	}
}
