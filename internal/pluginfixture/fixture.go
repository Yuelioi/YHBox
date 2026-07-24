// Package pluginfixture builds ephemeral signed example releases for local
// conformance and smoke runs. Its trust key is generated per fixture set and is
// never a production publisher authority.
package pluginfixture

import (
	"archive/zip"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/hostapi"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/nodepackage"
	"github.com/yottaapp/yotta/internal/pluginprotocol"
)

const Namespace = "https://fixtures.example.test/yotta"

type Set struct {
	Store           *nodepackage.Store
	ProcessManifest nodepackage.Manifest
	WasmManifest    nodepackage.Manifest
	StringType      datatype.Definition
}

func Create(ctx context.Context, root string, processExecutable, wasmModule []byte) (Set, error) {
	if ctx == nil || len(processExecutable) == 0 || len(wasmModule) == 0 {
		return Set{}, fmt.Errorf("plugin fixtures require context and both payloads")
	}
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return Set{}, err
	}
	policy, err := nodepackage.SealTrustPolicy(nodepackage.TrustPolicyDraft{
		Revision: 1, Publishers: []nodepackage.PublisherAuthorityDraft{{Namespace: Namespace, Keys: []ed25519.PublicKey{publicKey}}},
	})
	if err != nil {
		return Set{}, err
	}
	store, err := nodepackage.CreateStore(ctx, root, policy)
	if err != nil {
		return Set{}, err
	}
	stringType, err := sealStringType()
	if err != nil {
		return Set{}, err
	}
	processManifest, processArchive, err := release(root, privateKey, "process-uppercase", nodecontract.ABIProcess, processExecutable, stringType, true)
	if err != nil {
		return Set{}, err
	}
	wasmManifest, wasmArchive, err := release(root, privateKey, "wasm-minimal", nodecontract.ABIWIT, wasmModule, stringType, false)
	if err != nil {
		return Set{}, err
	}
	for _, archive := range []string{processArchive, wasmArchive} {
		if _, err := store.InstallArchive(ctx, archive); err != nil {
			return Set{}, err
		}
	}
	return Set{Store: store, ProcessManifest: processManifest, WasmManifest: wasmManifest, StringType: stringType}, nil
}

func MinimalWasmModule() ([]byte, error) {
	frame, err := pluginprotocol.MarshalFrame(&pluginprotocol.Frame{Protocol: pluginprotocol.Protocol, Sequence: 2,
		Payload: &pluginprotocol.Frame_Result{Result: &pluginprotocol.Result{
			Outcome: pluginprotocol.Outcome_OUTCOME_SUCCEEDED, TerminationStrength: "cooperative",
		}}})
	if err != nil {
		return nil, err
	}
	module := []byte{'\x00', 'a', 's', 'm', '\x01', '\x00', '\x00', '\x00'}
	types := []byte{3,
		0x60, 4, 0x7f, 0x7f, 0x7f, 0x7f, 1, 0x7e,
		0x60, 1, 0x7f, 1, 0x7f,
		0x60, 2, 0x7f, 0x7f, 1, 0x7f,
	}
	module = section(module, 1, types)
	imports := append([]byte{1}, name("yotta_plugin_v1")...)
	imports = append(imports, name("exchange")...)
	imports = append(imports, 0, 0)
	module = section(module, 2, imports)
	module = section(module, 3, []byte{2, 1, 2})
	module = section(module, 5, []byte{1, 0, 2})
	exports := []byte{3}
	exports = append(exports, name("memory")...)
	exports = append(exports, 2, 0)
	exports = append(exports, name("yotta_alloc")...)
	exports = append(exports, 0, 1)
	exports = append(exports, name("yotta_run")...)
	exports = append(exports, 0, 2)
	module = section(module, 7, exports)
	allocateBody := []byte{0, 0x41, 0x80, 0x08, 0x0b}
	runBody := []byte{0, 0x41, 0}
	runBody = append(runBody, 0x41)
	runBody = append(runBody, uleb(uint32(len(frame)))...)
	runBody = append(runBody, 0x41, 0, 0x41, 0, 0x10, 0, 0x1a, 0x41, 0, 0x0b)
	code := []byte{2}
	code = append(code, uleb(uint32(len(allocateBody)))...)
	code = append(code, allocateBody...)
	code = append(code, uleb(uint32(len(runBody)))...)
	code = append(code, runBody...)
	module = section(module, 10, code)
	data := []byte{1, 0, 0x41, 0, 0x0b}
	data = append(data, uleb(uint32(len(frame)))...)
	data = append(data, frame...)
	return section(module, 11, data), nil
}

func sealStringType() (datatype.Definition, error) {
	const typeID = Namespace + "/types/word/v1"
	return datatype.SealDefinition(datatype.DefinitionDraft{
		TypeID: typeID, SchemaDialect: datatype.JSONSchemaDialect, SchemaRoot: typeID + "/schema",
		SchemaBundle: []datatype.SchemaResource{{ID: typeID + "/schema", Schema: json.RawMessage(
			`{"$id":"https://fixtures.example.test/yotta/types/word/v1/schema","$schema":"https://json-schema.org/draft/2020-12/schema","type":"string"}`)}},
		Representations: []datatype.RepresentationSpec{{Kind: datatype.RepresentationInlineJSON, Codec: datatype.CodecJCSV1}},
		Authoring:       datatype.Authoring{TitleKey: "fixture.word.title"},
	})
}

