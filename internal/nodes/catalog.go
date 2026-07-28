// Package nodes declares Yotta's explicitly assembled built-in node catalog.
package nodes

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/configvalidator"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecatalog"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/stream"
)

const (
	BuiltinNodeVersion = "1.0.0"
	StringTypeID       = "https://schemas.yotta.dev/types/core/string/v1"
	BinaryTypeID       = "https://schemas.yotta.dev/types/core/binary/v1"
	NumberTypeID       = "https://schemas.yotta.dev/types/core/number/v1"
	IntegerTypeID      = "https://schemas.yotta.dev/types/core/integer/v1"
	BooleanTypeID      = "https://schemas.yotta.dev/types/core/boolean/v1"
	JSONTypeID         = "https://schemas.yotta.dev/types/core/json/v1"
	PointUnitTypeID    = "https://schemas.yotta.dev/types/geometry/point-unit/v1"
	PointTypeID        = "https://schemas.yotta.dev/types/geometry/point/v1"
	RegionTypeID       = "https://schemas.yotta.dev/types/geometry/region/v1"
	ConcatNodeID       = "https://schemas.yotta.dev/nodes/text/concat"
	BlobToStreamNodeID = "https://schemas.yotta.dev/nodes/conversion/blob-to-stream"
	StreamToBlobNodeID = "https://schemas.yotta.dev/nodes/conversion/stream-to-blob"
	AIGenerateNodeID   = "https://schemas.yotta.dev/nodes/ai/generate"
	AIExtractNodeID    = "https://schemas.yotta.dev/nodes/ai/extract"

	BlobReadCapabilityID     = "https://schemas.yotta.dev/capabilities/blob/read/v1"
	BlobWriteCapabilityID    = "https://schemas.yotta.dev/capabilities/blob/write/v1"
	StreamCapabilityID       = "https://schemas.yotta.dev/capabilities/stream/session/v1"
	AIGenerationCapabilityID = "https://schemas.yotta.dev/capabilities/ai/generation/v1"
	BlobToStreamEffectID     = "https://schemas.yotta.dev/effects/conversion/blob-to-stream/v1"
	StreamToBlobEffectID     = "https://schemas.yotta.dev/effects/conversion/stream-to-blob/v1"
	AIGenerateEffectID       = "https://schemas.yotta.dev/effects/ai/generate/v1"
	AIExtractEffectID        = "https://schemas.yotta.dev/effects/ai/extract/v1"

	concatEntrypoint                = "text.concat"
	blobToStreamEntrypoint          = "conversion.blob-to-stream"
	streamToBlobEntrypoint          = "conversion.stream-to-blob"
	concatImplementationVersion     = "v1"
	conversionImplementationVersion = "v2"
)

// InlineEvaluator is the narrow ABI shared by deterministic built-ins whose
// complete inputs and outputs are durable inline JSON. The generic runtime
// adapter seals every returned value against the exact output TypeRef.
type InlineEvaluator func(context.Context, map[string]json.RawMessage, map[string]any) (map[string]json.RawMessage, error)

// BuiltinDefinition binds one immutable Node Contract to the implementation
// manifest compiled into this build. A pure inline evaluator is optional;
// resource/effect families provide a specialized runtime installer instead.
type BuiltinDefinition struct {
	Contract       nodecontract.Contract
	Implementation nodecatalog.ImplementationLock
	EvaluateInline InlineEvaluator
}

type Builtins struct {
	Catalog                      nodecatalog.Snapshot
	StringType                   datatype.Definition
	BinaryType                   datatype.Definition
	ImageType                    datatype.Definition
	InputClipType                datatype.Definition
	MacroType                    datatype.Definition
	NumberType                   datatype.Definition
	IntegerType                  datatype.Definition
	BooleanType                  datatype.Definition
	JSONType                     datatype.Definition
	PointUnitType                datatype.Definition
	PointType                    datatype.Definition
	RegionType                   datatype.Definition
	TemplateMatchType            datatype.Definition
	QRCodeType                   datatype.Definition
	ColorRangeType               datatype.Definition
	ColorBlobType                datatype.Definition
	PointerButtonType            datatype.Definition
	KeyCodeType                  datatype.Definition
	HeldInputType                datatype.Definition
	RandomDistributionType       datatype.Definition
	DurationMillisecondsType     datatype.Definition
	FileMetadataType             datatype.Definition
	ObservabilityMessageType     datatype.Definition
	ConcatContract               nodecontract.Contract
	BlobToStreamContract         nodecontract.Contract
	StreamToBlobContract         nodecontract.Contract
	AIGenerateContract           nodecontract.Contract
	AIExtractContract            nodecontract.Contract
	AIGeneratePrompt             ai.PromptManifest
	AIExtractPrompt              ai.PromptManifest
	AIAuthoringPrompt            ai.PromptManifest
	AIAuthoringToolSet           ai.ToolSet
	ScriptExecuteContract        nodecontract.Contract
	FileReadTextContract         nodecontract.Contract
	FileReadJSONContract         nodecontract.Contract
	FileStatContract             nodecontract.Contract
	HTTPGetContract              nodecontract.Contract
	LaunchApplicationContract    nodecontract.Contract
	TerminateApplicationContract nodecontract.Contract
	AutomationInputContracts     []nodecontract.Contract
	AutomationHeldInputContracts []nodecontract.Contract
	AutomationTemplateContracts  []nodecontract.Contract
	ActivateWindowContract       nodecontract.Contract
	AutomationWindowContracts    []nodecontract.Contract
	CaptureWindowContract        nodecontract.Contract
	PlayInputClipContract        nodecontract.Contract
	PlayMacroContract            nodecontract.Contract
	MatchTemplateContract        nodecontract.Contract
	VisionAnalysisContracts      []nodecontract.Contract
	Types                        []datatype.Definition
	Contracts                    []nodecontract.Contract
	Capabilities                 []capability.Definition
	ConfigValidators             configvalidator.Registry
	definitions                  []BuiltinDefinition
	definitionByID               map[string]BuiltinDefinition
}

