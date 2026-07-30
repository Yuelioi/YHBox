package noderuntime

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodeadapter"
	"github.com/yottaapp/yotta/internal/nodes"
)

func activateWindow() nodeadapter.Adapter {
	return automationWindow(nodes.ActivateWindowNodeID, installed.OperationActivate, nodes.ActivateWindowEffectID, "automation.activate-window")
}

func getWindowState(builtins nodes.Builtins) nodeadapter.Adapter {
	return func(ctx context.Context, invocation nodeadapter.Invocation) (_ nodeadapter.AdapterResult, runErr error) {
		action := nodeadapter.AdapterAction{EffectID: nodes.GetWindowStateEffectID, Action: "automation.get-window-state", SummaryCode: "automation.get-window-state", Counters: map[string]int64{}, Facts: map[string]string{}}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, action, installed.CodeWindowFailed, runErr))
		}()
		handle, err := openConfiguredTarget(ctx, invocation, installed.KindWindow, []string{installed.OperationGetWindowState})
		if err != nil {
			return nodeadapter.AdapterResult{}, mapAutomationFailure(err)
		}
		defer func() { runErr = errors.Join(runErr, invocation.Targets.Drop(context.WithoutCancel(ctx), handle)) }()
		raw, err := invocation.Targets.Invoke(ctx, handle, installed.OperationGetWindowState, []byte(`{}`))
		if err != nil {
			return nodeadapter.AdapterResult{}, mapAutomationFailure(err)
		}
		response, err := installed.OpenWindowStateResponse(raw)
		if err != nil {
			return nodeadapter.AdapterResult{}, mapAutomationFailure(err)
		}
		values := map[string]any{
			"state": response.State, "foreground": response.Foreground,
			"x": response.X, "y": response.Y, "width": response.Width, "height": response.Height,
		}
		outputs := make(map[string]datatype.ValueEnvelope, len(values))
		for id, value := range values {
			resolved, ok := invocation.OutputTypes[id]
			if !ok {
				return nodeadapter.AdapterResult{}, automationFailure(installed.CodeContractViolation, errors.New("window state output type is missing"))
			}
			rawValue, err := json.Marshal(value)
			if err != nil {
				return nodeadapter.AdapterResult{}, automationFailure(installed.CodeContractViolation, err)
			}
			envelope, err := datatype.SealInlineJSON(builtins.Catalog, resolved, rawValue)
			if err != nil {
				return nodeadapter.AdapterResult{}, automationFailure(installed.CodeContractViolation, err)
			}
			outputs[id] = envelope
		}
		return nodeadapter.AdapterResult{Outputs: outputs, ExecOutputs: []string{"completed"}}, nil
	}
}

func stopTargetApp() nodeadapter.Adapter {
	return automationWindow(nodes.StopTargetAppNodeID, installed.OperationStopApp, nodes.StopTargetAppEffectID, "automation.stop-target-app")
}

func automationWindow(nodeID, operation, effectID, actionName string) nodeadapter.Adapter {
	return func(ctx context.Context, invocation nodeadapter.Invocation) (_ nodeadapter.AdapterResult, runErr error) {
		action := nodeadapter.AdapterAction{
			EffectID: effectID, Action: actionName, SummaryCode: actionName,
			Counters: map[string]int64{}, Facts: map[string]string{},
		}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, action, installed.CodeWindowFailed, runErr))
		}()

		handle, err := openConfiguredTarget(ctx, invocation, installed.KindWindow, []string{operation})
		if err != nil {
			return nodeadapter.AdapterResult{}, mapAutomationFailure(err)
		}
		defer func() { runErr = errors.Join(runErr, invocation.Targets.Drop(context.WithoutCancel(ctx), handle)) }()
		request, err := automationWindowRequest(invocation, nodeID, operation)
		if err != nil {
			return nodeadapter.AdapterResult{}, automationFailure(installed.CodeInvalidRequest, err)
		}
		if wait, ok := request.(installed.WaitWindowRequest); ok {
			action.Counters["timeout_ms"] = wait.TimeoutMilliseconds
		}
		payload, err := artifact.Marshal(request)
		if err != nil {
			return nodeadapter.AdapterResult{}, automationFailure(installed.CodeContractViolation, err)
		}
		raw, err := invocation.Targets.Invoke(ctx, handle, operation, payload)
		if err != nil {
			return nodeadapter.AdapterResult{}, mapAutomationFailure(err)
		}
		if operation == installed.OperationWaitWindow || operation == installed.OperationWaitWindowGone {
			response, err := installed.OpenWaitWindowResponse(raw)
			if err != nil {
				return nodeadapter.AdapterResult{}, mapAutomationFailure(err)
			}
			if !response.Matched {
				return nodeadapter.AdapterResult{ExecOutputs: []string{"timeout"}}, nil
			}
			if operation == installed.OperationWaitWindowGone {
				return nodeadapter.AdapterResult{ExecOutputs: []string{"gone"}}, nil
			}
			return nodeadapter.AdapterResult{ExecOutputs: []string{"found"}}, nil
		}
		if err := installed.OpenEffectResponse(raw); err != nil {
			return nodeadapter.AdapterResult{}, mapAutomationFailure(err)
		}
		return nodeadapter.AdapterResult{ExecOutputs: []string{"completed"}}, nil
	}
}

func automationWindowRequest(invocation nodeadapter.Invocation, nodeID, operation string) (any, error) {
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
