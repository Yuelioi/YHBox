package noderuntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/runid"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

func aiAgent(builtins nodes.Builtins, now func() time.Time) compiler.Adapter {
	toolExecutor, toolErr := ai.NewToolExecutor(builtins.AIAgentToolSet, []ai.ToolBinding{{
		Name: "text_length", Handler: func(_ context.Context, arguments json.RawMessage) (json.RawMessage, error) {
			var input struct {
				Text string `json:"text"`
			}
			if err := json.Unmarshal(arguments, &input); err != nil {
				return nil, err
			}
			return json.Marshal(map[string]int{"characters": utf8.RuneCountInString(input.Text)})
		},
	}})
	return func(ctx context.Context, invocation compiler.Invocation) (_ compiler.AdapterResult, runErr error) {
		action := compiler.AdapterAction{
			EffectID: nodes.AIAgentEffectID, Action: "ai.agent-terminal", SummaryCode: "ai.agent",
			Counters: map[string]int64{}, Facts: map[string]string{},
		}
		var tracker *ai.BudgetTracker
		defer func() {
			if tracker != nil {
				addAgentBudgetSummary(&action, tracker.Usage())
			}
			runErr = errors.Join(runErr, recordAdapterOutcome(context.WithoutCancel(ctx), invocation, action, agentFailureCode(runErr), runErr))
		}()
		if toolErr != nil || now == nil {
			return compiler.AdapterResult{}, errors.New("AI agent runtime is unavailable")
		}
		prompt, blocks, err := agentPromptBlocks(invocation)
		if err != nil {
			return compiler.AdapterResult{}, err
		}
		budget, maximum, err := agentBudget(invocation.Config)
		if err != nil {
			return compiler.AdapterResult{}, err
		}
		started := now()
		tracker, err = ai.NewBudgetTracker(budget, started)
		if err != nil {
			return compiler.AdapterResult{}, err
		}
		rendered, err := ai.RenderPrompt(builtins.AIAgentPrompt, append([]ai.PromptBlock{{Kind: ai.PromptBlockUser, Content: prompt}}, blocks...))
		if err != nil {
			return compiler.AdapterResult{}, err
		}
		resolvedTools, err := ai.ResolveToolSet(toolExecutor.ToolSet())
		if err != nil {
			return compiler.AdapterResult{}, err
		}
		addFact(action.Facts, "prompt_manifest", builtins.AIAgentPrompt.Digest().String())
		addFact(action.Facts, "tool_set", builtins.AIAgentToolSet.Digest().String())
		session := invocation.Sessions["model"]
		if session == nil {
			return compiler.AdapterResult{}, errors.New("AI model capability session is missing")
		}
		handle, err := session.Open(ctx, []string{ai.OperationAgentStart, ai.OperationAgentContinue}, []byte(`{}`))
		if err != nil {
			return compiler.AdapterResult{}, err
		}
		defer func() { runErr = errors.Join(runErr, session.Drop(context.WithoutCancel(ctx), handle)) }()
		runCtx, cancel := context.WithTimeout(ctx, time.Duration(budget.MaxWallTimeMillis)*time.Millisecond)
		defer cancel()
		attemptID, err := runid.New()
		if err != nil {
			return compiler.AdapterResult{}, err
		}
		start := ai.AgentStartRequest{
			AttemptID: attemptID, Prompt: rendered, ToolSet: resolvedTools,
			Limits: ai.GenerationLimits{MaxOutputTokens: &maximum}, MaxParallelism: budget.MaxParallelism, Retention: ai.RetentionNoApplicationState,
		}
		operation := ai.OperationAgentStart
		var request any = start
		for {
			if err := tracker.BeforeTurn(now()); err != nil {
				return compiler.AdapterResult{}, agentFailure(err)
			}
			if invocation.EmitStatus != nil {
				usage := tracker.Usage()
				if err := invocation.EmitStatus(runCtx, "ai.agent_turn", map[string]int64{"iteration": int64(usage.Iterations)}); err != nil {
					return compiler.AdapterResult{}, err
				}
			}
			payload, err := artifact.Marshal(request)
			if err != nil {
				return compiler.AdapterResult{}, err
			}
			raw, err := session.Invoke(runCtx, handle, operation, payload)
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					return compiler.AdapterResult{}, agentFailure(err)
				}
				return compiler.AdapterResult{}, err
			}
			outcome, err := ai.OpenOutcome(raw)
			if err != nil {
				return compiler.AdapterResult{}, err
			}
			calls, err := agentToolCalls(outcome)
			if err != nil {
				return compiler.AdapterResult{}, err
			}
			if err := tracker.ConsumeTurn(now(), outcome, len(calls)); err != nil {
				return compiler.AdapterResult{}, agentFailure(err)
			}
			addAgentOutcomeSummary(&action, outcome)
			switch outcome.Finish.Kind {
			case ai.FinishCompleted:
				result, err := agentTextResult(outcome)
				if err != nil {
					return compiler.AdapterResult{}, err
				}
				resolved := invocation.OutputTypes["result"]
				envelope, err := datatype.SealInlineJSON(builtins.Catalog, resolved, result)
				if err != nil {
					return compiler.AdapterResult{}, err
				}
				return compiler.AdapterResult{Outputs: map[string]datatype.ValueEnvelope{"result": envelope}, ExecOutputs: []string{"completed"}}, nil
			case ai.FinishToolCalls:
				results, err := toolExecutor.Execute(runCtx, calls, budget.MaxParallelism)
				if err != nil {
					return compiler.AdapterResult{}, agentFailure(err)
				}
				if invocation.EmitStatus != nil {
					if err := invocation.EmitStatus(runCtx, "ai.agent_tool_calls", map[string]int64{"calls": int64(len(results))}); err != nil {
						return compiler.AdapterResult{}, err
					}
				}
				turnID, err := runid.New()
				if err != nil {
					return compiler.AdapterResult{}, err
				}
				request = ai.AgentContinueRequest{AttemptID: turnID, Results: results}
				operation = ai.OperationAgentContinue
			default:
				return compiler.AdapterResult{}, &compiler.NodeFailure{Code: "ai.generation_failed", Output: "failed", Cause: fmt.Errorf("AI agent finished as %s", outcome.Finish.Kind)}
			}
		}
	}
}