func release(root string, key ed25519.PrivateKey, name string, kind nodecontract.ABIKind, payload []byte, stringType datatype.Definition, transform bool) (nodepackage.Manifest, string, error) {
	nodeID := Namespace + "/nodes/" + name
	featureID := pluginprotocol.ProcessIsolationHostFeatureID
	path, mediaType := "bin/plugin.exe", "application/vnd.microsoft.portable-executable"
	ports := nodecontract.PortSet{DataInputs: []nodecontract.DataInputPort{}, DataOutputs: []nodecontract.DataOutputPort{}, ExecInputs: []nodecontract.SignalPort{}, ExecOutputs: []nodecontract.SignalPort{}, ErrorOutputs: []nodecontract.SignalPort{}}
	if kind == nodecontract.ABIWIT {
		featureID, path, mediaType = pluginprotocol.WasmIsolationHostFeatureID, "bin/plugin.wasm", "application/wasm"
	}
	if transform {
		ports.DataInputs = []nodecontract.DataInputPort{{ID: "value", Type: datatype.RefExpression(stringType.TypeRef()), Required: true}}
		ports.DataOutputs = []nodecontract.DataOutputPort{{ID: "result", Type: datatype.RefExpression(stringType.TypeRef())}}
	}
	contract, err := nodecontract.Seal(nodecontract.Draft{
		NodeTypeID: nodeID, Version: "1.0.0", ConfigSchemaRoot: nodeID + "/config",
		ConfigSchemaBundle: []datatype.SchemaResource{{ID: nodeID + "/config", Schema: json.RawMessage(fmt.Sprintf(
			`{"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","additionalProperties":false}`, nodeID+"/config"))}},
		Ports: ports,
		Execution: nodecontract.ExecutionSpec{
			Class: nodecontract.ExecutionPureData, Effects: []nodecontract.EffectID{}, Determinism: nodecontract.Deterministic,
			Evaluation: nodecontract.EvaluationPull, Cache: nodecontract.CachePerRun, Retry: nodecontract.RetryNever,
			Cancellation: nodecontract.CancellationCooperative, Timeout: nodecontract.TimeoutNone,
		},
		Instruction: nodecontract.Invoke(), HostFeatureRequirements: []nodecontract.HostFeatureRequirement{{ID: "isolation", FeatureID: featureID}},
		CapabilityRequirements: []capability.Requirement{}, Errors: []nodecontract.ErrorSpec{}, StatusEvents: []nodecontract.StatusEventSpec{},
		ImplementationABI: []nodecontract.ABIRequirement{{Kind: kind, Version: "v1"}}, Authoring: nodecontract.Authoring{TitleKey: "fixture." + name + ".title", Tags: []string{"fixture"}},
	})
	if err != nil {
		return nodepackage.Manifest{}, "", err
	}
	digest := sha256.Sum256(payload)
	types := []datatype.Definition{}
	if transform {
		types = []datatype.Definition{stringType}
	}
	manifest, err := nodepackage.Seal(nodepackage.Draft{
		PublisherNamespace: Namespace, PackageID: Namespace + "/packages/" + name + "/v1", PackageVersion: "1.0.0",
		HostAPI: nodepackage.HostAPIRange{Min: hostapi.Current, MaxExclusive: hostapi.NextMajor}, Types: types,
		Nodes: []nodepackage.NodeDraft{{Contract: contract, Implementation: nodepackage.Implementation{
			ABI: nodecontract.ABIRequirement{Kind: kind, Version: "v1"}, Entrypoint: "fixture:" + name + "/node#run",
			Payload:   nodepackage.Payload{Path: path, Digest: artifact.Digest("sha256:" + hex.EncodeToString(digest[:])), Size: int64(len(payload)), MediaType: mediaType},
			Platforms: nodepackage.PlatformSupport{OperatingSystems: []string{runtime.GOOS}, Architectures: []string{runtime.GOARCH}},
		}}},
	})
	if err != nil {
		return nodepackage.Manifest{}, "", err
	}
	signature, err := nodepackage.SignManifest(manifest, key)
	if err != nil {
		return nodepackage.Manifest{}, "", err
	}
	archivePath := filepath.Join(root, name+".ynp")
	file, err := os.Create(archivePath)
	if err != nil {
		return nodepackage.Manifest{}, "", err
	}
	archive := zip.NewWriter(file)
	entries := []struct {
		name string
		data []byte
	}{{nodepackage.ArchiveManifestPath, manifest.Bytes()}, {nodepackage.ArchiveSignaturePath, signature.Bytes()}, {path, payload}}
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.name, Method: zip.Deflate}
		if entry.name == path && kind == nodecontract.ABIProcess {
			header.SetMode(0o755)
		}
		writer, createErr := archive.CreateHeader(header)
		if createErr != nil {
			_ = archive.Close()
			_ = file.Close()
			return nodepackage.Manifest{}, "", createErr
		}
		if _, writeErr := writer.Write(entry.data); writeErr != nil {
			_ = archive.Close()
			_ = file.Close()
			return nodepackage.Manifest{}, "", writeErr
		}
	}
	if err := archive.Close(); err != nil {
		_ = file.Close()
		return nodepackage.Manifest{}, "", err
	}
	if err := file.Close(); err != nil {
		return nodepackage.Manifest{}, "", err
	}
	return manifest, archivePath, nil
}

func section(module []byte, id byte, payload []byte) []byte {
	module = append(module, id)
	module = append(module, uleb(uint32(len(payload)))...)
	return append(module, payload...)
}

func name(value string) []byte {
	return append(uleb(uint32(len(value))), value...)
}

func uleb(value uint32) []byte {
	var scratch [binary.MaxVarintLen32]byte
	count := binary.PutUvarint(scratch[:], uint64(value))
	return append([]byte(nil), scratch[:count]...)
}
