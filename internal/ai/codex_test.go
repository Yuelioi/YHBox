package ai

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"
)

func TestCodexSubscriptionDynamicToolSmoke(t *testing.T) {
	if os.Getenv("YOTTA_CODEX_SMOKE") != "1" {
		t.Skip("set YOTTA_CODEX_SMOKE=1 to use the signed-in Codex subscription")
	}
	profile, err := SealModelProfile(ModelProfileDraft{
		Provider: ProviderCodexSubscription, Endpoint: "codex://subscription", Model: "gpt-5.6-luna",
		MaxOutputTokens: 128, Capabilities: ProfileCapabilities{StructuredOutput: true, ToolCalling: true, ParallelTools: true},
		Evaluation: EvaluationUnverified,
	})
	if err != nil {
		t.Fatal(err)
	}
	toolSet, err := SealToolSet(ToolSetDraft{ID: "yotta.ai.codex-smoke-tools", Version: "1.0.0", Owner: "test", Tools: []ToolManifestDraft{{
		Name: "echo", Description: "Echo one value for this required smoke test.", Authority: ToolAuthorityPure,
		InputSchema:  json.RawMessage(`{"type":"object","properties":{"value":{"type":"string"}},"required":["value"],"additionalProperties":false}`),
		OutputSchema: json.RawMessage(`{"type":"object","properties":{"value":{"type":"string"}},"required":["value"],"additionalProperties":false}`),
	}}})
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := SealPromptManifest(PromptManifestDraft{ID: "yotta.ai.codex-tool-smoke", Version: "1.0.0", Owner: "test", Instructions: "You must call the supplied echo tool exactly once, then report completion."})
	if err != nil {
		t.Fatal(err)
	}
	prompt, err := RenderPrompt(manifest, []PromptBlock{{Kind: PromptBlockUser, Content: "Call echo with value OK."}})
	if err != nil {
		t.Fatal(err)
	}
	provider, err := newCodexProvider(profile)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	maximum := int64(64)
	first, state, err := provider.StartAgent(ctx, "", AgentStartRequest{
		AttemptID: "codex-tool-start", Prompt: prompt,
		ToolSet: ToolSetArtifact{Digest: toolSet.Digest(), Manifest: json.RawMessage(toolSet.Bytes())},
		Limits:  GenerationLimits{MaxOutputTokens: &maximum}, MaxParallelism: 1, Retention: RetentionNoApplicationState,
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Finish.Kind != FinishToolCalls || len(first.Items) != 1 || first.Items[0].ToolCall == nil {
		t.Fatalf("first outcome = %#v", first)
	}
	call := first.Items[0].ToolCall
	final, next, err := provider.ContinueAgent(ctx, "", state, AgentContinueRequest{
		AttemptID: "codex-tool-continue", Results: []ToolResult{{CallID: call.CallID, Name: call.Name, Value: json.RawMessage(`{"value":"OK"}`)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if next != nil || final.Finish.Kind != FinishCompleted {
		t.Fatalf("final = %#v, state = %#v", final, next)
	}
}

func TestCodexSubscriptionSmoke(t *testing.T) {
	if os.Getenv("YOTTA_CODEX_SMOKE") != "1" {
		t.Skip("set YOTTA_CODEX_SMOKE=1 to use the signed-in Codex subscription")
	}
	profile, err := SealModelProfile(ModelProfileDraft{
		Provider: ProviderCodexSubscription, Endpoint: "codex://subscription", Model: "gpt-5.6-luna",
		MaxOutputTokens: 64, Capabilities: ProfileCapabilities{StructuredOutput: true, ToolCalling: true, ParallelTools: true},
		Evaluation: EvaluationUnverified,
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := SealPromptManifest(PromptManifestDraft{
		ID: "yotta.ai.codex-smoke", Version: "1.0.0", Owner: "test", Instructions: "Reply briefly.",
	})
	if err != nil {
		t.Fatal(err)
	}
	prompt, err := RenderPrompt(manifest, []PromptBlock{{Kind: PromptBlockUser, Content: "Reply with OK."}})
	if err != nil {
		t.Fatal(err)
	}
	provider, err := newCodexProvider(profile)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	maximum := int64(16)
	outcome, err := provider.Generate(ctx, "", GenerateRequest{
		AttemptID: "codex-smoke", Prompt: prompt, Limits: GenerationLimits{MaxOutputTokens: &maximum}, Retention: RetentionNoApplicationState,
	})
	if err != nil {
		var failure *ProviderFailure
		if errors.As(err, &failure) {
			t.Fatalf("%v: %s", err, failure.Message)
		}
		t.Fatal(err)
	}
	if outcome.Provider != ProviderCodexSubscription || outcome.Finish.Kind != FinishCompleted || len(outcome.Items) != 1 {
		t.Fatalf("outcome = %#v", outcome)
	}
}