func (b Builtins) AIEvaluationArtifacts() []artifact.Digest {
	return []artifact.Digest{
		b.AIGeneratePrompt.Digest(), b.AIExtractPrompt.Digest(),
		b.AIAuthoringPrompt.Digest(), b.AIAuthoringToolSet.Digest(),
		b.AIGenerateContract.NodeRef().SemanticDigest, b.AIExtractContract.NodeRef().SemanticDigest,
	}
}

func (b Builtins) Definitions() []BuiltinDefinition {
	return append([]BuiltinDefinition(nil), b.definitions...)
}

func (b Builtins) Definition(nodeTypeID string) (BuiltinDefinition, bool) {
	definition, ok := b.definitionByID[nodeTypeID]
	return definition, ok
}

func Build() (Builtins, error) {
	stringType, err := sealStringType()
	if err != nil {
		return Builtins{}, err
	}
	binaryType, err := sealBinaryType()
	if err != nil {
		return Builtins{}, err
	}
	imageType, err := sealImageType()
	if err != nil {
		return Builtins{}, err
	}
	inputClipType, err := sealInputClipType()
	if err != nil {
		return Builtins{}, err
	}
	macroType, err := sealMacroType()
	if err != nil {
		return Builtins{}, err
	}
	numberType, err := sealPrimitiveType(NumberTypeID, "number", "type.core.number", "#38bdf8", "decimal", coreValueTraits(true, true))
	if err != nil {
		return Builtins{}, err
	}
	integerType, err := sealSafeIntegerType(numberType.TypeRef())
	if err != nil {
		return Builtins{}, err
	}
	booleanType, err := sealPrimitiveType(BooleanTypeID, "boolean", "type.core.boolean", "#f59e0b", "toggle-right", coreValueTraits(false, false))
	if err != nil {
		return Builtins{}, err
	}
	jsonType, pointUnitType, pointType, regionType, err := sealExtendedTypes(numberType.TypeRef())
	if err != nil {
		return Builtins{}, err
	}
	visionTypes, err := sealVisionTypes(stringType.TypeRef(), numberType.TypeRef(), integerType.TypeRef(), pointType.TypeRef(), regionType.TypeRef())
	if err != nil {
		return Builtins{}, err
	}
	pointerButtonType, keyCodeType, err := sealAutomationInputTypes()
	if err != nil {
		return Builtins{}, err
	}
	heldInputType, err := sealHeldInputType()
	if err != nil {
		return Builtins{}, err
	}
	randomDistributionType, err := sealRandomDistributionType()
	if err != nil {
		return Builtins{}, err
	}
	durationMillisecondsType, err := sealDurationMillisecondsType()
	if err != nil {
		return Builtins{}, err
	}
	fileMetadataType, err := sealFileMetadataType(stringType.TypeRef(), integerType.TypeRef(), booleanType.TypeRef())
	if err != nil {
		return Builtins{}, err
	}
	observabilityMessageType, err := sealObservabilityMessageType()
	if err != nil {
		return Builtins{}, err
	}
	blobRead, err := sealCapability(BlobReadCapabilityID, []string{"read-range"}, "blob-store")
	if err != nil {
		return Builtins{}, err
	}
	blobWrite, err := sealCapability(BlobWriteCapabilityID, []string{"append", "cancel", "commit"}, "blob-store")
	if err != nil {
		return Builtins{}, err
	}
	streamSession, err := sealCapability(StreamCapabilityID, []string{stream.OperationCancel, stream.OperationFinish, stream.OperationReceive, stream.OperationSend}, "stream-session")
	if err != nil {
		return Builtins{}, err
	}
	aiGeneration, err := sealAIGenerationCapability()
	if err != nil {
		return Builtins{}, err
	}
	filesystem, err := sealFilesystemCapability()
	if err != nil {
		return Builtins{}, err
	}
	httpGetCapability, err := sealHTTPGetCapability()
	if err != nil {
		return Builtins{}, err
	}
	applicationLifecycle, err := sealApplicationLifecycleCapability()
	if err != nil {
		return Builtins{}, err
	}
	automationInput, err := sealAutomationInputCapability()
	if err != nil {
		return Builtins{}, err
	}
	automationDesktopInput, err := sealAutomationDesktopInputCapability()
	if err != nil {
		return Builtins{}, err
	}
	automationKeyInput, err := sealAutomationKeyInputCapability()
	if err != nil {
		return Builtins{}, err
	}
	automationHeldInput, err := sealAutomationHeldInputCapability()
	if err != nil {
		return Builtins{}, err
	}
	automationWindow, err := sealAutomationWindowCapability()
	if err != nil {
		return Builtins{}, err
	}
	automationAppLifecycle, err := sealAutomationAppLifecycleCapability()
	if err != nil {
		return Builtins{}, err
	}
	automationCapture, err := sealAutomationCaptureCapability()
	if err != nil {
		return Builtins{}, err
	}
	automationPlayback, err := sealAutomationPlaybackCapability()
	if err != nil {
		return Builtins{}, err
	}
	configValidators, err := sealBuiltinConfigValidators()
	if err != nil {
		return Builtins{}, err
	}
	aiArtifacts, err := sealAIArtifacts()
	if err != nil {
		return Builtins{}, err
	}
	activateWindowDefinition, activateWindowContract, err := defineActivateWindowNode(automationWindow)
	if err != nil {
		return Builtins{}, err
	}
	_ = activateWindowContract
	stopTargetAppDefinition, _, err := defineStopTargetAppNode(automationAppLifecycle)
	if err != nil {
		return Builtins{}, err
	}
	captureWindowDefinition, captureWindowContract, err := defineCaptureWindowNode(imageType.TypeRef(), automationCapture, blobWrite)
	if err != nil {
		return Builtins{}, err
	}
	playInputClipDefinition, playInputClipContract, err := definePlayInputClipNode(inputClipType.TypeRef(), automationPlayback, blobRead)
	if err != nil {
		return Builtins{}, err
	}
	playMacroDefinition, playMacroContract, err := definePlayMacroNode(macroType.TypeRef(), automationPlayback, blobRead)
	if err != nil {
		return Builtins{}, err
	}
	matchTemplateDefinition, matchTemplateContract, err := defineMatchTemplateNode(extendedTypes{
		stringRef: stringType.TypeRef(), numberRef: numberType.TypeRef(), integerRef: integerType.TypeRef(), booleanRef: booleanType.TypeRef(),
		jsonRef: jsonType.TypeRef(), pointUnitRef: pointUnitType.TypeRef(), pointRef: pointType.TypeRef(), regionRef: regionType.TypeRef(),
	}, imageType.TypeRef(), blobRead)
	if err != nil {
		return Builtins{}, err
	}
	visionAnalysisDefinitions, visionAnalysisContracts, err := defineVisionAnalysisNodes(extendedTypes{
		stringRef: stringType.TypeRef(), numberRef: numberType.TypeRef(), integerRef: integerType.TypeRef(), booleanRef: booleanType.TypeRef(),
		jsonRef: jsonType.TypeRef(), pointUnitRef: pointUnitType.TypeRef(), pointRef: pointType.TypeRef(), regionRef: regionType.TypeRef(),
	}, visionTypes, imageType.TypeRef(), blobRead)
	if err != nil {
		return Builtins{}, err
	}
	blobToStream, err := sealBlobToStream(binaryType.TypeRef(), blobRead, streamSession)
	if err != nil {
		return Builtins{}, err
	}
	streamToBlob, err := sealStreamToBlob(binaryType.TypeRef(), blobWrite, streamSession)
	if err != nil {
		return Builtins{}, err
	}
	concat, err := sealConcat(stringType.TypeRef())
	if err != nil {
		return Builtins{}, err
	}
	concatDefinition, err := defineBuiltin(concat, concatEntrypoint, concatImplementationVersion, "utf8-string-concatenation/a+b/result", func(ctx context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
		return Concat(ctx, inputs)
	})
	if err != nil {
		return Builtins{}, err
	}
	blobToStreamDefinition, err := defineBuiltin(blobToStream, blobToStreamEntrypoint, conversionImplementationVersion, "blob-range-to-bounded-stream/v1", nil)
	if err != nil {
		return Builtins{}, err
	}
	streamToBlobDefinition, err := defineBuiltin(streamToBlob, streamToBlobEntrypoint, conversionImplementationVersion, "bounded-stream-to-content-addressed-blob/v1", nil)
	if err != nil {
		return Builtins{}, err
	}
	primitiveDefinitions, err := definePrimitiveNodes(primitiveTypes{
		stringRef: stringType.TypeRef(), numberRef: numberType.TypeRef(), integerRef: integerType.TypeRef(), booleanRef: booleanType.TypeRef(),
	})
	if err != nil {
		return Builtins{}, err
	}
	collectionDefinitions, err := defineCollectionNodes(primitiveTypes{
		stringRef: stringType.TypeRef(), numberRef: numberType.TypeRef(), integerRef: integerType.TypeRef(), booleanRef: booleanType.TypeRef(),
	})
	if err != nil {
		return Builtins{}, err
	}
	extendedDefinitions, err := defineExtendedPureNodes(extendedTypes{
		stringRef: stringType.TypeRef(), numberRef: numberType.TypeRef(), integerRef: integerType.TypeRef(), booleanRef: booleanType.TypeRef(),
		jsonRef: jsonType.TypeRef(), pointUnitRef: pointUnitType.TypeRef(), pointRef: pointType.TypeRef(), regionRef: regionType.TypeRef(),
	})
	if err != nil {
		return Builtins{}, err
	}
	recordedObservationDefinitions, err := defineRecordedObservationNodes(primitiveTypes{
		stringRef: stringType.TypeRef(), numberRef: numberType.TypeRef(), integerRef: integerType.TypeRef(), booleanRef: booleanType.TypeRef(),
	}, randomDistributionType.TypeRef())
	if err != nil {
		return Builtins{}, err
	}
	stateDefinitions, err := defineStateNodes(primitiveTypes{
		stringRef: stringType.TypeRef(), numberRef: numberType.TypeRef(), integerRef: integerType.TypeRef(), booleanRef: booleanType.TypeRef(),
	})
	if err != nil {
		return Builtins{}, err
	}
	controlDefinitions, err := defineControlNodes(primitiveTypes{
		stringRef: stringType.TypeRef(), numberRef: numberType.TypeRef(), integerRef: integerType.TypeRef(), booleanRef: booleanType.TypeRef(),
	}, durationMillisecondsType.TypeRef())
	if err != nil {
		return Builtins{}, err
	}
	controlCapabilityDefinitions, err := defineControlCapabilityNodes(primitiveTypes{
		stringRef: stringType.TypeRef(), numberRef: numberType.TypeRef(), integerRef: integerType.TypeRef(), booleanRef: booleanType.TypeRef(),
	})
	if err != nil {
		return Builtins{}, err
	}
	aiDefinitions, aiGenerate, aiExtract, err := defineAINodes(
		stringType.TypeRef(), jsonType.TypeRef(), imageType.TypeRef(), aiGeneration, blobRead, aiArtifacts,
	)
	if err != nil {
		return Builtins{}, err
	}
	scriptDefinition, scriptExecute, err := defineScriptNode(jsonType.TypeRef())
	if err != nil {
		return Builtins{}, err
	}
	filesystemDefinitions, filesystemContracts, err := defineFilesystemNodes(extendedTypes{
		stringRef: stringType.TypeRef(), jsonRef: jsonType.TypeRef(),
	}, fileMetadataType.TypeRef(), imageType.TypeRef(), filesystem, blobRead, blobWrite)
	if err != nil {
		return Builtins{}, err
	}
	httpGetDefinition, httpGetContract, err := defineHTTPGetNode(extendedTypes{
		stringRef: stringType.TypeRef(), jsonRef: jsonType.TypeRef(),
	}, integerType.TypeRef(), httpGetCapability)
	if err != nil {
		return Builtins{}, err
	}
	applicationDefinitions, applicationContracts, err := defineApplicationNodes(integerType.TypeRef(), applicationLifecycle)
	if err != nil {
		return Builtins{}, err
	}
	automationInputDefinitions, automationInputContracts, err := defineAutomationInputNodes(automationInputTypes{
		stringRef: stringType.TypeRef(), integerRef: integerType.TypeRef(), booleanRef: booleanType.TypeRef(), pointRef: pointType.TypeRef(),
		durationRef: durationMillisecondsType.TypeRef(), buttonRef: pointerButtonType.TypeRef(), keyCodeRef: keyCodeType.TypeRef(),
	}, automationInput, automationDesktopInput, automationKeyInput)
	if err != nil {
		return Builtins{}, err
	}
	automationHeldInputDefinitions, automationHeldInputContracts, err := defineAutomationHeldInputNodes(automationInputTypes{
		stringRef: stringType.TypeRef(), integerRef: integerType.TypeRef(), booleanRef: booleanType.TypeRef(), pointRef: pointType.TypeRef(),
		durationRef: durationMillisecondsType.TypeRef(), buttonRef: pointerButtonType.TypeRef(), keyCodeRef: keyCodeType.TypeRef(),
	}, heldInputType.TypeRef(), automationHeldInput)
	if err != nil {
		return Builtins{}, err
	}
	automationWindowDefinitions, automationWindowContracts, err := defineDesktopWindowOperationNodes(stringType.TypeRef(), integerType.TypeRef(), booleanType.TypeRef(), durationMillisecondsType.TypeRef(), automationWindow)
	if err != nil {
		return Builtins{}, err
	}
	automationTemplateDefinitions, automationTemplateContracts, err := defineAutomationTemplateNodes(automationTemplateTypes{
		imageRef: imageType.TypeRef(), numberRef: numberType.TypeRef(), booleanRef: booleanType.TypeRef(), pointRef: pointType.TypeRef(),
		regionRef: regionType.TypeRef(), durationRef: durationMillisecondsType.TypeRef(), buttonRef: pointerButtonType.TypeRef(),
	}, automationCapture, automationInput, blobRead)
	if err != nil {
		return Builtins{}, err
	}
	automationObservationDefinitions, err := defineAutomationObservationNodes(automationTemplateTypes{
		imageRef: imageType.TypeRef(), numberRef: numberType.TypeRef(), integerRef: integerType.TypeRef(), booleanRef: booleanType.TypeRef(), pointRef: pointType.TypeRef(),
		regionRef: regionType.TypeRef(), durationRef: durationMillisecondsType.TypeRef(), buttonRef: pointerButtonType.TypeRef(),
	}, automationCapture)
	if err != nil {
		return Builtins{}, err
	}
	systemDefinitions, err := defineSystemNodes(observabilityMessageType.TypeRef())
	if err != nil {
		return Builtins{}, err
	}
	types := []datatype.Definition{
		stringType, binaryType, imageType, inputClipType, macroType, numberType, integerType, booleanType, jsonType, pointUnitType, pointType, regionType,
		visionTypes.templateMatch, visionTypes.qrCode, visionTypes.colorRange, visionTypes.colorBlob,
		pointerButtonType, keyCodeType, heldInputType, randomDistributionType, durationMillisecondsType, fileMetadataType, observabilityMessageType,
	}
	structureDefinitions, err := defineStructureNodes(types)
	if err != nil {
		return Builtins{}, err
	}
	definitions := []BuiltinDefinition{concatDefinition, blobToStreamDefinition, streamToBlobDefinition}
	definitions = append(definitions, primitiveDefinitions...)
	definitions = append(definitions, collectionDefinitions...)
	definitions = append(definitions, extendedDefinitions...)
	definitions = append(definitions, recordedObservationDefinitions...)
	definitions = append(definitions, stateDefinitions...)
	definitions = append(definitions, controlDefinitions...)
	definitions = append(definitions, controlCapabilityDefinitions...)
	definitions = append(definitions, aiDefinitions...)
	definitions = append(definitions, scriptDefinition)
	definitions = append(definitions, filesystemDefinitions...)
	definitions = append(definitions, httpGetDefinition)
	definitions = append(definitions, applicationDefinitions...)
	definitions = append(definitions, automationInputDefinitions...)
	definitions = append(definitions, automationHeldInputDefinitions...)
	definitions = append(definitions, automationWindowDefinitions...)
	definitions = append(definitions, automationTemplateDefinitions...)
	definitions = append(definitions, automationObservationDefinitions...)
	definitions = append(definitions, activateWindowDefinition)
	definitions = append(definitions, stopTargetAppDefinition)
	definitions = append(definitions, captureWindowDefinition)
	definitions = append(definitions, playInputClipDefinition)
	definitions = append(definitions, playMacroDefinition)
	definitions = append(definitions, matchTemplateDefinition)
	definitions = append(definitions, visionAnalysisDefinitions...)
	definitions = append(definitions, systemDefinitions...)
	definitions = append(definitions, structureDefinitions...)
	bindings := make([]nodecatalog.Binding, 0, len(definitions))
	contracts := make([]nodecontract.Contract, 0, len(definitions))
	definitionByID := make(map[string]BuiltinDefinition, len(definitions))
	for _, definition := range definitions {
		nodeTypeID := definition.Contract.NodeRef().NodeTypeID
		if _, exists := definitionByID[nodeTypeID]; exists {
			return Builtins{}, fmt.Errorf("duplicate built-in definition %q", nodeTypeID)
		}
		definitionByID[nodeTypeID] = definition
		bindings = append(bindings, nodecatalog.Binding{Contract: definition.Contract, Implementation: definition.Implementation})
		contracts = append(contracts, definition.Contract)
	}
	if _, err := validateTypeCapabilityClosure(types, contracts); err != nil {
		return Builtins{}, fmt.Errorf("validate built-in type capability closure: %w", err)
	}
	capabilities := []capability.Definition{blobRead, blobWrite, streamSession, aiGeneration, filesystem, httpGetCapability, applicationLifecycle, automationInput, automationDesktopInput, automationKeyInput, automationHeldInput, automationWindow, automationAppLifecycle, automationCapture, automationPlayback}
	catalog, err := nodecatalog.Seal(types, capabilities, bindings, "v1")
	if err != nil {
		return Builtins{}, err
	}
	return Builtins{
		Catalog: catalog, StringType: stringType, BinaryType: binaryType, ImageType: imageType, InputClipType: inputClipType, MacroType: macroType, NumberType: numberType,
		IntegerType: integerType, BooleanType: booleanType, JSONType: jsonType,
		PointUnitType: pointUnitType, PointType: pointType, RegionType: regionType, ConcatContract: concat,
		TemplateMatchType: visionTypes.templateMatch, QRCodeType: visionTypes.qrCode, ColorRangeType: visionTypes.colorRange, ColorBlobType: visionTypes.colorBlob,
		PointerButtonType: pointerButtonType, KeyCodeType: keyCodeType,
		HeldInputType:            heldInputType,
		RandomDistributionType:   randomDistributionType,
		DurationMillisecondsType: durationMillisecondsType,
		FileMetadataType:         fileMetadataType,
		ObservabilityMessageType: observabilityMessageType,
		BlobToStreamContract:     blobToStream, StreamToBlobContract: streamToBlob,
		AIGenerateContract: aiGenerate, AIExtractContract: aiExtract,
		AIGeneratePrompt: aiArtifacts.generate, AIExtractPrompt: aiArtifacts.extract,
		AIAuthoringPrompt: aiArtifacts.authoring, AIAuthoringToolSet: aiArtifacts.authoringTools,
		ScriptExecuteContract: scriptExecute,
		FileReadTextContract:  filesystemContracts[0], FileReadJSONContract: filesystemContracts[1], FileStatContract: filesystemContracts[2],
		HTTPGetContract:           httpGetContract,
		LaunchApplicationContract: applicationContracts[0], TerminateApplicationContract: applicationContracts[1],
		AutomationInputContracts:     automationInputContracts,
		AutomationHeldInputContracts: automationHeldInputContracts,
		AutomationTemplateContracts:  automationTemplateContracts,
		ActivateWindowContract:       activateWindowContract,
		AutomationWindowContracts:    automationWindowContracts,
		CaptureWindowContract:        captureWindowContract,
		PlayInputClipContract:        playInputClipContract,
		PlayMacroContract:            playMacroContract,
		MatchTemplateContract:        matchTemplateContract,
		VisionAnalysisContracts:      visionAnalysisContracts,
		Types:                        types, Contracts: contracts, Capabilities: capabilities, ConfigValidators: configValidators,
		definitions: definitions, definitionByID: definitionByID,
	}, nil
}

