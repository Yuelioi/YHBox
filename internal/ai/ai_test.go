package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/resource"
)

func TestModelProfileIsContentAddressedAndStrictlyReopened(t *testing.T) {
	profile := profileForTest(t, ProviderOpenAIResponses)
	reopened, err := OpenModelProfile(profile.Bytes(), profile.Digest())
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Digest() != profile.Digest() || reopened.Machine().Model != "model-snapshot-1" {
		t.Fatalf("reopened profile = %#v", reopened.Machine())
	}
	var document map[string]any
	if err := json.Unmarshal(profile.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	document["unknown"] = true
	tampered, err := artifact.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := OpenModelProfile(tampered, profile.Digest()); err == nil {
		t.Fatal("accepted a tampered AI model profile")
	}
}

func TestModelProfileRejectsParallelToolsWithoutToolCalling(t *testing.T) {
	_, err := SealModelProfile(ModelProfileDraft{
		Provider: ProviderOpenAIResponses, Model: "model-snapshot-1", MaxOutputTokens: 4096,
		Evaluation: EvaluationUnverified, Capabilities: ProfileCapabilities{ParallelTools: true},
	})
	if err == nil {
		t.Fatal("accepted parallel tool calls without tool calling")
	}
}

func TestModelProfileRequiresPinnedPricingForToolCalling(t *testing.T) {
	_, err := SealModelProfile(ModelProfileDraft{
		Provider: ProviderOpenAIResponses, Model: "model-snapshot-1", MaxOutputTokens: 4096,
		Evaluation: EvaluationUnverified, Capabilities: ProfileCapabilities{ToolCalling: true},
	})
	if err == nil {
		t.Fatal("tool-calling profile accepted without pinned pricing")
	}
}

func TestOpenAIChatProfileRejectsUnsupportedAgentToolCalling(t *testing.T) {
	_, err := SealModelProfile(ModelProfileDraft{
		Provider: ProviderOpenAIChatCompletions, Model: "deepseek-v4-flash", MaxOutputTokens: 4096,
		Evaluation: EvaluationUnverified, Capabilities: ProfileCapabilities{ToolCalling: true},
		Pricing: TokenPricing{InputMicrounitsPerMillion: 1, OutputMicrounitsPerMillion: 1},
	})
	if err == nil {
		t.Fatal("OpenAI Chat profile accepted unsupported agent tool calling")
	}
}

func TestModelProfileAllowsNoInstalledOutputTokenLimit(t *testing.T) {
	profile, err := SealModelProfile(ModelProfileDraft{
		Provider: ProviderOpenAIChatCompletions, Model: "deepseek-v4-flash",
		MaxOutputTokens: 0, Evaluation: EvaluationUnverified,
	})
	if err != nil {
		t.Fatal(err)
	}
	if profile.Machine().MaxOutputTokens != 0 {
		t.Fatalf("unlimited output token setting = %d", profile.Machine().MaxOutputTokens)
	}
}

func TestStructuredOutputCompilerRejectsPromptOnlyAndPartialSchemas(t *testing.T) {
	valid := json.RawMessage(`{
		"type":"object","properties":{
			"answer":{"type":"string"},
			"score":{"anyOf":[{"type":"number"},{"type":"null"}]}
		},"required":["answer","score"],"additionalProperties":false
	}`)
	spec, err := CompileStructuredOutput("result", valid)
	if err != nil {
		t.Fatal(err)
	}
	value, err := spec.ValidateValue(json.RawMessage(`{"score":0,"answer":"ok"}`))
	if err != nil || string(value) != `{"answer":"ok","score":0}` {
		t.Fatalf("validated value = %s, %v", value, err)
	}
	if _, err := spec.ValidateValue(json.RawMessage(`{"answer":"ok"}`)); err == nil {
		t.Fatal("accepted structured output with a missing required property")
	}
	for _, invalid := range []json.RawMessage{
		json.RawMessage(`{"type":"object","properties":{"answer":{"type":"string"}},"required":[],"additionalProperties":false}`),
		json.RawMessage(`{"type":"object","properties":{},"required":[],"additionalProperties":true}`),
		json.RawMessage(`{"type":"object","properties":{},"required":[],"additionalProperties":false,"patternProperties":{}}`),
		json.RawMessage(`{"anyOf":[{"type":"object","properties":{},"required":[],"additionalProperties":false},{"type":"null"}]}`),
	} {
		if _, err := CompileStructuredOutput("result", invalid); err == nil {
			t.Fatalf("accepted non-portable strict schema: %s", invalid)
		}
	}
}

func TestStructuredFieldsCompileFriendlyDefinitionsIntoStrictOutput(t *testing.T) {
	spec, err := CompileStructuredFields("result", []any{
		map[string]any{"name": "标题", "type": "string", "description": "文章标题"},
		map[string]any{"name": "score", "type": "number", "nullable": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(spec.Schema, &schema); err != nil {
		t.Fatal(err)
	}
	properties := schema["properties"].(map[string]any)
	if properties["标题"].(map[string]any)["type"] != "string" ||
		len(properties["score"].(map[string]any)["anyOf"].([]any)) != 2 ||
		schema["additionalProperties"] != false {
		t.Fatalf("compiled field schema = %#v", schema)
	}
	for _, invalid := range []any{
		[]any{},
		[]any{map[string]any{"name": "value", "type": "string"}, map[string]any{"name": "value", "type": "number"}},
		[]any{map[string]any{"name": " value ", "type": "string"}},
		[]any{map[string]any{"name": "value", "type": "object"}},
	} {
		if _, err := CompileStructuredFields("result", invalid); err == nil {
			t.Fatalf("accepted invalid structured fields %#v", invalid)
		}
	}
}

func TestOpenAIResponsesUsesNativeStructuredOutputAndExactUsage(t *testing.T) {
	spec := structuredSpecForTest(t)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer secret" || request.Header.Get("X-Client-Request-Id") != "attempt-1" {
			t.Fatalf("OpenAI headers = %#v", request.Header)
		}
		var body map[string]any
		decodeRequestForTest(t, request, &body)
		if body["model"] != "model-snapshot-1" || body["store"] != false {
			t.Fatalf("OpenAI request = %#v", body)
		}
		if body["instructions"] != "Trusted test instructions." {
			t.Fatalf("OpenAI prompt boundary = %#v", body)
		}
		input := body["input"].([]any)
		message := input[0].(map[string]any)
		content := message["content"].([]any)
		imagePart := content[0].(map[string]any)
		textPart := content[1].(map[string]any)
		if message["role"] != "user" ||
			imagePart["type"] != "input_image" ||
			!strings.HasPrefix(imagePart["image_url"].(string), "data:image/jpeg;base64,") ||
			textPart["type"] != "input_text" || textPart["text"] != "answer" {
			t.Fatalf("OpenAI multimodal input = %#v", body["input"])
		}
		text := body["text"].(map[string]any)
		format := text["format"].(map[string]any)
		if format["type"] != "json_schema" || format["strict"] != true || format["name"] != "result" {
			t.Fatalf("OpenAI structured format = %#v", format)
		}
		writer.Header().Set("x-request-id", "request-openai")
		_, _ = writer.Write([]byte(`{
			"id":"resp_1","model":"model-resolved","status":"completed",
			"output":[{"type":"message","content":[{"type":"output_text","text":"{\"answer\":\"ok\"}"}]}],
			"usage":{"input_tokens":10,"output_tokens":2,"input_tokens_details":{"cached_tokens":4},"output_tokens_details":{"reasoning_tokens":1}}
		}`))
	}))
	defer server.Close()
	provider := nativeProviderForTest(t, profileForTest(t, ProviderOpenAIResponses), server.URL)
	outcome, err := provider.Generate(context.Background(), "secret", GenerateRequest{
		AttemptID: "attempt-1", Prompt: renderedPromptForTest(t, "answer"),
		Image:  &ImageInput{MediaType: "image/jpeg", Data: []byte("jpeg")},
		Output: &spec, Retention: RetentionNoApplicationState,
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Finish.Kind != FinishCompleted || outcome.ProviderRequestID != "request-openai" || outcome.ResolvedModel != "model-resolved" ||
		len(outcome.Items) != 1 || outcome.Items[0].Kind != OutputStructured || string(outcome.Items[0].Structured.Value) != `{"answer":"ok"}` {
		t.Fatalf("OpenAI outcome = %#v", outcome)
	}
	if deref(outcome.Usage.InputTotal) != 10 || deref(outcome.Usage.InputUncached) != 6 || deref(outcome.Usage.CacheRead) != 4 ||
		deref(outcome.Usage.OutputTotal) != 2 || deref(outcome.Usage.ReasoningOutput) != 1 {
		t.Fatalf("OpenAI usage = %#v", outcome.Usage)
	}
}

func TestOpenAIChatCompletionsResolvesBaseURLAndGenerates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/chat/completions" {
			t.Fatalf("OpenAI Chat path = %q", request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("OpenAI Chat authorization = %q", request.Header.Get("Authorization"))
		}
		var body map[string]any
		decodeRequestForTest(t, request, &body)
		if body["model"] != "deepseek-v4-flash" {
			t.Fatalf("OpenAI Chat model = %#v", body["model"])
		}
		if _, limited := body["max_tokens"]; limited {
			t.Fatalf("unlimited OpenAI Chat request contains max_tokens = %#v", body["max_tokens"])
		}
		messages, ok := body["messages"].([]any)
		if !ok || len(messages) != 2 {
			t.Fatalf("OpenAI Chat messages = %#v", body["messages"])
		}
		user := messages[1].(map[string]any)
		content := user["content"].([]any)
		imagePart := content[0].(map[string]any)
		imageURL := imagePart["image_url"].(map[string]any)
		textPart := content[1].(map[string]any)
		if imagePart["type"] != "image_url" ||
			!strings.HasPrefix(imageURL["url"].(string), "data:image/png;base64,") ||
			textPart["type"] != "text" || textPart["text"] != "answer" {
			t.Fatalf("OpenAI Chat multimodal input = %#v", user["content"])
		}
		writer.Header().Set("x-request-id", "request-chat")
		_, _ = writer.Write([]byte(`{
			"id":"chat-1","model":"deepseek-v4-flash",
			"choices":[{"index":0,"message":{"role":"assistant","content":"OK"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":4,"completion_tokens":1,"total_tokens":5}
		}`))
	}))
	defer server.Close()

	profile, err := SealModelProfile(ModelProfileDraft{
		Provider: ProviderOpenAIChatCompletions, Endpoint: server.URL, AllowLocalHTTP: true,
		Model: "deepseek-v4-flash", MaxOutputTokens: 0, Evaluation: EvaluationUnverified,
	})
	if err != nil {
		t.Fatal(err)
	}
	provider, err := NewNativeProvider(profile, HTTPOptions{})
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := provider.Generate(context.Background(), "secret", GenerateRequest{
		AttemptID: "attempt-chat", Prompt: renderedPromptForTest(t, "answer"),
		Image:     &ImageInput{MediaType: "image/png", Data: []byte("png")},
		Retention: RetentionNoApplicationState,
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Provider != ProviderOpenAIChatCompletions || outcome.ProviderRequestID != "request-chat" ||
		outcome.ResolvedModel != "deepseek-v4-flash" || outcome.Finish.Kind != FinishCompleted ||
		len(outcome.Items) != 1 || outcome.Items[0].Text == nil || outcome.Items[0].Text.Text != "OK" {
		t.Fatalf("OpenAI Chat outcome = %#v", outcome)
	}
}

func TestAnthropicMessagesUsesOutputConfigAndCacheAwareUsage(t *testing.T) {
	profile := profileForTest(t, ProviderAnthropicMessages)
	spec := structuredSpecForTest(t)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("x-api-key") != "secret" || request.Header.Get("anthropic-version") != AnthropicAPIVersion {
			t.Fatalf("Anthropic headers = %#v", request.Header)
		}
		var body map[string]any
		decodeRequestForTest(t, request, &body)
		if _, legacy := body["output_format"]; legacy {
			t.Fatal("Anthropic request used legacy output_format")
		}
		if body["system"] != "Trusted test instructions." {
			t.Fatalf("Anthropic prompt boundary = %#v", body)
		}
		messages := body["messages"].([]any)
		content := messages[0].(map[string]any)["content"].([]any)
		imagePart := content[0].(map[string]any)
		source := imagePart["source"].(map[string]any)
		textPart := content[1].(map[string]any)
		if imagePart["type"] != "image" || source["type"] != "base64" ||
			source["media_type"] != "image/jpeg" || source["data"] == "" ||
			textPart["type"] != "text" || textPart["text"] != "answer" {
			t.Fatalf("Anthropic multimodal input = %#v", content)
		}
		output := body["output_config"].(map[string]any)
		format := output["format"].(map[string]any)
		if format["type"] != "json_schema" {
			t.Fatalf("Anthropic structured format = %#v", format)
		}
		writer.Header().Set("request-id", "request-anthropic")
		_, _ = writer.Write([]byte(`{
			"id":"msg_1","model":"claude-resolved","stop_reason":"end_turn",
			"content":[{"type":"text","text":"{\"answer\":\"ok\"}"}],
			"usage":{"input_tokens":5,"cache_creation_input_tokens":7,"cache_read_input_tokens":11,"output_tokens":3}
		}`))
	}))
	defer server.Close()
	provider := nativeProviderForTest(t, profile, server.URL)
	outcome, err := provider.Generate(context.Background(), "secret", GenerateRequest{
		AttemptID: "attempt-2", Prompt: renderedPromptForTest(t, "answer"),
		Image:  &ImageInput{MediaType: "image/jpeg", Data: []byte("jpeg")},
		Output: &spec, Retention: RetentionNoApplicationState,
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.ProviderRequestID != "request-anthropic" || outcome.Finish.Kind != FinishCompleted || deref(outcome.Usage.InputTotal) != 23 ||
		deref(outcome.Usage.InputUncached) != 5 || deref(outcome.Usage.CacheRead) != 11 || deref(outcome.Usage.CacheWrite) != 7 {
		t.Fatalf("Anthropic outcome = %#v", outcome)
	}
}

func TestProviderFailuresAndNonCompletedStructuredOutputFailClosed(t *testing.T) {
	t.Run("HTML success response identifies endpoint mismatch", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.Header().Set("Content-Type", "text/html; charset=utf-8")
			writer.Header().Set("x-request-id", "request-html")
			_, _ = writer.Write([]byte(`<!doctype html><title>NewAPI</title><p>Sign in</p>`))
		}))
		defer server.Close()
		provider := nativeProviderForTest(t, profileForTest(t, ProviderOpenAIResponses), server.URL)
		_, err := provider.Generate(context.Background(), "secret", GenerateRequest{AttemptID: "attempt-html", Prompt: renderedPromptForTest(t, "x"), Retention: RetentionProviderDefault})
		var failure *ProviderFailure
		if !errors.As(err, &failure) || failure.Stage != FailureContract || failure.Class != FailureInvalidResponse ||
			failure.HTTPStatus == nil || *failure.HTTPStatus != http.StatusOK || failure.ProviderCode != "html-response" ||
			failure.ProviderRequestID != "request-html" {
			t.Fatalf("failure = %#v", err)
		}
	})

	t.Run("overloaded", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.Header().Set("request-id", "request-529")
			writer.Header().Set("Retry-After", "2")
			writer.WriteHeader(529)
			_, _ = writer.Write([]byte(`{"type":"error","error":{"type":"overloaded_error","message":"busy"},"request_id":"request-529"}`))
		}))
		defer server.Close()
		provider := nativeProviderForTest(t, profileForTest(t, ProviderAnthropicMessages), server.URL)
		_, err := provider.Generate(context.Background(), "secret", GenerateRequest{AttemptID: "attempt-3", Prompt: renderedPromptForTest(t, "x"), Retention: RetentionProviderDefault})
		var failure *ProviderFailure
		if !errors.As(err, &failure) || failure.Class != FailureOverloaded || failure.Retry != RetryAfterHint || failure.RetryAfter == nil {
			t.Fatalf("failure = %#v", err)
		}
	})

	t.Run("refusal is not structured success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			_, _ = writer.Write([]byte(`{"id":"resp_2","model":"model-snapshot-1","status":"completed","output":[{"type":"message","content":[{"type":"refusal","refusal":"no"}]}],"usage":{}}`))
		}))
		defer server.Close()
		provider := nativeProviderForTest(t, profileForTest(t, ProviderOpenAIResponses), server.URL)
		spec := structuredSpecForTest(t)
		outcome, err := provider.Generate(context.Background(), "secret", GenerateRequest{AttemptID: "attempt-4", Prompt: renderedPromptForTest(t, "x"), Output: &spec, Retention: RetentionProviderDefault})
		if err != nil || outcome.Finish.Kind != FinishRefusal || outcome.Items[0].Kind != OutputRefusal {
			t.Fatalf("refusal outcome = %#v, %v", outcome, err)
		}
	})

	t.Run("unknown provider type", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			_, _ = writer.Write([]byte(`{"id":"resp_3","model":"model-snapshot-1","status":"completed","output":[{"type":"future_item"}],"usage":{}}`))
		}))
		defer server.Close()
		provider := nativeProviderForTest(t, profileForTest(t, ProviderOpenAIResponses), server.URL)
		_, err := provider.Generate(context.Background(), "secret", GenerateRequest{AttemptID: "attempt-5", Prompt: renderedPromptForTest(t, "x"), Retention: RetentionProviderDefault})
		var failure *ProviderFailure
		if !errors.As(err, &failure) || failure.Stage != FailureContract {
			t.Fatalf("unknown type error = %#v", err)
		}
	})
}

