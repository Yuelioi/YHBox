package noderuntime

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

func activateWindow() compiler.Adapter {
	return automationWindow(nodes.ActivateWindowNodeID, installed.OperationActivate, nodes.ActivateWindowEffectID, "automation.activate-window")
}

func getWindowState(builtins nodes.Builtins) compiler.Adapter {
	return func(ctx context.Context, invocation compiler.Invocation) (_ compiler.AdapterResult, runErr error) {
		action := compiler.AdapterAction{EffectID: nodes.GetWindowStateEffectID, Action: "automation.get-window-state", SummaryCode: "automation.get-window-state", Counters: map[string]int64{}, Facts: map[string]string{}}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, action, installed.CodeWindowFailed, runErr))
		}()
		session := invocation.Sessions["target"]
		if session == nil {
			return compiler.AdapterResult{}, automationFailure(installed.CodeContractViolation, errors.New("automation window capability session is missing"))
		}
		handle, err := session.Open(ctx, []string{installed.OperationGetWindowState}, []byte(`{}`))
		if err != nil {
			return compiler.AdapterResult{}, mapAutomationFailure(err)
		}
		defer func() { runErr = errors.Join(runErr, session.Drop(context.WithoutCancel(ctx), handle)) }()
		raw, err := session.Invoke(ctx, handle, installed.OperationGetWindowState, []byte(`{}`))
		if err != nil {
			return compiler.AdapterResult{}, mapAutomationFailure(err)
		}
		response, err := installed.OpenWindowStateResponse(raw)
		if err != nil {
			return compiler.AdapterResult{}, mapAutomationFailure(err)
		}
		values := map[string]any{
			"state": response.State, "foreground": response.Foreground,
			"x": response.X, "y": response.Y, "width": response.Width, "height": response.Height,
		}
		outputs := make(map[string]datatype.ValueEnvelope, len(values))
		for id, value := range values {
			resolved, ok := invocation.OutputTypes[id]
			if !ok {
				return compiler.AdapterResult{}, automationFailure(installed.CodeContractViolation, errors.New("window state output type is missing"))
			}
			rawValue, err := json.Marshal(value)
			if err != nil {
				return compiler.AdapterResult{}, automationFailure(installed.CodeContractViolation, err)
			}
			envelope, err := datatype.SealInlineJSON(builtins.Catalog, resolved, rawValue)
			if err != nil {
				return compiler.AdapterResult{}, automationFailure(installed.CodeContractViolation, err)
			}
			outputs[id] = envelope
		}
		return compiler.AdapterResult{Outputs: outputs, ExecOutputs: []string{"completed"}}, nil
	}
}

func stopTargetApp() compiler.Adapter {
	return automationWindow(nodes.StopTargetAppNodeID, installed.OperationStopApp, nodes.StopTargetAppEffectID, "automation.stop-target-app")
}

func automationWindow(nodeID, operation, effectID, actionName string) compiler.Adapter {
	return func(ctx context.Context, invocation compiler.Invocation) (_ compiler.AdapterResult, runErr error) {
		action := compiler.AdapterAction{
			EffectID: effectID, Action: actionName, SummaryCode: actionName,
			Counters: map[string]int64{}, Facts: map[string]string{},
		}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, action, installed.CodeWindowFailed, runErr))
		}()

		session := invocation.Sessions["target"]
		if session == nil {
			return compiler.AdapterResult{}, automationFailure(installed.CodeContractViolation, errors.New("automation window capability session is missing"))
		}
		handle, err := session.Open(ctx, []string{operation}, []byte(`{}`))
		if err != nil {
			return compiler.AdapterResult{}, mapAutomationFailure(err)
		}
		defer func() { runErr = errors.Join(runErr, session.Drop(context.WithoutCancel(ctx), handle)) }()
		request, err := automationWindowRequest(invocation, nodeID, operation)
		if err != nil {
			return compiler.AdapterResult{}, automationFailure(installed.CodeInvalidRequest, err)
		}
		if wait, ok := request.(installed.WaitWindowRequest); ok {
			action.Counters["timeout_ms"] = wait.TimeoutMilliseconds
		}
		payload, err := artifact.Marshal(request)
		if err != nil {
			return compiler.AdapterResult{}, automationFailure(installed.CodeContractViolation, err)
		}
		raw, err := session.Invoke(ctx, handle, operation, payload)
		if err != nil {
			return compiler.AdapterResult{}, mapAutomationFailure(err)
		}
		if operation == installed.OperationWaitWindow || operation == installed.OperationWaitWindowGone {
			response, err := installed.OpenWaitWindowResponse(raw)
			if err != nil {
				return compiler.AdapterResult{}, mapAutomationFailure(err)
			}
			if !response.Matched {
				return compiler.AdapterResult{ExecOutputs: []string{"timeout"}}, nil
			}
			if operation == installed.OperationWaitWindowGone {
				return compiler.AdapterResult{ExecOutputs: []string{"gone"}}, nil
			}
			return compiler.AdapterResult{ExecOutputs: []string{"found"}}, nil
		}
		if err := installed.OpenEffectResponse(raw); err != nil {
			return compiler.AdapterResult{}, mapAutomationFailure(err)
		}
		return compiler.AdapterResult{ExecOutputs: []string{"completed"}}, nil
	}
}

func automationWindowRequest(invocation compiler.Invocation, nodeID, operation string) (any, error) {
	switch operation {
	case installed.OperationActivate, installed.OperationCloseWindow, installed.OperationGetWindowState, installed.OperationStopApp:
		return struct{}{}, nil
	case installed.OperationMoveResizeWindow:
		var x, y, width, height int64
		for id, target := range map[string]*int64{"x": &x, "y": &y, "width": &width, "height": &height} {
			if err := decodeAutomationInput(invocation, id, target); err != nil {
				return nil, err
			}
		}
		return installed.MoveResizeWindowRequest{X: x, Y: y, Width: width, Height: height}, nil
	case installed.OperationSetWindowState:
		states := map[string]string{
			nodes.MaximizeWindowNodeID: "maximize", nodes.MinimizeWindowNodeID: "minimize", nodes.RestoreWindowNodeID: "restore",
		}
		state, ok := states[nodeID]
		if !ok {
			return nil, errors.New("automation window state node is not installed")
		}
		return installed.SetWindowStateRequest{State: state}, nil
	case installed.OperationWaitWindow, installed.OperationWaitWindowGone:
		var timeout int64
		if err := decodeAutomationInput(invocation, "timeout", &timeout); err != nil {
			return nil, err
		}
		return installed.WaitWindowRequest{TimeoutMilliseconds: timeout}, nil
	default:
		return nil, errors.New("automation window operation is not installed")
	}
}
