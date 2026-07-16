package nodepackage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRuntimePackagesProjectOnlyEnabledVerifiedHostCompatibleGenerations(t *testing.T) {
	ctx := context.Background()
	root := filepath.Join(t.TempDir(), "packages")
	policy, privateKey := lifecyclePolicy(t)
	store, err := CreateStore(ctx, root, policy)
	if err != nil {
		t.Fatal(err)
	}
	manifest, archivePath := lifecycleArchive(t, privateKey, "1.0.0", "process-v1")
	installed, err := store.InstallArchive(ctx, archivePath)
	if err != nil {
		t.Fatal(err)
	}

	packages, err := store.RuntimePackages(ctx, RuntimeHost{APIGeneration: "3.1", OperatingSystem: "windows", Architecture: "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	if len(packages) != 1 || packages[0].PackageID != installed.PackageID || packages[0].ManifestDigest != manifest.Digest() || len(packages[0].Nodes) != 1 {
		t.Fatalf("runtime packages = %#v", packages)
	}
	node := packages[0].Nodes[0]
	if node.Lock.PackageID != installed.PackageID || node.Lock.ArtifactDigest != manifest.Digest() || node.Lock.ABI != node.Implementation.ABI || node.Lock.Entrypoint != node.Implementation.Entrypoint {
		t.Fatalf("runtime node lock = %#v", node.Lock)
	}
	payload, err := node.Payload.Read(ctx, 1<<20)
	if err != nil || string(payload) != "process-v1" {
		t.Fatalf("runtime payload = %q, %v", payload, err)
	}

	if unsupported, err := store.RuntimePackages(ctx, RuntimeHost{APIGeneration: "4.0", OperatingSystem: "windows", Architecture: "amd64"}); err != nil || len(unsupported) != 0 {
		t.Fatalf("unsupported runtime projection = %#v, %v", unsupported, err)
	}
	if _, err := store.Disable(installed.PackageID); err != nil {
		t.Fatal(err)
	}
	if _, err := node.Payload.Read(ctx, 1<<20); err == nil {
		t.Fatal("previous runtime projection remained executable after disable")
	}
	if disabled, err := store.RuntimePackages(ctx, RuntimeHost{APIGeneration: "3.1", OperatingSystem: "windows", Architecture: "amd64"}); err != nil || len(disabled) != 0 {
		t.Fatalf("disabled runtime projection = %#v, %v", disabled, err)
	}
}

func TestRuntimePayloadRevalidatesIdentityAtReadBoundary(t *testing.T) {
	ctx := context.Background()
	root := filepath.Join(t.TempDir(), "packages")
	policy, privateKey := lifecyclePolicy(t)
	store, err := CreateStore(ctx, root, policy)
	if err != nil {
		t.Fatal(err)
	}
	manifest, archivePath := lifecycleArchive(t, privateKey, "1.0.0", "process-v1")
	if _, err := store.InstallArchive(ctx, archivePath); err != nil {
		t.Fatal(err)
	}
	packages, err := store.RuntimePackages(ctx, RuntimeHost{APIGeneration: "3.1", OperatingSystem: "windows", Architecture: "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	payload := packages[0].Nodes[0].Payload
	filename := filepath.Join(generationPath(root, manifest.Digest()), filepath.FromSlash(payload.Metadata().Path))
	if err := os.WriteFile(filename, []byte("PROCESS-V1"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := payload.Read(ctx, 1<<20); err == nil {
		t.Fatal("runtime payload read accepted tampered executable bytes")
	}
}
