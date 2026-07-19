package nodes

import (
	"encoding/json"
	"fmt"

	"github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

const (
	MacroTypeID       = "https://schemas.yotta.dev/types/automation/macro/v1"
	PlayMacroNodeID   = "https://schemas.yotta.dev/nodes/automation/play-macro"
	PlayMacroEffectID = "https://schemas.yotta.dev/effects/automation/play-macro/v1"
	MacroInvalidCode  = "macro.invalid"
)

func sealMacroType() (datatype.Definition, error) {
	const schemaID = MacroTypeID + "/schema"
	return datatype.SealDefinition(datatype.DefinitionDraft{
		TypeID: MacroTypeID, SchemaDialect: datatype.JSONSchemaDialect, SchemaRoot: schemaID,
		SchemaBundle: []datatype.SchemaResource{{ID: schemaID, Schema: json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema"
		}`, schemaID))}},
		Representations: []datatype.RepresentationSpec{{Kind: datatype.RepresentationBlobRef, Codec: datatype.CodecBlobRefV1}},
		Authoring: datatype.Authoring{
			TitleKey: "type.automation.macro.title", DescriptionKey: "type.automation.macro.description", Color: "#14b8a6", Icon: "list-check",
		},
	})
}

func definePlayMacroNode(macroRef datatype.TypeRef, playback, blobRead capability.Definition) (BuiltinDefinition, nodecontract.Contract, error) {
	const schemaID = PlayMacroNodeID + "/config"
	contract, err := nodecontract.Seal(nodecontract.Draft{Version: BuiltinNodeVersion,
		NodeTypeID: PlayMacroNodeID, ConfigSchemaRoot: schemaID,
		ConfigSchemaBundle: []datatype.SchemaResource{{ID: schemaID, Schema: json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object",
			"properties":{"slot":{"type":"string","minLength":1,"maxLength":128,"pattern":"^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$",
				"x-yotta-title-key":"node.automation.config.slot.title","x-yotta-description-key":"node.automation.config.slot.description"}},
			"required":["slot"],"additionalProperties":false
		}`, schemaID))}},
		Ports: nodecontract.PortSet{
			DataInputs: []nodecontract.DataInputPort{{ID: "macro", Type: datatype.RefExpression(macroRef), Required: true}},
			ExecInputs: signalList("in"), ExecOutputs: signalList("completed"), ErrorOutputs: signalList("failed"),
		},
		Execution: nodecontract.ExecutionSpec{
			Class: nodecontract.ExecutionEffect, Effects: []nodecontract.EffectID{PlayMacroEffectID}, Determinism: nodecontract.Recorded,
			Evaluation: nodecontract.EvaluationPush, Cache: nodecontract.CacheNone, Retry: nodecontract.RetryNever,
			Cancellation: nodecontract.CancellationCooperative, Timeout: nodecontract.TimeoutRequired,
		},
		Instruction: nodecontract.Invoke(),
		CapabilityRequirements: []capability.Requirement{
			{ID: "target", Capability: playback.Ref(), Operations: installed.PlaybackOperations(), TargetSlot: "target", Scope: json.RawMessage(`{"operation":"play"}`)},
			requirement(blobRead, "blob-read", []string{"read-range"}, "blob-store"),
		},
		RequirementBindings: []nodecontract.RequirementBindingSpec{{RequirementID: "target", TargetSlotConfigKey: "slot"}},
		Errors: []nodecontract.ErrorSpec{
			{Code: MacroInvalidCode, Category: "macro", RetryHint: false},
			{Code: installed.CodeIdentityChanged, Category: "automation", RetryHint: false},
			{Code: installed.CodeTargetNotFound, Category: "automation", RetryHint: true},
			{Code: installed.CodeTargetAmbiguous, Category: "automation", RetryHint: false},
			{Code: installed.CodePlaybackFailed, Category: "automation", RetryHint: true},
			{Code: installed.CodePlaybackBusy, Category: "automation", RetryHint: true},
			{Code: installed.CodeUnsupportedHost, Category: "automation", RetryHint: false},
			{Code: installed.CodeContractViolation, Category: "automation", RetryHint: false},
		},
		ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
		Authoring: nodecontract.Authoring{
			TitleKey: "node.automation.playMacro.title", DescriptionKey: "node.automation.playMacro.description", Category: "automation",
			Tags: []string{"automation", "input", "macro", "playback"}, Icon: "list-check",
		},
	})
	if err != nil {
		return BuiltinDefinition{}, nodecontract.Contract{}, err
	}
	definition, err := defineBuiltin(contract, "automation.play-macro", "v1", "content-addressed-atomic-macro/exact-target-playback/v1", nil)
	return definition, contract, err
}