func defineBuiltin(contract nodecontract.Contract, entrypoint, version, conformance string, evaluator InlineEvaluator) (BuiltinDefinition, error) {
	if !contract.Valid() || entrypoint == "" || version == "" || conformance == "" {
		return BuiltinDefinition{}, fmt.Errorf("built-in definition is incomplete")
	}
	abi := contract.Machine().ImplementationABI[0]
	digest, err := builtinImplementationDigest(entrypoint, version, conformance, abi)
	if err != nil {
		return BuiltinDefinition{}, err
	}
	return BuiltinDefinition{Contract: contract, Implementation: builtinLock(entrypoint, digest, abi), EvaluateInline: evaluator}, nil
}

func builtinImplementationDigest(entrypoint, version, conformance string, abi nodecontract.ABIRequirement) (artifact.Digest, error) {
	manifest, err := artifact.Marshal(map[string]any{
		"packageId":             "https://schemas.yotta.dev/packages/builtin/v1",
		"entrypoint":            entrypoint,
		"abi":                   abi,
		"implementationVersion": version,
		"conformance":           conformance,
	})
	if err != nil {
		return "", err
	}
	return artifact.Sum("yotta/builtin-implementation-manifest/v1", manifest)
}

func builtinLock(entrypoint string, digest artifact.Digest, abi nodecontract.ABIRequirement) nodecatalog.ImplementationLock {
	return nodecatalog.ImplementationLock{
		PackageID: "https://schemas.yotta.dev/packages/builtin/v1", ArtifactDigest: digest,
		ABI: abi, Entrypoint: entrypoint,
	}
}

