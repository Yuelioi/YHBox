package desktopapp

import (
	"path/filepath"
	"testing"

	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/services/asset"
)

func newTestAssetStore(t *testing.T, root string) *asset.Store {
	t.Helper()
	blobs, err := blob.Open(filepath.Join(root, "blobs"), blob.Limits{MaxBlobBytes: 1 << 20, MaxTotalBytes: 8 << 20})
	if err != nil {
		t.Fatal(err)
	}
	store, err := asset.NewStore(root, blobs)
	if err != nil {
		t.Fatal(err)
	}
	return store
}