func TestResourceProviderEnforcesGrantedScopeAndResolvesCredentialAtCallTime(t *testing.T) {
	native := &recordingProvider{outcome: Outcome{
		Provider: ProviderOpenAIResponses, RequestedModel: "model-snapshot-1", ResolvedModel: "model-snapshot-1",
		Items: []OutputItem{{Kind: OutputText, Text: &TextOutput{Text: "ok"}}}, Finish: Finish{Kind: FinishCompleted},
	}}
	provider, err := NewResourceProvider(profileForTest(t, ProviderOpenAIResponses), native, credentialMap{"slot-a": "secret"})
	if err != nil {
		t.Fatal(err)
	}
	scope, _ := artifact.Marshal(CapabilityScope{Retention: RetentionNoApplicationState, Structured: false})
	object, err := provider.Open(context.Background(), resource.ProviderOpenRequest{
		Kind: KindModelSession, Operations: []string{OperationGenerate}, Config: []byte(`{}`),
		CapabilityScope: scope, CredentialBindingID: "slot-a",
	})
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := artifact.Marshal(GenerateRequest{AttemptID: "attempt-6", Prompt: renderedPromptForTest(t, "x"), Retention: RetentionNoApplicationState})
	raw, err := provider.Invoke(context.Background(), object, OperationGenerate, payload)
	if err != nil || !bytes.Contains(raw, []byte(`"completed"`)) || native.credential != "secret" {
		t.Fatalf("resource invoke = %s, %v, credential=%q", raw, err, native.credential)
	}
	if _, err := provider.Invoke(context.Background(), object, OperationGenerateStructured, payload); err == nil {
		t.Fatal("resource provider widened the granted structured operation")
	}
	if err := provider.Close(context.Background(), object); err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Invoke(context.Background(), object, OperationGenerate, payload); err == nil {
		t.Fatal("closed AI session remained callable")
	}
}