func Concat(_ context.Context, inputs map[string]json.RawMessage) (map[string]json.RawMessage, error) {
	var a, b string
	if err := json.Unmarshal(inputs["a"], &a); err != nil {
		return nil, fmt.Errorf("decode concat input a: %w", err)
	}
	if err := json.Unmarshal(inputs["b"], &b); err != nil {
		return nil, fmt.Errorf("decode concat input b: %w", err)
	}
	result, err := json.Marshal(a + b)
	if err != nil {
		return nil, err
	}
	return map[string]json.RawMessage{"result": result}, nil
}

func sealStringType() (datatype.Definition, error) {
	const schemaID = StringTypeID + "/schema"
	return datatype.SealDefinition(datatype.DefinitionDraft{
		TypeID: StringTypeID, SchemaDialect: datatype.JSONSchemaDialect, SchemaRoot: schemaID,
		SchemaBundle: []datatype.SchemaResource{{ID: schemaID, Schema: json.RawMessage(`{
			"$id":"https://schemas.yotta.dev/types/core/string/v1/schema",
			"$schema":"https://json-schema.org/draft/2020-12/schema",
			"type":"string"
		}`)}},
		Representations: []datatype.RepresentationSpec{{Kind: datatype.RepresentationInlineJSON, Codec: datatype.CodecJCSV1}},
		Traits:          coreValueTraits(false, false),
		Authoring: datatype.Authoring{
			TitleKey: "type.core.string.title", DescriptionKey: "type.core.string.description", Color: "#8b5cf6", Icon: "text",
		},
	})
}

