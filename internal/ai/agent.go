package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
)

const (
	MaxAgentInputTokens    = 100_000_000
	MaxAgentOutputTokens   = 100_000_000
	MaxAgentCostMicrounits = 1_000_000_000_000
	MaxAgentWallTimeMillis = 3_600_000
	MaxAgentIterations     = 64
	MaxAgentToolCalls      = 256
	MaxAgentParallelism    = 32
)

var (
	ErrAgentBudgetExceeded = errors.New("AI agent budget exhausted")
	ErrAgentUnknownTool    = errors.New("AI agent requested an unknown tool")
	ErrAgentToolSchema     = errors.New("AI agent tool value violates its schema")
	ErrAgentToolApproval   = errors.New("AI agent tool lacks host approval")
)

type TokenPricing struct {
	InputMicrounitsPerMillion     int64 `json:"inputMicrounitsPerMillion"`
	CacheReadMicrounitsPerMillion int64 `json:"cacheReadMicrounitsPerMillion"`
	OutputMicrounitsPerMillion    int64 `json:"outputMicrounitsPerMillion"`
}

func (p TokenPricing) Validate() error {
	for _, value := range []int64{p.InputMicrounitsPerMillion, p.CacheReadMicrounitsPerMillion, p.OutputMicrounitsPerMillion} {
		if value < 0 || value > 1_000_000_000_000 {
			return errors.New("AI token pricing is outside the supported range")
		}
	}
	if p.InputMicrounitsPerMillion == 0 && p.OutputMicrounitsPerMillion == 0 {
		return errors.New("AI token pricing is unavailable")
	}
	return nil
}

type RunBudget struct {
	MaxInputTokens    int64 `json:"maxInputTokens"`
	MaxOutputTokens   int64 `json:"maxOutputTokens"`
	MaxCostMicrounits int64 `json:"maxCostMicrounits"`
	MaxWallTimeMillis int64 `json:"maxWallTimeMillis"`
	MaxIterations     int   `json:"maxIterations"`
	MaxToolCalls      int   `json:"maxToolCalls"`
	MaxParallelism    int   `json:"maxParallelism"`
}

func (b RunBudget) Validate() error {
	if b.MaxInputTokens <= 0 || b.MaxInputTokens > MaxAgentInputTokens ||
		b.MaxOutputTokens <= 0 || b.MaxOutputTokens > MaxAgentOutputTokens ||
		b.MaxCostMicrounits <= 0 || b.MaxCostMicrounits > MaxAgentCostMicrounits ||
		b.MaxWallTimeMillis <= 0 || b.MaxWallTimeMillis > MaxAgentWallTimeMillis ||
		b.MaxIterations <= 0 || b.MaxIterations > MaxAgentIterations || b.MaxToolCalls <= 0 || b.MaxToolCalls > MaxAgentToolCalls ||
		b.MaxParallelism <= 0 || b.MaxParallelism > MaxAgentParallelism || b.MaxParallelism > b.MaxToolCalls {
		return errors.New("invalid AI agent run budget")
	}
	return nil
}

type BudgetUsage struct {
	InputTokens    int64 `json:"inputTokens"`
	OutputTokens   int64 `json:"outputTokens"`
	CostMicrounits int64 `json:"costMicrounits"`
	WallTimeMillis int64 `json:"wallTimeMillis"`
	Iterations     int   `json:"iterations"`
	ToolCalls      int   `json:"toolCalls"`
	MaxParallelism int   `json:"maxParallelism"`
}

type BudgetTracker struct {
	budget  RunBudget
	started time.Time
	usage   BudgetUsage
}

func NewBudgetTracker(budget RunBudget, started time.Time) (*BudgetTracker, error) {
	if err := budget.Validate(); err != nil {
		return nil, err
	}
	if started.IsZero() {
		return nil, errors.New("AI agent budget requires a start time")
	}
	return &BudgetTracker{budget: budget, started: started}, nil
}

func (b *BudgetTracker) BeforeTurn(now time.Time) error {
	if b == nil || now.Before(b.started) || now.Sub(b.started) > time.Duration(b.budget.MaxWallTimeMillis)*time.Millisecond ||
		b.usage.Iterations >= b.budget.MaxIterations {
		return ErrAgentBudgetExceeded
	}
	b.usage.Iterations++
	b.usage.WallTimeMillis = now.Sub(b.started).Milliseconds()
	return nil
}