func agentPromptBlocks(invocation compiler.Invocation) (string, []ai.PromptBlock, error) {
	promptEnvelope, ok := invocation.Inputs["prompt"]
	if !ok {
		return "", nil, errors.New("AI agent prompt input is missing")
	}
	var prompt string
	if err := json.Unmarshal(promptEnvelope.InlineJSON(), &prompt); err != nil || prompt == "" {
		return "", nil, errors.New("AI agent prompt input must be a non-empty string")
	}
	contextEnvelope, ok := invocation.Inputs["context"]
	if !ok || len(contextEnvelope.InlineJSON()) == 0 {
		return prompt, nil, nil
	}
	return prompt, []ai.PromptBlock{{Kind: ai.PromptBlockContext, Content: string(contextEnvelope.InlineJSON())}}, nil
}

func agentBudget(config map[string]any) (ai.RunBudget, int64, error) {
	read := func(name string) (int64, error) {
		value, ok := config[name]
		if !ok {
			return 0, fmt.Errorf("AI agent config %q is missing", name)
		}
		return configInt64(value)
	}
	maximum, err := read("maxOutputTokens")
	if err != nil {
		return ai.RunBudget{}, 0, err
	}
	input, err := read("maxInputTokens")
	if err != nil {
		return ai.RunBudget{}, 0, err
	}
	output, err := read("maxTotalOutputTokens")
	if err != nil {
		return ai.RunBudget{}, 0, err
	}
	cost, err := read("maxCostMicrounits")
	if err != nil {
		return ai.RunBudget{}, 0, err
	}
	wall, err := read("maxWallTimeMillis")
	if err != nil {
		return ai.RunBudget{}, 0, err
	}
	iterations, err := read("maxIterations")
	if err != nil {
		return ai.RunBudget{}, 0, err
	}
	calls, err := read("maxToolCalls")
	if err != nil {
		return ai.RunBudget{}, 0, err
	}
	parallel, err := read("maxParallelism")
	if err != nil {
		return ai.RunBudget{}, 0, err
	}
	budget := ai.RunBudget{
		MaxInputTokens: input, MaxOutputTokens: output, MaxCostMicrounits: cost, MaxWallTimeMillis: wall,
		MaxIterations: int(iterations), MaxToolCalls: int(calls), MaxParallelism: int(parallel),
	}
	if err := budget.Validate(); err != nil {
		return ai.RunBudget{}, 0, err
	}
	return budget, maximum, nil
}

