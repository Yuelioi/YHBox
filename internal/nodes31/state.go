package nodes31

import (
	"encoding/json"
	"fmt"

	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

const (
	StateReadNodeID     = "https://schemas.yotta.dev/nodes/state/read/v1"
	StateWriteNodeID    = "https://schemas.yotta.dev/nodes/state/write/v1"
	StateMetadataNodeID = "https://schemas.yotta.dev/nodes/state/metadata/v1"

	StateReadEffectID  = "https://schemas.yotta.dev/effects/state/read/v1"
	StateWriteEffectID = "https://schemas.yotta.dev/effects/state/write/v1"

	StateReadFailedCode  = "state.read_failed"
	StateWriteFailedCode = "state.write_failed"
)

func defineStateNodes(types primitiveTypes) ([]BuiltinDefinition, error) {
	valueType := datatype.VariableExpression("T")
	integerType := datatype.RefExpression(types.integerRef)
	type stateNode struct {
		id, entrypoint, conformance, key, icon string
		ports                                  nodecontract.PortSet
		execution                              nodecontract.ExecutionSpec
		access                                 nodecontract.StateAccessSpec
		errors                                 []nodecontract.ErrorSpec
	}
	readAccess := nodecontract.StateAccessSpec{ID: "state", SlotConfigKey: "variable", Type: valueType, Mode: nodecontract.StateRead}
	writeAccess := readAccess
	writeAccess.Mode = nodecontract.StateWrite
	readExecution := stateExecution(StateReadEffectID, nodecontract.EvaluationPull)
	writeExecution := stateExecution(StateWriteEffectID, nodecontract.EvaluationPush)
	specs := []stateNode{
		{
			id: StateReadNodeID, entrypoint: "state.read", conformance: "run-owned-typed-state-read/v1",
			key: "node.state.read", icon: "database-export",
			ports: nodecontract.PortSet{
				DataInputs: []nodecontract.DataInputPort{}, DataOutputs: []nodecontract.DataOutputPort{{ID: "result", Type: valueType}},
				ExecInputs: []nodecontract.SignalPort{}, ExecOutputs: []nodecontract.SignalPort{}, ErrorOutputs: []nodecontract.SignalPort{},
			},
			execution: readExecution, access: readAccess,
			errors: []nodecontract.ErrorSpec{{Code: StateReadFailedCode, Category: "state", RetryHint: false}},
		},
		{
			id: StateWriteNodeID, entrypoint: "state.write", conformance: "run-owned-typed-state-write/v1",
			key: "node.state.write", icon: "database-import",
			ports: nodecontract.PortSet{
				DataInputs:  []nodecontract.DataInputPort{{ID: "value", Type: valueType, Required: true}},
				DataOutputs: []nodecontract.DataOutputPort{{ID: "result", Type: valueType}},
				ExecInputs:  []nodecontract.SignalPort{{ID: "in"}}, ExecOutputs: []nodecontract.SignalPort{{ID: "done"}}, ErrorOutputs: []nodecontract.SignalPort{},
			},
			execution: writeExecution, access: writeAccess,
			errors: []nodecontract.ErrorSpec{{Code: StateWriteFailedCode, Category: "state", RetryHint: false}},
		},
		{
			id: StateMetadataNodeID, entrypoint: "state.metadata", conformance: "run-owned-state-metadata/v1",
			key: "node.state.metadata", icon: "database-cog",
			ports: nodecontract.PortSet{
				DataInputs: []nodecontract.DataInputPort{}, DataOutputs: []nodecontract.DataOutputPort{
					{ID: "revision", Type: integerType}, {ID: "changed-at", Type: integerType},
				},
				ExecInputs: []nodecontract.SignalPort{}, ExecOutputs: []nodecontract.SignalPort{}, ErrorOutputs: []nodecontract.SignalPort{},
			},
			execution: readExecution, access: readAccess,
			errors: []nodecontract.ErrorSpec{{Code: StateReadFailedCode, Category: "state", RetryHint: false}},
		},
	}
	definitions := make([]BuiltinDefinition, 0, len(specs))
	for _, spec := range specs {
		configID := spec.id + "/config"
		contract, err := nodecontract.Seal(nodecontract.Draft{
			NodeTypeID: spec.id, ConfigSchemaRoot: configID,
			ConfigSchemaBundle: []datatype.SchemaResource{{ID: configID, Schema: json.RawMessage(fmt.Sprintf(`{
				"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object",
				"properties":{"variable":{"type":"string","minLength":1,"maxLength":128,
				"pattern":"^[A-Za-z0-9_][A-Za-z0-9._-]*$","x-yotta-control":"state-variable",
				"x-yotta-title-key":"node.state.variable.title","x-yotta-description-key":"node.state.variable.description"}},
				"required":["variable"],"additionalProperties":false
			}`, configID))}},
			Ports: spec.ports, Execution: spec.execution, Instruction: nodecontract.Invoke(), CapabilityRequirements: []capability.Requirement{},
			Errors: spec.errors, StatusEvents: []nodecontract.StatusEventSpec{}, StateAccesses: []nodecontract.StateAccessSpec{spec.access},
			ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
			Authoring: nodecontract.Authoring{
				TitleKey: spec.key + ".title", DescriptionKey: spec.key + ".description", Category: "state",
				Tags: []string{"state", "variable", "recorded"}, Icon: spec.icon,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("seal built-in %s: %w", spec.id, err)
		}
		definition, err := defineBuiltin(contract, spec.entrypoint, "v1", spec.conformance, nil)
		if err != nil {
			return nil, err
		}
		definitions = append(definitions, definition)
	}
	return definitions, nil
}

func stateExecution(effect nodecontract.EffectID, evaluation nodecontract.EvaluationMode) nodecontract.ExecutionSpec {
	return nodecontract.ExecutionSpec{
		Class: nodecontract.ExecutionEffect, Effects: []nodecontract.EffectID{effect}, Determinism: nodecontract.Recorded,
		Evaluation: evaluation, Cache: nodecontract.CacheNone, Retry: nodecontract.RetryNever,
		Cancellation: nodecontract.CancellationCooperative, Timeout: nodecontract.TimeoutNone,
	}
}
