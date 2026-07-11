package node

import (
	"context"
	"testing"
)

type fakeSubgraphCaller struct {
	gotID     string
	gotParams map[string]any
}

func (f *fakeSubgraphCaller) CallSubgraph(_ context.Context, sgID string, params map[string]any) (string, error) {
	f.gotID = sgID
	f.gotParams = params
	return "ok", nil
}

type subgraphCallingNode struct{}

func (subgraphCallingNode) Spec() Spec {
	return Spec{Kind: "SgCalling", Outputs: []OutputSpec{{Name: "Done", Type: "Exec"}}}
}

func (subgraphCallingNode) Run(ctx Ctx, _ Inputs) (Outputs, error) {
	exit, err := ctx.Services().Subgraphs.CallSubgraph(ctx.Context(), "sg-x", map[string]any{"k": 1.0})
	if err != nil {
		return nil, err
	}
	return ctx.Out("Done").Set("exit", exit).Fire(), nil
}

func TestCtx_Subgraphs_WiredFromBundle(t *testing.T) {
	registry := NewRegistry()
	registry.Register(subgraphCallingNode{})
	rn, _ := registry.Get("SgCalling")

	fake := &fakeSubgraphCaller{}
	services := StubServices()
	services.Subgraphs = fake

	r := RunNode(context.Background(), rn, nil, nil, nil, services, false)
	if r.Error != nil || r.Panic != nil {
		t.Fatalf("error/panic: %v / %v", r.Error, r.Panic)
	}
	if fake.gotID != "sg-x" {
		t.Errorf("CallSubgraph sgID = %q, want sg-x", fake.gotID)
	}
	if r.OutputData["exit"] != "ok" {
		t.Errorf("exit = %v, want ok", r.OutputData["exit"])
	}
}

func TestStubServices_SubgraphsNil(t *testing.T) {
	if StubServices().Subgraphs != nil {
		t.Error("StubServices().Subgraphs should be nil (非 runner 语境无子图调用能力)")
	}
}
