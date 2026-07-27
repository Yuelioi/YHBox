package noderuntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/yottaapp/yotta/internal/nodeadapter"
	"github.com/yottaapp/yotta/internal/nodes"
)

func branch() nodeadapter.Adapter {
	return func(_ context.Context, invocation nodeadapter.Invocation) (nodeadapter.AdapterResult, error) {
		input, ok := invocation.Inputs["condition"]
		if !ok {
			return nodeadapter.AdapterResult{}, errors.New("branch condition is missing")
		}
		var condition bool
		if err := json.Unmarshal(input.InlineJSON(), &condition); err != nil {
			return nodeadapter.AdapterResult{}, fmt.Errorf("decode branch condition: %w", err)
		}
		selected := "false"
		if condition {
			selected = "true"
		}
		return nodeadapter.AdapterResult{ExecOutputs: []string{selected}}, nil
	}
}

func delay() nodeadapter.Adapter {
	return func(ctx context.Context, invocation nodeadapter.Invocation) (_ nodeadapter.AdapterResult, runErr error) {
		counters := map[string]int64{}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, nodeadapter.AdapterAction{
				EffectID: nodes.DelayWaitEffectID, Action: "control.delay-completed", SummaryCode: "control.delay", Counters: counters,
			}, nodes.DelayFailedCode, runErr))
		}()
		duration, err := integerInput(invocation, "duration-milliseconds")
		if err != nil || duration < 0 || duration > nodes.MaxDelayMilliseconds {
			return nodeadapter.AdapterResult{}, delayFailure(errors.Join(errors.New("delay duration is outside its supported range"), err))
		}
		if invocation.Wait == nil || invocation.EmitStatus == nil {
			return nodeadapter.AdapterResult{}, delayFailure(errors.New("delay host functions are missing"))
		}
		counters["duration"] = duration
		if err := invocation.EmitStatus(ctx, nodes.DelayWaitingStatus, counters); err != nil {
			return nodeadapter.AdapterResult{}, delayFailure(err)
		}
		if err := invocation.Wait(ctx, time.Duration(duration)*time.Millisecond); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nodeadapter.AdapterResult{}, err
			}
			return nodeadapter.AdapterResult{}, delayFailure(err)
		}
		return nodeadapter.AdapterResult{ExecOutputs: []string{"done"}}, nil
	}
}

func delayFailure(err error) error {
	return &nodeadapter.NodeFailure{Code: nodes.DelayFailedCode, Output: "failed", Cause: err}
}

func endBranch() nodeadapter.Adapter {
	return func(context.Context, nodeadapter.Invocation) (nodeadapter.AdapterResult, error) {
		return nodeadapter.AdapterResult{}, nil
	}
}
