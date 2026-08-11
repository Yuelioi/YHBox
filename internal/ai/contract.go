// Package ai owns Yotta's provider-native model contract. It deliberately
// does not expose a generic Chat API: provider wire transcripts remain inside
// their adapters while stable generation outcomes cross the runtime boundary.
package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
)

const (
	MaxPromptBytes       = 1 << 20
	MaxImageInputBytes   = 1536 << 10
	MaxProviderRawBytes  = 1 << 20
	MaxOutputItems       = 1024
	MaxIdentifierBytes   = 128
	MaxProviderCodeBytes = 256
)

var attemptIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}$`)

type RetentionRequirement string

const (
	RetentionProviderDefault    RetentionRequirement = "provider-default"
	RetentionNoApplicationState RetentionRequirement = "no-application-state"
	RetentionZeroRequired       RetentionRequirement = "zero-retention-required"
)

type GenerationLimits struct {
	Temperature     *float64 `json:"temperature,omitempty"`
	MaxOutputTokens *int64   `json:"maxOutputTokens,omitempty"`
}

type ImageInput struct {
	MediaType string `json:"mediaType"`
	Data      []byte `json:"data"`
}

func (i ImageInput) Validate() error {
	if (i.MediaType != "image/jpeg" && i.MediaType != "image/png") || len(i.Data) == 0 || len(i.Data) > MaxImageInputBytes {
		return errors.New("AI image input must be a bounded JPEG or PNG")
	}
	return nil
}

type GenerateRequest struct {
	AttemptID string                `json:"attemptId"`
	Prompt    RenderedPrompt        `json:"prompt"`
	Image     *ImageInput           `json:"image,omitempty"`
	Output    *StructuredOutputSpec `json:"output,omitempty"`
	ToolSet   artifact.Digest       `json:"toolSet,omitempty"`
	Limits    GenerationLimits      `json:"limits"`
	Retention RetentionRequirement  `json:"retention"`
}

func (r GenerateRequest) Validate() error {
	if !attemptIDPattern.MatchString(r.AttemptID) {
		return errors.New("invalid AI generation identity or prompt budget")
	}
	if err := r.Prompt.Validate(); err != nil {
		return err
	}
	if r.Image != nil {
		if err := r.Image.Validate(); err != nil {
			return err
		}
	}
	if r.ToolSet != "" && !r.ToolSet.Valid() {
		return errors.New("invalid AI tool set identity")
	}
	if r.Limits.Temperature != nil && (*r.Limits.Temperature < 0 || *r.Limits.Temperature > 2) {
		return errors.New("AI temperature is outside the portable range")
	}
	if r.Limits.MaxOutputTokens != nil && (*r.Limits.MaxOutputTokens <= 0 || *r.Limits.MaxOutputTokens > 1_000_000) {
		return errors.New("AI output token limit is invalid")
	}
	switch r.Retention {
	case RetentionProviderDefault, RetentionNoApplicationState, RetentionZeroRequired:
	default:
		return errors.New("invalid AI retention requirement")
	}
	if r.Output != nil {
		if err := r.Output.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type OutputKind string

const (
	OutputText       OutputKind = "text"
	OutputStructured OutputKind = "structured"
	OutputToolCall   OutputKind = "tool-call"
	OutputRefusal    OutputKind = "refusal"
)

type TextOutput struct {
	Text string `json:"text"`
}

type StructuredOutput struct {
	Value json.RawMessage `json:"value"`
}

type ToolCall struct {
	CallID    string          `json:"callId"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type RefusalOutput struct {
	Reason string `json:"reason"`
}

// OutputItem is an exact tagged union. Exactly one payload must match Kind.
type OutputItem struct {
	Kind       OutputKind        `json:"kind"`
	Text       *TextOutput       `json:"text,omitempty"`
	Structured *StructuredOutput `json:"structured,omitempty"`
	ToolCall   *ToolCall         `json:"toolCall,omitempty"`
	Refusal    *RefusalOutput    `json:"refusal,omitempty"`
}

func (i OutputItem) Validate() error {
	payloads := 0
	for _, present := range []bool{i.Text != nil, i.Structured != nil, i.ToolCall != nil, i.Refusal != nil} {
		if present {
			payloads++
		}
	}
	if payloads != 1 {
		return errors.New("AI output item must contain exactly one payload")
	}
	switch i.Kind {
	case OutputText:
		if i.Text == nil {
			return errors.New("AI text output payload is missing")
		}
	case OutputStructured:
		if i.Structured == nil || len(i.Structured.Value) == 0 {
			return errors.New("AI structured output payload is missing")
		}
	case OutputToolCall:
		if i.ToolCall == nil || i.ToolCall.CallID == "" || i.ToolCall.Name == "" || len(i.ToolCall.Arguments) == 0 {
			return errors.New("AI tool call payload is invalid")
		}
	case OutputRefusal:
		if i.Refusal == nil {
			return errors.New("AI refusal payload is missing")
		}
	default:
		return errors.New("unknown AI output item kind")
	}
	return nil
}

type FinishKind string

const (
	FinishCompleted     FinishKind = "completed"
	FinishToolCalls     FinishKind = "tool-calls"
	FinishMaxOutput     FinishKind = "max-output"
	FinishContextLimit  FinishKind = "context-limit"
	FinishStopSequence  FinishKind = "stop-sequence"
	FinishRefusal       FinishKind = "refusal"
	FinishContentFilter FinishKind = "content-filter"
	FinishPaused        FinishKind = "paused"
	FinishCancelled     FinishKind = "cancelled"
	FinishFailed        FinishKind = "failed"
	FinishUnknown       FinishKind = "unknown"
)

