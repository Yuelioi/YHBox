package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/resource"
)

func TestBudgetTrackerEnforcesTokenCostTimeIterationCallAndParallelLimits(t *testing.T) {
	started := time.Date(2026, 7, 17, 3, 0, 0, 0, time.UTC)
	budget := RunBudget{
		MaxInputTokens: 100, MaxOutputTokens: 50, MaxCostMicrounits: 100,
		MaxWallTimeMillis: 1000, MaxIterations: 2, MaxToolCalls: 2, MaxParallelism: 1,
	}
	tracker, err := NewBudgetTracker(budget, started)
	if err != nil {
		t.Fatal(err)
	}
	if err := tracker.BeforeTurn(started); err != nil {
		t.Fatal(err)
	}
	input, cached, output, cost := int64(40), int64(10), int64(20), int64(71)
	outcome := Outcome{Usage: TokenUsage{InputTotal: &input, CacheRead: &cached, OutputTotal: &output, CostMicrounits: &cost}}
	if err := tracker.ConsumeTurn(started.Add(100*time.Millisecond), outcome, 1); err != nil {
		t.Fatal(err)
	}
	usage := tracker.Usage()
	if usage.CostMicrounits != 71 || usage.InputTokens != 40 || usage.OutputTokens != 20 || usage.ToolCalls != 1 || usage.MaxParallelism != 1 {
		t.Fatalf("budget usage = %#v", usage)
	}
	if err := tracker.BeforeTurn(started.Add(200 * time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if err := tracker.ConsumeTurn(started.Add(300*time.Millisecond), outcome, 1); !errors.Is(err, ErrAgentBudgetExceeded) {
		t.Fatalf("cost exhaustion = %v", err)
	}
	if err := tracker.BeforeTurn(started.Add(400 * time.Millisecond)); !errors.Is(err, ErrAgentBudgetExceeded) {
		t.Fatalf("iteration exhaustion = %v", err)
	}

	parallel, err := NewBudgetTracker(budget, started)
	if err != nil {
		t.Fatal(err)
	}
	if err := parallel.ConsumeTurn(started, outcome, 2); !errors.Is(err, ErrAgentBudgetExceeded) {
		t.Fatalf("parallel exhaustion = %v", err)
	}
	if err := parallel.BeforeTurn(started.Add(1001 * time.Millisecond)); !errors.Is(err, ErrAgentBudgetExceeded) {
		t.Fatalf("wall-time exhaustion = %v", err)
	}

	invalid := budget
	invalid.MaxWallTimeMillis = MaxAgentWallTimeMillis + 1
	if _, err := NewBudgetTracker(invalid, started); err == nil {
		t.Fatal("oversized wall-time budget accepted")
	}
	overflow, err := NewBudgetTracker(RunBudget{
		MaxInputTokens: MaxAgentInputTokens, MaxOutputTokens: MaxAgentOutputTokens, MaxCostMicrounits: MaxAgentCostMicrounits,
		MaxWallTimeMillis: 1000, MaxIterations: 1, MaxToolCalls: 1, MaxParallelism: 1,
	}, started)
	if err != nil {
		t.Fatal(err)
	}
	overflow.usage.InputTokens = int64(^uint64(0) >> 1)
	one, zero := int64(1), int64(0)
	if err := overflow.ConsumeTurn(started, Outcome{Usage: TokenUsage{InputTotal: &one, OutputTotal: &zero, CostMicrounits: &zero}}, 0); !errors.Is(err, ErrAgentBudgetExceeded) {
		t.Fatalf("counter overflow = %v", err)
	}
}

func TestOpenAIResponsesAgentReplaysNativeStateWhenStorageIsDisabled(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if requests == 1 {
			tools := body["tools"].([]any)
			tool := tools[0].(map[string]any)
			if body["instructions"] != "Trusted test instructions." || body["input"] != "run" || tool["name"] != "echo" || tool["strict"] != true {
				t.Fatalf("OpenAI agent start = %#v", body)
			}
			_, _ = writer.Write([]byte(`{"id":"resp_1","model":"model-snapshot-1","status":"completed","output":[{"type":"function_call","name":"echo","call_id":"call-1","arguments":"{\"value\":\"ok\"}"}],"usage":{"input_tokens":10,"output_tokens":5}}`))
			return
		}
		history := body["input"].([]any)
		output := history[2].(map[string]any)
		if _, stored := body["previous_response_id"]; stored || len(history) != 3 ||
			history[0].(map[string]any)["role"] != "user" || history[1].(map[string]any)["type"] != "function_call" ||
			output["type"] != "function_call_output" || output["call_id"] != "call-1" ||
			body["instructions"] != "Trusted test instructions." || len(body["include"].([]any)) != 1 {
			t.Fatalf("OpenAI agent continuation = %#v", body)
		}
		_, _ = writer.Write([]byte(`{"id":"resp_2","model":"model-snapshot-1","status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"done"}]}],"usage":{"input_tokens":12,"output_tokens":3}}`))
	}))
	defer server.Close()
	provider := nativeAgentProviderForTest(t, ProviderOpenAIResponses, server.URL)
	start := agentStartForTest(t)
	outcome, state, err := provider.StartAgent(context.Background(), "secret", start)
	if err != nil || outcome.Finish.Kind != FinishToolCalls || outcome.Usage.CostMicrounits == nil {
		t.Fatalf("OpenAI agent start outcome = %#v, %v", outcome, err)
	}
	previous := state.(*openAIAgentState)
	outcome, next, err := provider.ContinueAgent(context.Background(), "secret", state, AgentContinueRequest{
		AttemptID: "attempt-2", Results: []ToolResult{{CallID: "call-1", Name: "echo", Value: json.RawMessage(`{"value":"ok"}`)}},
	})
	continued := next.(*openAIAgentState)
	if err != nil || outcome.Finish.Kind != FinishCompleted || joinText(outcome.Items) != "done" || requests != 2 ||
		previous.previousResponseID != "resp_1" || len(previous.history) != 2 || continued.previousResponseID != "resp_2" || len(continued.history) != 4 {
		t.Fatalf("OpenAI agent completion = %#v, %v, requests=%d", outcome, err, requests)
	}
}

