package nodes

import (
	"encoding/json"
	"fmt"

	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/workspacefs"
)

const (
	FileMetadataTypeID      = "https://schemas.yotta.dev/types/filesystem/metadata/v1"
	BreakFileMetadataNodeID = "https://schemas.yotta.dev/nodes/structure/break-file-metadata"

	FileReadTextNodeID  = "https://schemas.yotta.dev/nodes/filesystem/read-text"
	FileReadJSONNodeID  = "https://schemas.yotta.dev/nodes/filesystem/read-json"
	FileStatNodeID      = "https://schemas.yotta.dev/nodes/filesystem/stat"
	FileLoadImageNodeID = "https://schemas.yotta.dev/nodes/filesystem/load-image"
	FileSaveImageNodeID = "https://schemas.yotta.dev/nodes/filesystem/save-image"

	FilesystemCapabilityID = "https://schemas.yotta.dev/capabilities/filesystem/workspace/v1"
	FileReadEffectID       = "https://schemas.yotta.dev/effects/filesystem/read/v1"
	FileStatEffectID       = "https://schemas.yotta.dev/effects/filesystem/stat/v1"
	FileLoadImageEffectID  = "https://schemas.yotta.dev/effects/filesystem/load-image/v1"
	FileSaveImageEffectID  = "https://schemas.yotta.dev/effects/filesystem/save-image/v1"

	DefaultFileReadBytes  = 1 << 20
	DefaultImageFileBytes = 32 << 20
)

func sealFileMetadataType(stringRef, integerRef, booleanRef datatype.TypeRef) (datatype.Definition, error) {
	return sealStructuredTypeWithStructure(
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
			BreakTitleKey: "node.structure.breakFileMetadata.title", BreakDescriptionKey: "node.structure.breakFileMetadata.description",
			Examples: []json.RawMessage{json.RawMessage(`{"path":".","name":".","extension":"","mediaType":"application/octet-stream","size":0,"modifiedUnixMillis":0,"isDirectory":false}`)},
		},
		&datatype.StructureSpec{BreakNodeTypeID: BreakFileMetadataNodeID, Fields: []datatype.StructureField{
			{ID: "extension", Type: datatype.RefExpression(stringRef)},
			{ID: "is-directory", JSONKey: "isDirectory", Type: datatype.RefExpression(booleanRef)},
			{ID: "media-type", JSONKey: "mediaType", Type: datatype.RefExpression(stringRef)},
			{ID: "modified-unix-millis", JSONKey: "modifiedUnixMillis", Type: datatype.RefExpression(integerRef)},
			{ID: "name", Type: datatype.RefExpression(stringRef)},
			{ID: "path", Type: datatype.RefExpression(stringRef)},
			{ID: "size", Type: datatype.RefExpression(integerRef)},
		}},
	)
}