type Finish struct {
	Kind              FinishKind `json:"kind"`
	RawProviderReason string     `json:"rawProviderReason,omitempty"`
}

type TokenUsage struct {
	InputTotal      *int64          `json:"inputTotal,omitempty"`
	InputUncached   *int64          `json:"inputUncached,omitempty"`
	CacheRead       *int64          `json:"cacheRead,omitempty"`
	CacheWrite      *int64          `json:"cacheWrite,omitempty"`
	OutputTotal     *int64          `json:"outputTotal,omitempty"`
	ReasoningOutput *int64          `json:"reasoningOutput,omitempty"`
	CostMicrounits  *int64          `json:"costMicrounits,omitempty"`
	ProviderExtras  json.RawMessage `json:"providerExtras,omitempty"`
}

type Cancellation struct {
	LocalStopped         bool   `json:"localStopped"`
	ProviderAcknowledged bool   `json:"providerAcknowledged"`
	ProviderStatus       string `json:"providerStatus,omitempty"`
}

type Outcome struct {
	Provider           ProviderKind `json:"provider"`
	RequestedModel     string       `json:"requestedModel"`
	ResolvedModel      string       `json:"resolvedModel"`
	ProviderRequestID  string       `json:"providerRequestId,omitempty"`
	ProviderResponseID string       `json:"providerResponseId,omitempty"`
	Items              []OutputItem `json:"items"`
	Finish             Finish       `json:"finish"`
	Usage              TokenUsage   `json:"usage"`
	Cancellation       Cancellation `json:"cancellation"`
}

func OpenOutcome(raw []byte) (Outcome, error) {
	var outcome Outcome
	if err := decodeExactJSON(raw, &outcome); err != nil {
		return Outcome{}, fmt.Errorf("decode AI outcome: %w", err)
	}
	if err := outcome.Validate(); err != nil {
		return Outcome{}, err
	}
	return outcome, nil
}

func (o Outcome) Validate() error {
	if o.Provider == "" || o.RequestedModel == "" || len(o.Items) > MaxOutputItems {
		return errors.New("invalid AI outcome identity or output budget")
	}
	for _, item := range o.Items {
		if err := item.Validate(); err != nil {
			return err
		}
	}
	switch o.Finish.Kind {
	case FinishCompleted, FinishToolCalls, FinishMaxOutput, FinishContextLimit, FinishStopSequence, FinishRefusal,
		FinishContentFilter, FinishPaused, FinishCancelled, FinishFailed, FinishUnknown:
	default:
		return errors.New("invalid AI finish kind")
	}
	return validateUsage(o.Usage)
}

func validateUsage(usage TokenUsage) error {
	for _, value := range []*int64{usage.InputTotal, usage.InputUncached, usage.CacheRead, usage.CacheWrite, usage.OutputTotal, usage.ReasoningOutput, usage.CostMicrounits} {
		if value != nil && *value < 0 {
			return errors.New("AI usage counters must be non-negative")
		}
	}
	if len(usage.ProviderExtras) > MaxProviderRawBytes {
		return errors.New("AI provider usage metadata exceeds byte budget")
	}
	return nil
}

type FailureStage string

const (
	FailureTransport  FailureStage = "transport"
	FailureHTTP       FailureStage = "http"
	FailureStream     FailureStage = "stream"
	FailureGeneration FailureStage = "generation"
	FailureContract   FailureStage = "contract"
)

type FailureClass string

const (
	FailureInvalidRequest  FailureClass = "invalid-request"
	FailureInvalidResponse FailureClass = "invalid-response"
	FailureAuthentication  FailureClass = "authentication"
	FailurePermission      FailureClass = "permission"
	FailureNotFound        FailureClass = "not-found"
	FailureConflict        FailureClass = "conflict"
	FailureRateLimit       FailureClass = "rate-limit"
	FailureOverloaded      FailureClass = "overloaded"
	FailureTimeout         FailureClass = "timeout"
	FailureServer          FailureClass = "server"
	FailureCancelled       FailureClass = "cancelled"
	FailureUnknown         FailureClass = "unknown"
)

type RetryDisposition string

const (
	RetryNever      RetryDisposition = "never"
	RetryAfterHint  RetryDisposition = "after-hint"
	RetryNewAttempt RetryDisposition = "new-attempt"
	RetryAmbiguous  RetryDisposition = "ambiguous"
)

// ProviderFailure is safe for control flow. Message and Raw are diagnostic
// material only and must be redacted before ordinary application logging.
type ProviderFailure struct {
	Stage             FailureStage     `json:"stage"`
	Class             FailureClass     `json:"class"`
	HTTPStatus        *int             `json:"httpStatus,omitempty"`
	ProviderCode      string           `json:"providerCode,omitempty"`
	ProviderRequestID string           `json:"providerRequestId,omitempty"`
	Message           string           `json:"message,omitempty"`
	RetryAfter        *time.Duration   `json:"retryAfter,omitempty"`
	Retry             RetryDisposition `json:"retry"`
	Raw               json.RawMessage  `json:"raw,omitempty"`
}

func (e *ProviderFailure) Error() string {
	if e == nil {
		return ""
	}
	if e.ProviderCode != "" {
		return fmt.Sprintf("AI provider failure: %s/%s", e.Class, e.ProviderCode)
	}
	return fmt.Sprintf("AI provider failure: %s", e.Class)
}

type Provider interface {
	Generate(context.Context, string, GenerateRequest) (Outcome, error)
}
