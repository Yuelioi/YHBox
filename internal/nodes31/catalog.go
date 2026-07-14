// Package nodes31 declares Yotta's explicitly assembled built-in Catalog 3.1.
package nodes31

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecatalog"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

const (
	StringTypeID = "https://schemas.yotta.dev/types/core/string/v1"
	ConcatNodeID = "https://schemas.yotta.dev/nodes/text/concat/v1"

	concatEntrypoint            = "text.concat"
	concatImplementationVersion = "v1"
)

type Builtins struct {
	Catalog        nodecatalog.Snapshot
	StringType     datatype.Definition
	ConcatContract nodecontract.Contract
}

func Build() (Builtins, error) {
	implementationArtifact, err := ConcatImplementationDigest()
	if err != nil {
		return Builtins{}, err
	}
	stringType, err := sealStringType()
	if err != nil {
		return Builtins{}, err
	}
	concat, err := sealConcat(stringType.TypeRef())
	if err != nil {
		return Builtins{}, err
	}
	catalog, err := nodecatalog.Seal([]datatype.Definition{stringType}, []nodecatalog.Binding{{
		Contract: concat,
		Implementation: nodecatalog.ImplementationLock{
			PackageID: "https://schemas.yotta.dev/packages/builtin/v1", ArtifactDigest: implementationArtifact,
			ABI: nodecontract.ABIRequirement{Kind: nodecontract.ABIBuiltin, Version: "v1"}, Entrypoint: concatEntrypoint,
		},
	}}, "v1")
	if err != nil {
		return Builtins{}, err
	}
	return Builtins{Catalog: catalog, StringType: stringType, ConcatContract: concat}, nil
}

// ConcatImplementationDigest identifies the installed builtin adapter
// manifest. The runtime additionally compares the complete lock before
// dispatch; bump the manifest version whenever its implementation/ABI changes.
func ConcatImplementationDigest() (artifact.Digest, error) {
	manifest, err := artifact.Marshal(map[string]any{
		"packageId":             "https://schemas.yotta.dev/packages/builtin/v1",
		"entrypoint":            concatEntrypoint,
		"abi":                   map[string]any{"kind": "builtin", "version": "v1"},
		"implementationVersion": concatImplementationVersion,
		"conformance":           "utf8-string-concatenation/a+b/result",
	})
	if err != nil {
		return "", err
	}
	return artifact.Sum("yotta/builtin-implementation-manifest/v1", manifest)
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
		Capabilities: []nodecontract.CapabilityID{}, Errors: []nodecontract.ErrorSpec{},
		ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
		Authoring: nodecontract.Authoring{
			TitleKey: "node.text.concat.title", DescriptionKey: "node.text.concat.description", Category: "text", Tags: []string{"text", "transform"}, Icon: "function",
		},
	})
}
