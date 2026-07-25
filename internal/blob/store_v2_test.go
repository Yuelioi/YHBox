package blob_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
)

type memoryInventory struct {
	mu      sync.Mutex
	objects map[artifact.Digest]blob.Object
	total   int64
}

func newMemoryInventory() *memoryInventory {
	return &memoryInventory{objects: make(map[artifact.Digest]blob.Object)}
}

func (inventory *memoryInventory) Observe(_ context.Context, object blob.Object) error {
	inventory.mu.Lock()
	defer inventory.mu.Unlock()
	if previous, found := inventory.objects[object.Digest]; found {
		if previous.Size != object.Size {
			return os.ErrInvalid
		}
		return nil
	}
	inventory.objects[object.Digest] = object
	inventory.total += object.Size
	return nil
}

func (inventory *memoryInventory) Forget(_ context.Context, digest artifact.Digest) error {
	inventory.mu.Lock()
	defer inventory.mu.Unlock()
	object, found := inventory.objects[digest]
	if !found {
		return os.ErrNotExist
	}
	delete(inventory.objects, digest)
	inventory.total -= object.Size
	return nil
}

func (inventory *memoryInventory) Objects(context.Context) ([]blob.Object, error) {
	inventory.mu.Lock()
	defer inventory.mu.Unlock()
	result := make([]blob.Object, 0, len(inventory.objects))
	for _, object := range inventory.objects {
		result = append(result, object)
	}
	return result, nil
}

func (inventory *memoryInventory) Object(
	_ context.Context,
	digest artifact.Digest,
) (blob.Object, bool, error) {
	inventory.mu.Lock()
	defer inventory.mu.Unlock()
	object, found := inventory.objects[digest]
	return object, found, nil
}

func (inventory *memoryInventory) TotalBytes(context.Context) (int64, error) {
	inventory.mu.Lock()
	defer inventory.mu.Unlock()
	return inventory.total, nil
}

func TestStoreV2UsesShardsAndCatalogInventoryAtStartup(t *testing.T) {
	root := t.TempDir()
	inventory := newMemoryInventory()
	store, err := blob.Open(
		root,
		blob.Limits{MaxBlobBytes: 1024, MaxTotalBytes: 4096},
		inventory,
	)
	if err != nil {
		t.Fatal(err)
	}
	ref, err := store.Put(context.Background(), "text/plain", strings.NewReader("sharded"))
	if err != nil {
		t.Fatal(err)
	}
	name := strings.TrimPrefix(ref.Digest.String(), "sha256:")
	path := filepath.Join(root, name[:2], name[2:4], name)
	if info, err := os.Stat(path); err != nil || info.Size() != ref.Size {
		t.Fatalf("sharded object = %v, %v", info, err)
	}
	// An indexed startup must not descend into every shard to rebuild quota.
	if err := os.WriteFile(filepath.Join(root, name[:2], name[2:4], "not-an-object"), []byte("ignored"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := blob.Open(
		root,
		blob.Limits{MaxBlobBytes: 1024, MaxTotalBytes: 4096},
		inventory,
	); err != nil {
		t.Fatalf("indexed Open() scanned the complete object tree: %v", err)
	}
}

func TestStoreV2RejectsUnsafeSecondLevelShardWithoutScanningObjects(t *testing.T) {
	root := t.TempDir()
	if _, err := blob.Open(root, blob.Limits{MaxBlobBytes: 1024, MaxTotalBytes: 4096}); err != nil {
		t.Fatal(err)
	}
	first := filepath.Join(root, "ab")
	if err := os.Mkdir(first, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(first, "cd"), []byte("not a shard directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := blob.Open(
		root,
		blob.Limits{MaxBlobBytes: 1024, MaxTotalBytes: 4096},
		newMemoryInventory(),
	); err == nil {
		t.Fatal("Open() accepted a non-directory second-level shard")
	}
}

func TestStoreV2RecoversPublishedStagingIntent(t *testing.T) {
	root := t.TempDir()
	inventory := newMemoryInventory()
	store, err := blob.Open(root, blob.Limits{MaxBlobBytes: 1024, MaxTotalBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}
	ref, err := store.Put(context.Background(), "text/plain", strings.NewReader("recover-stage"))
	if err != nil {
		t.Fatal(err)
	}
	name := strings.TrimPrefix(ref.Digest.String(), "sha256:")
	objectPath := filepath.Join(root, name[:2], name[2:4], name)
	stagingData := filepath.Join(root, ".staging", "put-crash.data")
	if err := os.Rename(objectPath, stagingData); err != nil {
		t.Fatal(err)
	}
	intent, err := json.Marshal(map[string]any{"digest": ref.Digest, "size": ref.Size})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".staging", "put-crash.json"), intent, 0o600); err != nil {
		t.Fatal(err)
	}
	reopened, err := blob.Open(
		root,
		blob.Limits{MaxBlobBytes: 1024, MaxTotalBytes: 4096},
		inventory,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := reopened.Verify(context.Background(), ref); err != nil {
		t.Fatalf("recovered object = %v", err)
	}
	if _, found, _ := inventory.Object(context.Background(), ref.Digest); !found {
		t.Fatal("recovered object was not indexed")
	}
}

func TestStoreV2RestoresIndexedTrashAndFinalizesForgottenTrash(t *testing.T) {
	root := t.TempDir()
	inventory := newMemoryInventory()
	store, err := blob.Open(
		root,
		blob.Limits{MaxBlobBytes: 1024, MaxTotalBytes: 4096},
		inventory,
	)
	if err != nil {
		t.Fatal(err)
	}
	ref, err := store.Put(context.Background(), "text/plain", strings.NewReader("recover-trash"))
	if err != nil {
		t.Fatal(err)
	}
	name := strings.TrimPrefix(ref.Digest.String(), "sha256:")
	objectPath := filepath.Join(root, name[:2], name[2:4], name)
	trashPath := filepath.Join(root, ".trash", name)
	if err := os.Rename(objectPath, trashPath); err != nil {
		t.Fatal(err)
	}
	reopened, err := blob.Open(
		root,
		blob.Limits{MaxBlobBytes: 1024, MaxTotalBytes: 4096},
		inventory,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := reopened.Verify(context.Background(), ref); err != nil {
		t.Fatalf("indexed trash was not restored: %v", err)
	}
	if err := os.Rename(objectPath, trashPath); err != nil {
		t.Fatal(err)
	}
	if err := inventory.Forget(context.Background(), ref.Digest); err != nil {
		t.Fatal(err)
	}
	if _, err := blob.Open(
		root,
		blob.Limits{MaxBlobBytes: 1024, MaxTotalBytes: 4096},
		inventory,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(trashPath); !os.IsNotExist(err) {
		t.Fatalf("forgotten trash survived recovery: %v", err)
	}
}
