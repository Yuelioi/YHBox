package datatype

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/resource"
	"github.com/yottaapp/yotta/internal/stream"
)

type valueTypes map[string]Definition

func (types valueTypes) LookupType(typeID string) (Definition, bool) {
	definition, ok := types[typeID]
	return definition, ok
}

func TestInlineValueEnvelopeRoundTripsTypeAndCanonicalValue(t *testing.T) {
	definition, err := SealDefinition(DefinitionDraft{
		TypeID: "https://schemas.yotta.dev/types/test/string/v1", SchemaDialect: JSONSchemaDialect,
		SchemaRoot:      "https://schemas.yotta.dev/types/test/string/v1/schema",
		SchemaBundle:    []SchemaResource{{ID: "https://schemas.yotta.dev/types/test/string/v1/schema", Schema: []byte(`{"$id":"https://schemas.yotta.dev/types/test/string/v1/schema","$schema":"https://json-schema.org/draft/2020-12/schema","type":"string"}`)}},
		Representations: []RepresentationSpec{{Kind: RepresentationInlineJSON, Codec: CodecJCSV1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	wantType := RefResolvedType(definition.TypeRef())
	types := valueTypes{definition.TypeRef().TypeID: definition}
	sealed, err := SealInlineJSON(types, wantType, []byte(`"hello"`))
	if err != nil {
		t.Fatal(err)
	}
	opened, err := OpenValueEnvelope(types, sealed.Artifact())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(opened.InlineJSON(), []byte(`"hello"`)) || opened.Type().Ref == nil || *opened.Type().Ref != definition.TypeRef() {
		t.Fatalf("opened envelope lost type or value: %#v %s", opened.Type(), opened.InlineJSON())
	}
	forged := append([]byte(nil), sealed.Artifact()...)
	forged[len(forged)-2] ^= 1
	if _, err := OpenValueEnvelope(types, forged); err == nil {
		t.Fatal("accepted forged value envelope")
	}
}

func TestInlineValueEnvelopeValidatesResolvedListsRecursively(t *testing.T) {
	definition, err := SealDefinition(DefinitionDraft{
		TypeID: "https://schemas.yotta.dev/types/test/string/v1", SchemaDialect: JSONSchemaDialect,
		SchemaRoot:      "https://schemas.yotta.dev/types/test/string/v1/schema",
		SchemaBundle:    []SchemaResource{{ID: "https://schemas.yotta.dev/types/test/string/v1/schema", Schema: []byte(`{"$id":"https://schemas.yotta.dev/types/test/string/v1/schema","$schema":"https://json-schema.org/draft/2020-12/schema","type":"string"}`)}},
		Representations: []RepresentationSpec{{Kind: RepresentationInlineJSON, Codec: CodecJCSV1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	types := valueTypes{definition.TypeRef().TypeID: definition}
	resolved := ListResolvedType(RefResolvedType(definition.TypeRef()))
	sealed, err := SealInlineJSON(types, resolved, []byte(`["a","节点"]`))
	if err != nil {
		t.Fatal(err)
	}
	opened, err := OpenValueEnvelope(types, sealed.Artifact())
	if err != nil || !bytes.Equal(opened.InlineJSON(), []byte(`["a","节点"]`)) {
		t.Fatalf("list round trip = %s, %v", opened.InlineJSON(), err)
	}
	if _, err := SealInlineJSON(types, resolved, []byte(`["a",2]`)); err == nil {
		t.Fatal("resolved string list accepted a number element")
	}
}

func TestExternalValueEnvelopeCarriersRoundTripWithoutRawResources(t *testing.T) {
	definition, err := SealDefinition(DefinitionDraft{
		TypeID: "https://schemas.yotta.dev/types/test/external/v1", SchemaDialect: JSONSchemaDialect,
		SchemaRoot:   "https://schemas.yotta.dev/types/test/external/v1/schema",
		SchemaBundle: []SchemaResource{{ID: "https://schemas.yotta.dev/types/test/external/v1/schema", Schema: []byte(`{"$id":"https://schemas.yotta.dev/types/test/external/v1/schema","$schema":"https://json-schema.org/draft/2020-12/schema"}`)}},
		Representations: []RepresentationSpec{
			{Kind: RepresentationBlobRef, Codec: CodecBlobRefV1},
			{Kind: RepresentationStreamRef, Codec: CodecStreamRefV1},
			{Kind: RepresentationHandleRef, Codec: CodecHandleRefV1},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	resolved := RefResolvedType(definition.TypeRef())
	types := valueTypes{definition.TypeRef().TypeID: definition}
	blobRef := blob.BlobRef{
		MediaType: "application/octet-stream",
		Digest:    artifact.Digest("sha256:" + string(bytes.Repeat([]byte{'1'}, 64))),
		Size:      12,
	}
	blobEnvelope, err := SealBlobRef(types, resolved, blobRef)
	if err != nil {
		t.Fatal(err)
	}
	openedBlob, err := OpenValueEnvelope(types, blobEnvelope.Artifact())
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := openedBlob.BlobRef(); !ok || got != blobRef || openedBlob.Representation() != RepresentationBlobRef {
		t.Fatalf("blob carrier = %#v, %v", got, ok)
	}

	streamHandle := resource.Handle{
		Token: base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32)),
		Kind:  stream.Kind,
	}
	streamEnvelope, err := SealStreamRef(types, resolved, streamHandle)
	if err != nil {
		t.Fatal(err)
	}
	openedStream, err := OpenValueEnvelope(types, streamEnvelope.RuntimeArtifact())
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := openedStream.StreamRef(); !ok || got != streamHandle || openedStream.Representation() != RepresentationStreamRef || openedStream.Durable() {
		t.Fatalf("stream carrier = %#v, %v", got, ok)
	}
	if openedStream.Artifact() != nil {
		t.Fatal("runtime stream authority was exposed as a durable artifact")
	}
	handle := streamHandle
	handle.Kind = "test/session"
	handleEnvelope, err := SealHandleRef(types, resolved, handle)
	if err != nil {
		t.Fatal(err)
	}
	openedHandle, err := OpenValueEnvelope(types, handleEnvelope.RuntimeArtifact())
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := openedHandle.HandleRef(); !ok || got != handle || openedHandle.Representation() != RepresentationHandleRef || openedHandle.Durable() {
		t.Fatalf("handle carrier = %#v, %v", got, ok)
	}
	if openedHandle.Artifact() != nil {
		t.Fatal("runtime handle authority was exposed as a durable artifact")
	}
	_, isBlob := openedHandle.BlobRef()
	if openedHandle.InlineJSON() != nil || isBlob {
		t.Fatal("external carrier leaked through another representation accessor")
	}
}
