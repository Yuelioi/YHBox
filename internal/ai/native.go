package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
)

const (
	OpenAIResponsesEndpoint   = "https://api.openai.com/v1/responses"
	AnthropicMessagesEndpoint = "https://api.anthropic.com/v1/messages"
	AnthropicAPIVersion       = "2023-06-01"
	MaxProviderResponseBytes  = 16 << 20
)

type HTTPOptions struct {
	Client   *http.Client
	Endpoint string
}

type nativeProvider struct {
	profile  ModelProfile
	client   *http.Client
	endpoint string
}

func (p *nativeProvider) CloseIdleConnections() { p.client.CloseIdleConnections() }

func NewNativeProvider(profile ModelProfile, options HTTPOptions) (Provider, error) {
	if !profile.Valid() {
		return nil, errors.New("native AI provider requires a model profile")
	}
	if options.Client == nil {
		options.Client = &http.Client{Timeout: 2 * time.Minute}
	}
	if options.Endpoint == "" {
		switch profile.Machine().Provider {
		case ProviderOpenAIResponses:
			options.Endpoint = OpenAIResponsesEndpoint
		case ProviderAnthropicMessages:
			options.Endpoint = AnthropicMessagesEndpoint
		}
	}
	if options.Endpoint == "" {
		return nil, errors.New("native AI provider endpoint is unavailable")
	}
	return &nativeProvider{profile: profile, client: options.Client, endpoint: options.Endpoint}, nil
}

func (p *nativeProvider) Generate(ctx context.Context, credential string, request GenerateRequest) (Outcome, error) {
	if ctx == nil || credential == "" {
		return Outcome{}, contractFailure("AI provider credential is unavailable")
	}
	if err := request.Validate(); err != nil {
		return Outcome{}, contractFailure(err.Error())
	}
	profile := p.profile.Machine()
	if request.Limits.MaxOutputTokens != nil && *request.Limits.MaxOutputTokens > profile.MaxOutputTokens {
		return Outcome{}, contractFailure("AI request exceeds the installed model output budget")
	}
	if request.Output != nil && !profile.Capabilities.StructuredOutput {
		return Outcome{}, contractFailure("installed AI model does not support native structured output")
	}
	if request.Retention == RetentionZeroRequired && !profile.Capabilities.ZeroRetention {
		return Outcome{}, contractFailure("installed AI connection has no verified zero-retention entitlement")
	}
	var outcome Outcome
	var err error
	switch profile.Provider {
	case ProviderOpenAIResponses:
		outcome, err = p.generateOpenAI(ctx, credential, request, profile)
	case ProviderAnthropicMessages:
		outcome, err = p.generateAnthropic(ctx, credential, request, profile)
	default:
		return Outcome{}, contractFailure("unsupported native AI provider")
	}
	if err != nil {
		return Outcome{}, err
	}
	if err := outcome.Validate(); err != nil {
		return Outcome{}, contractFailure(err.Error())
	}
	return outcome, nil
}

type openAIRequest struct {
	Model           string            `json:"model"`
	Instructions    string            `json:"instructions,omitempty"`
	Input           string            `json:"input"`
	Store           bool              `json:"store"`
	Temperature     *float64          `json:"temperature,omitempty"`
	MaxOutputTokens *int64            `json:"max_output_tokens,omitempty"`
	Text            *openAITextConfig `json:"text,omitempty"`
}

type openAITextConfig struct {
	Format openAIFormat `json:"format"`
}

type openAIFormat struct {
	Type   string          `json:"type"`
	Name   string          `json:"name"`
	Strict bool            `json:"strict"`
	Schema json.RawMessage `json:"schema"`
}

type openAIResponse struct {
	ID                string         `json:"id"`
	Model             string         `json:"model"`
	Status            string         `json:"status"`
	Output            []openAIOutput `json:"output"`
	Usage             openAIUsage    `json:"usage"`
	Error             *providerError `json:"error"`
	IncompleteDetails *struct {
		Reason string `json:"reason"`
	} `json:"incomplete_details"`
}

type openAIOutput struct {
	Type      string          `json:"type"`
	Name      string          `json:"name"`
	CallID    string          `json:"call_id"`
	Arguments string          `json:"arguments"`
	Content   []openAIContent `json:"content"`
}

type openAIContent struct {
	Type    string `json:"type"`
	Text    string `json:"text"`
	Refusal string `json:"refusal"`
}

