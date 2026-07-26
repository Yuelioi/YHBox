package noderuntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

func stateRead(_ nodes.Builtins) compiler.Adapter {
	return func(ctx context.Context, invocation compiler.Invocation) (_ compiler.AdapterResult, runErr error) {
		counters := map[string]int64{}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, compiler.AdapterAction{
				EffectID: nodes.StateReadEffectID, Action: "state.read", SummaryCode: "state.read", Counters: counters,
			}, nodes.StateReadFailedCode, runErr))
		}()
		binding, ok := invocation.State["state"]
		if !ok {
			return compiler.AdapterResult{}, stateFailure(nodes.StateReadFailedCode, errors.New("state read binding is missing"))
		}
		snapshot, err := binding.Read()
		if err != nil {
			return compiler.AdapterResult{}, stateFailure(nodes.StateReadFailedCode, err)
		}
		counters["revision"] = snapshot.Revision
		return compiler.AdapterResult{Outputs: map[string]datatype.ValueEnvelope{"result": snapshot.Value}}, nil
	}
}

func stateWrite(_ nodes.Builtins) compiler.Adapter {
	return func(ctx context.Context, invocation compiler.Invocation) (_ compiler.AdapterResult, runErr error) {
		counters := map[string]int64{}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, compiler.AdapterAction{
				EffectID: nodes.StateWriteEffectID, Action: "state.written", SummaryCode: "state.write", Counters: counters,
			}, nodes.StateWriteFailedCode, runErr))
		}()
		binding, bindingOK := invocation.State["state"]
		value, valueOK := invocation.Inputs["value"]
		if !bindingOK || !valueOK {
			return compiler.AdapterResult{}, stateFailure(nodes.StateWriteFailedCode, errors.New("state write binding or value is missing"))
		}
		snapshot, err := binding.Write(value)
		if err != nil {
			return compiler.AdapterResult{}, stateFailure(nodes.StateWriteFailedCode, err)
		}
		counters["revision"] = snapshot.Revision
		return compiler.AdapterResult{
			Outputs: map[string]datatype.ValueEnvelope{"result": snapshot.Value}, ExecOutputs: []string{"done"},
		}, nil
	}
}

func stateMetadata(builtins nodes.Builtins) compiler.Adapter {
	return func(ctx context.Context, invocation compiler.Invocation) (_ compiler.AdapterResult, runErr error) {
		counters := map[string]int64{}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, compiler.AdapterAction{
				EffectID: nodes.StateReadEffectID, Action: "state.metadata-read", SummaryCode: "state.metadata", Counters: counters,
			}, nodes.StateReadFailedCode, runErr))
		}()
		binding, ok := invocation.State["state"]
		if !ok {
			return compiler.AdapterResult{}, stateFailure(nodes.StateReadFailedCode, errors.New("state metadata binding is missing"))
		}
		snapshot, err := binding.Read()
		if err != nil {
			return compiler.AdapterResult{}, stateFailure(nodes.StateReadFailedCode, err)
		}
		counters["revision"] = snapshot.Revision
		revision, err := sealStateOutput(builtins, invocation, "revision", snapshot.Revision)
		if err != nil {
			return compiler.AdapterResult{}, stateFailure(nodes.StateReadFailedCode, err)
		}
		changedAt, err := sealStateOutput(builtins, invocation, "changed-at", snapshot.ChangedAt.UnixMilli())
		if err != nil {
			return compiler.AdapterResult{}, stateFailure(nodes.StateReadFailedCode, err)
		}
		return compiler.AdapterResult{Outputs: map[string]datatype.ValueEnvelope{"revision": revision, "changed-at": changedAt}}, nil
	}
}

