package workspacefs

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/resource"
)

func TestProviderReadsOnlyRelativeFilesInsideWorkspaceRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "docs"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "hello.txt"), []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	provider, err := NewProvider(root, Limits{MaxReadBytes: 16})
	if err != nil {
		t.Fatal(err)
	}
	state := openTestSession(t, provider, []string{OperationRead, OperationStat})

	request, err := artifact.Marshal(ReadRequest{Path: "docs/hello.txt", MaxBytes: 16})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := provider.Invoke(context.Background(), state, OperationRead, request)
	if err != nil {
		t.Fatal(err)
	}
	var response ReadResponse
	if err := decodeExact(raw, &response); err != nil {
		t.Fatal(err)
	}
	if string(response.Data) != "hello" || response.Metadata.Path != "docs/hello.txt" || response.Metadata.Name != "hello.txt" || response.Metadata.Size != 5 || response.Metadata.IsDirectory {
		t.Fatalf("read response = %#v", response)
	}

	statRequest, err := artifact.Marshal(StatRequest{Path: "docs/hello.txt"})
	if err != nil {
		t.Fatal(err)
	}
	raw, err = provider.Invoke(context.Background(), state, OperationStat, statRequest)
	if err != nil {
		t.Fatal(err)
	}
	var metadata Metadata
	if err := decodeExact(raw, &metadata); err != nil || metadata.Path != "docs/hello.txt" {
		t.Fatalf("stat metadata = %#v, %v", metadata, err)
	}
	if err := provider.Close(context.Background(), state); err != nil {
		t.Fatal(err)
	}
}

func TestProviderRejectsAmbientPathsBudgetsAndForgedOpenScope(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "large.txt"), []byte("0123456789"), 0o600); err != nil {
		t.Fatal(err)
	}
	provider, err := NewProvider(root, Limits{MaxReadBytes: 8})
	if err != nil {
		t.Fatal(err)
	}
	state := openTestSession(t, provider, []string{OperationRead})

	for _, request := range []ReadRequest{
		{Path: "../outside.txt", MaxBytes: 8},
		{Path: filepath.Join(root, "large.txt"), MaxBytes: 8},
		{Path: "large.txt", MaxBytes: 9},
		{Path: "large.txt", MaxBytes: 8},
	} {
		raw, marshalErr := artifact.Marshal(request)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		if _, invokeErr := provider.Invoke(context.Background(), state, OperationRead, raw); invokeErr == nil {
			t.Fatalf("read accepted forbidden request %#v", request)
		}
	}

	badScope, err := artifact.Marshal(Scope{Root: "host-root"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Open(context.Background(), resource.ProviderOpenRequest{
		ProviderID: ProviderID, TargetID: TargetID, Kind: Kind, Operations: []string{OperationRead},
		Config: []byte(`{}`), CapabilityScope: badScope,
	}); err == nil {
		t.Fatal("provider accepted forged capability scope")
	}
	if _, err := provider.Open(context.Background(), resource.ProviderOpenRequest{
		ProviderID: ProviderID, TargetID: TargetID, Kind: Kind, Operations: []string{OperationRead, OperationRead},
		Config: []byte(`{}`), CapabilityScope: mustScope(t),
	}); err == nil {
		t.Fatal("provider accepted duplicate operations")
	}
}

func TestProviderRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(root, "escape.txt")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	provider, err := NewProvider(root, Limits{MaxReadBytes: 16})
	if err != nil {
		t.Fatal(err)
	}
	state := openTestSession(t, provider, []string{OperationRead})
	request, err := artifact.Marshal(ReadRequest{Path: "escape.txt", MaxBytes: 16})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Invoke(context.Background(), state, OperationRead, request); err == nil {
		t.Fatal("provider followed a symlink outside the workspace root")
	}
}

func openTestSession(t *testing.T, provider *Provider, operations []string) any {
	t.Helper()
	state, err := provider.Open(context.Background(), resource.ProviderOpenRequest{
		ProviderID: ProviderID, TargetID: TargetID, Kind: Kind, Operations: operations,
		Config: []byte(`{}`), CapabilityScope: mustScope(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func mustScope(t *testing.T) []byte {
	t.Helper()
	raw, err := artifact.Marshal(Scope{Root: ScopeRoot})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