type openAIUsage struct {
	InputTokens  *int64 `json:"input_tokens"`
	OutputTokens *int64 `json:"output_tokens"`
	InputDetails *struct {
		CachedTokens *int64 `json:"cached_tokens"`
	} `json:"input_tokens_details"`
	OutputDetails *struct {
		ReasoningTokens *int64 `json:"reasoning_tokens"`
	} `json:"output_tokens_details"`
}

type providerError struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (p *nativeProvider) generateOpenAI(ctx context.Context, credential string, request GenerateRequest, profile ModelProfileDraft) (Outcome, error) {
	manifest, err := request.Prompt.OpenManifest()
	if err != nil {
		return Outcome{}, contractFailure(err.Error())
	}
	input, err := request.Prompt.ProviderInput()
	if err != nil {
		return Outcome{}, contractFailure(err.Error())
	}
	payload := openAIRequest{
		Model: profile.Model, Instructions: manifest.Machine().Instructions, Input: input,
		Store: request.Retention == RetentionProviderDefault, Temperature: request.Limits.Temperature,
		MaxOutputTokens: boundedOutputTokens(request.Limits.MaxOutputTokens, profile.MaxOutputTokens),
	}
	if request.Output != nil {
		payload.Text = &openAITextConfig{Format: openAIFormat{
			Type: "json_schema", Name: request.Output.Name, Strict: true, Schema: request.Output.Schema,
		}}
	}
	var response openAIResponse
	requestID, raw, err := p.doJSON(ctx, credential, request.AttemptID, payload, &response, ProviderOpenAIResponses)
	if err != nil {
		return Outcome{}, err
	}
	if response.Error != nil || response.Status == "failed" {
		code, message := "generation_failed", "provider reported generation failure"
		if response.Error != nil {
			if response.Error.Code != "" {
				code = response.Error.Code
			} else if response.Error.Type != "" {
				code = response.Error.Type
			}
			message = response.Error.Message
		}
		return Outcome{}, &ProviderFailure{Stage: FailureGeneration, Class: FailureUnknown, ProviderCode: code, ProviderRequestID: requestID, Message: message, Retry: RetryNever, Raw: raw}
	}
	items, hasToolCall, hasRefusal, err := openAIItems(response.Output)
	if err != nil {
		return Outcome{}, contractFailureWithRequest(err.Error(), requestID)
	}
	finish := openAIFinish(response.Status, response.IncompleteDetails, hasToolCall, hasRefusal)
	if request.Output != nil {
		items, err = sealStructuredItems(*request.Output, items, finish)
		if err != nil {
			return Outcome{}, contractFailureWithRequest(err.Error(), requestID)
		}
	}
	usageRaw, _ := artifact.Marshal(response.Usage)
	usage := TokenUsage{InputTotal: response.Usage.InputTokens, OutputTotal: response.Usage.OutputTokens, ProviderExtras: usageRaw}
	if response.Usage.InputDetails != nil {
		usage.CacheRead = response.Usage.InputDetails.CachedTokens
		usage.InputUncached = subtractUsage(response.Usage.InputTokens, response.Usage.InputDetails.CachedTokens)
	}
	if response.Usage.OutputDetails != nil {
		usage.ReasoningOutput = response.Usage.OutputDetails.ReasoningTokens
	}
	return Outcome{
		Provider: ProviderOpenAIResponses, RequestedModel: profile.Model, ResolvedModel: response.Model,
		ProviderRequestID: requestID, ProviderResponseID: response.ID, Items: items, Finish: finish, Usage: usage,
	}, nil
}

func openAIItems(source []openAIOutput) ([]OutputItem, bool, bool, error) {
	items := make([]OutputItem, 0)
	hasToolCall, hasRefusal := false, false
	for _, output := range source {
		switch output.Type {
		case "message":
			for _, content := range output.Content {
				switch content.Type {
				case "output_text":
					items = append(items, OutputItem{Kind: OutputText, Text: &TextOutput{Text: content.Text}})
				case "refusal":
					hasRefusal = true
					items = append(items, OutputItem{Kind: OutputRefusal, Refusal: &RefusalOutput{Reason: content.Refusal}})
				default:
					return nil, false, false, fmt.Errorf("unknown OpenAI content type %q", content.Type)
				}
			}
		case "function_call":
			arguments, err := canonicalJSONObject([]byte(output.Arguments))
			if err != nil {
				return nil, false, false, fmt.Errorf("invalid OpenAI function arguments: %w", err)
			}
			hasToolCall = true
			items = append(items, OutputItem{Kind: OutputToolCall, ToolCall: &ToolCall{CallID: output.CallID, Name: output.Name, Arguments: arguments}})
		case "reasoning":
			// Reasoning is provider-owned continuation state, never user-visible text.
		default:
			return nil, false, false, fmt.Errorf("unknown OpenAI output type %q", output.Type)
		}
	}
	return items, hasToolCall, hasRefusal, nil
}

