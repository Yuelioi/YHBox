package nodes

import (
	"encoding/json"
	"fmt"

	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

const (
	FindTemplateMatchesNodeID = "https://schemas.yotta.dev/nodes/vision/find-template-matches"
	CompareImagesNodeID       = "https://schemas.yotta.dev/nodes/vision/compare-images"
	DecodeQRNodeID            = "https://schemas.yotta.dev/nodes/vision/decode-qr"
	AnalyzeColorNodeID        = "https://schemas.yotta.dev/nodes/vision/analyze-color"
	FindColorBlobsNodeID      = "https://schemas.yotta.dev/nodes/vision/find-color-blobs"

	FindTemplateMatchesEffectID = "https://schemas.yotta.dev/effects/vision/find-template-matches/v1"
	CompareImagesEffectID       = "https://schemas.yotta.dev/effects/vision/compare-images/v1"
	DecodeQREffectID            = "https://schemas.yotta.dev/effects/vision/decode-qr/v1"
	AnalyzeColorEffectID        = "https://schemas.yotta.dev/effects/vision/analyze-color/v1"
	FindColorBlobsEffectID      = "https://schemas.yotta.dev/effects/vision/find-color-blobs/v1"

	VisionColorRangeInvalidCode = "vision.color_range_invalid"
	VisionAnalysisFailedCode    = "vision.analysis_failed"
)

type visionNodeSpec struct {
	id, effect, entrypoint, conformance, key, icon string
	inputs                                         []nodecontract.DataInputPort
	outputs                                        []nodecontract.DataOutputPort
	errors                                         []nodecontract.ErrorSpec
	portAdapters                                   map[string]string
}

func defineVisionAnalysisNodes(types extendedTypes, visionTypes visionTypeSet, imageRef datatype.TypeRef, blobRead capability.Definition) ([]BuiltinDefinition, []nodecontract.Contract, error) {
	defaultRegion := json.RawMessage(`{"x":0,"y":0,"width":1,"height":1,"unit":"ratio"}`)
	defaultThreshold := json.RawMessage(`0.8`)
	defaultGridSize := json.RawMessage(`32`)
	defaultCellDelta := json.RawMessage(`12`)
	defaultMinimumDistance := json.RawMessage(`0`)
	defaultMinimumArea := json.RawMessage(`1`)
	imageType := datatype.RefExpression(imageRef)
	regionType := datatype.RefExpression(types.regionRef)
	numberType := datatype.RefExpression(types.numberRef)
	integerType := datatype.RefExpression(types.integerRef)
	pointType := datatype.RefExpression(types.pointRef)
	commonErrors := []nodecontract.ErrorSpec{
		{Code: VisionImageInvalidCode, Category: "vision", RetryHint: false},
		{Code: VisionRegionInvalidCode, Category: "vision", RetryHint: false},
		{Code: VisionAnalysisFailedCode, Category: "vision", RetryHint: false},
	}
	specs := []visionNodeSpec{
		{
			id: FindTemplateMatchesNodeID, effect: FindTemplateMatchesEffectID, entrypoint: "vision.find-template-matches",
			conformance: "explicit-png-roi-tm-ccoeff-normed-local-max-nms/v1", key: "node.vision.findTemplateMatches", icon: "list-search",
			inputs: []nodecontract.DataInputPort{
				{ID: "image", Type: imageType, Required: true}, {ID: "template", Type: imageType, Required: true},
				{ID: "region", Type: regionType, Required: true, Default: &defaultRegion},
				{ID: "threshold", Type: numberType, Required: true, Default: &defaultThreshold},
				{ID: "minimum-distance", Type: integerType, Required: true, Default: &defaultMinimumDistance},
			},
			outputs:      []nodecontract.DataOutputPort{{ID: "matches", Type: datatype.ListExpression(datatype.RefExpression(visionTypes.templateMatch.TypeRef()))}},
			errors:       append([]nodecontract.ErrorSpec{{Code: VisionTemplateInvalidCode, Category: "vision", RetryHint: false}}, commonErrors...),
			portAdapters: map[string]string{"template": "template-image"},
		},
		{
			id: CompareImagesNodeID, effect: CompareImagesEffectID, entrypoint: "vision.compare-images",
			conformance: "explicit-png-roi-box-grid-difference/v1", key: "node.vision.compareImages", icon: "arrows-diff",
			inputs: []nodecontract.DataInputPort{
				{ID: "before", Type: imageType, Required: true}, {ID: "after", Type: imageType, Required: true},
				{ID: "region", Type: regionType, Required: true, Default: &defaultRegion},
				{ID: "grid-size", Type: integerType, Required: true, Default: &defaultGridSize},
				{ID: "cell-delta", Type: integerType, Required: true, Default: &defaultCellDelta},
			},
			outputs: []nodecontract.DataOutputPort{{ID: "changed-ratio", Type: numberType}, {ID: "mean-difference", Type: numberType}},
			errors:  commonErrors,
		},
		{
			id: DecodeQRNodeID, effect: DecodeQREffectID, entrypoint: "vision.decode-qr",
			conformance: "explicit-png-roi-gozxing-multi-qr/v1", key: "node.vision.decodeQR", icon: "qrcode",
			inputs:  []nodecontract.DataInputPort{{ID: "image", Type: imageType, Required: true}, {ID: "region", Type: regionType, Required: true, Default: &defaultRegion}},
			outputs: []nodecontract.DataOutputPort{{ID: "codes", Type: datatype.ListExpression(datatype.RefExpression(visionTypes.qrCode.TypeRef()))}},
			errors:  commonErrors,
		},
		{
			id: AnalyzeColorNodeID, effect: AnalyzeColorEffectID, entrypoint: "vision.analyze-color",
			conformance: "explicit-png-roi-inclusive-rgb-hsv-statistics/v1", key: "node.vision.analyzeColor", icon: "color-filter",
			inputs: []nodecontract.DataInputPort{
				{ID: "image", Type: imageType, Required: true}, {ID: "range", Type: datatype.RefExpression(visionTypes.colorRange.TypeRef()), Required: true},
				{ID: "region", Type: regionType, Required: true, Default: &defaultRegion},
			},
			outputs: []nodecontract.DataOutputPort{{ID: "pixel-count", Type: integerType}, {ID: "fraction", Type: numberType}, {ID: "centroid", Type: pointType}},
			errors:  append([]nodecontract.ErrorSpec{{Code: VisionColorRangeInvalidCode, Category: "vision", RetryHint: false}}, commonErrors...),
		},
		{
			id: FindColorBlobsNodeID, effect: FindColorBlobsEffectID, entrypoint: "vision.find-color-blobs",
			conformance: "explicit-png-roi-inclusive-rgb-hsv-four-connected-components/v1", key: "node.vision.findColorBlobs", icon: "box-multiple",
			inputs: []nodecontract.DataInputPort{
				{ID: "image", Type: imageType, Required: true}, {ID: "range", Type: datatype.RefExpression(visionTypes.colorRange.TypeRef()), Required: true},
				{ID: "region", Type: regionType, Required: true, Default: &defaultRegion},
				{ID: "minimum-area", Type: integerType, Required: true, Default: &defaultMinimumArea},
			},
			outputs: []nodecontract.DataOutputPort{{ID: "blobs", Type: datatype.ListExpression(datatype.RefExpression(visionTypes.colorBlob.TypeRef()))}},
			errors:  append([]nodecontract.ErrorSpec{{Code: VisionColorRangeInvalidCode, Category: "vision", RetryHint: false}}, commonErrors...),
		},
	}
	definitions := make([]BuiltinDefinition, 0, len(specs))
	contracts := make([]nodecontract.Contract, 0, len(specs))
	for _, spec := range specs {
		configID := spec.id + "/config"
		contract, err := nodecontract.Seal(nodecontract.Draft{Version: BuiltinNodeVersion,
			NodeTypeID: spec.id, ConfigSchemaRoot: configID, ConfigSchemaBundle: emptyConfigSchema(configID),
			Ports: nodecontract.PortSet{
				DataInputs: spec.inputs, DataOutputs: spec.outputs,
				ExecInputs: []nodecontract.SignalPort{}, ExecOutputs: []nodecontract.SignalPort{}, ErrorOutputs: []nodecontract.SignalPort{},
			},
			Execution: conversionExecution(nodecontract.EffectID(spec.effect)), Instruction: nodecontract.Invoke(),
			CapabilityRequirements: []capability.Requirement{requirement(blobRead, "blob-read", []string{"read-range"}, "blob-store")},
			Errors:                 spec.errors, StatusEvents: []nodecontract.StatusEventSpec{},
			ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
			Authoring: nodecontract.Authoring{
				TitleKey: spec.key + ".title", DescriptionKey: spec.key + ".description", Category: "vision",
				Tags: []string{"analysis", "image", "vision"}, Icon: spec.icon,
				Ports: visionPortHints(spec.key, spec.inputs, spec.outputs, spec.portAdapters),
			},
		})
		if err != nil {
			return nil, nil, fmt.Errorf("seal vision node %s: %w", spec.id, err)
		}
		definition, err := defineBuiltin(contract, spec.entrypoint, "v1", spec.conformance, nil)
		if err != nil {
			return nil, nil, err
		}
		definitions = append(definitions, definition)
		contracts = append(contracts, contract)
	}
	return definitions, contracts, nil
}

func visionPortHints(key string, inputs []nodecontract.DataInputPort, outputs []nodecontract.DataOutputPort, adapters map[string]string) []nodecontract.PortAuthoring {
	result := make([]nodecontract.PortAuthoring, 0, len(inputs)+len(outputs))
	for _, port := range inputs {
		result = append(result, nodecontract.PortAuthoring{
			ID: port.ID, TitleKey: key + ".input." + port.ID + ".title", DescriptionKey: key + ".input." + port.ID + ".description",
			EditorAdapter: adapters[port.ID],
		})
	}
	for _, port := range outputs {
		result = append(result, nodecontract.PortAuthoring{
			ID: port.ID, TitleKey: key + ".output." + port.ID + ".title", DescriptionKey: key + ".output." + port.ID + ".description",
		})
	}
	return result
}
