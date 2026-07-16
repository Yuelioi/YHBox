package nodepackage

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

func TestStoreInstallUpdateReopenQuarantineRollbackAndUninstall(t *testing.T) {
	ctx := context.Background()
	root := filepath.Join(t.TempDir(), "packages")
	policy, privateKey := lifecyclePolicy(t)
	store, err := CreateStore(ctx, root, policy)
	if err != nil {
		t.Fatal(err)
	}
	firstManifest, firstArchive := lifecycleArchive(t, privateKey, "1.0.0", "process-v1")
	first, err := store.InstallArchive(ctx, firstArchive)
	if err != nil {
		t.Fatal(err)
	}
	if first.Current != firstManifest.Digest() || !first.Enabled || first.Rollback.Valid() || len(first.Releases) != 1 {
		t.Fatalf("first installation = %#v", first)
	}
	first.Releases[0].QuarantineReason = "mutated"
	got, _ := store.Get(first.PackageID)
	if got.Releases[0].QuarantineReason != "" {
		t.Fatal("Get returned mutable store state")
	}

	secondManifest, secondArchive := lifecycleArchive(t, privateKey, "2.0.0", "process-v2")
	second, err := store.InstallArchive(ctx, secondArchive)
	if err != nil {
		t.Fatal(err)
	}
	if second.Current != secondManifest.Digest() || second.Rollback != firstManifest.Digest() || !second.Enabled || len(second.Releases) != 2 {
		t.Fatalf("updated installation = %#v", second)
	}
	reopened, err := OpenStore(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if list := reopened.List(); len(list) != 1 || list[0].Current != secondManifest.Digest() {
		t.Fatalf("reopened installations = %#v", list)
	}
	if _, err := reopened.Disable(second.PackageID); err != nil {
		t.Fatal(err)
	}
	if enabled, err := reopened.Enable(second.PackageID); err != nil || !enabled.Enabled {
		t.Fatalf("enable = %#v, %v", enabled, err)
	}
	quarantined, err := reopened.Quarantine(second.PackageID, secondManifest.Digest(), "security.revoked")
	if err != nil || quarantined.Enabled {
		t.Fatalf("quarantine = %#v, %v", quarantined, err)
	}
	if _, err := reopened.Enable(second.PackageID); err == nil {
		t.Fatal("quarantined current generation was enabled")
	}
	rolledBack, err := reopened.Rollback(ctx, second.PackageID)
	if err != nil {
		t.Fatal(err)
	}
	if rolledBack.Current != firstManifest.Digest() || rolledBack.Rollback != secondManifest.Digest() || rolledBack.Enabled {
		t.Fatalf("rollback = %#v", rolledBack)
	}
	if enabled, err := reopened.Enable(second.PackageID); err != nil || !enabled.Enabled {
		t.Fatalf("enable rollback = %#v, %v", enabled, err)
	}
	if err := reopened.Uninstall(second.PackageID); err != nil {
		t.Fatal(err)
	}
	finalStore, err := OpenStore(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if len(finalStore.List()) != 0 {
		t.Fatalf("store after uninstall = %#v", finalStore.List())
	}
}

func TestStoreRejectsUnknownPublisherAndTamperedGeneration(t *testing.T) {
	ctx := context.Background()
	root := filepath.Join(t.TempDir(), "packages")
	policy, privateKey := lifecyclePolicy(t)
	store, err := CreateStore(ctx, root, policy)
	if err != nil {
		t.Fatal(err)
	}
	manifest, archivePath := lifecycleArchive(t, privateKey, "1.0.0", "process-v1")
	_, unknownPrivateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	_, unknownArchive := lifecycleArchive(t, unknownPrivateKey, "1.0.0", "process-v1")
	if _, err := store.InstallArchive(ctx, unknownArchive); err == nil {
		t.Fatal("unknown publisher key installed a package")
	}
	if len(store.List()) != 0 {
		t.Fatal("failed signature verification published registry state")
	}
	if _, err := store.InstallArchive(ctx, archivePath); err != nil {
		t.Fatal(err)
	}
	payload := filepath.Join(generationPath(root, manifest.Digest()), "bin", "plugin.exe")
	if err := os.WriteFile(payload, []byte("PROCESS-V1"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenStore(ctx, root); err == nil {
		t.Fatal("store reopened a tampered generation")
	}
}

func TestStoreCleansInterruptedIncomingAndOrphanGeneration(t *testing.T) {
	root := filepath.Join(t.TempDir(), "packages")
	if err := os.MkdirAll(filepath.Join(root, generationsDir), 0o700); err != nil {
		t.Fatal(err)
	}
	incoming := filepath.Join(root, ".incoming-crashed")
	if err := os.Mkdir(incoming, 0o700); err != nil {
		t.Fatal(err)
	}
	orphanDigest, err := artifact.Sum("yotta/test/orphan-generation/v1", []byte("orphan"))
	if err != nil {
		t.Fatal(err)
	}
	orphan := generationPath(root, orphanDigest)
	if err := os.Mkdir(orphan, 0o700); err != nil {
		t.Fatal(err)
	}
	policy, _ := lifecyclePolicy(t)
	if _, err := CreateStore(context.Background(), root, policy); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{incoming, orphan} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("crash residue %s remains: %v", path, err)
		}
	}
}

func TestStoreDoesNotPublishMemoryWhenRegistryCommitFails(t *testing.T) {
	ctx := context.Background()
	root := filepath.Join(t.TempDir(), "packages")
	policy, privateKey := lifecyclePolicy(t)
	store, err := CreateStore(ctx, root, policy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, registryFilename)); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, registryFilename), 0o700); err != nil {
		t.Fatal(err)
	}
	_, archivePath := lifecycleArchive(t, privateKey, "1.0.0", "process-v1")
	if _, err := store.InstallArchive(ctx, archivePath); err == nil {
		t.Fatal("registry commit failure installed a package")
	}
	if len(store.List()) != 0 {
		t.Fatal("registry commit failure published in-memory authority")
	}
	if err := os.RemoveAll(filepath.Join(root, registryFilename)); err != nil {
		t.Fatal(err)
	}
	reopened, err := CreateStore(ctx, root, policy)
	if err != nil {
		t.Fatal(err)
	}
	if len(reopened.List()) != 0 {
		t.Fatalf("reopen after failed commit = %#v", reopened.List())
	}
}

