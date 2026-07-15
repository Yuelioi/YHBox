package nodes31

import (
	"encoding/json"
	"fmt"

	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/scriptengine"
)

const (
	ScriptExecuteNodeID   = "https://schemas.yotta.dev/nodes/script/execute/v1"
	ScriptExecuteEffectID = "https://schemas.yotta.dev/effects/script/execute/v1"

	scriptEntrypoint            = "script.execute"
	scriptImplementationVersion = "v1"
)

func defineScriptNode(jsonRef datatype.TypeRef) (BuiltinDefinition, nodecontract.Contract, error) {
	contract, err := sealScriptNode(jsonRef)
	if err != nil {
		return BuiltinDefinition{}, nodecontract.Contract{}, err
	}
	definition, err := defineBuiltin(
		contract,
		scriptEntrypoint,
		scriptImplementationVersion,
		"isolated-one-shot-canonical-json-worker/3.1",
		nil,
	)
	if err != nil {
		return BuiltinDefinition{}, nodecontract.Contract{}, err
	}
	return definition, contract, nil
}

func sealScriptNode(jsonRef datatype.TypeRef) (nodecontract.Contract, error) {
	const schemaID = ScriptExecuteNodeID + "/config"
	defaultInput := json.RawMessage(`{}`)
	errors := make([]nodecontract.ErrorSpec, 0, len(scriptFailureCodes()))
	for _, code := range scriptFailureCodes() {
		errors = append(errors, nodecontract.ErrorSpec{Code: code, Category: "script", RetryHint: false})
	}
	return nodecontract.Seal(nodecontract.Draft{
		NodeTypeID:       ScriptExecuteNodeID,
		ConfigSchemaRoot: schemaID,
		ConfigSchemaBundle: []datatype.SchemaResource{{ID: schemaID, Schema: json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema",
			"type":"object","properties":{
				"source":{"type":"string","minLength":1,"maxLength":%d,"x-yotta-control":"code",
					"x-yotta-title-key":"node.script.execute.config.source.title",
					"x-yotta-description-key":"node.script.execute.config.source.description"},
				"timeoutMilliseconds":{"type":"integer","minimum":%d,"maximum":%d,
					"x-yotta-title-key":"node.script.execute.config.timeoutMilliseconds.title",
					"x-yotta-description-key":"node.script.execute.config.timeoutMilliseconds.description"}
			},"required":["source","timeoutMilliseconds"],"additionalProperties":false
		}`, schemaID, scriptengine.MaxSourceBytes, scriptengine.MinTimeoutMillis, scriptengine.MaxTimeoutMillis))}},
		Ports: nodecontract.PortSet{
			DataInputs: []nodecontract.DataInputPort{{
				ID: "input", Type: datatype.RefExpression(jsonRef), Required: true, Default: &defaultInput,
			}},
			DataOutputs:  []nodecontract.DataOutputPort{{ID: "result", Type: datatype.RefExpression(jsonRef)}},
			ExecInputs:   []nodecontract.SignalPort{{ID: "in"}},
			ExecOutputs:  []nodecontract.SignalPort{{ID: "completed"}},
			ErrorOutputs: []nodecontract.SignalPort{{ID: "failed"}},
		},
		Execution: nodecontract.ExecutionSpec{
			Class: nodecontract.ExecutionEffect, Effects: []nodecontract.EffectID{nodecontract.EffectID(ScriptExecuteEffectID)},
			Determinism: nodecontract.Recorded, Evaluation: nodecontract.EvaluationPush,
			Cache: nodecontract.CacheNone, Retry: nodecontract.RetryNever,
			Cancellation: nodecontract.CancellationImmediate, Timeout: nodecontract.TimeoutRequired,
		},
		Instruction:             nodecontract.Invoke(),
		HostFeatureRequirements: []nodecontract.HostFeatureRequirement{{ID: "isolation", FeatureID: scriptengine.IsolationHostFeatureID}},
		CapabilityRequirements:  []capability.Requirement{},
		Errors:                  errors,
		StatusEvents:            []nodecontract.StatusEventSpec{},
		ImplementationABI:       []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
		Authoring: nodecontract.Authoring{
			TitleKey: "node.script.execute.title", DescriptionKey: "node.script.execute.description",
			Category: "script", Tags: []string{"script", "javascript", "json", "isolated"}, Icon: "code",
		},
	})
}

func scriptFailureCodes() []string {
	return []string{
		scriptengine.CodeSourceInvalid,
		scriptengine.CodeGuestThrown,
		scriptengine.CodeDeadlineExceeded,
		scriptengine.CodeStackExceeded,
		scriptengine.CodeContractViolation,
		scriptengine.CodeRunnerProtocolViolation,
		scriptengine.CodeRunnerCrashed,
		scriptengine.CodeIsolationUnavailable,
	}
}
