package compiler_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/yottaapp/yotta/internal/datatype"
	run "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/storage"
	storagecatalog "github.com/yottaapp/yotta/internal/storage/catalog"
)

func newCompilerIntegrationRunStore(
	t *testing.T,
	valueCatalog datatype.ValueTypeCatalog,
	options run.StoreOptions,
) (*run.Store, error) {
	t.Helper()
	roots, err := storage.Resolve(filepath.Join(t.TempDir(), "profile"))
	if err != nil {
		return nil, err
	}
	foundation, err := storagecatalog.Open(context.Background(), roots)
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() {
		if err := foundation.Close(); err != nil {
			t.Errorf("close test Run Ledger: %v", err)
		}
	})
	return run.OpenStore(foundation.Runs(), valueCatalog, options)
}
