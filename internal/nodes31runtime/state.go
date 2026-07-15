package nodes31runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodes31"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

func stateRead(_ nodes31.Builtins) compiler.Adapter {
	return func(ctx context.Context, invocation compiler.Invocation) (_ compiler.AdapterResult, runErr error) {
		counters := map[string]int64{}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, compiler.AdapterAction{
				EffectID: nodes31.StateReadEffectID, Action: "state.read", SummaryCode: "state.read", Counters: counters,
			}, nodes31.StateReadFailedCode, runErr))
		}()
		binding, ok := invocation.State["state"]
		if !ok {
			return compiler.AdapterResult{}, stateFailure(nodes31.StateReadFailedCode, errors.New("state read binding is missing"))
		}
		snapshot, err := binding.Read()
		if err != nil {
			return compiler.AdapterResult{}, stateFailure(nodes31.StateReadFailedCode, err)
		}
		counters["revision"] = snapshot.Revision
		return compiler.AdapterResult{Outputs: map[string]datatype.ValueEnvelope{"result": snapshot.Value}}, nil
	}
}

func stateWrite(_ nodes31.Builtins) compiler.Adapter {
	return func(ctx context.Context, invocation compiler.Invocation) (_ compiler.AdapterResult, runErr error) {
		counters := map[string]int64{}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, compiler.AdapterAction{
				EffectID: nodes31.StateWriteEffectID, Action: "state.written", SummaryCode: "state.write", Counters: counters,
			}, nodes31.StateWriteFailedCode, runErr))
		}()
		binding, bindingOK := invocation.State["state"]
		value, valueOK := invocation.Inputs["value"]
		if !bindingOK || !valueOK {
			return compiler.AdapterResult{}, stateFailure(nodes31.StateWriteFailedCode, errors.New("state write binding or value is missing"))
		}
		snapshot, err := binding.Write(value)
		if err != nil {
			return compiler.AdapterResult{}, stateFailure(nodes31.StateWriteFailedCode, err)
		}
		counters["revision"] = snapshot.Revision
		return compiler.AdapterResult{
			Outputs: map[string]datatype.ValueEnvelope{"result": snapshot.Value}, ExecOutputs: []string{"done"},
		}, nil
	}
}

func stateMetadata(builtins nodes31.Builtins) compiler.Adapter {
	return func(ctx context.Context, invocation compiler.Invocation) (_ compiler.AdapterResult, runErr error) {
		counters := map[string]int64{}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, compiler.AdapterAction{
				EffectID: nodes31.StateReadEffectID, Action: "state.metadata-read", SummaryCode: "state.metadata", Counters: counters,
			}, nodes31.StateReadFailedCode, runErr))
		}()
		binding, ok := invocation.State["state"]
		if !ok {
			return compiler.AdapterResult{}, stateFailure(nodes31.StateReadFailedCode, errors.New("state metadata binding is missing"))
		}
		snapshot, err := binding.Read()
		if err != nil {
			return compiler.AdapterResult{}, stateFailure(nodes31.StateReadFailedCode, err)
		}
		counters["revision"] = snapshot.Revision
		revision, err := sealStateOutput(builtins, invocation, "revision", snapshot.Revision)
		if err != nil {
			return compiler.AdapterResult{}, stateFailure(nodes31.StateReadFailedCode, err)
		}
		changedAt, err := sealStateOutput(builtins, invocation, "changed-at", snapshot.ChangedAt.UnixMilli())
		if err != nil {
			return compiler.AdapterResult{}, stateFailure(nodes31.StateReadFailedCode, err)
		}
		return compiler.AdapterResult{Outputs: map[string]datatype.ValueEnvelope{"revision": revision, "changed-at": changedAt}}, nil
	}
}

func sealStateOutput(builtins nodes31.Builtins, invocation compiler.Invocation, portID string, value any) (datatype.ValueEnvelope, error) {
	resolved, ok := invocation.OutputTypes[portID]
	if !ok {
		return datatype.ValueEnvelope{}, fmt.Errorf("state output %q has no resolved type", portID)
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return datatype.ValueEnvelope{}, err
	}
	return datatype.SealInlineJSON(builtins.Catalog, resolved, raw)
}

func stateFailure(code string, cause error) error {
	return &compiler.NodeFailure{Code: code, Cause: cause}
}
