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
	ImageTypeID           = "https://schemas.yotta.dev/types/media/image/v1"
	CaptureWindowNodeID   = "https://schemas.yotta.dev/nodes/automation/capture-window"
	CaptureWindowEffectID = "https://schemas.yotta.dev/effects/automation/capture-window/v1"
)

func sealImageType() (datatype.Definition, error) {
	const schemaID = ImageTypeID + "/schema"
	return datatype.SealDefinition(datatype.DefinitionDraft{
		TypeID: ImageTypeID, SchemaDialect: datatype.JSONSchemaDialect, SchemaRoot: schemaID,
		SchemaBundle: []datatype.SchemaResource{{ID: schemaID, Schema: json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema"
		}`, schemaID))}},
		Representations: []datatype.RepresentationSpec{{Kind: datatype.RepresentationBlobRef, Codec: datatype.CodecBlobRefV1}},
		Authoring: datatype.Authoring{
			TitleKey: "type.media.image.title", DescriptionKey: "type.media.image.description", Color: "#22c55e", Icon: "photo",
		},
	})
}

func defineCaptureWindowNode(imageRef datatype.TypeRef, blobWrite capability.Definition) (BuiltinDefinition, nodecontract.Contract, error) {
	const schemaID = CaptureWindowNodeID + "/config"
	contract, err := nodecontract.Seal(nodecontract.Draft{Version: BuiltinNodeVersion,
		NodeTypeID: CaptureWindowNodeID, ConfigSchemaRoot: schemaID,
		ConfigSchemaBundle: []datatype.SchemaResource{{ID: schemaID, Schema: json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object",
			"properties":{"slot":{"type":"string","minLength":1,"maxLength":128,"pattern":"^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$",
				"x-yotta-title-key":"node.automation.config.slot.title","x-yotta-description-key":"node.automation.config.slot.description"}},
			"required":["slot"],"additionalProperties":false
		}`, schemaID))}},
		Ports: nodecontract.PortSet{
			DataOutputs: []nodecontract.DataOutputPort{{ID: "image", Type: datatype.RefExpression(imageRef)}},
			ExecInputs:  signalList("in"), ExecOutputs: signalList("completed"), ErrorOutputs: signalList("failed"),
		},
		Execution: nodecontract.ExecutionSpec{
			Class: nodecontract.ExecutionEffect, Effects: []nodecontract.EffectID{CaptureWindowEffectID}, Determinism: nodecontract.Recorded,
			Evaluation: nodecontract.EvaluationPush, Cache: nodecontract.CacheNone, Retry: nodecontract.RetryNever,
			Cancellation: nodecontract.CancellationCooperative, Timeout: nodecontract.TimeoutRequired,
		},
		Instruction:            nodecontract.Invoke(),
		CapabilityRequirements: []capability.Requirement{requirement(blobWrite, "blob-write", []string{"append", "cancel", "commit"}, "blob-store")},
		ConfiguredTargets:      automationTargetSpec("target", installed.TargetKindDesktopWindow, installed.TargetKindAndroidDevice, installed.TargetKindBrowserCDP),
		Errors: []nodecontract.ErrorSpec{
			{Code: installed.CodeTargetNotFound, Category: "automation", RetryHint: true},
			{Code: installed.CodeTargetAmbiguous, Category: "automation", RetryHint: false},
			{Code: installed.CodeCaptureFailed, Category: "automation", RetryHint: true},
			{Code: installed.CodeUnsupportedHost, Category: "automation", RetryHint: false},
			{Code: installed.CodeContractViolation, Category: "automation", RetryHint: false},
		},
		ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
		Authoring: nodecontract.Authoring{
			TitleKey: "node.automation.captureWindow.title", DescriptionKey: "node.automation.captureWindow.description", Category: "automation",
			Tags: []string{"automation", "window", "capture", "image"}, Icon: "camera",
		},
	})
	if err != nil {
		return BuiltinDefinition{}, nodecontract.Contract{}, err
	}
	definition, err := defineBuiltin(contract, "automation.capture-window", "v1", "configured-target/png-capture-to-blob/v1", nil)
	return definition, contract, err
}
