package nodes31runtime

import (
	"context"
	"errors"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/nodes31"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

func activateWindow() compiler.Adapter {
	return func(ctx context.Context, invocation compiler.Invocation) (_ compiler.AdapterResult, runErr error) {
		action := compiler.AdapterAction{
			EffectID: nodes31.ActivateWindowEffectID, Action: "automation.activate-window", SummaryCode: "automation.activate-window",
			Counters: map[string]int64{}, Facts: map[string]string{},
		}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, action, installed.CodeWindowFailed, runErr))
		}()

		session := invocation.Sessions["target"]
		if session == nil {
			return compiler.AdapterResult{}, automationFailure(installed.CodeContractViolation, errors.New("automation window capability session is missing"))
		}
		handle, err := session.Open(ctx, []string{installed.OperationActivate}, []byte(`{}`))
		if err != nil {
			return compiler.AdapterResult{}, mapAutomationFailure(err)
		}
		defer func() { runErr = errors.Join(runErr, session.Drop(context.WithoutCancel(ctx), handle)) }()
		payload, err := artifact.Marshal(struct{}{})
		if err != nil {
			return compiler.AdapterResult{}, automationFailure(installed.CodeContractViolation, err)
		}
		raw, err := session.Invoke(ctx, handle, installed.OperationActivate, payload)
		if err != nil {
			return compiler.AdapterResult{}, mapAutomationFailure(err)
		}
		if err := installed.OpenEffectResponse(raw); err != nil {
			return compiler.AdapterResult{}, mapAutomationFailure(err)
		}
		return compiler.AdapterResult{ExecOutputs: []string{"completed"}}, nil
	}
}