func (b *BudgetTracker) ConsumeTurn(now time.Time, outcome Outcome, toolCalls int) error {
	if b == nil || outcome.Usage.InputTotal == nil || outcome.Usage.OutputTotal == nil || outcome.Usage.CostMicrounits == nil ||
		toolCalls < 0 || toolCalls > b.budget.MaxParallelism {
		return ErrAgentBudgetExceeded
	}
	input, ok := checkedAgentCounter(b.usage.InputTokens, *outcome.Usage.InputTotal)
	if !ok {
		return ErrAgentBudgetExceeded
	}
	output, ok := checkedAgentCounter(b.usage.OutputTokens, *outcome.Usage.OutputTotal)
	if !ok {
		return ErrAgentBudgetExceeded
	}
	cost, ok := checkedAgentCounter(b.usage.CostMicrounits, *outcome.Usage.CostMicrounits)
	if !ok {
		return ErrAgentBudgetExceeded
	}
	b.usage.InputTokens = input
	b.usage.OutputTokens = output
	b.usage.CostMicrounits = cost
	b.usage.ToolCalls += toolCalls
	if toolCalls > b.usage.MaxParallelism {
		b.usage.MaxParallelism = toolCalls
	}
	if now.Before(b.started) {
		return ErrAgentBudgetExceeded
	}
	b.usage.WallTimeMillis = now.Sub(b.started).Milliseconds()
	if b.usage.InputTokens > b.budget.MaxInputTokens || b.usage.OutputTokens > b.budget.MaxOutputTokens ||
		b.usage.CostMicrounits > b.budget.MaxCostMicrounits || b.usage.WallTimeMillis > b.budget.MaxWallTimeMillis ||
		b.usage.ToolCalls > b.budget.MaxToolCalls {
		return ErrAgentBudgetExceeded
	}
	return nil
}

func checkedAgentCounter(current, increment int64) (int64, bool) {
	if current < 0 || increment < 0 || increment > int64(^uint64(0)>>1)-current {
		return 0, false
	}
	return current + increment, true
}

func (b *BudgetTracker) Usage() BudgetUsage {
	if b == nil {
		return BudgetUsage{}
	}
	return b.usage
}

func estimateCost(pricing TokenPricing, usage TokenUsage) (int64, error) {
	if usage.InputTotal == nil || usage.OutputTotal == nil {
		return 0, errors.New("AI token usage is incomplete")
	}
	cached := int64(0)
	if usage.CacheRead != nil {
		cached = *usage.CacheRead
	}
	if cached < 0 || cached > *usage.InputTotal {
		return 0, errors.New("AI cached token usage is inconsistent")
	}
	uncached := *usage.InputTotal - cached
	weighted := int64(0)
	for _, component := range []struct{ tokens, rate int64 }{
		{uncached, pricing.InputMicrounitsPerMillion},
		{cached, pricing.CacheReadMicrounitsPerMillion},
		{*usage.OutputTotal, pricing.OutputMicrounitsPerMillion},
	} {
		if component.tokens != 0 && component.rate > (int64(^uint64(0)>>1)-weighted)/component.tokens {
			return 0, errors.New("AI estimated cost overflow")
		}
		weighted += component.tokens * component.rate
	}
	if weighted > int64(^uint64(0)>>1)-999_999 {
		return 0, errors.New("AI estimated cost overflow")
	}
	return (weighted + 999_999) / 1_000_000, nil
}

type ToolSetArtifact struct {
	Digest   artifact.Digest `json:"digest"`
	Manifest json.RawMessage `json:"manifest"`
}

func ResolveToolSet(toolSet ToolSet) (ToolSetArtifact, error) {
	if !toolSet.Valid() {
		return ToolSetArtifact{}, errors.New("AI tool set artifact is unavailable")
	}
	return ToolSetArtifact{Digest: toolSet.Digest(), Manifest: toolSet.Bytes()}, nil
}

func (a ToolSetArtifact) Open() (ToolSet, error) { return OpenToolSet(a.Manifest, a.Digest) }

type AgentStartRequest struct {
	AttemptID      string               `json:"attemptId"`
	Prompt         RenderedPrompt       `json:"prompt"`
	ToolSet        ToolSetArtifact      `json:"toolSet"`
	Limits         GenerationLimits     `json:"limits"`
	MaxParallelism int                  `json:"maxParallelism"`
	Retention      RetentionRequirement `json:"retention"`
}

func (r AgentStartRequest) Validate() error {
	toolSet, err := r.ToolSet.Open()
	if err != nil {
		return err
	}
	request := GenerateRequest{AttemptID: r.AttemptID, Prompt: r.Prompt, ToolSet: toolSet.Digest(), Limits: r.Limits, Retention: r.Retention}
	if err := request.Validate(); err != nil {
		return err
	}
	if r.MaxParallelism <= 0 || r.MaxParallelism > MaxAgentParallelism {
		return errors.New("invalid AI agent parallelism")
	}
	return nil
}

