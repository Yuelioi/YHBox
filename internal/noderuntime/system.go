package noderuntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

type LogEntry struct {
	Level        string
	Message      string
	GraphID      string
	NodeID       string
	InvocationID string
	Attempt      int
}

type LogEmitter interface {
	EmitWorkflowLog(context.Context, LogEntry) error
}

type LogEmitterFunc func(context.Context, LogEntry) error

func (f LogEmitterFunc) EmitWorkflowLog(ctx context.Context, entry LogEntry) error {
	return f(ctx, entry)
}

func writeLog(emitter LogEmitter) compiler.Adapter {
	return func(ctx context.Context, invocation compiler.Invocation) (_ compiler.AdapterResult, runErr error) {
		action := compiler.AdapterAction{
			EffectID: nodes.LogWriteEffectID, Action: "observability.log-written", SummaryCode: "observability.log",
			Counters: map[string]int64{}, Facts: map[string]string{},
		}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, action, nodes.LogWriteFailed, runErr))
		}()
		envelope, ok := invocation.Inputs["message"]
		if !ok || len(envelope.InlineJSON()) == 0 {
			return compiler.AdapterResult{}, logFailure(nodes.LogContractError, errors.New("log message is missing"))
		}
		var message string
		if err := json.Unmarshal(envelope.InlineJSON(), &message); err != nil || utf8.RuneCountInString(message) > nodes.MaxObservabilityMessageRunes {
			return compiler.AdapterResult{}, logFailure(nodes.LogContractError, errors.New("log message is invalid"))
		}
		level := "info"
		if configured, ok := invocation.Config["level"].(string); ok && configured != "" {
			level = configured
		}
		switch level {
		case "debug", "info", "warn", "error":
		default:
			return compiler.AdapterResult{}, logFailure(nodes.LogContractError, fmt.Errorf("log level %q is invalid", level))
		}
		digest, err := artifact.Sum("yotta/workflow-log-message/v1", []byte(message))
		if err != nil {
			return compiler.AdapterResult{}, logFailure(nodes.LogContractError, err)
		}
		action.Counters["message_bytes"] = int64(len(message))
		action.Facts["level"] = level
		action.Facts["message_digest"] = digest.String()
		if err := emitter.EmitWorkflowLog(ctx, LogEntry{
			Level: level, Message: message, GraphID: invocation.GraphID, NodeID: invocation.NodeID,
			InvocationID: invocation.InvocationID, Attempt: invocation.Attempt,
		}); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return compiler.AdapterResult{}, err
			}
			return compiler.AdapterResult{}, logFailure(nodes.LogWriteFailed, err)
		}
		return compiler.AdapterResult{ExecOutputs: []string{"completed"}}, nil
	}
}

func throwFailure() compiler.Adapter {
	return func(_ context.Context, invocation compiler.Invocation) (compiler.AdapterResult, error) {
		envelope, ok := invocation.Inputs["message"]
		if !ok || len(envelope.InlineJSON()) == 0 {
			return compiler.AdapterResult{}, &compiler.NodeFailure{Code: nodes.ControlThrown, Cause: errors.New("workflow explicitly failed")}
		}
		var message string
		if err := json.Unmarshal(envelope.InlineJSON(), &message); err != nil {
			return compiler.AdapterResult{}, &compiler.NodeFailure{Code: nodes.ControlThrown, Cause: errors.New("workflow explicitly failed")}
		}
		if message == "" {
			message = "workflow explicitly failed"
		}
		return compiler.AdapterResult{}, &compiler.NodeFailure{Code: nodes.ControlThrown, Cause: errors.New(message)}
	}
}

func logFailure(code string, cause error) error {
	return &compiler.NodeFailure{Code: code, Output: "failed", Cause: cause}
}
