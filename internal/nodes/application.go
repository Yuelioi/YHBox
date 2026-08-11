package nodes

import (
	"encoding/json"
	"fmt"

	"github.com/yottaapp/yotta/internal/appcontrol"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

const (
	LaunchApplicationNodeID      = "https://schemas.yotta.dev/nodes/application/launch"
	TerminateApplicationNodeID   = "https://schemas.yotta.dev/nodes/application/terminate"
	LaunchApplicationEffectID    = "https://schemas.yotta.dev/effects/application/launch/v1"
	TerminateApplicationEffectID = "https://schemas.yotta.dev/effects/application/terminate/v1"
)

func defineApplicationNodes(integerRef datatype.TypeRef) ([]BuiltinDefinition, []nodecontract.Contract, error) {
	launch, err := sealApplicationNode(LaunchApplicationNodeID, "application.launch", "node.application.launch", "rocket", appcontrol.OperationLaunch, LaunchApplicationEffectID, integerRef, false)
	if err != nil {
		return nil, nil, err
	}
	terminate, err := sealApplicationNode(TerminateApplicationNodeID, "application.terminate", "node.application.terminate", "player-stop", appcontrol.OperationTerminate, TerminateApplicationEffectID, integerRef, true)
	if err != nil {
		return nil, nil, err
	}
	launchDefinition, err := defineBuiltin(launch, "application.launch", "v1", "configured-command/direct-launch/v1", nil)
	if err != nil {
		return nil, nil, err
	}
	terminateDefinition, err := defineBuiltin(terminate, "application.terminate", "v1", "configured-command/process-terminate/v1", nil)
	if err != nil {
		return nil, nil, err
	}
	return []BuiltinDefinition{launchDefinition, terminateDefinition}, []nodecontract.Contract{launch, terminate}, nil
}

func sealApplicationNode(nodeID, entrypoint, titleKey, icon, operation, effectID string, integerRef datatype.TypeRef, countOutput bool) (nodecontract.Contract, error) {
	schemaID := nodeID + "/config"
	inputs := []nodecontract.DataInputPort{}
	outputs := []nodecontract.DataOutputPort{}
	if countOutput {
		outputs = append(outputs, nodecontract.DataOutputPort{ID: "terminated-count", Type: datatype.RefExpression(integerRef)})
	}
	return nodecontract.Seal(nodecontract.Draft{Version: BuiltinNodeVersion,
		NodeTypeID: nodeID, ConfigSchemaRoot: schemaID,
		ConfigSchemaBundle: []datatype.SchemaResource{{ID: schemaID, Schema: json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object",
			"properties":{"slot":{"type":"string","minLength":1,"maxLength":128,"pattern":"^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$",
				"x-yotta-title-key":"node.application.config.slot.title","x-yotta-description-key":"node.application.config.slot.description"}},
			"required":["slot"],"additionalProperties":false
		}`, schemaID))}},
		Ports: nodecontract.PortSet{DataInputs: inputs, DataOutputs: outputs, ExecInputs: signalList("in"), ExecOutputs: signalList("completed"), ErrorOutputs: signalList("failed")},
		Execution: nodecontract.ExecutionSpec{
			Class: nodecontract.ExecutionEffect, Effects: []nodecontract.EffectID{nodecontract.EffectID(effectID)}, Determinism: nodecontract.Recorded,
			Evaluation: nodecontract.EvaluationPush, Cache: nodecontract.CacheNone, Retry: nodecontract.RetryNever,
			Cancellation: nodecontract.CancellationCooperative, Timeout: nodecontract.TimeoutRequired,
		},
		Instruction: nodecontract.Invoke(),
		ConfiguredTargets: []nodecontract.ConfiguredTargetSpec{{
			ID: "application", TargetSlot: "application", SlotConfigKey: "slot", TargetKinds: []string{appcontrol.TargetKind},
		}},
		Errors: applicationErrors(), StatusEvents: []nodecontract.StatusEventSpec{},
		ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
		Authoring: nodecontract.Authoring{
			TitleKey: titleKey + ".title", DescriptionKey: titleKey + ".description", Category: "application",
			Tags: []string{"application", "process", operation}, Icon: icon,
			Ports: dataPortHints(titleKey, inputs, outputs, nil),
		},
	})
}

func applicationErrors() []nodecontract.ErrorSpec {
	codes := []string{appcontrol.CodeInvalidRequest, appcontrol.CodeLaunchFailed, appcontrol.CodeTerminateFailed, appcontrol.CodeUnsupportedHost, appcontrol.CodeContractViolation}
	result := make([]nodecontract.ErrorSpec, 0, len(codes))
	for _, code := range codes {
		result = append(result, nodecontract.ErrorSpec{Code: code, Category: "application", RetryHint: false})
	}
	return result
}
