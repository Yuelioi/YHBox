package desktopapp

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/services/asset"
	"github.com/yottaapp/yotta/internal/storage"
	"github.com/yottaapp/yotta/internal/storage/catalog"
)

func newTestAssetStore(t *testing.T, root string) *asset.Store {
	t.Helper()
	roots, err := storage.Resolve(filepath.Join(root, "profile"))
	if err != nil {
		t.Fatal(err)
	}
	foundation, err := catalog.Open(context.Background(), roots)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = foundation.Close() })
	blobs, err := blob.Open(
		filepath.Join(root, "blobs"),
		blob.Limits{MaxBlobBytes: 1 << 20, MaxTotalBytes: 8 << 20},
		foundation.Objects(),
	)
	if err != nil {
		t.Fatal(err)
	}
	store, err := asset.NewStore(
		foundation.Assets(),
		foundation.Objects(),
		blobs,
		asset.WithGCGracePeriod(0),
	)
	if err != nil {
		t.Fatal(err)
	}
	return store
}
