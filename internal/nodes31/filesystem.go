package nodes31

import (
	"encoding/json"
	"fmt"

	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/workspacefs"
)

const (
	FileMetadataTypeID = "https://schemas.yotta.dev/types/filesystem/metadata/v1"

	FileReadTextNodeID = "https://schemas.yotta.dev/nodes/filesystem/read-text/v1"
	FileReadJSONNodeID = "https://schemas.yotta.dev/nodes/filesystem/read-json/v1"
	FileStatNodeID     = "https://schemas.yotta.dev/nodes/filesystem/stat/v1"

	FilesystemReadCapabilityID = "https://schemas.yotta.dev/capabilities/filesystem/read/v1"
	FileReadEffectID           = "https://schemas.yotta.dev/effects/filesystem/read/v1"
	FileStatEffectID           = "https://schemas.yotta.dev/effects/filesystem/stat/v1"

	DefaultFileReadBytes = 1 << 20
)

func sealFileMetadataType() (datatype.Definition, error) {
	return sealStructuredType(
		FileMetadataTypeID,
		json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object",
			"properties":{
				"path":{"type":"string","minLength":1,"maxLength":4096},
				"name":{"type":"string","minLength":1,"maxLength":255},
				"extension":{"type":"string","maxLength":255},
				"mediaType":{"type":"string","minLength":1,"maxLength":255},
				"size":{"type":"integer","minimum":0,"maximum":9007199254740991},
				"modifiedUnixMillis":{"type":"integer","minimum":-9007199254740991,"maximum":9007199254740991},
				"isDirectory":{"type":"boolean"}
			},
			"required":["path","name","extension","mediaType","size","modifiedUnixMillis","isDirectory"],
			"additionalProperties":false
		}`, FileMetadataTypeID+"/schema")),
		datatype.Authoring{
			TitleKey: "type.filesystem.metadata.title", DescriptionKey: "type.filesystem.metadata.description",
			Color: "#06b6d4", Icon: "file-info",
		},
	)
}

func sealFilesystemReadCapability() (capability.Definition, error) {
	const scopeID = FilesystemReadCapabilityID + "/scope"
	return capability.SealDefinition(capability.DefinitionDraft{
		CapabilityID:    FilesystemReadCapabilityID,
		Operations:      []string{workspacefs.OperationRead, workspacefs.OperationStat},
		TargetKinds:     []string{workspacefs.TargetKind},
		ScopeSchemaRoot: scopeID,
		ScopeSchemaBundle: []datatype.SchemaResource{{ID: scopeID, Schema: json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema",
			"type":"object","properties":{"root":{"const":%q}},"required":["root"],"additionalProperties":false
		}`, scopeID, workspacefs.ScopeRoot))}},
		Credential:  capability.CredentialNone,
		Risk:        capability.RiskLow,
		Consent:     capability.ConsentNone,
		ProviderABI: workspacefs.ProviderABI,
	})
}

