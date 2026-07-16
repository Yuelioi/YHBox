package nodes31

import (
	"encoding/json"
	"fmt"

	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

const (
	MatchTemplateNodeID   = "https://schemas.yotta.dev/nodes/vision/match-template/v1"
	MatchTemplateEffectID = "https://schemas.yotta.dev/effects/vision/match-template/v1"

	VisionImageInvalidCode    = "vision.image_invalid"
	VisionTemplateInvalidCode = "vision.template_invalid"
	VisionRegionInvalidCode   = "vision.region_invalid"
	VisionMatchFailedCode     = "vision.match_failed"
)

func defineMatchTemplateNode(types extendedTypes, imageRef datatype.TypeRef, blobRead capability.Definition) (BuiltinDefinition, nodecontract.Contract, error) {
	const schemaID = MatchTemplateNodeID + "/config"
	defaultRegion := json.RawMessage(`{"x":0,"y":0,"width":1,"height":1,"unit":"ratio"}`)
	defaultThreshold := json.RawMessage(`0.8`)
	contract, err := nodecontract.Seal(nodecontract.Draft{
		NodeTypeID: MatchTemplateNodeID, ConfigSchemaRoot: schemaID, ConfigSchemaBundle: emptyConfigSchema(schemaID),
		Ports: nodecontract.PortSet{
			DataInputs: []nodecontract.DataInputPort{
				{ID: "image", Type: datatype.RefExpression(imageRef), Required: true},
				{ID: "template", Type: datatype.RefExpression(imageRef), Required: true},
				{ID: "region", Type: datatype.RefExpression(types.regionRef), Required: true, Default: &defaultRegion},
				{ID: "threshold", Type: datatype.RefExpression(types.numberRef), Required: true, Default: &defaultThreshold},
			},
			DataOutputs: []nodecontract.DataOutputPort{
				{ID: "matched", Type: datatype.RefExpression(types.booleanRef)},
				{ID: "score", Type: datatype.RefExpression(types.numberRef)},
				{ID: "center", Type: datatype.RefExpression(types.pointRef)},
				{ID: "bounds", Type: datatype.RefExpression(types.regionRef)},
			},
			ExecInputs: []nodecontract.SignalPort{}, ExecOutputs: []nodecontract.SignalPort{}, ErrorOutputs: []nodecontract.SignalPort{},
		},
		Execution: conversionExecution(MatchTemplateEffectID), Instruction: nodecontract.Invoke(),
		CapabilityRequirements: []capability.Requirement{requirement(blobRead, "blob-read", []string{"read-range"}, "blob-store")},
		Errors: []nodecontract.ErrorSpec{
			{Code: VisionImageInvalidCode, Category: "vision", RetryHint: false},
			{Code: VisionTemplateInvalidCode, Category: "vision", RetryHint: false},
			{Code: VisionRegionInvalidCode, Category: "vision", RetryHint: false},
			{Code: VisionMatchFailedCode, Category: "vision", RetryHint: false},
		},
		StatusEvents:      []nodecontract.StatusEventSpec{},
		ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
		Authoring: nodecontract.Authoring{
			TitleKey: "node.vision.matchTemplate.title", DescriptionKey: "node.vision.matchTemplate.description", Category: "vision",
			Tags: []string{"image", "template", "vision"}, Icon: "scan",
		},
	})
	if err != nil {
		return BuiltinDefinition{}, nodecontract.Contract{}, fmt.Errorf("seal match template: %w", err)
	}
	definition, err := defineBuiltin(contract, "vision.match-template", "v1", "explicit-png-roi-tm-ccoeff-normed/v1", nil)
	return definition, contract, err
}
