package noderuntime

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strconv"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodeadapter"
	"github.com/yottaapp/yotta/internal/nodeinstance"
	"github.com/yottaapp/yotta/internal/nodes"
)

func typedSwitch() nodeadapter.Adapter {
	return func(_ context.Context, invocation nodeadapter.Invocation) (nodeadapter.AdapterResult, error) {
		value, ok := invocation.Inputs["value"]
		if !ok || len(value.InlineJSON()) == 0 {
			return nodeadapter.AdapterResult{}, &nodeadapter.NodeFailure{Code: nodes.SwitchFailedCode, Output: "failed", Cause: errors.New("switch value is missing")}
		}
		caseCount, err := nodeinstance.SwitchCaseCount(invocation.Config)
		if err != nil {
			return nodeadapter.AdapterResult{}, &nodeadapter.NodeFailure{Code: nodes.SwitchFailedCode, Output: "failed", Cause: err}
		}
		for index := 1; index <= caseCount; index++ {
			id := "case-" + strconv.Itoa(index)
			candidate, exists := invocation.Inputs[id]
			if !exists {
				continue
			}
			if candidate.Type().Validate() != nil || candidate.Representation() != datatype.RepresentationInlineJSON || !reflect.DeepEqual(candidate.Type(), value.Type()) || !bytes.Equal(candidate.InlineJSON(), value.InlineJSON()) {
				if !reflect.DeepEqual(candidate.Type(), value.Type()) {
					return nodeadapter.AdapterResult{}, &nodeadapter.NodeFailure{Code: nodes.SwitchFailedCode, Output: "failed", Cause: errors.New("switch case type does not match the value")}
				}
				continue
			}
			return nodeadapter.AdapterResult{ExecOutputs: []string{id}}, nil
		}
		return nodeadapter.AdapterResult{ExecOutputs: []string{"default"}}, nil
	}
}

func stopwatch(builtins nodes.Builtins, nodeTypeID string) nodeadapter.Adapter {
	action := map[string]string{
		nodes.StopwatchStartNodeID: "start",
		nodes.StopwatchReadNodeID:  "read",
		nodes.StopwatchStopNodeID:  "stop",
	}[nodeTypeID]
	return func(ctx context.Context, invocation nodeadapter.Invocation) (_ nodeadapter.AdapterResult, runErr error) {
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, nodeadapter.AdapterAction{
				EffectID: nodes.StopwatchEffectID, Action: "time.stopwatch", SummaryCode: action,
			}, nodes.StopwatchFailedCode, runErr))
		}()
		if invocation.ObservedAt.IsZero() {
			return nodeadapter.AdapterResult{}, stopwatchFailure(errors.New("stopwatch observation time is missing"))
		}
		now := invocation.ObservedAt.UnixMilli()
		if nodeTypeID == nodes.StopwatchStartNodeID {
			value, err := sealStateOutput(builtins, invocation, "started-at", now)
			if err != nil {
				return nodeadapter.AdapterResult{}, stopwatchFailure(err)
			}
			return nodeadapter.AdapterResult{Outputs: map[string]datatype.ValueEnvelope{"started-at": value}, ExecOutputs: []string{"completed"}}, nil
		}
		started, err := integerInput(invocation, "started-at")
		if err != nil || started < 0 || started > now {
			return nodeadapter.AdapterResult{}, stopwatchFailure(errors.Join(err, errors.New("stopwatch start instant is invalid")))
		}
		elapsed := now - started
		if elapsed < 0 || elapsed > nodes.StopwatchMaximumElapsed {
			return nodeadapter.AdapterResult{}, stopwatchFailure(errors.New("stopwatch elapsed value exceeds its supported range"))
		}
		value, err := sealStateOutput(builtins, invocation, "elapsed", elapsed)
		if err != nil {
			return nodeadapter.AdapterResult{}, stopwatchFailure(err)
		}
		result := nodeadapter.AdapterResult{Outputs: map[string]datatype.ValueEnvelope{"elapsed": value}}
		if nodeTypeID == nodes.StopwatchStopNodeID {
			result.ExecOutputs = []string{"completed"}
		}
		return result, nil
	}
}

func stopwatchFailure(cause error) error {
	return &nodeadapter.NodeFailure{Code: nodes.StopwatchFailedCode, Output: "failed", Cause: cause}
}
