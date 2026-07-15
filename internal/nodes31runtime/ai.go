package nodes31runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodes31"
	"github.com/yottaapp/yotta/internal/runid"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

func aiGenerate(builtins nodes31.Builtins, structured bool) compiler.Adapter {
	effectID := nodes31.AIGenerateEffectID
	operation := ai.OperationGenerate
	if structured {
		effectID = nodes31.AIExtractEffectID
		operation = ai.OperationGenerateStructured
	}
	return func(ctx context.Context, invocation compiler.Invocation) (_ compiler.AdapterResult, runErr error) {
		action := compiler.AdapterAction{
			EffectID: effectID, Action: "ai.provider-response", SummaryCode: "ai.generation",
			Counters: map[string]int64{}, Facts: map[string]string{},
		}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, action, "ai.generation_failed", runErr))
		}()
		promptEnvelope, ok := invocation.Inputs["prompt"]
		if !ok || len(promptEnvelope.InlineJSON()) == 0 {
			return compiler.AdapterResult{}, errors.New("AI prompt input is missing")
		}
		var prompt string
		if err := json.Unmarshal(promptEnvelope.InlineJSON(), &prompt); err != nil || prompt == "" {
			return compiler.AdapterResult{}, errors.New("AI prompt input must be a non-empty string")
		}
		request, err := aiRequest(invocation.Config, prompt, structured)
		if err != nil {
			return compiler.AdapterResult{}, err
		}
		session := invocation.Sessions["model"]
		if session == nil {
			return compiler.AdapterResult{}, errors.New("AI model capability session is missing")
		}
		handle, err := session.Open(ctx, []string{operation}, []byte(`{}`))
		if err != nil {
			return compiler.AdapterResult{}, err
		}
		defer func() { runErr = errors.Join(runErr, session.Drop(context.WithoutCancel(ctx), handle)) }()
		payload, err := artifact.Marshal(request)
		if err != nil {
			return compiler.AdapterResult{}, err
		}
		rawOutcome, err := session.Invoke(ctx, handle, operation, payload)
		if err != nil {
			return compiler.AdapterResult{}, err
		}
		outcome, err := ai.OpenOutcome(rawOutcome)
		if err != nil {
			return compiler.AdapterResult{}, err
		}
		addAIOutcomeSummary(&action, outcome)
		if outcome.Finish.Kind != ai.FinishCompleted {
			return compiler.AdapterResult{}, &compiler.NodeFailure{
				Code: "ai.generation_failed", Output: "failed", Cause: fmt.Errorf("AI generation finished as %s", outcome.Finish.Kind),
			}
		}
		var rawValue json.RawMessage
		if structured {
			if len(outcome.Items) != 1 || outcome.Items[0].Kind != ai.OutputStructured || outcome.Items[0].Structured == nil {
				return compiler.AdapterResult{}, errors.New("AI structured generation returned no exact structured item")
			}
			rawValue = outcome.Items[0].Structured.Value
		} else {
			var text strings.Builder
			for _, item := range outcome.Items {
				if item.Kind != ai.OutputText || item.Text == nil {
					return compiler.AdapterResult{}, errors.New("AI text generation returned a non-text item")
				}
				text.WriteString(item.Text.Text)
			}
			if text.Len() == 0 {
				return compiler.AdapterResult{}, errors.New("AI text generation returned an empty result")
			}
			rawValue, err = json.Marshal(text.String())
			if err != nil {
				return compiler.AdapterResult{}, err
			}
		}
		resolved, ok := invocation.OutputTypes["result"]
		if !ok {
			return compiler.AdapterResult{}, errors.New("AI result output type is unresolved")
		}
		envelope, err := datatype.SealInlineJSON(builtins.Catalog, resolved, rawValue)
		if err != nil {
			return compiler.AdapterResult{}, err
		}
		return compiler.AdapterResult{Outputs: map[string]datatype.ValueEnvelope{"result": envelope}, ExecOutputs: []string{"completed"}}, nil
	}
}

func aiRequest(config map[string]any, prompt string, structured bool) (ai.GenerateRequest, error) {
	attemptID, err := runid.New()
	if err != nil {
		return ai.GenerateRequest{}, err
	}
	request := ai.GenerateRequest{AttemptID: attemptID, Prompt: prompt, Retention: ai.RetentionNoApplicationState}
	if instructions, ok := config["instructions"].(string); ok {
		request.Instructions = instructions
	}
	if value, exists := config["temperature"]; exists {
		temperature, err := configFloat(value)
		if err != nil {
			return ai.GenerateRequest{}, errors.New("AI temperature config is invalid")
		}
		request.Limits.Temperature = &temperature
	}
	if value, exists := config["maxOutputTokens"]; exists {
		maximum, err := configInt64(value)
		if err != nil {
			return ai.GenerateRequest{}, errors.New("AI max output tokens config is invalid")
		}
		request.Limits.MaxOutputTokens = &maximum
	}
	if structured {
		rawSchema, ok := config["schema"].(string)
		if !ok {
			return ai.GenerateRequest{}, errors.New("AI Extract schema config is missing")
		}
		spec, err := ai.CompileStructuredOutput("result", json.RawMessage(rawSchema))
		if err != nil {
			return ai.GenerateRequest{}, err
		}
		request.Output = &spec
	}
	if err := request.Validate(); err != nil {
		return ai.GenerateRequest{}, err
	}
	return request, nil
}

func configFloat(value any) (float64, error) {
	switch typed := value.(type) {
	case json.Number:
		return typed.Float64()
	case float64:
		return typed, nil
	default:
		return 0, errors.New("not a number")
	}
}

func configInt64(value any) (int64, error) {
	switch typed := value.(type) {
	case json.Number:
		return typed.Int64()
	case float64:
		if typed != float64(int64(typed)) {
			return 0, errors.New("not an integer")
		}
		return int64(typed), nil
	default:
		return 0, errors.New("not an integer")
	}
}

func addAIOutcomeSummary(action *compiler.AdapterAction, outcome ai.Outcome) {
	addCounter := func(name string, value *int64) {
		if value != nil {
			action.Counters[name] = *value
		}
	}
	addCounter("input_tokens", outcome.Usage.InputTotal)
	addCounter("cache_read_tokens", outcome.Usage.CacheRead)
	addCounter("cache_write_tokens", outcome.Usage.CacheWrite)
	addCounter("output_tokens", outcome.Usage.OutputTotal)
	addCounter("reasoning_tokens", outcome.Usage.ReasoningOutput)
	addFact(action.Facts, "provider", string(outcome.Provider))
	addFact(action.Facts, "finish", string(outcome.Finish.Kind))
	addFact(action.Facts, "provider_request_id", outcome.ProviderRequestID)
	addFact(action.Facts, "provider_response_id", outcome.ProviderResponseID)
}

func addFact(target map[string]string, name, value string) {
	if value == "" || len(value) > 256 {
		return
	}
	for _, character := range value {
		if character < 0x20 || character > 0x7e {
			return
		}
	}
	target[name] = value
}
