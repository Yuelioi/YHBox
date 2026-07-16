package noderuntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

func automationInput(nodeTypeID, operation string) compiler.Adapter {
	return func(ctx context.Context, invocation compiler.Invocation) (_ compiler.AdapterResult, runErr error) {
		effectID, ok := nodes.AutomationInputEffectID(nodeTypeID)
		if !ok {
			return compiler.AdapterResult{}, automationFailure(installed.CodeContractViolation, errors.New("automation input effect is not installed"))
		}
		action := compiler.AdapterAction{
			EffectID: effectID, Action: "automation." + operation, SummaryCode: "automation." + operation,
			Counters: map[string]int64{}, Facts: map[string]string{},
		}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, action, installed.CodeInputFailed, runErr))
		}()

		request, counters, err := automationInputRequest(invocation, operation)
		if err != nil {
			return compiler.AdapterResult{}, automationFailure(installed.CodeInvalidRequest, err)
		}
		for key, value := range counters {
			action.Counters[key] = value
		}
		payload, err := artifact.Marshal(request)
		if err != nil {
			return compiler.AdapterResult{}, automationFailure(installed.CodeContractViolation, err)
		}
		session := invocation.Sessions["target"]
		if session == nil {
			return compiler.AdapterResult{}, automationFailure(installed.CodeContractViolation, errors.New("automation input capability session is missing"))
		}
		handle, err := session.Open(ctx, []string{operation}, []byte(`{}`))
		if err != nil {
			return compiler.AdapterResult{}, mapAutomationFailure(err)
		}
		defer func() { runErr = errors.Join(runErr, session.Drop(context.WithoutCancel(ctx), handle)) }()
		raw, err := session.Invoke(ctx, handle, operation, payload)
		if err != nil {
			return compiler.AdapterResult{}, mapAutomationFailure(err)
		}
		if err := installed.OpenEffectResponse(raw); err != nil {
			return compiler.AdapterResult{}, mapAutomationFailure(err)
		}
		return compiler.AdapterResult{ExecOutputs: []string{"completed"}}, nil
	}
}

func automationInputRequest(invocation compiler.Invocation, operation string) (any, map[string]int64, error) {
	counters := map[string]int64{}
	switch operation {
	case installed.OperationClick:
		var point installed.Point
		var button string
		var duration int64
		if err := decodeAutomationInput(invocation, "point", &point); err != nil {
			return nil, nil, err
		}
		if err := decodeAutomationInput(invocation, "button", &button); err != nil {
			return nil, nil, err
		}
		if err := decodeAutomationInput(invocation, "hold-duration", &duration); err != nil {
			return nil, nil, err
		}
		counters["duration_ms"] = duration
		return installed.ClickRequest{Point: point, Button: button, DurationMilliseconds: duration}, counters, nil
	case installed.OperationMove:
		var point installed.Point
		if err := decodeAutomationInput(invocation, "point", &point); err != nil {
			return nil, nil, err
		}
		return installed.MoveRequest{Point: point}, counters, nil
	case installed.OperationScroll:
		var point installed.Point
		var notches int64
		var horizontal bool
		if err := decodeAutomationInput(invocation, "point", &point); err != nil {
			return nil, nil, err
		}
		if err := decodeAutomationInput(invocation, "notches", &notches); err != nil {
			return nil, nil, err
		}
		if err := decodeAutomationInput(invocation, "horizontal", &horizontal); err != nil {
			return nil, nil, err
		}
		counters["notches"] = notches
		if horizontal {
			counters["horizontal"] = 1
		}
		return installed.ScrollRequest{Point: point, Notches: notches, Horizontal: horizontal}, counters, nil
	case installed.OperationDrag:
		var from, to installed.Point
		var button string
		var duration int64
		if err := decodeAutomationInput(invocation, "from", &from); err != nil {
			return nil, nil, err
		}
		if err := decodeAutomationInput(invocation, "to", &to); err != nil {
			return nil, nil, err
		}
		if err := decodeAutomationInput(invocation, "button", &button); err != nil {
			return nil, nil, err
		}
		if err := decodeAutomationInput(invocation, "duration", &duration); err != nil {
			return nil, nil, err
		}
		counters["duration_ms"] = duration
		return installed.DragRequest{From: from, To: to, Button: button, DurationMilliseconds: duration}, counters, nil
	case installed.OperationMoveRelative:
		var deltaX, deltaY, duration int64
		if err := decodeAutomationInput(invocation, "delta-x", &deltaX); err != nil {
			return nil, nil, err
		}
		if err := decodeAutomationInput(invocation, "delta-y", &deltaY); err != nil {
			return nil, nil, err
		}
		if err := decodeAutomationInput(invocation, "duration", &duration); err != nil {
			return nil, nil, err
		}
		counters["duration_ms"] = duration
		return installed.RelativeMoveRequest{DeltaX: deltaX, DeltaY: deltaY, DurationMilliseconds: duration}, counters, nil
	case installed.OperationPressKeys:
		var keys []string
		var duration int64
		if err := decodeAutomationInput(invocation, "keys", &keys); err != nil {
			return nil, nil, err
		}
		if err := decodeAutomationInput(invocation, "hold-duration", &duration); err != nil {
			return nil, nil, err
		}
		counters["key_count"] = int64(len(keys))
		counters["duration_ms"] = duration
		return installed.PressKeysRequest{Keys: keys, DurationMilliseconds: duration}, counters, nil
	case installed.OperationTypeText:
		var value string
		if err := decodeAutomationInput(invocation, "text", &value); err != nil {
			return nil, nil, err
		}
		counters["text_bytes"] = int64(len(value))
		counters["text_runes"] = int64(utf8.RuneCountInString(value))
		return installed.TypeTextRequest{Text: value}, counters, nil
	default:
		return nil, nil, errors.New("automation input operation is not installed")
	}
}

func decodeAutomationInput(invocation compiler.Invocation, id string, target any) error {
	input, ok := invocation.Inputs[id]
	if !ok || len(input.InlineJSON()) == 0 {
		return fmt.Errorf("automation input %q is missing", id)
	}
	if err := json.Unmarshal(input.InlineJSON(), target); err != nil {
		return fmt.Errorf("decode automation input %q: %w", id, err)
	}
	return nil
}

func mapAutomationFailure(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	var failure *installed.Failure
	if errors.As(err, &failure) && failure.Code != "" {
		return automationFailure(failure.Code, err)
	}
	return automationFailure(installed.CodeContractViolation, err)
}

func automationFailure(code string, cause error) error {
	return &compiler.NodeFailure{Code: code, Output: "failed", Cause: fmt.Errorf("automation: %w", cause)}
}