func TestOpenAIResponsesAgentUsesStoredContinuationWithTrustedContract(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["store"] != true || body["instructions"] != "Trusted test instructions." || len(body["tools"].([]any)) != 1 {
			t.Fatalf("OpenAI stored agent contract = %#v", body)
		}
		if requests == 1 {
			if _, found := body["previous_response_id"]; found {
				t.Fatalf("OpenAI stored start = %#v", body)
			}
			_, _ = writer.Write([]byte(`{"id":"resp_1","model":"model-snapshot-1","status":"completed","output":[{"type":"function_call","name":"echo","call_id":"call-1","arguments":"{\"value\":\"ok\"}"}],"usage":{"input_tokens":10,"output_tokens":5}}`))
			return
		}
		outputs := body["input"].([]any)
		if body["previous_response_id"] != "resp_1" || len(outputs) != 1 || outputs[0].(map[string]any)["call_id"] != "call-1" {
			t.Fatalf("OpenAI stored continuation = %#v", body)
		}
		_, _ = writer.Write([]byte(`{"id":"resp_2","model":"model-snapshot-1","status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"done"}]}],"usage":{"input_tokens":12,"output_tokens":3}}`))
	}))
	defer server.Close()
	provider := nativeAgentProviderForTest(t, ProviderOpenAIResponses, server.URL)
	start := agentStartForTest(t)
	start.Retention = RetentionProviderDefault
	outcome, state, err := provider.StartAgent(context.Background(), "secret", start)
	if err != nil || outcome.Finish.Kind != FinishToolCalls {
		t.Fatalf("OpenAI stored agent start = %#v, %v", outcome, err)
	}
	previous := state.(*openAIAgentState)
	outcome, next, err := provider.ContinueAgent(context.Background(), "secret", state, AgentContinueRequest{
		AttemptID: "attempt-2", Results: []ToolResult{{CallID: "call-1", Name: "echo", Value: json.RawMessage(`{"value":"ok"}`)}},
	})
	continued := next.(*openAIAgentState)
	if err != nil || outcome.Finish.Kind != FinishCompleted || joinText(outcome.Items) != "done" || requests != 2 ||
		previous.previousResponseID != "resp_1" || continued.previousResponseID != "resp_2" {
		t.Fatalf("OpenAI stored agent completion = %#v, %v, requests=%d", outcome, err, requests)
	}
}

