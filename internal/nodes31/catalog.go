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

type Builtins struct {
	Catalog              nodecatalog.Snapshot
	StringType           datatype.Definition
	BinaryType           datatype.Definition
	ConcatContract       nodecontract.Contract
	BlobToStreamContract nodecontract.Contract
	StreamToBlobContract nodecontract.Contract
	Types                []datatype.Definition
	Contracts            []nodecontract.Contract
}

func Build() (Builtins, error) {
	stringType, err := sealStringType()
	if err != nil {
		return Builtins{}, err
	}
	concat, err := sealConcat(stringType.TypeRef())
	if err != nil {
		return Builtins{}, err
	}
	binaryType, err := sealBinaryType()
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
	concatLock, err := BuiltinImplementationLock(ConcatNodeID)
	if err != nil {
		return Builtins{}, err
	}
	blobToStreamLock, err := BuiltinImplementationLock(BlobToStreamNodeID)
	if err != nil {
		return Builtins{}, err
	}
	streamToBlobLock, err := BuiltinImplementationLock(StreamToBlobNodeID)
	if err != nil {
		return Builtins{}, err
	}
	bindings := []nodecatalog.Binding{
		{Contract: concat, Implementation: concatLock},
		{Contract: blobToStream, Implementation: blobToStreamLock},
		{Contract: streamToBlob, Implementation: streamToBlobLock},
	}
	types := []datatype.Definition{stringType, binaryType}
	contracts := []nodecontract.Contract{concat, blobToStream, streamToBlob}
	catalog, err := nodecatalog.Seal(types, []capability.Definition{blobRead, blobWrite, streamSession}, bindings, "v1")
	if err != nil {
		return Builtins{}, err
	}
	return Builtins{
		Catalog: catalog, StringType: stringType, BinaryType: binaryType, ConcatContract: concat,
		BlobToStreamContract: blobToStream, StreamToBlobContract: streamToBlob,
		Types: types, Contracts: contracts,
	}, nil
}

// ConcatImplementationDigest identifies the installed builtin adapter
// manifest. The runtime additionally compares the complete lock before
// dispatch; bump the manifest version whenever its implementation/ABI changes.
func ConcatImplementationDigest() (artifact.Digest, error) {
	return builtinImplementationDigest(concatEntrypoint, concatImplementationVersion, "utf8-string-concatenation/a+b/result")
}

// BuiltinImplementationLock returns the independently trusted manifest lock
// for code compiled into this Yotta build. Runtime installation must compare
// this value with the Catalog instead of relabeling code with Catalog data.
func BuiltinImplementationLock(nodeTypeID string) (nodecatalog.ImplementationLock, error) {
	var entrypoint, version, conformance string
	switch nodeTypeID {
	case ConcatNodeID:
		entrypoint, version, conformance = concatEntrypoint, concatImplementationVersion, "utf8-string-concatenation/a+b/result"
	case BlobToStreamNodeID:
		entrypoint, version, conformance = blobToStreamEntrypoint, conversionImplementationVersion, "blob-range-to-bounded-stream/v1"
	case StreamToBlobNodeID:
		entrypoint, version, conformance = streamToBlobEntrypoint, conversionImplementationVersion, "bounded-stream-to-content-addressed-blob/v1"
	default:
		return nodecatalog.ImplementationLock{}, fmt.Errorf("unknown built-in node type %q", nodeTypeID)
	}
	digest, err := builtinImplementationDigest(entrypoint, version, conformance)
	if err != nil {
		return nodecatalog.ImplementationLock{}, err
	}
	return builtinLock(entrypoint, digest), nil
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
		TypeID: StringTypeID, SchemaDialect: datatype.JSONSchemaDialect,
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
		TypeID: BinaryTypeID, SchemaDialect: datatype.JSONSchemaDialect,
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
			ErrorOutputs: []nodecontract.SignalPort{}, StatusOutputs: []nodecontract.SignalPort{},
		},
		Execution: conversionExecution(BlobToStreamEffectID),
		CapabilityRequirements: []capability.Requirement{
			requirement(blobRead, "blob-read", []string{"read-range"}, "blob-store"),
			requirement(streamSession, "stream", []string{stream.OperationCancel, stream.OperationFinish, stream.OperationReceive, stream.OperationSend}, "stream-session"),
		},
		Errors:            []nodecontract.ErrorSpec{{Code: "conversion.blob_to_stream_failed", Category: "adapter", RetryHint: false}},
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
			"type":"object","properties":{"mediaType":{"type":"string","minLength":3,"maxLength":255,
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
			ErrorOutputs: []nodecontract.SignalPort{}, StatusOutputs: []nodecontract.SignalPort{},
		},
		Execution: conversionExecution(StreamToBlobEffectID),
		CapabilityRequirements: []capability.Requirement{
			requirement(blobWrite, "blob-write", []string{"append", "cancel", "commit"}, "blob-store"),
			requirement(streamSession, "stream", []string{stream.OperationCancel, stream.OperationReceive}, "stream-session"),
		},
		Errors:            []nodecontract.ErrorSpec{{Code: "conversion.stream_to_blob_failed", Category: "adapter", RetryHint: false}},
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
			ErrorOutputs: []nodecontract.SignalPort{}, StatusOutputs: []nodecontract.SignalPort{},
		},
		Execution: nodecontract.ExecutionSpec{
			Class: nodecontract.ExecutionPureData, Effects: []nodecontract.EffectID{}, Determinism: nodecontract.Deterministic,
			Evaluation: nodecontract.EvaluationPull, Cache: nodecontract.CachePerRun, Retry: nodecontract.RetryNever,
			Cancellation: nodecontract.CancellationCooperative, Timeout: nodecontract.TimeoutNone,
		},
		CapabilityRequirements: []capability.Requirement{}, Errors: []nodecontract.ErrorSpec{},
		ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
		Authoring: nodecontract.Authoring{
			TitleKey: "node.text.concat.title", DescriptionKey: "node.text.concat.description", Category: "text", Tags: []string{"text", "transform"}, Icon: "function",
		},
	})
}
