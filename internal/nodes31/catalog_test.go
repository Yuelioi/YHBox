package nodes31

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodeauthoring"
	"github.com/yottaapp/yotta/internal/nodecatalog"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

func TestBuiltinsExposeConcatAsPureDataOnly(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := builtins.Catalog.Lookup(ConcatNodeID)
	if !ok {
		t.Fatal("missing concat")
	}
	ports := entry.Contract.Machine().Ports
	if len(ports.DataInputs) != 2 || len(ports.DataOutputs) != 1 || len(ports.ExecInputs)+len(ports.ExecOutputs) != 0 {
		t.Fatalf("concat ports = %#v", ports)
	}
	outputs, err := Concat(context.Background(), map[string]json.RawMessage{"a": json.RawMessage(`"a"`), "b": json.RawMessage(`"b"`)})
	if err != nil || string(outputs["result"]) != `"ab"` {
		t.Fatalf("outputs=%s err=%v", outputs["result"], err)
	}
}

func TestGeneratedArtifactsShareOneContractAndDocumentNoExecOut(t *testing.T) {
	artifacts, err := GenerateArtifacts()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(artifacts.Catalog, []byte(`"version":"3.1"`)) ||
		!bytes.Contains(artifacts.Authoring, []byte(`"format":"yotta.node-authoring-projection"`)) {
		t.Fatalf("generated artifacts are not pinned to 3.1")
	}
	if _, err := nodecatalog.Open(artifacts.Catalog); err != nil {
		t.Fatalf("generated machine catalog is not a strict canonical artifact: %v", err)
	}
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := nodeauthoring.Open(artifacts.Authoring, nodeauthoring.Input{
		Catalog: builtins.Catalog, Types: builtins.Types, Capabilities: builtins.Capabilities,
		Contracts: builtins.Contracts, GeneratorVersion: GeneratorVersion,
	}); err != nil {
		t.Fatalf("generated authoring projection cannot strict-open: %v", err)
	}
	if !bytes.Contains(artifacts.Documentation, []byte("Exec and Error ports: none.")) ||
		!bytes.Contains(artifacts.Documentation, []byte("Status events: none.")) ||
		bytes.Contains(artifacts.Documentation, []byte("| exec | output | `out`")) {
		t.Fatalf("generated docs invented an exec output:\n%s", artifacts.Documentation)
	}
	for _, fact := range []string{
		"Authoring projection:",
		"Availability: `target-required`",
		"`mediaType` | `text` | yes | `minLength: 3, maxLength: 255",
		"`stream` | `https://schemas.yotta.dev/types/core/binary/v1` | `durable-or-runtime` | `runtime`",
		"target `blob-store`",
		"Run state access:",
		"`state`: `write` slot selected by config `variable`; type `$T`",
	} {
		if !bytes.Contains(artifacts.Documentation, []byte(fact)) {
			t.Fatalf("generated docs omit projected fact %q:\n%s", fact, artifacts.Documentation)
		}
	}
}

func TestBuiltinsExposeExplicitBlobStreamConversions(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	binary := builtins.BinaryType.Machine()
	if len(binary.Representations) != 2 || binary.Representations[0].Kind != datatype.RepresentationBlobRef || binary.Representations[1].Kind != datatype.RepresentationStreamRef {
		t.Fatalf("binary representations = %#v", binary.Representations)
	}
	for _, nodeID := range []string{BlobToStreamNodeID, StreamToBlobNodeID} {
		entry, ok := builtins.Catalog.Lookup(nodeID)
		if !ok {
			t.Fatalf("missing conversion %s", nodeID)
		}
		machine := entry.Contract.Machine()
		if machine.Execution.Class != nodecontract.ExecutionEffect || len(machine.Ports.ExecInputs)+len(machine.Ports.ExecOutputs) != 0 ||
			len(machine.Ports.ErrorOutputs) != 0 || len(machine.StatusEvents) != 0 || len(machine.CapabilityRequirements) != 2 {
			t.Fatalf("conversion contract = %#v", machine)
		}
	}
	outputLease := builtins.BlobToStreamContract.Machine().Ports.DataOutputs[0].ResourceLease
	inputLease := builtins.StreamToBlobContract.Machine().Ports.DataInputs[0].ResourceLease
	if outputLease == nil || inputLease == nil || outputLease.RequirementID != "stream" || inputLease.RequirementID != "stream" {
		t.Fatalf("conversion leases = %#v / %#v", outputLease, inputLease)
	}
}