func TestAnthropicAgentUsesNativeToolResultContinuation(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		messages := body["messages"].([]any)
		if requests == 1 {
			tools := body["tools"].([]any)
			if body["system"] != "Trusted test instructions." || len(messages) != 1 || tools[0].(map[string]any)["name"] != "echo" {
				t.Fatalf("Anthropic agent start = %#v", body)
			}
			_, _ = writer.Write([]byte(`{"id":"msg_1","model":"model-snapshot-1","content":[{"type":"thinking","thinking":"private","signature":"signed","future_field":"preserve"},{"type":"tool_use","id":"call-1","name":"echo","input":{"value":"ok"}}],"stop_reason":"tool_use","usage":{"input_tokens":10,"output_tokens":5}}`))
			return
		}
		if len(messages) != 3 {
			t.Fatalf("Anthropic continuation messages = %#v", messages)
		}
		resultBlocks := messages[2].(map[string]any)["content"].([]any)
		assistantBlocks := messages[1].(map[string]any)["content"].([]any)
		if resultBlocks[0].(map[string]any)["tool_use_id"] != "call-1" || assistantBlocks[0].(map[string]any)["future_field"] != "preserve" {
			t.Fatalf("Anthropic tool result = %#v", resultBlocks)
		}
		_, _ = writer.Write([]byte(`{"id":"msg_2","model":"model-snapshot-1","content":[{"type":"text","text":"done"}],"stop_reason":"end_turn","usage":{"input_tokens":12,"output_tokens":3}}`))
	}))
	defer server.Close()
	provider := nativeAgentProviderForTest(t, ProviderAnthropicMessages, server.URL)
	outcome, state, err := provider.StartAgent(context.Background(), "secret", agentStartForTest(t))
	if err != nil || outcome.Finish.Kind != FinishToolCalls {
		t.Fatalf("Anthropic agent start outcome = %#v, %v", outcome, err)
	}
	previous := state.(*anthropicAgentState)
	outcome, next, err := provider.ContinueAgent(context.Background(), "secret", state, AgentContinueRequest{
		AttemptID: "attempt-2", Results: []ToolResult{{CallID: "call-1", Name: "echo", Value: json.RawMessage(`{"value":"ok"}`)}},
	})
	continued := next.(*anthropicAgentState)
	if err != nil || outcome.Finish.Kind != FinishCompleted || joinText(outcome.Items) != "done" || requests != 2 ||
		len(previous.messages) != 2 || len(continued.messages) != 4 {
		t.Fatalf("Anthropic agent completion = %#v, %v, requests=%d", outcome, err, requests)
	}
}

