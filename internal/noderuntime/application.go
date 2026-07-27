package noderuntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/yottaapp/yotta/internal/appcontrol"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodeadapter"
	"github.com/yottaapp/yotta/internal/nodes"
)

func launchApplication() nodeadapter.Adapter { return applicationLifecycle(appcontrol.OperationLaunch) }
func terminateApplication(builtins nodes.Builtins) nodeadapter.Adapter {
	return applicationLifecycleWithOutput(builtins, appcontrol.OperationTerminate)
}

func applicationLifecycle(operation string) nodeadapter.Adapter {
	return func(ctx context.Context, invocation nodeadapter.Invocation) (_ nodeadapter.AdapterResult, runErr error) {
		action := applicationAction(operation)
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, action, appcontrol.CodeContractViolation, runErr))
		}()
		raw, err := invokeApplicationOperation(ctx, invocation, operation)
		if err != nil {
			return nodeadapter.AdapterResult{}, err
		}
		response, err := appcontrol.OpenLaunchResponse(raw)
		if err != nil {
			return nodeadapter.AdapterResult{}, mapApplicationFailure(err)
		}
		if response.ProcessID == 0 {
			return nodeadapter.AdapterResult{}, applicationFailure(appcontrol.CodeContractViolation, errors.New("application launch returned no process identity"))
		}
		action.Counters["launched"] = 1
		return nodeadapter.AdapterResult{ExecOutputs: []string{"completed"}}, nil
	}
}

func applicationLifecycleWithOutput(builtins nodes.Builtins, operation string) nodeadapter.Adapter {
	return func(ctx context.Context, invocation nodeadapter.Invocation) (_ nodeadapter.AdapterResult, runErr error) {
		action := applicationAction(operation)
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, action, appcontrol.CodeContractViolation, runErr))
		}()
		raw, err := invokeApplicationOperation(ctx, invocation, operation)
		if err != nil {
			return nodeadapter.AdapterResult{}, err
		}
		response, err := appcontrol.OpenTerminateResponse(raw)
		if err != nil {
			return nodeadapter.AdapterResult{}, mapApplicationFailure(err)
		}
		action.Counters["terminated_count"] = int64(response.TerminatedCount)
		resolved, ok := invocation.OutputTypes["terminated-count"]
		if !ok {
			return nodeadapter.AdapterResult{}, applicationFailure(appcontrol.CodeContractViolation, errors.New("application terminate output is unresolved"))
		}
		rawCount, err := json.Marshal(response.TerminatedCount)
		if err != nil {
			return nodeadapter.AdapterResult{}, applicationFailure(appcontrol.CodeContractViolation, err)
		}
		value, err := datatype.SealInlineJSON(builtins.Catalog, resolved, rawCount)
		if err != nil {
			return nodeadapter.AdapterResult{}, applicationFailure(appcontrol.CodeContractViolation, err)
		}
		return nodeadapter.AdapterResult{Outputs: map[string]datatype.ValueEnvelope{"terminated-count": value}, ExecOutputs: []string{"completed"}}, nil
	}
}

func invokeApplicationOperation(ctx context.Context, invocation nodeadapter.Invocation, operation string) ([]byte, error) {
	session := invocation.Sessions["application"]
	if session == nil {
		return nil, applicationFailure(appcontrol.CodeContractViolation, errors.New("application lifecycle capability session is missing"))
	}
	handle, err := session.Open(ctx, []string{operation}, []byte(`{}`))
	if err != nil {
		return nil, mapApplicationFailure(err)
	}
	raw, invokeErr := session.Invoke(ctx, handle, operation, []byte(`{}`))
	dropErr := session.Drop(context.WithoutCancel(ctx), handle)
	if err := errors.Join(invokeErr, dropErr); err != nil {
		return nil, mapApplicationFailure(err)
	}
	return raw, nil
}

func applicationAction(operation string) nodeadapter.AdapterAction {
	effect, action := nodes.LaunchApplicationEffectID, "application.launched"
	if operation == appcontrol.OperationTerminate {
		effect, action = nodes.TerminateApplicationEffectID, "application.terminated"
	}
	return nodeadapter.AdapterAction{EffectID: effect, Action: action, SummaryCode: "application." + operation, Counters: map[string]int64{}, Facts: map[string]string{}}
}

func mapApplicationFailure(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	var failure *appcontrol.Failure
	if errors.As(err, &failure) && failure.Code != "" {
		return applicationFailure(failure.Code, err)
	}
	return applicationFailure(appcontrol.CodeContractViolation, err)
}
func applicationFailure(code string, cause error) error {
	return &nodeadapter.NodeFailure{Code: code, Output: "failed", Cause: fmt.Errorf("application lifecycle: %w", cause)}
}