func agentToolCalls(outcome ai.Outcome) ([]ai.ToolCall, error) {
	calls := make([]ai.ToolCall, 0)
	for _, item := range outcome.Items {
		if item.Kind == ai.OutputToolCall {
			calls = append(calls, *item.ToolCall)
		} else if outcome.Finish.Kind == ai.FinishToolCalls {
			return nil, errors.New("AI agent tool turn contained non-tool output")
		}
	}
	if (outcome.Finish.Kind == ai.FinishToolCalls) != (len(calls) > 0) {
		return nil, errors.New("AI agent tool calls do not match finish state")
	}
	return calls, nil
}

func agentTextResult(outcome ai.Outcome) (json.RawMessage, error) {
	text := ""
	for _, item := range outcome.Items {
		if item.Kind != ai.OutputText || item.Text == nil {
			return nil, errors.New("AI agent completion contained non-text output")
		}
		text += item.Text.Text
	}
	if text == "" {
		return nil, errors.New("AI agent returned an empty result")
	}
	return json.Marshal(text)
}

func agentFailure(err error) error {
	return &compiler.NodeFailure{Code: agentFailureCode(err), Output: "failed", Cause: err}
}

func agentFailureCode(err error) string {
	switch {
	case errors.Is(err, ai.ErrAgentBudgetExceeded), errors.Is(err, context.DeadlineExceeded):
		return "ai.agent_budget_exhausted"
	case errors.Is(err, ai.ErrAgentUnknownTool):
		return "ai.agent_unknown_tool"
	case errors.Is(err, ai.ErrAgentToolSchema):
		return "ai.agent_tool_schema"
	case errors.Is(err, ai.ErrAgentToolApproval):
		return "ai.agent_tool_approval"
	default:
		return "ai.generation_failed"
	}
}

func addAgentOutcomeSummary(action *compiler.AdapterAction, outcome ai.Outcome) {
	addFact(action.Facts, "provider", string(outcome.Provider))
	addFact(action.Facts, "requested_model", outcome.RequestedModel)
	addFact(action.Facts, "resolved_model", outcome.ResolvedModel)
	addFact(action.Facts, "finish", string(outcome.Finish.Kind))
	addFact(action.Facts, "provider_request_id", outcome.ProviderRequestID)
	addFact(action.Facts, "provider_response_id", outcome.ProviderResponseID)
	for name, value := range map[string]*int64{
		"input_tokens": outcome.Usage.InputTotal, "output_tokens": outcome.Usage.OutputTotal,
		"cache_read_tokens": outcome.Usage.CacheRead, "cache_write_tokens": outcome.Usage.CacheWrite,
		"reasoning_tokens": outcome.Usage.ReasoningOutput, "cost_microunits": outcome.Usage.CostMicrounits,
	} {
		if value != nil {
			action.Counters[name] += *value
		}
	}
}

func addAgentBudgetSummary(action *compiler.AdapterAction, usage ai.BudgetUsage) {
	action.Counters["budget_input_tokens"] = usage.InputTokens
	action.Counters["budget_output_tokens"] = usage.OutputTokens
	action.Counters["budget_cost_microunits"] = usage.CostMicrounits
	action.Counters["budget_wall_time_ms"] = usage.WallTimeMillis
	action.Counters["budget_iterations"] = int64(usage.Iterations)
	action.Counters["budget_tool_calls"] = int64(usage.ToolCalls)
	action.Counters["budget_max_parallelism"] = int64(usage.MaxParallelism)
}
