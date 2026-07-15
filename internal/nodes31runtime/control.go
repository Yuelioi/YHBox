package nodes31runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/yottaapp/yotta/internal/nodes31"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

func runStarted() compiler.Adapter {
	return func(context.Context, compiler.Invocation) (compiler.AdapterResult, error) {
		return compiler.AdapterResult{ExecOutputs: []string{"started"}}, nil
	}
}

func branch() compiler.Adapter {
	return func(_ context.Context, invocation compiler.Invocation) (compiler.AdapterResult, error) {
		input, ok := invocation.Inputs["condition"]
		if !ok {
			return compiler.AdapterResult{}, errors.New("branch condition is missing")
		}
		var condition bool
		if err := json.Unmarshal(input.InlineJSON(), &condition); err != nil {
			return compiler.AdapterResult{}, fmt.Errorf("decode branch condition: %w", err)
		}
		selected := "false"
		if condition {
			selected = "true"
		}
		return compiler.AdapterResult{ExecOutputs: []string{selected}}, nil
	}
}

func delay() compiler.Adapter {
	return func(ctx context.Context, invocation compiler.Invocation) (_ compiler.AdapterResult, runErr error) {
		counters := map[string]int64{}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, compiler.AdapterAction{
				EffectID: nodes31.DelayWaitEffectID, Action: "control.delay-completed", SummaryCode: "control.delay", Counters: counters,
			}, nodes31.DelayFailedCode, runErr))
		}()
		duration, err := integerInput(invocation, "duration-milliseconds")
		if err != nil || duration < 0 || duration > nodes31.MaxDelayMilliseconds {
			return compiler.AdapterResult{}, errors.Join(errors.New("delay duration is outside its supported range"), err)
		}
		if invocation.Wait == nil || invocation.EmitStatus == nil {
			return compiler.AdapterResult{}, errors.New("delay host functions are missing")
		}
		counters["duration"] = duration
		if err := invocation.EmitStatus(ctx, nodes31.DelayWaitingStatus, counters); err != nil {
			return compiler.AdapterResult{}, err
		}
		if err := invocation.Wait(ctx, time.Duration(duration)*time.Millisecond); err != nil {
			return compiler.AdapterResult{}, err
		}
		return compiler.AdapterResult{ExecOutputs: []string{"done"}}, nil
	}
}

func endBranch() compiler.Adapter {
	return func(context.Context, compiler.Invocation) (compiler.AdapterResult, error) {
		return compiler.AdapterResult{}, nil
	}
}