type credentialMap map[string]string

func (m credentialMap) Get(id string) (string, error) {
	value := m[id]
	if value == "" {
		return "", errors.New("missing")
	}
	return value, nil
}

type recordingProvider struct {
	credential string
	outcome    Outcome
}

func (p *recordingProvider) Generate(_ context.Context, credential string, _ GenerateRequest) (Outcome, error) {
	p.credential = credential
	return p.outcome, nil
}

func profileForTest(t *testing.T, provider ProviderKind) ModelProfile {
	t.Helper()
	profile, err := SealModelProfile(ModelProfileDraft{
		Provider: provider, Model: "model-snapshot-1", MaxOutputTokens: 4096, Evaluation: EvaluationUnverified,
		Capabilities: ProfileCapabilities{StructuredOutput: true}, ProviderMetadata: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	return profile
}

func structuredSpecForTest(t *testing.T) StructuredOutputSpec {
	t.Helper()
	spec, err := CompileStructuredOutput("result", json.RawMessage(`{"type":"object","properties":{"answer":{"type":"string"}},"required":["answer"],"additionalProperties":false}`))
	if err != nil {
		t.Fatal(err)
	}
	return spec
}

func renderedPromptForTest(t *testing.T, content string) RenderedPrompt {
	t.Helper()
	manifest, err := SealPromptManifest(PromptManifestDraft{
		ID: "yotta.test.prompt", Version: "1.0.0", Owner: "tests", Instructions: "Trusted test instructions.",
	})
	if err != nil {
		t.Fatal(err)
	}
	prompt, err := RenderPrompt(manifest, []PromptBlock{{Kind: PromptBlockUser, Content: content}})
	if err != nil {
		t.Fatal(err)
	}
	return prompt
}

func nativeProviderForTest(t *testing.T, profile ModelProfile, endpoint string) Provider {
	t.Helper()
	provider, err := NewNativeProvider(profile, HTTPOptions{Endpoint: endpoint})
	if err != nil {
		t.Fatal(err)
	}
	return provider
}

func decodeRequestForTest(t *testing.T, request *http.Request, target any) {
	t.Helper()
	decoder := json.NewDecoder(request.Body)
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		t.Fatal(err)
	}
}

func deref(value *int64) int64 {
	if value == nil {
		return -1
	}
	return *value
}
