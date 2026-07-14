package nodes31

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/yottaapp/yotta/internal/nodecatalog"
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
		!bytes.Contains(artifacts.Presentation, []byte(`"format":"yotta.node-presentation"`)) {
		t.Fatalf("generated artifacts are not pinned to 3.1")
	}
	if _, err := nodecatalog.Open(artifacts.Catalog); err != nil {
		t.Fatalf("generated machine catalog is not a strict canonical artifact: %v", err)
	}
	if !bytes.Contains(artifacts.Documentation, []byte("Exec, Error, and Status ports: none.")) ||
		bytes.Contains(artifacts.Documentation, []byte("| exec | output | `out`")) {
		t.Fatalf("generated docs invented an exec output:\n%s", artifacts.Documentation)
	}
}