func openAIFinish(status string, incomplete *struct {
	Reason string `json:"reason"`
}, toolCall, refusal bool) Finish {
	if refusal {
		return Finish{Kind: FinishRefusal, RawProviderReason: status}
	}
	if toolCall {
		return Finish{Kind: FinishToolCalls, RawProviderReason: status}
	}
	switch status {
	case "completed":
		return Finish{Kind: FinishCompleted, RawProviderReason: status}
	case "cancelled":
		return Finish{Kind: FinishCancelled, RawProviderReason: status}
	case "incomplete":
		reason := ""
		if incomplete != nil {
			reason = incomplete.Reason
		}
		switch reason {
		case "max_output_tokens":
			return Finish{Kind: FinishMaxOutput, RawProviderReason: reason}
		case "content_filter":
			return Finish{Kind: FinishContentFilter, RawProviderReason: reason}
		default:
			return Finish{Kind: FinishUnknown, RawProviderReason: reason}
		}
	default:
		return Finish{Kind: FinishUnknown, RawProviderReason: status}
	}
}

type anthropicRequest struct {
	Model        string                 `json:"model"`
	MaxTokens    int64                  `json:"max_tokens"`
	System       string                 `json:"system,omitempty"`
	Messages     []anthropicInput       `json:"messages"`
	Temperature  *float64               `json:"temperature,omitempty"`
	OutputConfig *anthropicOutputConfig `json:"output_config,omitempty"`
}

type anthropicInput struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicOutputConfig struct {
	Format anthropicFormat `json:"format"`
}

type anthropicFormat struct {
	Type   string          `json:"type"`
	Schema json.RawMessage `json:"schema"`
}

type anthropicResponse struct {
	ID         string             `json:"id"`
	Model      string             `json:"model"`
	Content    []anthropicContent `json:"content"`
	StopReason string             `json:"stop_reason"`
	Usage      anthropicUsage     `json:"usage"`
}

