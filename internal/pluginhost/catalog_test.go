package pluginhost

import (
	"encoding/json"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecatalog"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/nodepackage"
	"github.com/yottaapp/yotta/internal/nodes"
)

func TestMergeCatalogPinsPackageGenerationAndRejectsEntrypointConflicts(t *testing.T) {
	builtins, err := nodes.Build()
	if err != nil {
		t.Fatal(err)
	}
	contract := pluginContract(t, builtins.StringType.TypeRef())
	manifestDigest, err := artifact.Sum("yotta/test/plugin-manifest/v1", []byte("plugin"))
	if err != nil {
		t.Fatal(err)
	}
	implementation := nodepackage.Implementation{
		ABI:        nodecontract.ABIRequirement{Kind: nodecontract.ABIWIT, Version: "v1"},
		Entrypoint: "acme:transform/uppercase#run",
	}
	lock := nodecatalog.ImplementationLock{
		PackageID: "https://packages.example.test/acme/packages/transform/v1", ArtifactDigest: manifestDigest,
		ABI: implementation.ABI, Entrypoint: implementation.Entrypoint,
	}
	packages := []nodepackage.RuntimePackage{{
		PackageID: lock.PackageID, PackageVersion: "1.0.0", ManifestDigest: manifestDigest,
		Nodes: []nodepackage.RuntimeNode{{Contract: contract, Implementation: implementation, Lock: lock}},
	}}

	projection, err := MergeCatalog(builtins.Catalog, packages, "v1")
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := projection.Catalog.Lookup(contract.NodeRef().NodeTypeID)
	if !ok || entry.Contract.NodeRef() != contract.NodeRef() || entry.Implementation != lock {
		t.Fatalf("merged plugin entry = %#v, found=%v", entry, ok)
	}
	packages[0].Nodes = nil
	if len(projection.Packages[0].Nodes) != 1 {
		t.Fatal("CatalogProjection retained mutable package slices")
	}

	conflicting := projection.Packages[0]
	conflicting.Nodes[0].Lock.Entrypoint = builtins.Definitions()[0].Implementation.Entrypoint
	conflicting.Nodes[0].Implementation.Entrypoint = conflicting.Nodes[0].Lock.Entrypoint
	if _, err := MergeCatalog(builtins.Catalog, []nodepackage.RuntimePackage{conflicting}, "v1"); err == nil {
		t.Fatal("plugin Catalog accepted an entrypoint conflict")
	}
}

func pluginContract(t *testing.T, stringRef datatype.TypeRef) nodecontract.Contract {
	t.Helper()
	const nodeID = "https://packages.example.test/acme/nodes/uppercase"
	const schemaID = nodeID + "/config"
	contract, err := nodecontract.Seal(nodecontract.Draft{
		NodeTypeID: nodeID, Version: "1.0.0", ConfigSchemaRoot: schemaID,
		ConfigSchemaBundle: []datatype.SchemaResource{{ID: schemaID, Schema: json.RawMessage(
			`{"$id":"https://packages.example.test/acme/nodes/uppercase/config","$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","additionalProperties":false}`,
		)}},
		Ports: nodecontract.PortSet{
			DataInputs:  []nodecontract.DataInputPort{{ID: "value", Type: datatype.RefExpression(stringRef), Required: true}},
			DataOutputs: []nodecontract.DataOutputPort{{ID: "result", Type: datatype.RefExpression(stringRef)}},
			ExecInputs:  []nodecontract.SignalPort{}, ExecOutputs: []nodecontract.SignalPort{}, ErrorOutputs: []nodecontract.SignalPort{},
		},
		Execution: nodecontract.ExecutionSpec{
			Class: nodecontract.ExecutionPureData, Effects: []nodecontract.EffectID{}, Determinism: nodecontract.Deterministic,
			Evaluation: nodecontract.EvaluationPull, Cache: nodecontract.CachePerRun, Retry: nodecontract.RetryNever,
			Cancellation: nodecontract.CancellationCooperative, Timeout: nodecontract.TimeoutNone,
		},
		Instruction: nodecontract.Invoke(), CapabilityRequirements: []capability.Requirement{},
		Errors: []nodecontract.ErrorSpec{}, StatusEvents: []nodecontract.StatusEventSpec{},
		ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIWIT, Version: "v1"}},
		Authoring:         nodecontract.Authoring{TitleKey: "acme.uppercase.title", Tags: []string{"text"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return contract
}
