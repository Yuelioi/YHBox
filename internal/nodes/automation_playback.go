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
	InputClipTypeID                = "https://schemas.yotta.dev/types/automation/input-clip/v1"
	AutomationPlaybackCapabilityID = installed.CapabilityPlaybackID
	PlayInputClipNodeID            = "https://schemas.yotta.dev/nodes/automation/play-input-clip"
	PlayInputClipEffectID          = "https://schemas.yotta.dev/effects/automation/play-input-clip/v1"
	InputClipInvalidCode           = "inputclip.invalid"
)

func sealInputClipType() (datatype.Definition, error) {
	const schemaID = InputClipTypeID + "/schema"
	return datatype.SealDefinition(datatype.DefinitionDraft{
		TypeID: InputClipTypeID, SchemaDialect: datatype.JSONSchemaDialect, SchemaRoot: schemaID,
		SchemaBundle: []datatype.SchemaResource{{ID: schemaID, Schema: json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema"
		}`, schemaID))}},
		Representations: []datatype.RepresentationSpec{{Kind: datatype.RepresentationBlobRef, Codec: datatype.CodecBlobRefV1}},
		Authoring: datatype.Authoring{
			TitleKey: "type.automation.inputClip.title", DescriptionKey: "type.automation.inputClip.description", Color: "#f97316", Icon: "player-play",
		},
	})
}

func sealAutomationPlaybackCapability() (capability.Definition, error) {
	const scopeID = AutomationPlaybackCapabilityID + "/scope"
	return capability.SealDefinition(capability.DefinitionDraft{
		CapabilityID: AutomationPlaybackCapabilityID, Operations: installed.PlaybackOperations(), TargetKinds: []string{installed.TargetKindDesktopWindow, installed.TargetKindAndroidDevice},
		ScopeSchemaRoot: scopeID, ScopeSchemaBundle: []datatype.SchemaResource{{ID: scopeID, Schema: json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object",
			"properties":{"operation":{"const":"play"}},"required":["operation"],"additionalProperties":false
		}`, scopeID))}},
		Credential: capability.CredentialNone, Risk: capability.RiskDangerous, Consent: capability.ConsentNone,
		ProviderABI: installed.ProviderABI,
	})
}

func definePlayInputClipNode(inputClipRef datatype.TypeRef, playback, blobRead capability.Definition) (BuiltinDefinition, nodecontract.Contract, error) {
	const schemaID = PlayInputClipNodeID + "/config"
	contract, err := nodecontract.Seal(nodecontract.Draft{Version: BuiltinNodeVersion,
		NodeTypeID: PlayInputClipNodeID, ConfigSchemaRoot: schemaID,
		ConfigSchemaBundle: []datatype.SchemaResource{{ID: schemaID, Schema: json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object",
			"properties":{"slot":{"type":"string","minLength":1,"maxLength":128,"pattern":"^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$",
				"x-yotta-title-key":"node.automation.config.slot.title","x-yotta-description-key":"node.automation.config.slot.description"}},
			"required":["slot"],"additionalProperties":false
		}`, schemaID))}},
		Ports: nodecontract.PortSet{
			DataInputs: []nodecontract.DataInputPort{{ID: "clip", Type: datatype.RefExpression(inputClipRef), Required: true}},
			ExecInputs: signalList("in"), ExecOutputs: signalList("completed"), ErrorOutputs: signalList("failed"),
		},
		Execution: nodecontract.ExecutionSpec{
			Class: nodecontract.ExecutionEffect, Effects: []nodecontract.EffectID{PlayInputClipEffectID}, Determinism: nodecontract.Recorded,
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
			{Code: InputClipInvalidCode, Category: "inputclip", RetryHint: false},
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
			TitleKey: "node.automation.playInputClip.title", DescriptionKey: "node.automation.playInputClip.description", Category: "automation",
			Tags: []string{"automation", "input", "recording", "playback"}, Icon: "player-play",
		},
	})
	if err != nil {
		return BuiltinDefinition{}, nodecontract.Contract{}, err
	}
	definition, err := defineBuiltin(contract, "automation.play-input-clip", "v1", "content-addressed-input-clip/exact-target-playback/v1", nil)
	return definition, contract, err
}