type anthropicContent struct {
	Type  string          `json:"type"`
	Text  string          `json:"text"`
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

type anthropicUsage struct {
	InputTokens              *int64 `json:"input_tokens"`
	OutputTokens             *int64 `json:"output_tokens"`
	CacheCreationInputTokens *int64 `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     *int64 `json:"cache_read_input_tokens"`
}

func (p *nativeProvider) generateAnthropic(ctx context.Context, credential string, request GenerateRequest, profile ModelProfileDraft) (Outcome, error) {
	manifest, err := request.Prompt.OpenManifest()
	if err != nil {
		return Outcome{}, contractFailure(err.Error())
	}
	input, err := request.Prompt.ProviderInput()
	if err != nil {
		return Outcome{}, contractFailure(err.Error())
	}
	payload := anthropicRequest{
		Model: profile.Model, MaxTokens: valueOr(request.Limits.MaxOutputTokens, profile.MaxOutputTokens),
		System: manifest.Machine().Instructions, Messages: []anthropicInput{{Role: "user", Content: input}}, Temperature: request.Limits.Temperature,
	}
	if request.Output != nil {
		payload.OutputConfig = &anthropicOutputConfig{Format: anthropicFormat{Type: "json_schema", Schema: request.Output.Schema}}
	}
	var response anthropicResponse
	requestID, _, err := p.doJSON(ctx, credential, request.AttemptID, payload, &response, ProviderAnthropicMessages)
	if err != nil {
		return Outcome{}, err
	}
	items := make([]OutputItem, 0, len(response.Content))
	hasToolCall := false
	for _, content := range response.Content {
		switch content.Type {
		case "text":
			items = append(items, OutputItem{Kind: OutputText, Text: &TextOutput{Text: content.Text}})
		case "tool_use":
			arguments, err := canonicalJSONObject(content.Input)
			if err != nil {
				return Outcome{}, contractFailureWithRequest("invalid Anthropic tool input", requestID)
			}
			hasToolCall = true
			items = append(items, OutputItem{Kind: OutputToolCall, ToolCall: &ToolCall{CallID: content.ID, Name: content.Name, Arguments: arguments}})
		case "thinking", "redacted_thinking":
			// Thinking blocks remain provider-owned continuation state.
		default:
			return Outcome{}, contractFailureWithRequest(fmt.Sprintf("unknown Anthropic content type %q", content.Type), requestID)
		}
	}
	finish := anthropicFinish(response.StopReason, hasToolCall)
	if finish.Kind == FinishRefusal {
		reason := joinText(items)
		items = []OutputItem{{Kind: OutputRefusal, Refusal: &RefusalOutput{Reason: reason}}}
	}
	if request.Output != nil {
		items, err = sealStructuredItems(*request.Output, items, finish)
		if err != nil {
			return Outcome{}, contractFailureWithRequest(err.Error(), requestID)
		}
	}
	inputTotal := sumUsage(response.Usage.InputTokens, response.Usage.CacheCreationInputTokens, response.Usage.CacheReadInputTokens)
	usageRaw, _ := artifact.Marshal(response.Usage)
	return Outcome{
		Provider: ProviderAnthropicMessages, RequestedModel: profile.Model, ResolvedModel: response.Model,
		ProviderRequestID: requestID, ProviderResponseID: response.ID, Items: items, Finish: finish,
		Usage: TokenUsage{InputTotal: inputTotal, InputUncached: response.Usage.InputTokens, CacheRead: response.Usage.CacheReadInputTokens,
			CacheWrite: response.Usage.CacheCreationInputTokens, OutputTotal: response.Usage.OutputTokens, ProviderExtras: usageRaw},
	}, nil
}

func anthropicFinish(reason string, toolCall bool) Finish {
	if toolCall || reason == "tool_use" {
		return Finish{Kind: FinishToolCalls, RawProviderReason: reason}
	}
	switch reason {
	case "end_turn":
		return Finish{Kind: FinishCompleted, RawProviderReason: reason}
	case "max_tokens":
		return Finish{Kind: FinishMaxOutput, RawProviderReason: reason}
	case "model_context_window_exceeded":
		return Finish{Kind: FinishContextLimit, RawProviderReason: reason}
	case "stop_sequence":
		return Finish{Kind: FinishStopSequence, RawProviderReason: reason}
	case "pause_turn":
		return Finish{Kind: FinishPaused, RawProviderReason: reason}
	case "refusal":
		return Finish{Kind: FinishRefusal, RawProviderReason: reason}
	default:
		return Finish{Kind: FinishUnknown, RawProviderReason: reason}
	}
}

func sealStructuredItems(spec StructuredOutputSpec, items []OutputItem, finish Finish) ([]OutputItem, error) {
	if finish.Kind != FinishCompleted {
		return items, nil
	}
	text := joinText(items)
	if text == "" {
		return nil, errors.New("provider completed structured output without JSON content")
	}
	value, err := spec.ValidateValue([]byte(text))
	if err != nil {
		return nil, err
	}
	return []OutputItem{{Kind: OutputStructured, Structured: &StructuredOutput{Value: value}}}, nil
}

func joinText(items []OutputItem) string {
	var builder strings.Builder
	for _, item := range items {
		if item.Kind == OutputText && item.Text != nil {
			builder.WriteString(item.Text.Text)
		}
	}
	return builder.String()
}

func (p *nativeProvider) doJSON(ctx context.Context, credential, attemptID string, payload any, target any, provider ProviderKind) (string, json.RawMessage, error) {
	body, err := artifact.Marshal(payload)
	if err != nil {
		return "", nil, contractFailure(err.Error())
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(body))
	if err != nil {
		return "", nil, contractFailure(err.Error())
	}
	request.Header.Set("Content-Type", "application/json")
	switch provider {
	case ProviderOpenAIResponses:
		request.Header.Set("Authorization", "Bearer "+credential)
		request.Header.Set("X-Client-Request-Id", attemptID)
	case ProviderAnthropicMessages:
		request.Header.Set("x-api-key", credential)
		request.Header.Set("anthropic-version", AnthropicAPIVersion)
	}
	response, err := p.client.Do(request)
	if err != nil {
		class := FailureUnknown
		retry := RetryAmbiguous
		if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
			class, retry = FailureCancelled, RetryNever
		} else if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			class = FailureTimeout
		}
		return "", nil, &ProviderFailure{Stage: FailureTransport, Class: class, Message: err.Error(), Retry: retry}
	}
	defer response.Body.Close()
	requestID := response.Header.Get("x-request-id")
	if requestID == "" {
		requestID = response.Header.Get("request-id")
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, MaxProviderResponseBytes+1))
	if err != nil {
		return requestID, nil, &ProviderFailure{Stage: FailureTransport, Class: FailureUnknown, ProviderRequestID: requestID, Message: err.Error(), Retry: RetryAmbiguous}
	}
	if len(raw) > MaxProviderResponseBytes {
		return requestID, nil, contractFailureWithRequest("AI provider response exceeds byte budget", requestID)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return requestID, boundedRaw(raw), httpFailure(response, requestID, raw)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return requestID, boundedRaw(raw), contractFailureWithRequest("decode AI provider response: "+err.Error(), requestID)
	}
	return requestID, boundedRaw(raw), nil
}

func httpFailure(response *http.Response, requestID string, raw []byte) error {
	var envelope struct {
		Type      string        `json:"type"`
		RequestID string        `json:"request_id"`
		Error     providerError `json:"error"`
	}
	_ = json.Unmarshal(raw, &envelope)
	if requestID == "" {
		requestID = envelope.RequestID
	}
	code := envelope.Error.Code
	if code == "" {
		code = envelope.Error.Type
	}
	if code == "" {
		code = envelope.Type
	}
	class, retry := classifyHTTP(response.StatusCode)
	retryAfter := parseRetryAfter(response.Header.Get("Retry-After"), time.Now())
	if retryAfter != nil {
		retry = RetryAfterHint
	}
	status := response.StatusCode
	return &ProviderFailure{
		Stage: FailureHTTP, Class: class, HTTPStatus: &status, ProviderCode: code, ProviderRequestID: requestID,
		Message: envelope.Error.Message, RetryAfter: retryAfter, Retry: retry, Raw: boundedRaw(raw),
	}
}

func classifyHTTP(status int) (FailureClass, RetryDisposition) {
	switch status {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return FailureInvalidRequest, RetryNever
	case http.StatusUnauthorized:
		return FailureAuthentication, RetryNever
	case http.StatusForbidden:
		return FailurePermission, RetryNever
	case http.StatusNotFound:
		return FailureNotFound, RetryNever
	case http.StatusConflict:
		return FailureConflict, RetryNever
	case http.StatusTooManyRequests:
		return FailureRateLimit, RetryNewAttempt
	case http.StatusGatewayTimeout:
		return FailureTimeout, RetryNewAttempt
	case 529:
		return FailureOverloaded, RetryNewAttempt
	default:
		if status >= 500 {
			return FailureServer, RetryNewAttempt
		}
		return FailureUnknown, RetryNever
	}
}

func parseRetryAfter(value string, now time.Time) *time.Duration {
	if value == "" {
		return nil
	}
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil && seconds >= 0 {
		duration := time.Duration(seconds) * time.Second
		return &duration
	}
	if at, err := http.ParseTime(value); err == nil && at.After(now) {
		duration := at.Sub(now)
		return &duration
	}
	return nil
}

func contractFailure(message string) *ProviderFailure {
	return &ProviderFailure{Stage: FailureContract, Class: FailureInvalidRequest, Message: message, Retry: RetryNever}
}

func contractFailureWithRequest(message, requestID string) *ProviderFailure {
	failure := contractFailure(message)
	failure.ProviderRequestID = requestID
	return failure
}

func boundedRaw(raw []byte) json.RawMessage {
	if len(raw) > MaxProviderRawBytes {
		return nil
	}
	return append(json.RawMessage(nil), raw...)
}

func boundedOutputTokens(requested *int64, maximum int64) *int64 {
	value := maximum
	if requested != nil {
		value = *requested
	}
	return &value
}

func valueOr(requested *int64, maximum int64) int64 {
	if requested != nil {
		return *requested
	}
	return maximum
}

func subtractUsage(total, subset *int64) *int64 {
	if total == nil || subset == nil || *subset > *total {
		return nil
	}
	value := *total - *subset
	return &value
}

func sumUsage(values ...*int64) *int64 {
	var total int64
	seen := false
	for _, value := range values {
		if value != nil {
			total += *value
			seen = true
		}
	}
	if !seen {
		return nil
	}
	return &total
}

func canonicalJSONObject(raw []byte) (json.RawMessage, error) {
	var value map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil || value == nil {
		return nil, errors.New("value is not a JSON object")
	}
	return artifact.Canonicalize(raw)
}
