package nodes

import (
	"encoding/json"
	"fmt"

	"github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

const (
	ActivateWindowNodeID     = "https://schemas.yotta.dev/nodes/automation/activate-window"
	ActivateWindowEffectID   = "https://schemas.yotta.dev/effects/automation/activate-window/v1"
	StopTargetAppNodeID      = "https://schemas.yotta.dev/nodes/automation/stop-target-app"
	StopTargetAppEffectID    = "https://schemas.yotta.dev/effects/automation/stop-target-app/v1"
	CloseWindowNodeID        = "https://schemas.yotta.dev/nodes/automation/close-window"
	MoveResizeWindowNodeID   = "https://schemas.yotta.dev/nodes/automation/move-resize-window"
	MaximizeWindowNodeID     = "https://schemas.yotta.dev/nodes/automation/maximize-window"
	MinimizeWindowNodeID     = "https://schemas.yotta.dev/nodes/automation/minimize-window"
	RestoreWindowNodeID      = "https://schemas.yotta.dev/nodes/automation/restore-window"
	GetWindowStateNodeID     = "https://schemas.yotta.dev/nodes/automation/get-window-state"
	WaitWindowNodeID         = "https://schemas.yotta.dev/nodes/automation/wait-window"
	WaitWindowGoneNodeID     = "https://schemas.yotta.dev/nodes/automation/wait-window-gone"
	CloseWindowEffectID      = "https://schemas.yotta.dev/effects/automation/close-window/v1"
	MoveResizeWindowEffectID = "https://schemas.yotta.dev/effects/automation/move-resize-window/v1"
	MaximizeWindowEffectID   = "https://schemas.yotta.dev/effects/automation/maximize-window/v1"
	MinimizeWindowEffectID   = "https://schemas.yotta.dev/effects/automation/minimize-window/v1"
	RestoreWindowEffectID    = "https://schemas.yotta.dev/effects/automation/restore-window/v1"
	GetWindowStateEffectID   = "https://schemas.yotta.dev/effects/automation/get-window-state/v1"
	WaitWindowEffectID       = "https://schemas.yotta.dev/effects/automation/wait-window/v1"
	WaitWindowGoneEffectID   = "https://schemas.yotta.dev/effects/automation/wait-window-gone/v1"
)

func defineActivateWindowNode() (BuiltinDefinition, nodecontract.Contract, error) {
	return defineAutomationWindowNode(ActivateWindowNodeID, ActivateWindowEffectID, installed.OperationActivate, "automation.activate-window", "node.automation.activateWindow", "window-maximize", "configured-target/target-activation/v1", []string{installed.TargetKindDesktopWindow, installed.TargetKindAndroidDevice})
}

func defineStopTargetAppNode() (BuiltinDefinition, nodecontract.Contract, error) {
	return defineAutomationWindowNode(StopTargetAppNodeID, StopTargetAppEffectID, installed.OperationStopApp, "automation.stop-target-app", "node.automation.stopTargetApp", "player-stop", "configured-target/target-app-stop/v1", []string{installed.TargetKindAndroidDevice})
}

func defineDesktopWindowOperationNodes(stringRef, integerRef, booleanRef, durationRef datatype.TypeRef) ([]BuiltinDefinition, []nodecontract.Contract, error) {
	stringType := datatype.RefExpression(stringRef)
	integerType := datatype.RefExpression(integerRef)
	booleanType := datatype.RefExpression(booleanRef)
	durationType := datatype.RefExpression(durationRef)
	type spec struct {
		id, effectID, operation, entrypoint, titleKey, icon, conformance string
		inputs                                                           []nodecontract.DataInputPort
		outputs                                                          []nodecontract.DataOutputPort
		execOutputs                                                      []string
	}
	specs := []spec{
		{id: CloseWindowNodeID, effectID: CloseWindowEffectID, operation: installed.OperationCloseWindow, entrypoint: "automation.close-window", titleKey: "node.automation.closeWindow", icon: "x", conformance: "exact-target/window-close/v1", execOutputs: []string{"completed"}},
		{id: MoveResizeWindowNodeID, effectID: MoveResizeWindowEffectID, operation: installed.OperationMoveResizeWindow, entrypoint: "automation.move-resize-window", titleKey: "node.automation.moveResizeWindow", icon: "arrows-maximize", conformance: "exact-target/window-move-resize/v1", inputs: []nodecontract.DataInputPort{
			{ID: "x", Type: integerType, Required: true}, {ID: "y", Type: integerType, Required: true}, {ID: "width", Type: integerType, Required: true}, {ID: "height", Type: integerType, Required: true},
		}, execOutputs: []string{"completed"}},
		{id: MaximizeWindowNodeID, effectID: MaximizeWindowEffectID, operation: installed.OperationSetWindowState, entrypoint: "automation.maximize-window", titleKey: "node.automation.maximizeWindow", icon: "square", conformance: "exact-target/window-maximize/v1", execOutputs: []string{"completed"}},
		{id: MinimizeWindowNodeID, effectID: MinimizeWindowEffectID, operation: installed.OperationSetWindowState, entrypoint: "automation.minimize-window", titleKey: "node.automation.minimizeWindow", icon: "minus", conformance: "exact-target/window-minimize/v1", execOutputs: []string{"completed"}},
		{id: RestoreWindowNodeID, effectID: RestoreWindowEffectID, operation: installed.OperationSetWindowState, entrypoint: "automation.restore-window", titleKey: "node.automation.restoreWindow", icon: "copy", conformance: "exact-target/window-restore/v1", execOutputs: []string{"completed"}},
		{id: GetWindowStateNodeID, effectID: GetWindowStateEffectID, operation: installed.OperationGetWindowState, entrypoint: "automation.get-window-state", titleKey: "node.automation.getWindowState", icon: "info-circle", conformance: "exact-target/window-state-observation/v1", outputs: []nodecontract.DataOutputPort{
			{ID: "state", Type: stringType}, {ID: "foreground", Type: booleanType}, {ID: "x", Type: integerType}, {ID: "y", Type: integerType}, {ID: "width", Type: integerType}, {ID: "height", Type: integerType},
		}, execOutputs: []string{"completed"}},
		{id: WaitWindowNodeID, effectID: WaitWindowEffectID, operation: installed.OperationWaitWindow, entrypoint: "automation.wait-window", titleKey: "node.automation.waitWindow", icon: "clock", conformance: "exact-target/cooperative-window-wait/v1", inputs: []nodecontract.DataInputPort{{ID: "timeout", Type: durationType, Required: true, Default: rawDefault("10000")}}, execOutputs: []string{"found", "timeout"}},
		{id: WaitWindowGoneNodeID, effectID: WaitWindowGoneEffectID, operation: installed.OperationWaitWindowGone, entrypoint: "automation.wait-window-gone", titleKey: "node.automation.waitWindowGone", icon: "clock-off", conformance: "exact-target/cooperative-window-gone-wait/v1", inputs: []nodecontract.DataInputPort{{ID: "timeout", Type: durationType, Required: true, Default: rawDefault("10000")}}, execOutputs: []string{"gone", "timeout"}},
	}
	definitions := make([]BuiltinDefinition, 0, len(specs))
	contracts := make([]nodecontract.Contract, 0, len(specs))
	for _, item := range specs {
		contract, err := sealAutomationWindowContract(item.id, item.effectID, item.operation, item.titleKey, item.icon, item.inputs, item.outputs, item.execOutputs, []string{installed.TargetKindDesktopWindow})
		if err != nil {
			return nil, nil, err
		}
		definition, err := defineBuiltin(contract, item.entrypoint, "v1", item.conformance, nil)
		if err != nil {
			return nil, nil, err
		}
		definitions = append(definitions, definition)
		contracts = append(contracts, contract)
	}
	return definitions, contracts, nil
}

func defineAutomationWindowNode(nodeID, effectID, operation, entrypoint, titleKey, icon, conformance string, targetKinds []string) (BuiltinDefinition, nodecontract.Contract, error) {
	contract, err := sealAutomationWindowContract(nodeID, effectID, operation, titleKey, icon, nil, nil, []string{"completed"}, targetKinds)
	if err != nil {
		return BuiltinDefinition{}, nodecontract.Contract{}, err
	}
	definition, err := defineBuiltin(contract, entrypoint, "v1", conformance, nil)
	return definition, contract, err
}

func sealAutomationWindowContract(nodeID, effectID, operation, titleKey, icon string, inputs []nodecontract.DataInputPort, outputs []nodecontract.DataOutputPort, execOutputIDs, targetKinds []string) (nodecontract.Contract, error) {
	schemaID := nodeID + "/config"
	execOutputs := make([]nodecontract.SignalPort, 0, len(execOutputIDs))
	for _, id := range execOutputIDs {
		execOutputs = append(execOutputs, nodecontract.SignalPort{ID: id})
	}
	return nodecontract.Seal(nodecontract.Draft{Version: BuiltinNodeVersion,
		NodeTypeID: nodeID, ConfigSchemaRoot: schemaID,
		ConfigSchemaBundle: []datatype.SchemaResource{{ID: schemaID, Schema: json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object",
			"properties":{"slot":{"type":"string","minLength":1,"maxLength":128,"pattern":"^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$",
				"x-yotta-title-key":"node.automation.config.slot.title","x-yotta-description-key":"node.automation.config.slot.description"}},
			"required":["slot"],"additionalProperties":false
		}`, schemaID))}},
		Ports: nodecontract.PortSet{
			DataInputs: inputs, DataOutputs: outputs, ExecInputs: signalList("in"), ExecOutputs: execOutputs, ErrorOutputs: signalList("failed"),
		},
		Execution: nodecontract.ExecutionSpec{
			Class: nodecontract.ExecutionEffect, Effects: []nodecontract.EffectID{nodecontract.EffectID(effectID)}, Determinism: nodecontract.Recorded,
			Evaluation: nodecontract.EvaluationPush, Cache: nodecontract.CacheNone, Retry: nodecontract.RetryNever,
			Cancellation: nodecontract.CancellationCooperative, Timeout: nodecontract.TimeoutRequired,
		},
		Instruction:       nodecontract.Invoke(),
		ConfiguredTargets: automationTargetSpec("target", targetKinds...),
		Errors: []nodecontract.ErrorSpec{
			{Code: installed.CodeInvalidRequest, Category: "automation", RetryHint: false},
			{Code: installed.CodeTargetNotFound, Category: "automation", RetryHint: true},
			{Code: installed.CodeTargetAmbiguous, Category: "automation", RetryHint: false},
			{Code: installed.CodeWindowFailed, Category: "automation", RetryHint: true},
			{Code: installed.CodeUnsupportedHost, Category: "automation", RetryHint: false},
			{Code: installed.CodeContractViolation, Category: "automation", RetryHint: false},
		},
		ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
		Authoring: nodecontract.Authoring{
			TitleKey: titleKey + ".title", DescriptionKey: titleKey + ".description", Category: "automation",
			Tags: []string{"automation", "target", operation}, Icon: icon,
		},
	})
}
