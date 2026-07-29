package noderuntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/nodeadapter"
	"github.com/yottaapp/yotta/internal/nodes"
)

type LogEntry struct {
	Level        string
	Message      string
	GraphID      string
	NodeID       string
	InvocationID string
	Attempt      int
	Failure      *LogFailure
}

type LogFailure struct {
	Code         string `json:"code"`
	Category     string `json:"category"`
	RetryHint    bool   `json:"retryHint"`
	SourceNodeID string `json:"sourceNodeId"`
	SourcePortID string `json:"sourcePortId"`
	Attempt      int    `json:"attempt"`
}

type LogEmitter interface {
	EmitWorkflowLog(context.Context, LogEntry) error
}

type LogEmitterFunc func(context.Context, LogEntry) error

func (f LogEmitterFunc) EmitWorkflowLog(ctx context.Context, entry LogEntry) error {
	return f(ctx, entry)
}

func writeLog(emitter LogEmitter) nodeadapter.Adapter {
	return func(ctx context.Context, invocation nodeadapter.Invocation) (_ nodeadapter.AdapterResult, runErr error) {
		action := nodeadapter.AdapterAction{
			EffectID: nodes.LogWriteEffectID, Action: "observability.log-written", SummaryCode: "observability.log",
			Counters: map[string]int64{}, Facts: map[string]string{},
		}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, action, nodes.LogWriteFailed, runErr))
		}()
		message, _ := invocation.Config["message"].(string)
		if envelope, ok := invocation.Inputs["message"]; ok && len(envelope.InlineJSON()) != 0 {
			raw := envelope.InlineJSON()
			if err := json.Unmarshal(raw, &message); err != nil {
				message = string(raw)
			}
		}
		if utf8.RuneCountInString(message) > nodes.MaxObservabilityMessageRunes {
			return nodeadapter.AdapterResult{}, logFailure(nodes.LogContractError, errors.New("log message is invalid"))
		}
		level := "info"
		if configured, ok := invocation.Config["level"].(string); ok && configured != "" {
			level = configured
		}
		switch level {
		case "debug", "info", "warn", "error":
		default:
			return nodeadapter.AdapterResult{}, logFailure(nodes.LogContractError, fmt.Errorf("log level %q is invalid", level))
		}
		digest, err := artifact.Sum("yotta/workflow-log-message/v1", []byte(message))
		if err != nil {
			return nodeadapter.AdapterResult{}, logFailure(nodes.LogContractError, err)
		}
		action.Counters["message_bytes"] = int64(len(message))
		action.Facts["level"] = level
		action.Facts["message_digest"] = digest.String()
		entry := LogEntry{
			Level: level, Message: message, GraphID: invocation.GraphID, NodeID: invocation.NodeID,
			InvocationID: invocation.InvocationID, Attempt: invocation.Attempt,
		}
		if invocation.Trigger != nil && invocation.Trigger.Failure != nil {
			failure := invocation.Trigger.Failure
			entry.Failure = &LogFailure{
				Code: failure.Code, Category: failure.Category, RetryHint: failure.RetryHint,
				SourceNodeID: failure.SourceNodeID, SourcePortID: failure.SourcePortID, Attempt: failure.Attempt,
			}
		}
		if err := emitter.EmitWorkflowLog(ctx, entry); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nodeadapter.AdapterResult{}, err
			}
			return nodeadapter.AdapterResult{}, logFailure(nodes.LogWriteFailed, err)
		}
		return nodeadapter.AdapterResult{ExecOutputs: []string{"completed"}}, nil
	}
}

func throwFailure() nodeadapter.Adapter {
	return func(_ context.Context, invocation nodeadapter.Invocation) (nodeadapter.AdapterResult, error) {
		envelope, ok := invocation.Inputs["message"]
		if !ok || len(envelope.InlineJSON()) == 0 {
			return nodeadapter.AdapterResult{}, &nodeadapter.NodeFailure{Code: nodes.ControlThrown, Cause: errors.New("workflow explicitly failed")}
		}
		var message string
		if err := json.Unmarshal(envelope.InlineJSON(), &message); err != nil {
			return nodeadapter.AdapterResult{}, &nodeadapter.NodeFailure{Code: nodes.ControlThrown, Cause: errors.New("workflow explicitly failed")}
		}
		if message == "" {
			message = "workflow explicitly failed"
		}
		return nodeadapter.AdapterResult{}, &nodeadapter.NodeFailure{Code: nodes.ControlThrown, Cause: errors.New(message)}
	}
}

func logFailure(code string, cause error) error {
	return &nodeadapter.NodeFailure{Code: code, Output: "failed", Cause: cause}
}