func sealFilesystemCapability() (capability.Definition, error) {
	const scopeID = FilesystemCapabilityID + "/scope"
	return capability.SealDefinition(capability.DefinitionDraft{
		CapabilityID: FilesystemCapabilityID,
		Operations: []string{
			workspacefs.OperationRead, workspacefs.OperationReadRange, workspacefs.OperationStat,
			workspacefs.OperationWriteAppend, workspacefs.OperationWriteCancel, workspacefs.OperationWriteCommit,
		},
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

func defineFilesystemNodes(types extendedTypes, metadataRef, imageRef datatype.TypeRef, readCapability, blobRead, blobWrite capability.Definition) ([]BuiltinDefinition, []nodecontract.Contract, error) {
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
		contract, err := nodecontract.Seal(nodecontract.Draft{Version: BuiltinNodeVersion,
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
	imageDefinitions, imageContracts, err := defineImageFileNodes(types.stringRef, metadataRef, imageRef, readCapability, blobRead, blobWrite)
	if err != nil {
		return nil, nil, err
	}
	definitions = append(definitions, imageDefinitions...)
	contracts = append(contracts, imageContracts...)
	return definitions, contracts, nil
}

func defineImageFileNodes(stringRef, metadataRef, imageRef datatype.TypeRef, workspace, blobRead, blobWrite capability.Definition) ([]BuiltinDefinition, []nodecontract.Contract, error) {
	stringType := datatype.RefExpression(stringRef)
	metadataType := datatype.RefExpression(metadataRef)
	imageType := datatype.RefExpression(imageRef)
	scope := json.RawMessage(fmt.Sprintf(`{"root":%q}`, workspacefs.ScopeRoot))
	type spec struct {
		id, entrypoint, key, icon, effect, conformance string
		ports                                          nodecontract.PortSet
		config                                         []datatype.SchemaResource
		requirements                                   []capability.Requirement
	}
	loadConfigID := FileLoadImageNodeID + "/config"
	saveConfigID := FileSaveImageNodeID + "/config"
	specs := []spec{
		{
			id: FileLoadImageNodeID, entrypoint: "filesystem.load-image", key: "node.filesystem.loadImage", icon: "photo-down", effect: FileLoadImageEffectID,
			conformance: "workspace-rooted-image-to-blob/v1",
			ports: nodecontract.PortSet{
				DataInputs:  []nodecontract.DataInputPort{{ID: "path", Type: stringType, Required: true}},
				DataOutputs: []nodecontract.DataOutputPort{{ID: "image", Type: imageType}, {ID: "metadata", Type: metadataType}},
				ExecInputs:  signalList("in"), ExecOutputs: signalList("completed"), ErrorOutputs: signalList("failed"),
			},
			config: []datatype.SchemaResource{{ID: loadConfigID, Schema: json.RawMessage(fmt.Sprintf(`{
				"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object",
				"properties":{"maxBytes":{"type":"integer","minimum":1,"maximum":%d,"default":%d,
				"x-yotta-title-key":"node.filesystem.config.maxBytes.title","x-yotta-description-key":"node.filesystem.config.maxBytes.description"}},
				"additionalProperties":false
			}`, loadConfigID, DefaultImageFileBytes, DefaultImageFileBytes))}},
			requirements: []capability.Requirement{
				{ID: "workspace-files", Capability: workspace.Ref(), Operations: []string{workspacefs.OperationReadRange, workspacefs.OperationStat}, TargetSlot: "workspace-files", Scope: scope},
				requirement(blobWrite, "blob-write", []string{"append", "cancel", "commit"}, "blob-store"),
			},
		},
		{
			id: FileSaveImageNodeID, entrypoint: "filesystem.save-image", key: "node.filesystem.saveImage", icon: "photo-up", effect: FileSaveImageEffectID,
			conformance: "blob-image-to-workspace-root/v1",
			ports: nodecontract.PortSet{
				DataInputs:  []nodecontract.DataInputPort{{ID: "image", Type: imageType, Required: true}, {ID: "path", Type: stringType, Required: true}},
				DataOutputs: []nodecontract.DataOutputPort{{ID: "metadata", Type: metadataType}},
				ExecInputs:  signalList("in"), ExecOutputs: signalList("completed"), ErrorOutputs: signalList("failed"),
			},
			config: []datatype.SchemaResource{{ID: saveConfigID, Schema: json.RawMessage(fmt.Sprintf(`{
				"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object",
				"properties":{"overwrite":{"type":"boolean","default":false,
				"x-yotta-title-key":"node.filesystem.saveImage.config.overwrite.title","x-yotta-description-key":"node.filesystem.saveImage.config.overwrite.description"}},
				"additionalProperties":false
			}`, saveConfigID))}},
			requirements: []capability.Requirement{
				{ID: "workspace-files", Capability: workspace.Ref(), Operations: []string{workspacefs.OperationWriteAppend, workspacefs.OperationWriteCancel, workspacefs.OperationWriteCommit}, TargetSlot: "workspace-files", Scope: scope},
				requirement(blobRead, "blob-read", []string{"read-range"}, "blob-store"),
			},
		},
	}
	definitions := make([]BuiltinDefinition, 0, len(specs))
	contracts := make([]nodecontract.Contract, 0, len(specs))
	for _, item := range specs {
		contract, err := nodecontract.Seal(nodecontract.Draft{
			Version: BuiltinNodeVersion, NodeTypeID: item.id, ConfigSchemaRoot: item.id + "/config", ConfigSchemaBundle: item.config,
			Ports: item.ports,
			Execution: nodecontract.ExecutionSpec{
				Class: nodecontract.ExecutionEffect, Effects: []nodecontract.EffectID{nodecontract.EffectID(item.effect)}, Determinism: nodecontract.Recorded,
				Evaluation: nodecontract.EvaluationPush, Cache: nodecontract.CacheNone, Retry: nodecontract.RetryNever,
				Cancellation: nodecontract.CancellationCooperative, Timeout: nodecontract.TimeoutNone,
			},
			Instruction: nodecontract.Invoke(), CapabilityRequirements: item.requirements,
			Errors: filesystemErrors(), StatusEvents: []nodecontract.StatusEventSpec{},
			ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
			Authoring: nodecontract.Authoring{
				TitleKey: item.key + ".title", DescriptionKey: item.key + ".description", Category: "io",
				Tags: []string{"file", "image", "io", "workspace"}, Icon: item.icon,
				Ports: dataPortHints(item.key, item.ports.DataInputs, item.ports.DataOutputs, nil),
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
		workspacefs.CodeWriteFailed,
		"filesystem.decode_failed", "filesystem.invalid_json",
	}
	result := make([]nodecontract.ErrorSpec, 0, len(codes))
	for _, code := range codes {
		result = append(result, nodecontract.ErrorSpec{Code: code, Category: "filesystem", RetryHint: false})
	}
	return result
}