func stateLastChange(builtins nodes.Builtins) compiler.Adapter {
	return func(ctx context.Context, invocation compiler.Invocation) (_ compiler.AdapterResult, runErr error) {
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, compiler.AdapterAction{
				EffectID: nodes.StateReadEffectID, Action: "state.last-change-read", SummaryCode: "state.last-change",
			}, nodes.StateReadFailedCode, runErr))
		}()
		binding, ok := invocation.State["state"]
		if !ok {
			return compiler.AdapterResult{}, stateFailure(nodes.StateReadFailedCode, errors.New("state last-change binding is missing"))
		}
		snapshot, err := binding.Read()
		if err != nil {
			return compiler.AdapterResult{}, stateFailure(nodes.StateReadFailedCode, err)
		}
		changedAt, err := sealStateOutput(builtins, invocation, "changed-at", snapshot.ChangedAt.UnixMilli())
		if err != nil {
			return compiler.AdapterResult{}, stateFailure(nodes.StateReadFailedCode, err)
		}
		return compiler.AdapterResult{Outputs: map[string]datatype.ValueEnvelope{"changed-at": changedAt}}, nil
	}
}

func stateIncrement(builtins nodes.Builtins) compiler.Adapter {
	return func(ctx context.Context, invocation compiler.Invocation) (_ compiler.AdapterResult, runErr error) {
		counters := map[string]int64{}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, compiler.AdapterAction{
				EffectID: nodes.StateWriteEffectID, Action: "state.incremented", SummaryCode: "state.increment", Counters: counters,
			}, nodes.StateUpdateFailedCode, runErr))
		}()
		binding, bindingOK := invocation.State["state"]
		delta, deltaOK := invocation.Inputs["delta"]
		if !bindingOK || !deltaOK {
			return compiler.AdapterResult{}, stateUpdateFailure(errors.New("state increment binding or delta is missing"))
		}
		snapshot, err := binding.Update(func(current datatype.ValueEnvelope) (datatype.ValueEnvelope, error) {
			if current.Type().Kind != datatype.ResolvedTypeRef || current.Type().Ref == nil {
				return datatype.ValueEnvelope{}, errors.New("state increment requires a concrete numeric slot")
			}
			switch current.Type().Ref.TypeID {
			case nodes.IntegerTypeID:
				var value, amount int64
				if err := json.Unmarshal(current.InlineJSON(), &value); err != nil {
					return datatype.ValueEnvelope{}, err
				}
				if err := json.Unmarshal(delta.InlineJSON(), &amount); err != nil {
					return datatype.ValueEnvelope{}, err
				}
				if amount > 0 && value > 9_007_199_254_740_991-amount || amount < 0 && value < -9_007_199_254_740_991-amount {
					return datatype.ValueEnvelope{}, errors.New("state increment exceeds the safe integer range")
				}
				return datatype.SealInlineJSON(builtins.Catalog, current.Type(), []byte(fmt.Sprint(value+amount)))
			case nodes.NumberTypeID:
				var value, amount float64
				if err := json.Unmarshal(current.InlineJSON(), &value); err != nil {
					return datatype.ValueEnvelope{}, err
				}
				if err := json.Unmarshal(delta.InlineJSON(), &amount); err != nil {
					return datatype.ValueEnvelope{}, err
				}
				result := value + amount
				if math.IsNaN(result) || math.IsInf(result, 0) {
					return datatype.ValueEnvelope{}, errors.New("state increment produced a non-finite number")
				}
				raw, err := json.Marshal(result)
				if err != nil {
					return datatype.ValueEnvelope{}, err
				}
				return datatype.SealInlineJSON(builtins.Catalog, current.Type(), raw)
			default:
				return datatype.ValueEnvelope{}, errors.New("state increment supports Integer and Number slots")
			}
		})
		if err != nil {
			return compiler.AdapterResult{}, stateUpdateFailure(err)
		}
		counters["revision"] = snapshot.Revision
		return compiler.AdapterResult{Outputs: map[string]datatype.ValueEnvelope{"result": snapshot.Value}, ExecOutputs: []string{"done"}}, nil
	}
}

func stateUpdateFailure(cause error) error {
	return &compiler.NodeFailure{Code: nodes.StateUpdateFailedCode, Output: "failed", Cause: cause}
}

func sealStateOutput(builtins nodes.Builtins, invocation compiler.Invocation, portID string, value any) (datatype.ValueEnvelope, error) {
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
