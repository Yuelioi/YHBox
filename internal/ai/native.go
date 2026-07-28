package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
)

const (
	OpenAIResponsesEndpoint      = "https://api.openai.com/v1/responses"
	OpenAIChatCompletionsBaseURL = "https://api.openai.com/v1"
	AnthropicMessagesEndpoint    = "https://api.anthropic.com/v1/messages"
	AnthropicAPIVersion          = "2023-06-01"
	MaxProviderResponseBytes     = 16 << 20
	MaxAgentProviderStateBytes   = 16 << 20
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
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.Proxy = nil
		options.Client = &http.Client{
			Timeout:   2 * time.Minute,
			Transport: transport,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}
	if options.Endpoint == "" {
		var err error
		options.Endpoint, err = providerRequestEndpoint(profile.Machine().Provider, profile.Machine().Endpoint)
		if err != nil {
			return nil, err
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
	if exceedsInstalledOutputLimit(request.Limits.MaxOutputTokens, profile.MaxOutputTokens) {
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
	case ProviderOpenAIChatCompletions:
		outcome, err = p.generateOpenAIChat(ctx, credential, request, profile)
	case ProviderAnthropicMessages:
		outcome, err = p.generateAnthropic(ctx, credential, request, profile)
	default:
		return Outcome{}, contractFailure("unsupported native AI provider")
	}
	if err != nil {
		return Outcome{}, err
	}
	if profile.Pricing.InputMicrounitsPerMillion != 0 || profile.Pricing.OutputMicrounitsPerMillion != 0 {
		if err := attachEstimatedCost(profile.Pricing, &outcome); err != nil {
			return Outcome{}, contractFailure(err.Error())
		}
	}
	if err := outcome.Validate(); err != nil {
		return Outcome{}, contractFailure(err.Error())
	}
	return outcome, nil
}

type openAIRequest struct {
	Model              string            `json:"model"`
	Instructions       string            `json:"instructions,omitempty"`
	Input              any               `json:"input"`
	Store              bool              `json:"store"`
	PreviousResponseID string            `json:"previous_response_id,omitempty"`
	Temperature        *float64          `json:"temperature,omitempty"`
	MaxOutputTokens    *int64            `json:"max_output_tokens,omitempty"`
	Text               *openAITextConfig `json:"text,omitempty"`
	Tools              []openAITool      `json:"tools,omitempty"`
	ParallelToolCalls  *bool             `json:"parallel_tool_calls,omitempty"`
	Include            []string          `json:"include,omitempty"`
}

type openAIChatRequest struct {
	Model          string                    `json:"model"`
	Messages       []openAIChatMessage       `json:"messages"`
	Temperature    *float64                  `json:"temperature,omitempty"`
	MaxTokens      *int64                    `json:"max_tokens,omitempty"`
	ResponseFormat *openAIChatResponseFormat `json:"response_format,omitempty"`
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type openAIResponsesMessage struct {
	Role    string                       `json:"role"`
	Content []openAIResponsesContentPart `json:"content"`
}

type openAIResponsesContentPart struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

type openAIChatContentPart struct {
	Type     string              `json:"type"`
	Text     string              `json:"text,omitempty"`
	ImageURL *openAIChatImageURL `json:"image_url,omitempty"`
}

type openAIChatImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

type openAIChatResponseFormat struct {
	Type       string               `json:"type"`
	JSONSchema openAIChatJSONSchema `json:"json_schema"`
}

type openAIChatJSONSchema struct {
	Name   string          `json:"name"`
	Strict bool            `json:"strict"`
	Schema json.RawMessage `json:"schema"`
}

type openAIChatResponse struct {
	ID      string             `json:"id"`
	Model   string             `json:"model"`
	Choices []openAIChatChoice `json:"choices"`
	Usage   openAIChatUsage    `json:"usage"`
}

type openAIChatChoice struct {
	Message      openAIChatResponseMessage `json:"message"`
	FinishReason string                    `json:"finish_reason"`
}

type openAIChatResponseMessage struct {
	Content string `json:"content"`
	Refusal string `json:"refusal"`
}

type openAIChatUsage struct {
	PromptTokens     *int64 `json:"prompt_tokens"`
	CompletionTokens *int64 `json:"completion_tokens"`
	PromptDetails    *struct {
		CachedTokens *int64 `json:"cached_tokens"`
	} `json:"prompt_tokens_details"`
	CompletionDetails *struct {
		ReasoningTokens *int64 `json:"reasoning_tokens"`
	} `json:"completion_tokens_details"`
}

type openAITool struct {
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
	Strict      bool            `json:"strict"`
}

type openAIFunctionOutput struct {
	Type   string `json:"type"`
	CallID string `json:"call_id"`
	Output string `json:"output"`
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
	ID                string            `json:"id"`
	Model             string            `json:"model"`
	Status            string            `json:"status"`
	Output            []json.RawMessage `json:"output"`
	Usage             openAIUsage       `json:"usage"`
	Error             *providerError    `json:"error"`
	IncompleteDetails *struct {
		Reason string `json:"reason"`
	} `json:"incomplete_details"`
}

type openAIOutput struct {
	Type             string          `json:"type"`
	Name             string          `json:"name"`
	CallID           string          `json:"call_id"`
	Arguments        string          `json:"arguments"`
	Content          []openAIContent `json:"content"`
	EncryptedContent string          `json:"encrypted_content,omitempty"`
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

func openAIResponsesInput(text string, image *ImageInput) any {
	if image == nil {
		return text
	}
	return []openAIResponsesMessage{{
		Role: "user",
		Content: []openAIResponsesContentPart{
			{Type: "input_image", ImageURL: imageDataURL(*image), Detail: "auto"},
			{Type: "input_text", Text: text},
		},
	}}
}

func openAIChatInput(text string, image *ImageInput) any {
	if image == nil {
		return text
	}
	return []openAIChatContentPart{
		{Type: "image_url", ImageURL: &openAIChatImageURL{URL: imageDataURL(*image), Detail: "auto"}},
		{Type: "text", Text: text},
	}
}

func anthropicUserInput(text string, image *ImageInput) any {
	if image == nil {
		return text
	}
	return []any{
		anthropicImageBlock{Type: "image", Source: anthropicImageSource{
			Type: "base64", MediaType: image.MediaType, Data: base64.StdEncoding.EncodeToString(image.Data),
		}},
		anthropicTextBlock{Type: "text", Text: text},
	}
}

func imageDataURL(image ImageInput) string {
	return "data:" + image.MediaType + ";base64," + base64.StdEncoding.EncodeToString(image.Data)
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
		Model: profile.Model, Instructions: manifest.Machine().Instructions, Input: openAIResponsesInput(input, request.Image),
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
	return openAIOutcome(profile, request.Output, requestID, raw, response)
}

func (p *nativeProvider) generateOpenAIChat(ctx context.Context, credential string, request GenerateRequest, profile ModelProfileDraft) (Outcome, error) {
	manifest, err := request.Prompt.OpenManifest()
	if err != nil {
		return Outcome{}, contractFailure(err.Error())
	}
	input, err := request.Prompt.ProviderInput()
	if err != nil {
		return Outcome{}, contractFailure(err.Error())
	}
	payload := openAIChatRequest{
		Model: profile.Model,
		Messages: []openAIChatMessage{
			{Role: "system", Content: manifest.Machine().Instructions},
			{Role: "user", Content: openAIChatInput(input, request.Image)},
		},
		Temperature: request.Limits.Temperature,
		MaxTokens:   boundedOutputTokens(request.Limits.MaxOutputTokens, profile.MaxOutputTokens),
	}
	if request.Output != nil {
		payload.ResponseFormat = &openAIChatResponseFormat{
			Type: "json_schema",
			JSONSchema: openAIChatJSONSchema{
				Name: request.Output.Name, Strict: true, Schema: request.Output.Schema,
			},
		}
	}
	var response openAIChatResponse
	requestID, raw, err := p.doJSON(ctx, credential, request.AttemptID, payload, &response, ProviderOpenAIChatCompletions)
	if err != nil {
		return Outcome{}, err
	}
	return openAIChatOutcome(profile, request.Output, requestID, raw, response)
}

func openAIChatOutcome(profile ModelProfileDraft, structured *StructuredOutputSpec, requestID string, raw []byte, response openAIChatResponse) (Outcome, error) {
	if len(response.Choices) == 0 {
		return Outcome{}, contractFailureWithRequest("OpenAI Chat response has no choices", requestID)
	}
	choice := response.Choices[0]
	items := make([]OutputItem, 0, 1)
	finish := openAIChatFinish(choice.FinishReason)
	if choice.Message.Refusal != "" {
		items = append(items, OutputItem{Kind: OutputRefusal, Refusal: &RefusalOutput{Reason: choice.Message.Refusal}})
		finish = Finish{Kind: FinishRefusal, RawProviderReason: choice.FinishReason}
	} else if choice.Message.Content != "" {
		items = append(items, OutputItem{Kind: OutputText, Text: &TextOutput{Text: choice.Message.Content}})
	}
	if structured != nil {
		var err error
		items, err = sealStructuredItems(*structured, items, finish)
		if err != nil {
			return Outcome{}, contractFailureWithRequest(err.Error(), requestID)
		}
	}
	usageRaw, _ := artifact.Marshal(response.Usage)
	usage := TokenUsage{
		InputTotal: response.Usage.PromptTokens, OutputTotal: response.Usage.CompletionTokens,
		ProviderExtras: usageRaw,
	}
	if response.Usage.PromptDetails != nil {
		usage.CacheRead = response.Usage.PromptDetails.CachedTokens
		usage.InputUncached = subtractUsage(response.Usage.PromptTokens, response.Usage.PromptDetails.CachedTokens)
	}
	if response.Usage.CompletionDetails != nil {
		usage.ReasoningOutput = response.Usage.CompletionDetails.ReasoningTokens
	}
	return Outcome{
		Provider: ProviderOpenAIChatCompletions, RequestedModel: profile.Model, ResolvedModel: response.Model,
		ProviderRequestID: requestID, ProviderResponseID: response.ID, Items: items, Finish: finish, Usage: usage,
	}, nil
}

func openAIChatFinish(reason string) Finish {
	finish := Finish{RawProviderReason: reason}
	switch reason {
	case "stop":
		finish.Kind = FinishCompleted
	case "length":
		finish.Kind = FinishMaxOutput
	case "content_filter":
		finish.Kind = FinishContentFilter
	case "tool_calls":
		finish.Kind = FinishToolCalls
	default:
		finish.Kind = FinishUnknown
	}
	return finish
}

func openAIOutcome(profile ModelProfileDraft, structured *StructuredOutputSpec, requestID string, raw []byte, response openAIResponse) (Outcome, error) {
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
	if structured != nil {
		items, err = sealStructuredItems(*structured, items, finish)
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

func attachEstimatedCost(pricing TokenPricing, outcome *Outcome) error {
	if outcome == nil {
		return errors.New("AI outcome is unavailable")
	}
	if err := pricing.Validate(); err != nil {
		return err
	}
	cost, err := estimateCost(pricing, outcome.Usage)
	if err != nil {
		return err
	}
	outcome.Usage.CostMicrounits = &cost
	return nil
}

type openAIAgentState struct {
	start              AgentStartRequest
	previousResponseID string
	history            []json.RawMessage
}

func (p *nativeProvider) StartAgent(ctx context.Context, credential string, request AgentStartRequest) (Outcome, any, error) {
	if ctx == nil || credential == "" {
		return Outcome{}, nil, contractFailure("AI provider credential is unavailable")
	}
	if err := request.Validate(); err != nil {
		return Outcome{}, nil, contractFailure(err.Error())
	}
	profile := p.profile.Machine()
	if !profile.Capabilities.ToolCalling {
		return Outcome{}, nil, contractFailure("installed AI model does not support native tool calling")
	}
	if exceedsInstalledOutputLimit(request.Limits.MaxOutputTokens, profile.MaxOutputTokens) {
		return Outcome{}, nil, contractFailure("AI request exceeds the installed model output budget")
	}
	if request.Retention == RetentionZeroRequired && !profile.Capabilities.ZeroRetention {
		return Outcome{}, nil, contractFailure("installed AI connection has no verified zero-retention entitlement")
	}
	if request.MaxParallelism > 1 && !profile.Capabilities.ParallelTools {
		return Outcome{}, nil, contractFailure("installed AI model does not support parallel tool calls")
	}
	manifest, _ := request.Prompt.OpenManifest()
	input, _ := request.Prompt.ProviderInput()
	tools, err := openAITools(request.ToolSet)
	if err != nil {
		return Outcome{}, nil, contractFailure(err.Error())
	}
	if profile.Provider == ProviderAnthropicMessages {
		return p.startAnthropicAgent(ctx, credential, request, profile)
	}
	if profile.Provider == ProviderOpenAIChatCompletions {
		return Outcome{}, nil, contractFailure("OpenAI Chat Completions agent continuation is not supported")
	}
	payload := openAIRequest{
		Model: profile.Model, Instructions: manifest.Machine().Instructions, Input: input, Tools: tools,
		Store: request.Retention == RetentionProviderDefault, Temperature: request.Limits.Temperature,
		MaxOutputTokens: boundedOutputTokens(request.Limits.MaxOutputTokens, profile.MaxOutputTokens),
	}
	if request.Retention != RetentionProviderDefault {
		payload.Include = []string{"reasoning.encrypted_content"}
	}
	parallel := request.MaxParallelism > 1
	payload.ParallelToolCalls = &parallel
	var response openAIResponse
	requestID, raw, err := p.doJSON(ctx, credential, request.AttemptID, payload, &response, ProviderOpenAIResponses)
	if err != nil {
		return Outcome{}, nil, err
	}
	outcome, err := openAIOutcome(profile, nil, requestID, raw, response)
	if err != nil {
		return Outcome{}, nil, err
	}
	if err := attachEstimatedCost(profile.Pricing, &outcome); err != nil {
		return Outcome{}, nil, contractFailure(err.Error())
	}
	state := &openAIAgentState{start: request, previousResponseID: response.ID}
	if request.Retention != RetentionProviderDefault {
		userItem, marshalErr := artifact.Marshal(map[string]any{"role": "user", "content": input})
		if marshalErr != nil {
			return Outcome{}, nil, contractFailure(marshalErr.Error())
		}
		state.history, err = appendOpenAIState(nil, append([]json.RawMessage{userItem}, response.Output...)...)
		if err != nil {
			return Outcome{}, nil, contractFailure(err.Error())
		}
	}
	return outcome, state, nil
}

func (p *nativeProvider) ContinueAgent(ctx context.Context, credential string, state any, request AgentContinueRequest) (Outcome, any, error) {
	if ctx == nil || credential == "" {
		return Outcome{}, nil, contractFailure("AI provider credential is unavailable")
	}
	if err := request.Validate(); err != nil {
		return Outcome{}, nil, contractFailure(err.Error())
	}
	if anthropic, ok := state.(*anthropicAgentState); ok {
		return p.continueAnthropicAgent(ctx, credential, anthropic, request)
	}
	previous, ok := state.(*openAIAgentState)
	if !ok || previous.previousResponseID == "" {
		return Outcome{}, nil, contractFailure("AI provider continuation state is invalid")
	}
	currentValue := *previous
	currentValue.history = append([]json.RawMessage(nil), previous.history...)
	current := &currentValue
	profile := p.profile.Machine()
	manifest, err := current.start.Prompt.OpenManifest()
	if err != nil {
		return Outcome{}, nil, contractFailure(err.Error())
	}
	tools, err := openAITools(current.start.ToolSet)
	if err != nil {
		return Outcome{}, nil, contractFailure(err.Error())
	}
	outputs := make([]openAIFunctionOutput, 0, len(request.Results))
	for _, result := range request.Results {
		outputs = append(outputs, openAIFunctionOutput{Type: "function_call_output", CallID: result.CallID, Output: string(result.Value)})
	}
	payload := openAIRequest{
		Model: profile.Model, Instructions: manifest.Machine().Instructions, Input: outputs, PreviousResponseID: current.previousResponseID,
		Store: current.start.Retention == RetentionProviderDefault, Temperature: current.start.Limits.Temperature,
		MaxOutputTokens: boundedOutputTokens(current.start.Limits.MaxOutputTokens, profile.MaxOutputTokens), Tools: tools,
	}
	parallel := current.start.MaxParallelism > 1
	payload.ParallelToolCalls = &parallel
	if current.start.Retention != RetentionProviderDefault {
		toolOutputItems := make([]json.RawMessage, 0, len(outputs))
		for _, output := range outputs {
			raw, marshalErr := artifact.Marshal(output)
			if marshalErr != nil {
				return Outcome{}, nil, contractFailure(marshalErr.Error())
			}
			toolOutputItems = append(toolOutputItems, raw)
		}
		history, historyErr := appendOpenAIState(current.history, toolOutputItems...)
		if historyErr != nil {
			return Outcome{}, nil, contractFailure(historyErr.Error())
		}
		payload.Input = history
		payload.PreviousResponseID = ""
		payload.Include = []string{"reasoning.encrypted_content"}
	}
	var response openAIResponse
	requestID, raw, err := p.doJSON(ctx, credential, request.AttemptID, payload, &response, ProviderOpenAIResponses)
	if err != nil {
		return Outcome{}, nil, err
	}
	outcome, err := openAIOutcome(profile, nil, requestID, raw, response)
	if err != nil {
		return Outcome{}, nil, err
	}
	if err := attachEstimatedCost(profile.Pricing, &outcome); err != nil {
		return Outcome{}, nil, contractFailure(err.Error())
	}
	current.previousResponseID = response.ID
	if current.start.Retention != RetentionProviderDefault {
		current.history, err = appendOpenAIState(payload.Input.([]json.RawMessage), response.Output...)
		if err != nil {
			return Outcome{}, nil, contractFailure(err.Error())
		}
	}
	return outcome, current, nil
}

func appendOpenAIState(existing []json.RawMessage, items ...json.RawMessage) ([]json.RawMessage, error) {
	result := append([]json.RawMessage(nil), existing...)
	total := 0
	for _, item := range result {
		total += len(item)
	}
	for _, item := range items {
		canonical, err := artifact.Canonicalize(item)
		if err != nil {
			return nil, errors.New("OpenAI continuation item is invalid")
		}
		total += len(canonical)
		if total > MaxAgentProviderStateBytes {
			return nil, errors.New("OpenAI continuation state exceeds its byte budget")
		}
		result = append(result, canonical)
	}
	return result, nil
}

func openAITools(artifact ToolSetArtifact) ([]openAITool, error) {
	toolSet, err := artifact.Open()
	if err != nil {
		return nil, err
	}
	tools := make([]openAITool, 0, len(toolSet.Machine().Tools))
	for _, tool := range toolSet.Machine().Tools {
		tools = append(tools, openAITool{Type: "function", Name: tool.Name, Description: tool.Description, Parameters: tool.InputSchema, Strict: true})
	}
	return tools, nil
}

func openAIItems(source []json.RawMessage) ([]OutputItem, bool, bool, error) {
	items := make([]OutputItem, 0)
	hasToolCall, hasRefusal := false, false
	for _, raw := range source {
		var output openAIOutput
		if err := json.Unmarshal(raw, &output); err != nil {
			return nil, false, false, errors.New("invalid OpenAI output item")
		}
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
	MaxTokens    *int64                 `json:"max_tokens,omitempty"`
	System       string                 `json:"system,omitempty"`
	Messages     []anthropicInput       `json:"messages"`
	Temperature  *float64               `json:"temperature,omitempty"`
	OutputConfig *anthropicOutputConfig `json:"output_config,omitempty"`
	Tools        []anthropicTool        `json:"tools,omitempty"`
	ToolChoice   *anthropicToolChoice   `json:"tool_choice,omitempty"`
}

type anthropicInput struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type anthropicImageBlock struct {
	Type   string               `json:"type"`
	Source anthropicImageSource `json:"source"`
}

type anthropicImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type anthropicTextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type anthropicToolResult struct {
	Type      string `json:"type"`
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
}

type anthropicToolChoice struct {
	Type                   string `json:"type"`
	DisableParallelToolUse bool   `json:"disable_parallel_tool_use"`
}

type anthropicOutputConfig struct {
	Format anthropicFormat `json:"format"`
}

type anthropicFormat struct {
	Type   string          `json:"type"`
	Schema json.RawMessage `json:"schema"`
}

type anthropicResponse struct {
	ID         string            `json:"id"`
	Model      string            `json:"model"`
	Content    []json.RawMessage `json:"content"`
	StopReason string            `json:"stop_reason"`
	Usage      anthropicUsage    `json:"usage"`
}

type anthropicContent struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	Thinking  string          `json:"thinking,omitempty"`
	Signature string          `json:"signature,omitempty"`
	Data      string          `json:"data,omitempty"`
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
		Model: profile.Model, MaxTokens: boundedOutputTokens(request.Limits.MaxOutputTokens, profile.MaxOutputTokens),
		System: manifest.Machine().Instructions, Messages: []anthropicInput{{Role: "user", Content: anthropicUserInput(input, request.Image)}}, Temperature: request.Limits.Temperature,
	}
	if request.Output != nil {
		payload.OutputConfig = &anthropicOutputConfig{Format: anthropicFormat{Type: "json_schema", Schema: request.Output.Schema}}
	}
	var response anthropicResponse
	requestID, _, err := p.doJSON(ctx, credential, request.AttemptID, payload, &response, ProviderAnthropicMessages)
	if err != nil {
		return Outcome{}, err
	}
	return anthropicOutcome(profile, request.Output, requestID, response)
}

func anthropicOutcome(profile ModelProfileDraft, structured *StructuredOutputSpec, requestID string, response anthropicResponse) (Outcome, error) {
	items := make([]OutputItem, 0, len(response.Content))
	hasToolCall := false
	for _, raw := range response.Content {
		var content anthropicContent
		if err := json.Unmarshal(raw, &content); err != nil {
			return Outcome{}, contractFailureWithRequest("invalid Anthropic content block", requestID)
		}
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
	if structured != nil {
		var err error
		items, err = sealStructuredItems(*structured, items, finish)
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

type anthropicAgentState struct {
	start    AgentStartRequest
	messages []anthropicInput
}

func (p *nativeProvider) startAnthropicAgent(ctx context.Context, credential string, request AgentStartRequest, profile ModelProfileDraft) (Outcome, any, error) {
	manifest, _ := request.Prompt.OpenManifest()
	input, _ := request.Prompt.ProviderInput()
	tools, err := anthropicTools(request.ToolSet)
	if err != nil {
		return Outcome{}, nil, contractFailure(err.Error())
	}
	messages := []anthropicInput{{Role: "user", Content: input}}
	payload := anthropicRequest{
		Model: profile.Model, MaxTokens: boundedOutputTokens(request.Limits.MaxOutputTokens, profile.MaxOutputTokens),
		System: manifest.Machine().Instructions, Messages: messages, Tools: tools, Temperature: request.Limits.Temperature,
		ToolChoice: &anthropicToolChoice{Type: "auto", DisableParallelToolUse: request.MaxParallelism == 1},
	}
	var response anthropicResponse
	requestID, _, err := p.doJSON(ctx, credential, request.AttemptID, payload, &response, ProviderAnthropicMessages)
	if err != nil {
		return Outcome{}, nil, err
	}
	outcome, err := anthropicOutcome(profile, nil, requestID, response)
	if err != nil {
		return Outcome{}, nil, err
	}
	if err := attachEstimatedCost(profile.Pricing, &outcome); err != nil {
		return Outcome{}, nil, contractFailure(err.Error())
	}
	content, err := canonicalAnthropicContent(response.Content)
	if err != nil {
		return Outcome{}, nil, contractFailure(err.Error())
	}
	messages = append(messages, anthropicInput{Role: "assistant", Content: content})
	if err := validateAgentProviderState("anthropic", messages); err != nil {
		return Outcome{}, nil, contractFailure(err.Error())
	}
	return outcome, &anthropicAgentState{start: request, messages: messages}, nil
}

func (p *nativeProvider) continueAnthropicAgent(ctx context.Context, credential string, current *anthropicAgentState, request AgentContinueRequest) (Outcome, any, error) {
	if current == nil || len(current.messages) == 0 {
		return Outcome{}, nil, contractFailure("AI provider continuation state is invalid")
	}
	profile := p.profile.Machine()
	manifest, _ := current.start.Prompt.OpenManifest()
	tools, err := anthropicTools(current.start.ToolSet)
	if err != nil {
		return Outcome{}, nil, contractFailure(err.Error())
	}
	results := make([]anthropicToolResult, 0, len(request.Results))
	for _, result := range request.Results {
		results = append(results, anthropicToolResult{Type: "tool_result", ToolUseID: result.CallID, Content: string(result.Value)})
	}
	messages := append([]anthropicInput(nil), current.messages...)
	messages = append(messages, anthropicInput{Role: "user", Content: results})
	payload := anthropicRequest{
		Model: profile.Model, MaxTokens: boundedOutputTokens(current.start.Limits.MaxOutputTokens, profile.MaxOutputTokens),
		System: manifest.Machine().Instructions, Messages: messages, Tools: tools, Temperature: current.start.Limits.Temperature,
		ToolChoice: &anthropicToolChoice{Type: "auto", DisableParallelToolUse: current.start.MaxParallelism == 1},
	}
	var response anthropicResponse
	requestID, _, err := p.doJSON(ctx, credential, request.AttemptID, payload, &response, ProviderAnthropicMessages)
	if err != nil {
		return Outcome{}, nil, err
	}
	outcome, err := anthropicOutcome(profile, nil, requestID, response)
	if err != nil {
		return Outcome{}, nil, err
	}
	if err := attachEstimatedCost(profile.Pricing, &outcome); err != nil {
		return Outcome{}, nil, contractFailure(err.Error())
	}
	content, err := canonicalAnthropicContent(response.Content)
	if err != nil {
		return Outcome{}, nil, contractFailure(err.Error())
	}
	messages = append(messages, anthropicInput{Role: "assistant", Content: content})
	if err := validateAgentProviderState("anthropic", messages); err != nil {
		return Outcome{}, nil, contractFailure(err.Error())
	}
	return outcome, &anthropicAgentState{start: current.start, messages: messages}, nil
}

func canonicalAnthropicContent(source []json.RawMessage) ([]json.RawMessage, error) {
	result := make([]json.RawMessage, 0, len(source))
	for _, item := range source {
		canonical, err := artifact.Canonicalize(item)
		if err != nil {
			return nil, errors.New("anthropic continuation block is invalid")
		}
		result = append(result, canonical)
	}
	return result, nil
}

func validateAgentProviderState(provider string, state any) error {
	raw, err := artifact.Marshal(state)
	if err != nil {
		return fmt.Errorf("%s continuation state is invalid", provider)
	}
	if len(raw) > MaxAgentProviderStateBytes {
		return fmt.Errorf("%s continuation state exceeds its byte budget", provider)
	}
	return nil
}

func anthropicTools(artifact ToolSetArtifact) ([]anthropicTool, error) {
	toolSet, err := artifact.Open()
	if err != nil {
		return nil, err
	}
	tools := make([]anthropicTool, 0, len(toolSet.Machine().Tools))
	for _, tool := range toolSet.Machine().Tools {
		tools = append(tools, anthropicTool{Name: tool.Name, Description: tool.Description, InputSchema: tool.InputSchema})
	}
	return tools, nil
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
	case ProviderOpenAIResponses, ProviderOpenAIChatCompletions:
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

func providerRequestEndpoint(provider ProviderKind, configured string) (string, error) {
	parsed, err := url.Parse(configured)
	if err != nil || !parsed.IsAbs() || parsed.Host == "" {
		return "", errors.New("AI provider base URL is invalid")
	}
	var suffix string
	switch provider {
	case ProviderOpenAIResponses:
		suffix = "/responses"
	case ProviderOpenAIChatCompletions:
		suffix = "/chat/completions"
	case ProviderAnthropicMessages:
		suffix = "/v1/messages"
	default:
		return "", errors.New("unsupported native AI provider")
	}
	path := strings.TrimRight(parsed.Path, "/")
	if !strings.HasSuffix(path, suffix) {
		path += suffix
	}
	parsed.Path = path
	parsed.RawPath = ""
	return parsed.String(), nil
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
	if requested != nil {
		value := *requested
		return &value
	}
	if maximum <= 0 {
		return nil
	}
	value := maximum
	return &value
}

func exceedsInstalledOutputLimit(requested *int64, maximum int64) bool {
	return requested != nil && maximum > 0 && *requested > maximum
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
