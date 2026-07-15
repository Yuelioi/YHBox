// Package nodes31 declares Yotta's explicitly assembled built-in Catalog 3.1.
package nodes31

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecatalog"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/stream"
)

const (
	StringTypeID       = "https://schemas.yotta.dev/types/core/string/v1"
	BinaryTypeID       = "https://schemas.yotta.dev/types/core/binary/v1"
	NumberTypeID       = "https://schemas.yotta.dev/types/core/number/v1"
	IntegerTypeID      = "https://schemas.yotta.dev/types/core/integer/v1"
	BooleanTypeID      = "https://schemas.yotta.dev/types/core/boolean/v1"
	ConcatNodeID       = "https://schemas.yotta.dev/nodes/text/concat/v1"
	BlobToStreamNodeID = "https://schemas.yotta.dev/nodes/conversion/blob-to-stream/v1"
	StreamToBlobNodeID = "https://schemas.yotta.dev/nodes/conversion/stream-to-blob/v1"

	BlobReadCapabilityID  = "https://schemas.yotta.dev/capabilities/blob/read/v1"
	BlobWriteCapabilityID = "https://schemas.yotta.dev/capabilities/blob/write/v1"
	StreamCapabilityID    = "https://schemas.yotta.dev/capabilities/stream/session/v1"
	BlobToStreamEffectID  = "https://schemas.yotta.dev/effects/conversion/blob-to-stream/v1"
	StreamToBlobEffectID  = "https://schemas.yotta.dev/effects/conversion/stream-to-blob/v1"

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
	Catalog              nodecatalog.Snapshot
	StringType           datatype.Definition
	BinaryType           datatype.Definition
	NumberType           datatype.Definition
	IntegerType          datatype.Definition
	BooleanType          datatype.Definition
	ConcatContract       nodecontract.Contract
	BlobToStreamContract nodecontract.Contract
	StreamToBlobContract nodecontract.Contract
	Types                []datatype.Definition
	Contracts            []nodecontract.Contract
	Capabilities         []capability.Definition
	definitions          []BuiltinDefinition
	definitionByID       map[string]BuiltinDefinition
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
	numberType, err := sealPrimitiveType(NumberTypeID, "number", "type.core.number", "#38bdf8", "decimal")
	if err != nil {
		return Builtins{}, err
	}
	integerType, err := sealPrimitiveType(IntegerTypeID, "integer", "type.core.integer", "#06b6d4", "number-123")
	if err != nil {
		return Builtins{}, err
	}
	booleanType, err := sealPrimitiveType(BooleanTypeID, "boolean", "type.core.boolean", "#f59e0b", "toggle-right")
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
	definitions := []BuiltinDefinition{concatDefinition, blobToStreamDefinition, streamToBlobDefinition}
	definitions = append(definitions, primitiveDefinitions...)
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
	types := []datatype.Definition{stringType, binaryType, numberType, integerType, booleanType}
	capabilities := []capability.Definition{blobRead, blobWrite, streamSession}
	catalog, err := nodecatalog.Seal(types, capabilities, bindings, "v1")
	if err != nil {
		return Builtins{}, err
	}
	return Builtins{
		Catalog: catalog, StringType: stringType, BinaryType: binaryType, NumberType: numberType,
		IntegerType: integerType, BooleanType: booleanType, ConcatContract: concat,
		BlobToStreamContract: blobToStream, StreamToBlobContract: streamToBlob,
		Types: types, Contracts: contracts, Capabilities: capabilities,
		definitions: definitions, definitionByID: definitionByID,
	}, nil
}

func defineBuiltin(contract nodecontract.Contract, entrypoint, version, conformance string, evaluator InlineEvaluator) (BuiltinDefinition, error) {
	if !contract.Valid() || entrypoint == "" || version == "" || conformance == "" {
		return BuiltinDefinition{}, fmt.Errorf("built-in definition is incomplete")
	}
	digest, err := builtinImplementationDigest(entrypoint, version, conformance)
	if err != nil {
		return BuiltinDefinition{}, err
	}
	return BuiltinDefinition{Contract: contract, Implementation: builtinLock(entrypoint, digest), EvaluateInline: evaluator}, nil
}

func builtinImplementationDigest(entrypoint, version, conformance string) (artifact.Digest, error) {
	manifest, err := artifact.Marshal(map[string]any{
		"packageId":             "https://schemas.yotta.dev/packages/builtin/v1",
		"entrypoint":            entrypoint,
		"abi":                   map[string]any{"kind": "builtin", "version": "v1"},
		"implementationVersion": version,
		"conformance":           conformance,
	})
	if err != nil {
		return "", err
	}
	return artifact.Sum("yotta/builtin-implementation-manifest/v1", manifest)
}

func builtinLock(entrypoint string, digest artifact.Digest) nodecatalog.ImplementationLock {
	return nodecatalog.ImplementationLock{
		PackageID: "https://schemas.yotta.dev/packages/builtin/v1", ArtifactDigest: digest,
		ABI: nodecontract.ABIRequirement{Kind: nodecontract.ABIBuiltin, Version: "v1"}, Entrypoint: entrypoint,
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
	return nodecontract.Seal(nodecontract.Draft{
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
		Execution: conversionExecution(BlobToStreamEffectID),
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
	return nodecontract.Seal(nodecontract.Draft{
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
		Execution: conversionExecution(StreamToBlobEffectID),
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
	return nodecontract.Seal(nodecontract.Draft{
		NodeTypeID: ConcatNodeID, ConfigSchemaRoot: schemaID,
		ConfigSchemaBundle: []datatype.SchemaResource{{ID: schemaID, Schema: json.RawMessage(`{
			"$id":"https://schemas.yotta.dev/nodes/text/concat/v1/config",
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
		CapabilityRequirements: []capability.Requirement{}, Errors: []nodecontract.ErrorSpec{}, StatusEvents: []nodecontract.StatusEventSpec{},
		ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
		Authoring: nodecontract.Authoring{
			TitleKey: "node.text.concat.title", DescriptionKey: "node.text.concat.description", Category: "text", Tags: []string{"text", "transform"}, Icon: "function",
		},
	})
}