type ToolResult struct {
	CallID string          `json:"callId"`
	Name   string          `json:"name"`
	Value  json.RawMessage `json:"value"`
}

type AgentContinueRequest struct {
	AttemptID string       `json:"attemptId"`
	Results   []ToolResult `json:"results"`
}

func (r AgentContinueRequest) Validate() error {
	if !attemptIDPattern.MatchString(r.AttemptID) || len(r.Results) == 0 || len(r.Results) > MaxAgentParallelism {
		return errors.New("invalid AI agent continuation identity or result budget")
	}
	seen := make(map[string]struct{}, len(r.Results))
	for _, result := range r.Results {
		if !attemptIDPattern.MatchString(result.CallID) || !toolNamePattern.MatchString(result.Name) || len(result.Value) == 0 || len(result.Value) > MaxPromptBytes {
			return errors.New("invalid AI agent tool result")
		}
		if _, duplicate := seen[result.CallID]; duplicate {
			return errors.New("duplicate AI agent tool result")
		}
		seen[result.CallID] = struct{}{}
	}
	return nil
}

type ToolHandler func(context.Context, json.RawMessage) (json.RawMessage, error)

type ToolBinding struct {
	Name     string
	Approval artifact.Digest
	Handler  ToolHandler
}

type ToolExecutor struct {
	toolSet  ToolSet
	tools    map[string]ToolManifestDraft
	bindings map[string]ToolBinding
}

func NewToolExecutor(toolSet ToolSet, bindings []ToolBinding) (ToolExecutor, error) {
	if !toolSet.Valid() {
		return ToolExecutor{}, errors.New("AI tool executor requires an exact tool set")
	}
	tools := make(map[string]ToolManifestDraft)
	for _, tool := range toolSet.Machine().Tools {
		tools[tool.Name] = tool
	}
	resolved := make(map[string]ToolBinding, len(bindings))
	for _, binding := range bindings {
		tool, ok := tools[binding.Name]
		if !ok || binding.Handler == nil {
			return ToolExecutor{}, ErrAgentUnknownTool
		}
		if _, duplicate := resolved[binding.Name]; duplicate {
			return ToolExecutor{}, errors.New("duplicate AI tool binding")
		}
		if tool.Authority == ToolAuthorityCapability && binding.Approval != tool.Capability {
			return ToolExecutor{}, ErrAgentToolApproval
		}
		if tool.Authority == ToolAuthorityPure && binding.Approval != "" {
			return ToolExecutor{}, ErrAgentToolApproval
		}
		resolved[binding.Name] = binding
	}
	if len(resolved) != len(tools) {
		return ToolExecutor{}, ErrAgentUnknownTool
	}
	return ToolExecutor{toolSet: toolSet, tools: tools, bindings: resolved}, nil
}

func (e ToolExecutor) ToolSet() ToolSet { return e.toolSet }

func (e ToolExecutor) Execute(ctx context.Context, calls []ToolCall, maxParallelism int) ([]ToolResult, error) {
	if ctx == nil || !e.toolSet.Valid() || len(calls) == 0 || len(calls) > maxParallelism || maxParallelism <= 0 {
		return nil, ErrAgentBudgetExceeded
	}
	results := make([]ToolResult, 0, len(calls))
	seen := make(map[string]struct{}, len(calls))
	for _, call := range calls {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if _, duplicate := seen[call.CallID]; duplicate {
			return nil, fmt.Errorf("%w: duplicate call identity", ErrAgentToolSchema)
		}
		seen[call.CallID] = struct{}{}
		tool, ok := e.tools[call.Name]
		if !ok {
			return nil, ErrAgentUnknownTool
		}
		input, err := CompileStructuredOutput(call.Name+"_input", tool.InputSchema)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrAgentToolSchema, err)
		}
		arguments, err := input.ValidateValue(call.Arguments)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrAgentToolSchema, err)
		}
		value, err := e.bindings[call.Name].Handler(ctx, arguments)
		if err != nil {
			return nil, err
		}
		output, err := CompileStructuredOutput(call.Name+"_output", tool.OutputSchema)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrAgentToolSchema, err)
		}
		value, err = output.ValidateValue(value)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrAgentToolSchema, err)
		}
		results = append(results, ToolResult{CallID: call.CallID, Name: call.Name, Value: value})
	}
	return results, nil
}