func sealBinaryType() (datatype.Definition, error) {
	const schemaID = BinaryTypeID + "/schema"
	return datatype.SealDefinition(datatype.DefinitionDraft{
		TypeID: BinaryTypeID, SchemaDialect: datatype.JSONSchemaDialect, SchemaRoot: schemaID,
		SchemaBundle: []datatype.SchemaResource{{ID: schemaID, Schema: json.RawMessage(`{
			"$id":"https://schemas.yotta.dev/types/core/binary/v1/schema",
			"$schema":"https://json-schema.org/draft/2020-12/schema"
		}`)}},
		Representations: []datatype.RepresentationSpec{
			{Kind: datatype.RepresentationBlobRef, Codec: datatype.CodecBlobRefV1},
			{Kind: datatype.RepresentationStreamRef, Codec: datatype.CodecStreamRefV1},
		},
		Authoring: datatype.Authoring{
			TitleKey: "type.core.binary.title", DescriptionKey: "type.core.binary.description", Color: "#0ea5e9", Icon: "binary",
		},
	})
}

func sealCapability(id string, operations []string, targetKind string) (capability.Definition, error) {
	scopeID := id + "/scope"
	return capability.SealDefinition(capability.DefinitionDraft{
		CapabilityID: id, Operations: operations, TargetKinds: []string{targetKind},
		ScopeSchemaRoot: scopeID, ScopeSchemaBundle: []datatype.SchemaResource{{ID: scopeID, Schema: json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema",
			"type":"object","additionalProperties":false
		}`, scopeID))}},
		Credential: capability.CredentialNone, Risk: capability.RiskLow, Consent: capability.ConsentNone,
		ProviderABI: "https://schemas.yotta.dev/provider-abi/resource/v1",
	})
}

func sealBlobToStream(binaryRef datatype.TypeRef, blobRead, streamSession capability.Definition) (nodecontract.Contract, error) {
	const schemaID = BlobToStreamNodeID + "/config"
	binaryType := datatype.RefExpression(binaryRef)
	return nodecontract.Seal(nodecontract.Draft{Version: BuiltinNodeVersion,
		NodeTypeID: BlobToStreamNodeID, ConfigSchemaRoot: schemaID,
		ConfigSchemaBundle: emptyConfigSchema(schemaID),
		Ports: nodecontract.PortSet{
			DataInputs: []nodecontract.DataInputPort{{ID: "blob", Type: binaryType, Required: true}},
			DataOutputs: []nodecontract.DataOutputPort{{
				ID: "stream", Type: binaryType,
				ResourceLease: &nodecontract.ResourceLeaseBinding{RequirementID: "stream", Operations: []string{stream.OperationCancel, stream.OperationReceive}},
			}},
			ExecInputs: []nodecontract.SignalPort{}, ExecOutputs: []nodecontract.SignalPort{},
			ErrorOutputs: []nodecontract.SignalPort{},
		},
		Execution: conversionExecution(BlobToStreamEffectID), Instruction: nodecontract.Invoke(),
		CapabilityRequirements: []capability.Requirement{
			requirement(blobRead, "blob-read", []string{"read-range"}, "blob-store"),
			requirement(streamSession, "stream", []string{stream.OperationCancel, stream.OperationFinish, stream.OperationReceive, stream.OperationSend}, "stream-session"),
		},
		Errors:            []nodecontract.ErrorSpec{{Code: "conversion.blob_to_stream_failed", Category: "adapter", RetryHint: false}},
		StatusEvents:      []nodecontract.StatusEventSpec{},
		ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
		Authoring: nodecontract.Authoring{
			TitleKey: "node.conversion.blobToStream.title", DescriptionKey: "node.conversion.blobToStream.description",
			Category: "conversion", Tags: []string{"blob", "stream", "conversion"}, Icon: "arrows-transfer-down",
		},
	})
}

func sealStreamToBlob(binaryRef datatype.TypeRef, blobWrite, streamSession capability.Definition) (nodecontract.Contract, error) {
	const schemaID = StreamToBlobNodeID + "/config"
	binaryType := datatype.RefExpression(binaryRef)
	return nodecontract.Seal(nodecontract.Draft{Version: BuiltinNodeVersion,
		NodeTypeID: StreamToBlobNodeID, ConfigSchemaRoot: schemaID,
		ConfigSchemaBundle: []datatype.SchemaResource{{ID: schemaID, Schema: json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema",
			"type":"object","properties":{"mediaType":{"type":"string",
			"x-yotta-title-key":"node.conversion.streamToBlob.config.mediaType.title",
			"x-yotta-description-key":"node.conversion.streamToBlob.config.mediaType.description",
			"examples":["application/octet-stream","image/png"],"minLength":3,"maxLength":255,
			"pattern":"^[a-z0-9][a-z0-9!#$&^_.+-]+/[a-z0-9][a-z0-9!#$&^_.+-]+$"}},
			"required":["mediaType"],"additionalProperties":false
		}`, schemaID))}},
		Ports: nodecontract.PortSet{
			DataInputs: []nodecontract.DataInputPort{{
				ID: "stream", Type: binaryType, Required: true,
				ResourceLease: &nodecontract.ResourceLeaseBinding{RequirementID: "stream", Operations: []string{stream.OperationCancel, stream.OperationReceive}},
			}},
			DataOutputs: []nodecontract.DataOutputPort{{ID: "blob", Type: binaryType}},
			ExecInputs:  []nodecontract.SignalPort{}, ExecOutputs: []nodecontract.SignalPort{},
			ErrorOutputs: []nodecontract.SignalPort{},
		},
		Execution: conversionExecution(StreamToBlobEffectID), Instruction: nodecontract.Invoke(),
		CapabilityRequirements: []capability.Requirement{
			requirement(blobWrite, "blob-write", []string{"append", "cancel", "commit"}, "blob-store"),
			requirement(streamSession, "stream", []string{stream.OperationCancel, stream.OperationReceive}, "stream-session"),
		},
		Errors:            []nodecontract.ErrorSpec{{Code: "conversion.stream_to_blob_failed", Category: "adapter", RetryHint: false}},
		StatusEvents:      []nodecontract.StatusEventSpec{},
		ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
		Authoring: nodecontract.Authoring{
			TitleKey: "node.conversion.streamToBlob.title", DescriptionKey: "node.conversion.streamToBlob.description",
			Category: "conversion", Tags: []string{"stream", "blob", "conversion"}, Icon: "arrows-transfer-up",
		},
	})
}