func TestStoreSerializesConcurrentPackageUpdates(t *testing.T) {
	ctx := context.Background()
	policy, privateKey := lifecyclePolicy(t)
	store, err := CreateStore(ctx, filepath.Join(t.TempDir(), "packages"), policy)
	if err != nil {
		t.Fatal(err)
	}
	firstManifest, firstArchive := lifecycleArchive(t, privateKey, "1.0.0", "process-v1")
	secondManifest, secondArchive := lifecycleArchive(t, privateKey, "2.0.0", "process-v2")
	type request struct {
		manifest Manifest
		archive  string
	}
	requests := []request{{firstManifest, firstArchive}, {secondManifest, secondArchive}}
	start := make(chan struct{})
	errorsSeen := make(chan error, len(requests))
	var wait sync.WaitGroup
	for _, candidate := range requests {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			_, err := store.InstallArchive(ctx, candidate.archive)
			errorsSeen <- err
		}()
	}
	close(start)
	wait.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		if err != nil {
			t.Fatal(err)
		}
	}
	installed, found := store.Get(firstManifest.PackageID())
	if !found || len(installed.Releases) != 2 || !installed.Current.Valid() || !installed.Rollback.Valid() || installed.Current == installed.Rollback {
		t.Fatalf("concurrent updates = %#v, found=%v", installed, found)
	}
}

func lifecycleArchive(t *testing.T, privateKey ed25519.PrivateKey, version, payload string) (Manifest, string) {
	t.Helper()
	draft := testDraft(t, nodecontract.ABIProcess)
	draft.PackageVersion = version
	draft.Nodes[0].Implementation.Payload = testPayload(t, "bin/plugin.exe", "application/vnd.microsoft.portable-executable", payload)
	manifest, err := Seal(draft)
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := SignManifest(manifest, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	archivePath := writeArchive(t, []archiveTestEntry{
		{name: ArchiveManifestPath, data: manifest.Bytes()},
		{name: ArchiveSignaturePath, data: envelope.Bytes()},
		{name: "bin/plugin.exe", data: []byte(payload)},
	})
	return manifest, archivePath
}

func lifecyclePolicy(t *testing.T) (TrustPolicy, ed25519.PrivateKey) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := SealTrustPolicy(TrustPolicyDraft{
		Revision:   1,
		Publishers: []PublisherAuthorityDraft{{Namespace: testNamespace, Keys: []ed25519.PublicKey{publicKey}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return policy, privateKey
}
