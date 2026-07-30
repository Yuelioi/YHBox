package blob_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/resource"
)

type allowBlobProvider struct{}

func (allowBlobProvider) AuthorizeOpen(context.Context, resource.OpenRequest) (resource.OpenAuthorization, error) {
	return resource.OpenAuthorization{CapabilityScope: []byte(`{}`)}, nil
}
func (allowBlobProvider) AuthorizeBorrow(context.Context, resource.BorrowRequest) error { return nil }
func (allowBlobProvider) AuthorizeCall(context.Context, resource.AuthorizationCall) error {
	return nil
}

func TestBlobProviderStreamsWriterAndReaderThroughBroker(t *testing.T) {
	store, err := blob.Open(t.TempDir(), blob.Limits{MaxBlobBytes: 1024, MaxTotalBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}
	provider, err := blob.NewProvider(store, blob.ProviderLimits{MaxChunkBytes: 8, QueueCapacity: 1})
	if err != nil {
		t.Fatal(err)
	}
	broker, err := resource.New(allowBlobProvider{}, map[string]resource.Provider{blob.ProviderID: provider}, resource.Options{MaxPayloadBytes: 1024})
	if err != nil {
		t.Fatal(err)
	}
	scope := blobProviderScope()
	writeConfig, _ := json.Marshal(blob.WriteConfig{MediaType: "text/plain"})
	writer, err := broker.Open(context.Background(), resource.OpenRequest{
		Scope: scope, ProviderID: blob.ProviderID, TargetID: "workspace", Kind: blob.KindWriter,
		Operations: []string{blob.OperationAppend, blob.OperationCancel, blob.OperationCommit},
		Config:     writeConfig,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, chunk := range []string{"hello ", "world"} {
		if _, err := broker.Invoke(context.Background(), resource.Call{Scope: scope, Handle: writer, Operation: blob.OperationAppend, Payload: []byte(chunk)}); err != nil {
			t.Fatal(err)
		}
	}
	rawRef, err := broker.Invoke(context.Background(), resource.Call{Scope: scope, Handle: writer, Operation: blob.OperationCommit})
	if err != nil {
		t.Fatal(err)
	}
	var ref blob.BlobRef
	if err := json.Unmarshal(rawRef, &ref); err != nil {
		t.Fatal(err)
	}
	if err := broker.Drop(context.Background(), scope, writer); err != nil {
		t.Fatal(err)
	}
	readConfig, _ := json.Marshal(blob.ReadConfig{Blob: ref})
	reader, err := broker.Open(context.Background(), resource.OpenRequest{
		Scope: scope, ProviderID: blob.ProviderID, TargetID: "workspace", Kind: blob.KindReader,
		Operations: []string{blob.OperationReadRange}, Config: readConfig,
	})
	if err != nil {
		t.Fatal(err)
	}
	rangePayload, _ := json.Marshal(blob.RangeRequest{Offset: 6, Length: 5})
	got, err := broker.Invoke(context.Background(), resource.Call{Scope: scope, Handle: reader, Operation: blob.OperationReadRange, Payload: rangePayload})
	if err != nil || string(got) != "world" {
		t.Fatalf("read = %q, %v", got, err)
	}
	if err := broker.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func blobProviderScope() resource.Scope {
	return resource.Scope{
		ProgramHash: artifact.Digest("sha256:" + strings.Repeat("1", 64)), CapabilityPlanDigest: artifact.Digest("sha256:" + strings.Repeat("2", 64)),
		GrantDigest: artifact.Digest("sha256:" + strings.Repeat("3", 64)), PolicyGeneration: "policy-1", RunID: "run-1", Principal: "user-1",
		PluginInstanceID: "builtin", SessionID: "session-1", GraphID: "main", NodeID: "node-1", RequirementID: "blob", InvocationID: "invoke-1",
	}
}