func TestToolExecutorRequiresExactBindingsApprovalAndSchemas(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"value":{"type":"string"}},"required":["value"],"additionalProperties":false}`)
	approval, err := artifact.Sum("yotta/test/agent-approval/v1", []byte("approved"))
	if err != nil {
		t.Fatal(err)
	}
	toolSet, err := SealToolSet(ToolSetDraft{
		ID: "yotta.test.agent-tools", Version: "1.0.0", Owner: "tests",
		Tools: []ToolManifestDraft{
			{Name: "echo", Description: "Echo one value.", Authority: ToolAuthorityPure, InputSchema: schema, OutputSchema: schema},
			{Name: "approved_echo", Description: "Echo with host approval.", Authority: ToolAuthorityCapability, Capability: approval, InputSchema: schema, OutputSchema: schema},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := func(_ context.Context, input json.RawMessage) (json.RawMessage, error) { return input, nil }
	if _, err := NewToolExecutor(toolSet, []ToolBinding{{Name: "echo", Handler: handler}}); !errors.Is(err, ErrAgentUnknownTool) {
		t.Fatalf("missing exact binding = %v", err)
	}
	if _, err := NewToolExecutor(toolSet, []ToolBinding{{Name: "echo", Handler: handler}, {Name: "approved_echo", Handler: handler}}); !errors.Is(err, ErrAgentToolApproval) {
		t.Fatalf("missing capability approval = %v", err)
	}
	executor, err := NewToolExecutor(toolSet, []ToolBinding{{Name: "echo", Handler: handler}, {Name: "approved_echo", Approval: approval, Handler: handler}})
	if err != nil {
		t.Fatal(err)
	}
	results, err := executor.Execute(context.Background(), []ToolCall{{CallID: "call-1", Name: "echo", Arguments: json.RawMessage(`{"value":"ok"}`)}}, 1)
	if err != nil || len(results) != 1 || string(results[0].Value) != `{"value":"ok"}` {
		t.Fatalf("tool results = %#v, %v", results, err)
	}
	if _, err := executor.Execute(context.Background(), []ToolCall{{CallID: "call-2", Name: "missing", Arguments: json.RawMessage(`{}`)}}, 1); !errors.Is(err, ErrAgentUnknownTool) {
		t.Fatalf("unknown tool = %v", err)
	}
	if _, err := executor.Execute(context.Background(), []ToolCall{{CallID: "call-3", Name: "echo", Arguments: json.RawMessage(`{"extra":true}`)}}, 1); !errors.Is(err, ErrAgentToolSchema) {
		t.Fatalf("invalid arguments = %v", err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := executor.Execute(cancelled, []ToolCall{{CallID: "call-4", Name: "echo", Arguments: json.RawMessage(`{"value":"ok"}`)}}, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled tool execution = %v", err)
	}
}

func TestAgentRequestsStrictOpenToolSetAndRejectDuplicateResults(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{},"required":[],"additionalProperties":false}`)
	toolSet, err := SealToolSet(ToolSetDraft{ID: "yotta.test.agent", Version: "1.0.0", Owner: "tests", Tools: []ToolManifestDraft{{
		Name: "noop", Description: "Return an empty value.", Authority: ToolAuthorityPure, InputSchema: schema, OutputSchema: schema,
	}}})
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := ResolveToolSet(toolSet)
	if err != nil {
		t.Fatal(err)
	}
	request := AgentStartRequest{AttemptID: "attempt-1", Prompt: renderedPromptForTest(t, "run"), ToolSet: resolved, MaxParallelism: 1, Retention: RetentionNoApplicationState}
	if err := request.Validate(); err != nil {
		t.Fatal(err)
	}
	request.ToolSet.Manifest[0] ^= 1
	if err := request.Validate(); err == nil {
		t.Fatal("accepted a tampered agent ToolSet")
	}
	continuation := AgentContinueRequest{AttemptID: "attempt-1", Results: []ToolResult{
		{CallID: "call-1", Name: "noop", Value: json.RawMessage(`{}`)},
		{CallID: "call-1", Name: "noop", Value: json.RawMessage(`{}`)},
	}}
	if err := continuation.Validate(); err == nil {
		t.Fatal("accepted duplicate tool results")
	}
}

