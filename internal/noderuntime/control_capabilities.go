package noderuntime

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

func typedSwitch() compiler.Adapter {
	return func(_ context.Context, invocation compiler.Invocation) (compiler.AdapterResult, error) {
		value, ok := invocation.Inputs["value"]
		if !ok || len(value.InlineJSON()) == 0 {
			return compiler.AdapterResult{}, &compiler.NodeFailure{Code: nodes.SwitchFailedCode, Output: "failed", Cause: errors.New("switch value is missing")}
		}
		for index := 1; index <= nodes.SwitchCaseCount; index++ {
			id := fmt.Sprintf("case-%d", index)
			candidate, exists := invocation.Inputs[id]
			if !exists {
				continue
			}
			if candidate.Type().Validate() != nil || candidate.Representation() != datatype.RepresentationInlineJSON || !reflect.DeepEqual(candidate.Type(), value.Type()) || !bytes.Equal(candidate.InlineJSON(), value.InlineJSON()) {
				if !reflect.DeepEqual(candidate.Type(), value.Type()) {
					return compiler.AdapterResult{}, &compiler.NodeFailure{Code: nodes.SwitchFailedCode, Output: "failed", Cause: errors.New("switch case type does not match the value")}
				}
				continue
			}
			return compiler.AdapterResult{ExecOutputs: []string{id}}, nil
		}
		return compiler.AdapterResult{ExecOutputs: []string{"default"}}, nil
	}
}

func stopwatch(builtins nodes.Builtins, nodeTypeID string) compiler.Adapter {
	action := map[string]string{
		nodes.StopwatchStartNodeID: "start",
		nodes.StopwatchReadNodeID:  "read",
		nodes.StopwatchStopNodeID:  "stop",
	}[nodeTypeID]
	return func(ctx context.Context, invocation compiler.Invocation) (_ compiler.AdapterResult, runErr error) {
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, compiler.AdapterAction{
				EffectID: nodes.StopwatchEffectID, Action: "time.stopwatch", SummaryCode: action,
			}, nodes.StopwatchFailedCode, runErr))
		}()
		if invocation.ObservedAt.IsZero() {
			return compiler.AdapterResult{}, stopwatchFailure(errors.New("stopwatch observation time is missing"))
		}
		now := invocation.ObservedAt.UnixMilli()
		if nodeTypeID == nodes.StopwatchStartNodeID {
			value, err := sealStateOutput(builtins, invocation, "started-at", now)
			if err != nil {
				return compiler.AdapterResult{}, stopwatchFailure(err)
			}
			return compiler.AdapterResult{Outputs: map[string]datatype.ValueEnvelope{"started-at": value}, ExecOutputs: []string{"completed"}}, nil
		}
		started, err := integerInput(invocation, "started-at")
		if err != nil || started < 0 || started > now {
			return compiler.AdapterResult{}, stopwatchFailure(errors.Join(err, errors.New("stopwatch start instant is invalid")))
		}
		elapsed := now - started
		if elapsed < 0 || elapsed > nodes.StopwatchMaximumElapsed {
			return compiler.AdapterResult{}, stopwatchFailure(errors.New("stopwatch elapsed value exceeds its supported range"))
		}
		value, err := sealStateOutput(builtins, invocation, "elapsed", elapsed)
		if err != nil {
			return compiler.AdapterResult{}, stopwatchFailure(err)
		}
		result := compiler.AdapterResult{Outputs: map[string]datatype.ValueEnvelope{"elapsed": value}}
		if nodeTypeID == nodes.StopwatchStopNodeID {
			result.ExecOutputs = []string{"completed"}
		}
		return result, nil
	}
}

func stopwatchFailure(cause error) error {
	return &compiler.NodeFailure{Code: nodes.StopwatchFailedCode, Output: "failed", Cause: cause}
}