func defineFilesystemNodes(types extendedTypes, metadataRef datatype.TypeRef, readCapability capability.Definition) ([]BuiltinDefinition, []nodecontract.Contract, error) {
	stringType := datatype.RefExpression(types.stringRef)
	jsonType := datatype.RefExpression(types.jsonRef)
	metadataType := datatype.RefExpression(metadataRef)
	pathInput := nodecontract.DataInputPort{ID: "path", Type: stringType, Required: true}

	type filesystemNode struct {
		id, entrypoint, conformance, key, icon, effect string
		operations                                     []string
		config                                         []datatype.SchemaResource
		outputs                                        []nodecontract.DataOutputPort
	}
	nodes := []filesystemNode{
		{
			id: FileReadTextNodeID, entrypoint: "filesystem.read-text", conformance: "workspace-rooted-bounded-text-read/v1",
			key: "node.filesystem.readText", icon: "file-text", effect: FileReadEffectID,
			operations: []string{workspacefs.OperationRead}, config: fileReadConfig(FileReadTextNodeID, true),
			outputs: []nodecontract.DataOutputPort{{ID: "text", Type: stringType}, {ID: "metadata", Type: metadataType}},
		},
		{
			id: FileReadJSONNodeID, entrypoint: "filesystem.read-json", conformance: "workspace-rooted-strict-json-read/v1",
			key: "node.filesystem.readJSON", icon: "json", effect: FileReadEffectID,
			operations: []string{workspacefs.OperationRead}, config: fileReadConfig(FileReadJSONNodeID, false),
			outputs: []nodecontract.DataOutputPort{{ID: "value", Type: jsonType}, {ID: "text", Type: stringType}, {ID: "metadata", Type: metadataType}},
		},
		{
			id: FileStatNodeID, entrypoint: "filesystem.stat", conformance: "workspace-rooted-file-metadata/v1",
			key: "node.filesystem.stat", icon: "file-info", effect: FileStatEffectID,
			operations: []string{workspacefs.OperationStat}, config: emptyConfigSchema(FileStatNodeID + "/config"),
			outputs: []nodecontract.DataOutputPort{{ID: "metadata", Type: metadataType}},
		},
	}

	definitions := make([]BuiltinDefinition, 0, len(nodes))
	contracts := make([]nodecontract.Contract, 0, len(nodes))
	for _, item := range nodes {
		scope := json.RawMessage(fmt.Sprintf(`{"root":%q}`, workspacefs.ScopeRoot))
		contract, err := nodecontract.Seal(nodecontract.Draft{
			NodeTypeID: item.id, ConfigSchemaRoot: item.id + "/config", ConfigSchemaBundle: item.config,
			Ports: nodecontract.PortSet{
				DataInputs: []nodecontract.DataInputPort{pathInput}, DataOutputs: item.outputs,
				ExecInputs: signalList("in"), ExecOutputs: signalList("completed"), ErrorOutputs: signalList("failed"),
			},
			Execution: nodecontract.ExecutionSpec{
				Class: nodecontract.ExecutionEffect, Effects: []nodecontract.EffectID{nodecontract.EffectID(item.effect)},
				Determinism: nodecontract.Recorded, Evaluation: nodecontract.EvaluationPush, Cache: nodecontract.CacheNone,
				Retry: nodecontract.RetryNever, Cancellation: nodecontract.CancellationCooperative, Timeout: nodecontract.TimeoutNone,
			},
			Instruction: nodecontract.Invoke(),
			CapabilityRequirements: []capability.Requirement{{
				ID: "workspace-files", Capability: readCapability.Ref(), Operations: item.operations,
				TargetSlot: "workspace-files", Scope: scope,
			}},
			Errors: filesystemErrors(), StatusEvents: []nodecontract.StatusEventSpec{},
			ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
			Authoring: nodecontract.Authoring{
				TitleKey: item.key + ".title", DescriptionKey: item.key + ".description", Category: "io",
				Tags: []string{"file", "filesystem", "io", "workspace"}, Icon: item.icon,
			},
		})
		if err != nil {
			return nil, nil, fmt.Errorf("seal filesystem node %s: %w", item.id, err)
		}
		definition, err := defineBuiltin(contract, item.entrypoint, "v1", item.conformance, nil)
		if err != nil {
			return nil, nil, err
		}
		definitions = append(definitions, definition)
		contracts = append(contracts, contract)
	}
	return definitions, contracts, nil
}

func fileReadConfig(nodeID string, withEncoding bool) []datatype.SchemaResource {
	schemaID := nodeID + "/config"
	encoding := ""
	if withEncoding {
		encoding = `,"encoding":{"type":"string","enum":["auto","utf-8","gbk"],"default":"auto",
			"x-yotta-title-key":"node.filesystem.config.encoding.title","x-yotta-description-key":"node.filesystem.config.encoding.description"}`
	}
	return []datatype.SchemaResource{{ID: schemaID, Schema: json.RawMessage(fmt.Sprintf(`{
		"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object",
		"properties":{"maxBytes":{"type":"integer","minimum":1,"maximum":%d,"default":%d,
			"x-yotta-title-key":"node.filesystem.config.maxBytes.title","x-yotta-description-key":"node.filesystem.config.maxBytes.description"}%s},
		"additionalProperties":false
	}`, schemaID, DefaultFileReadBytes, DefaultFileReadBytes, encoding))}}
}

func filesystemErrors() []nodecontract.ErrorSpec {
	codes := []string{
		workspacefs.CodeInvalidPath, workspacefs.CodeNotFound, workspacefs.CodeBudgetExceeded,
		workspacefs.CodeIsDirectory, workspacefs.CodeReadFailed, workspacefs.CodeContractViolation,
		"filesystem.decode_failed", "filesystem.invalid_json",
	}
	result := make([]nodecontract.ErrorSpec, 0, len(codes))
	for _, code := range codes {
		result = append(result, nodecontract.ErrorSpec{Code: code, Category: "filesystem", RetryHint: false})
	}
	return result
}