func TestResourceAgentSessionMatchesPendingCallsAndPreservesAmbiguousContinuation(t *testing.T) {
	profile, err := SealModelProfile(ModelProfileDraft{
		Provider: ProviderOpenAIResponses, Model: "model-snapshot-1", MaxOutputTokens: 4096, Evaluation: EvaluationUnverified,
		Capabilities: ProfileCapabilities{ToolCalling: true},
		Pricing:      TokenPricing{InputMicrounitsPerMillion: 1, OutputMicrounitsPerMillion: 1}, ProviderMetadata: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	native := &ambiguousAgentProvider{}
	provider, err := NewResourceProvider(profile, native, credentialMap{"agent-key": "secret"})
	if err != nil {
		t.Fatal(err)
	}
	nonAgentScope, _ := artifact.Marshal(CapabilityScope{Retention: RetentionNoApplicationState})
	if _, err := provider.Open(context.Background(), resource.ProviderOpenRequest{
		Kind: KindModelSession, Operations: []string{OperationAgentStart, OperationAgentContinue}, Config: []byte(`{}`),
		CapabilityScope: nonAgentScope, CredentialBindingID: "agent-key",
	}); err == nil {
		t.Fatal("agent operations accepted without agent scope")
	}
	scope, _ := artifact.Marshal(CapabilityScope{Retention: RetentionNoApplicationState, Agent: true})
	if _, err := provider.Open(context.Background(), resource.ProviderOpenRequest{
		Kind: KindModelSession, Operations: []string{OperationGenerate, OperationAgentStart, OperationAgentContinue}, Config: []byte(`{}`),
		CapabilityScope: scope, CredentialBindingID: "agent-key",
	}); err == nil {
		t.Fatal("generation operation accepted in isolated agent scope")
	}
	object, err := provider.Open(context.Background(), resource.ProviderOpenRequest{
		Kind: KindModelSession, Operations: []string{OperationAgentStart, OperationAgentContinue}, Config: []byte(`{}`),
		CapabilityScope: scope, CredentialBindingID: "agent-key",
	})
	if err != nil {
		t.Fatal(err)
	}
	startPayload, _ := artifact.Marshal(agentStartForTest(t))
	raw, err := provider.Invoke(context.Background(), object, OperationAgentStart, startPayload)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := OpenOutcome(raw)
	if err != nil || outcome.Finish.Kind != FinishToolCalls {
		t.Fatalf("resource Agent start = %#v, %v", outcome, err)
	}
	wrong, _ := artifact.Marshal(AgentContinueRequest{AttemptID: "turn-wrong", Results: []ToolResult{{CallID: "call-1", Name: "other", Value: json.RawMessage(`{"value":"ok"}`)}}})
	if _, err := provider.Invoke(context.Background(), object, OperationAgentContinue, wrong); err == nil || native.continuations != 0 {
		t.Fatalf("mismatched continuation = %v, calls=%d", err, native.continuations)
	}
	valid, _ := artifact.Marshal(AgentContinueRequest{AttemptID: "turn-2", Results: []ToolResult{{CallID: "call-1", Name: "echo", Value: json.RawMessage(`{"value":"ok"}`)}}})
	if _, err := provider.Invoke(context.Background(), object, OperationAgentContinue, valid); err == nil {
		t.Fatal("ambiguous provider failure was accepted")
	} else {
		var failure *ProviderFailure
		if !errors.As(err, &failure) || failure.Retry != RetryAmbiguous {
			t.Fatalf("ambiguous continuation = %#v", err)
		}
	}
	raw, err = provider.Invoke(context.Background(), object, OperationAgentContinue, valid)
	if err != nil {
		t.Fatalf("pending continuation state was lost: %v", err)
	}
	outcome, err = OpenOutcome(raw)
	if err != nil || outcome.Finish.Kind != FinishCompleted || native.continuations != 2 {
		t.Fatalf("resource Agent completion = %#v, %v, calls=%d", outcome, err, native.continuations)
	}
	if _, err := provider.Invoke(context.Background(), object, OperationAgentContinue, valid); err == nil {
		t.Fatal("completed Agent session retained pending calls")
	}
}

type ambiguousAgentProvider struct{ continuations int }

func (*ambiguousAgentProvider) Generate(context.Context, string, GenerateRequest) (Outcome, error) {
	return Outcome{}, errors.New("unexpected generate")
}

func (*ambiguousAgentProvider) StartAgent(context.Context, string, AgentStartRequest) (Outcome, any, error) {
	input, output, cost := int64(1), int64(1), int64(1)
	return Outcome{
		Provider: ProviderOpenAIResponses, RequestedModel: "model-snapshot-1", ResolvedModel: "model-snapshot-1",
		Items:  []OutputItem{{Kind: OutputToolCall, ToolCall: &ToolCall{CallID: "call-1", Name: "echo", Arguments: json.RawMessage(`{"value":"ok"}`)}}},
		Finish: Finish{Kind: FinishToolCalls}, Usage: TokenUsage{InputTotal: &input, OutputTotal: &output, CostMicrounits: &cost},
	}, "opaque-state", nil
}

func (p *ambiguousAgentProvider) ContinueAgent(_ context.Context, _ string, state any, _ AgentContinueRequest) (Outcome, any, error) {
	p.continuations++
	if state != "opaque-state" {
		return Outcome{}, nil, errors.New("unexpected continuation state")
	}
	if p.continuations == 1 {
		return Outcome{}, nil, &ProviderFailure{Stage: FailureTransport, Class: FailureTimeout, Retry: RetryAmbiguous}
	}
	input, output, cost := int64(1), int64(1), int64(1)
	return Outcome{
		Provider: ProviderOpenAIResponses, RequestedModel: "model-snapshot-1", ResolvedModel: "model-snapshot-1",
		Items: []OutputItem{{Kind: OutputText, Text: &TextOutput{Text: "done"}}}, Finish: Finish{Kind: FinishCompleted},
		Usage: TokenUsage{InputTotal: &input, OutputTotal: &output, CostMicrounits: &cost},
	}, state, nil
}

func agentStartForTest(t *testing.T) AgentStartRequest {
	t.Helper()
	schema := json.RawMessage(`{"type":"object","properties":{"value":{"type":"string"}},"required":["value"],"additionalProperties":false}`)
	toolSet, err := SealToolSet(ToolSetDraft{ID: "yotta.test.native-agent", Version: "1.0.0", Owner: "tests", Tools: []ToolManifestDraft{{
		Name: "echo", Description: "Echo a value.", Authority: ToolAuthorityPure, InputSchema: schema, OutputSchema: schema,
	}}})
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := ResolveToolSet(toolSet)
	if err != nil {
		t.Fatal(err)
	}
	maximum := int64(128)
	return AgentStartRequest{
		AttemptID: "attempt-1", Prompt: renderedPromptForTest(t, "run"), ToolSet: resolved,
		Limits: GenerationLimits{MaxOutputTokens: &maximum}, MaxParallelism: 1, Retention: RetentionNoApplicationState,
	}
}

func nativeAgentProviderForTest(t *testing.T, provider ProviderKind, endpoint string) nativeAgentProvider {
	t.Helper()
	profile, err := SealModelProfile(ModelProfileDraft{
		Provider: provider, Model: "model-snapshot-1", MaxOutputTokens: 4096, Evaluation: EvaluationUnverified,
		Capabilities:     ProfileCapabilities{ToolCalling: true},
		Pricing:          TokenPricing{InputMicrounitsPerMillion: 1_000_000, CacheReadMicrounitsPerMillion: 100_000, OutputMicrounitsPerMillion: 2_000_000},
		ProviderMetadata: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := NewNativeProvider(profile, HTTPOptions{Endpoint: endpoint})
	if err != nil {
		t.Fatal(err)
	}
	native, ok := result.(nativeAgentProvider)
	if !ok {
		t.Fatal("native provider does not implement agent turns")
	}
	return native
}
