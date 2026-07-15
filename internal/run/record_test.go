package run_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/resource"
	run31 "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/stream"
)

type valueCatalog map[string]datatype.Definition

func (catalog valueCatalog) LookupType(id string) (datatype.Definition, bool) {
	definition, ok := catalog[id]
	return definition, ok
}

func TestRunRecordStateMachineRoundTripsDurableValues(t *testing.T) {
	catalog, definition := stringValueCatalog(t)
	queuedAt := time.Date(2026, 7, 15, 1, 0, 0, 0, time.UTC)
	record := queuedRecord(t, queuedAt)
	running, err := record.Start(queuedAt.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := datatype.SealInlineJSON(catalog, datatype.RefResolvedType(definition.TypeRef()), []byte(`"done"`))
	if err != nil {
		t.Fatal(err)
	}
	succeeded, err := running.Succeed(queuedAt.Add(2*time.Second), catalog, []run31.ProducedValue{{
		ValueID: "value-1", GraphID: "main", NodeID: "concat-1", PortID: "result", Attempt: 1, Envelope: envelope,
	}})
	if err != nil {
		t.Fatal(err)
	}
	opened, err := run31.OpenRecord(succeeded.Bytes(), catalog)
	if err != nil {
		t.Fatal(err)
	}
	if opened.Status() != run31.StatusSucceeded || opened.Digest() != succeeded.Digest() || !bytes.Equal(opened.Bytes(), succeeded.Bytes()) {
		t.Fatalf("opened run record = %s %s", opened.Status(), opened.Digest())
	}
	if _, err := opened.Start(queuedAt.Add(3 * time.Second)); err == nil {
		t.Fatal("terminal RunRecord transitioned again")
	}
}

func TestRunRecordRejectsRuntimeAuthorityAndTampering(t *testing.T) {
	catalog, definition := externalValueCatalog(t)
	queuedAt := time.Date(2026, 7, 15, 1, 0, 0, 0, time.UTC)
	record := queuedRecord(t, queuedAt)
	running, err := record.Start(queuedAt.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	handle := resource.Handle{Token: base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32)), Kind: stream.Kind, ExpiresAt: queuedAt.Add(time.Minute)}
	envelope, err := datatype.SealStreamRef(catalog, datatype.RefResolvedType(definition.TypeRef()), handle)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := running.Succeed(queuedAt.Add(2*time.Second), catalog, []run31.ProducedValue{{
		ValueID: "value-1", GraphID: "main", NodeID: "stream-1", PortID: "result", Attempt: 1, Envelope: envelope,
	}}); err == nil {
		t.Fatal("RunRecord persisted runtime-only authority")
	}
	failed, err := running.Fail(queuedAt.Add(2*time.Second), run31.RunError{Code: "node.failed", Category: run31.ErrorCategoryNode, Retryable: false, GraphID: "main", NodeID: "stream-1", Attempt: 1})
	if err != nil {
		t.Fatal(err)
	}
	tampered := bytes.Replace(failed.Bytes(), []byte(`"node.failed"`), []byte(`"node.forged"`), 1)
	if _, err := run31.OpenRecord(tampered, catalog); err == nil {
		t.Fatal("accepted tampered RunRecord")
	}
}

func TestRunRecordCancellationPreservesTemporalOrder(t *testing.T) {
	queuedAt := time.Date(2026, 7, 15, 1, 0, 0, 0, time.UTC)
	queued := queuedRecord(t, queuedAt)
	if cancelled, err := queued.Cancel(queuedAt); err != nil || cancelled.Status() != run31.StatusCancelled {
		t.Fatalf("queued cancellation = %s, %v", cancelled.Status(), err)
	}
	running, err := queued.Start(queuedAt.Add(2 * time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := running.Cancel(queuedAt.Add(time.Second)); err == nil {
		t.Fatal("running record accepted cancellation before start")
	}
}

func stringValueCatalog(t *testing.T) (valueCatalog, datatype.Definition) {
	t.Helper()
	const id = "https://schemas.yotta.dev/types/test/string/v1"
	definition, err := datatype.SealDefinition(datatype.DefinitionDraft{
		TypeID: id, SchemaDialect: datatype.JSONSchemaDialect,
		SchemaBundle:    []datatype.SchemaResource{{ID: id + "/schema", Schema: json.RawMessage(`{"$id":"https://schemas.yotta.dev/types/test/string/v1/schema","$schema":"https://json-schema.org/draft/2020-12/schema","type":"string"}`)}},
		Representations: []datatype.RepresentationSpec{{Kind: datatype.RepresentationInlineJSON, Codec: datatype.CodecJCSV1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return valueCatalog{id: definition}, definition
}

func externalValueCatalog(t *testing.T) (valueCatalog, datatype.Definition) {
	t.Helper()
	const id = "https://schemas.yotta.dev/types/test/stream/v1"
	definition, err := datatype.SealDefinition(datatype.DefinitionDraft{
		TypeID: id, SchemaDialect: datatype.JSONSchemaDialect,
		SchemaBundle:    []datatype.SchemaResource{{ID: id + "/schema", Schema: json.RawMessage(`{"$id":"https://schemas.yotta.dev/types/test/stream/v1/schema","$schema":"https://json-schema.org/draft/2020-12/schema"}`)}},
		Representations: []datatype.RepresentationSpec{{Kind: datatype.RepresentationStreamRef, Codec: datatype.CodecStreamRefV1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return valueCatalog{id: definition}, definition
}