func emptyConfigSchema(id string) []datatype.SchemaResource {
	return []datatype.SchemaResource{{ID: id, Schema: json.RawMessage(fmt.Sprintf(`{
		"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema",
		"type":"object","additionalProperties":false
	}`, id))}}
}

func conversionExecution(effect nodecontract.EffectID) nodecontract.ExecutionSpec {
	return nodecontract.ExecutionSpec{
		Class: nodecontract.ExecutionEffect, Effects: []nodecontract.EffectID{effect}, Determinism: nodecontract.Recorded,
		Evaluation: nodecontract.EvaluationPull, Cache: nodecontract.CacheNone, Retry: nodecontract.RetryNever,
		Cancellation: nodecontract.CancellationCooperative, Timeout: nodecontract.TimeoutNone,
	}
}

func requirement(definition capability.Definition, id string, operations []string, target string) capability.Requirement {
	return capability.Requirement{
		ID: id, Capability: definition.Ref(), Operations: operations, TargetSlot: target, Scope: json.RawMessage(`{}`),
	}
}

func sealConcat(stringRef datatype.TypeRef) (nodecontract.Contract, error) {
	const schemaID = ConcatNodeID + "/config"
	stringType := datatype.RefExpression(stringRef)
	return nodecontract.Seal(nodecontract.Draft{Version: BuiltinNodeVersion,
		NodeTypeID: ConcatNodeID, ConfigSchemaRoot: schemaID,
		ConfigSchemaBundle: []datatype.SchemaResource{{ID: schemaID, Schema: json.RawMessage(`{
			"$id":"https://schemas.yotta.dev/nodes/text/concat/config",
			"$schema":"https://json-schema.org/draft/2020-12/schema",
			"type":"object","additionalProperties":false
		}`)}},
		Ports: nodecontract.PortSet{
			DataInputs: []nodecontract.DataInputPort{
				{ID: "a", Type: stringType, Required: true},
				{ID: "b", Type: stringType, Required: true},
			},
			DataOutputs: []nodecontract.DataOutputPort{{ID: "result", Type: stringType}},
			ExecInputs:  []nodecontract.SignalPort{}, ExecOutputs: []nodecontract.SignalPort{},
			ErrorOutputs: []nodecontract.SignalPort{},
		},
		Execution: nodecontract.ExecutionSpec{
			Class: nodecontract.ExecutionPureData, Effects: []nodecontract.EffectID{}, Determinism: nodecontract.Deterministic,
			Evaluation: nodecontract.EvaluationPull, Cache: nodecontract.CachePerRun, Retry: nodecontract.RetryNever,
			Cancellation: nodecontract.CancellationCooperative, Timeout: nodecontract.TimeoutNone,
		},
		Instruction:            nodecontract.Invoke(),
		CapabilityRequirements: []capability.Requirement{}, Errors: []nodecontract.ErrorSpec{}, StatusEvents: []nodecontract.StatusEventSpec{},
		ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
		Authoring: nodecontract.Authoring{
			TitleKey: "node.text.concat.title", DescriptionKey: "node.text.concat.description", Category: "text", Tags: []string{"text", "transform"}, Icon: "function",
		},
	})
}
