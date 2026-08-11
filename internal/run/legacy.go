package run

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/durablefs"
	"github.com/yottaapp/yotta/internal/runid"
	"github.com/yottaapp/yotta/internal/storage/catalog"
)

const (
	legacyRunStoreMarker         = ".yotta-run-store"
	legacyRunStoreMarkerContents = "yotta/run-store/1\n"
	legacyRunImportMarker        = ".yotta-run-ledger-imported"
	legacyRunImportContents      = "yotta/run-ledger-import/1\n"
)

type LegacyImportError struct {
	Name string
	Err  error
}

func (e *LegacyImportError) Error() string {
	return fmt.Sprintf("legacy Run record %q: %v", e.Name, e.Err)
}

func (e *LegacyImportError) Unwrap() error { return e.Err }

// ImportLegacyStore is the explicit root-migration action for the released
// v1 one-JSON-per-Run layout. Store construction never calls it implicitly.
// Exact records are idempotent and the legacy bytes remain in place.
func ImportLegacyStore(
	ctx context.Context,
	repository *catalog.RunRepository,
	valueCatalog datatype.ValueTypeCatalog,
	root string,
	maxRecords int,
) error {
	if strings.TrimSpace(root) == "" {
		return nil
	}
	if repository == nil || valueCatalog == nil || maxRecords <= 0 {
		return errors.New("legacy Run import requires a repository, type catalog, and positive record limit")
	}
	resolved, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	info, err := os.Lstat(resolved)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("legacy Run Store root is not a trusted directory")
	}
	if imported, err := exactLegacyMarker(filepath.Join(resolved, legacyRunImportMarker), legacyRunImportContents); err != nil {
		return err
	} else if imported {
		return nil
	}
	if claimed, err := exactLegacyMarker(filepath.Join(resolved, legacyRunStoreMarker), legacyRunStoreMarkerContents); err != nil {
		return err
	} else if !claimed {
		return errors.New("legacy Run Store has an unsupported ownership marker")
	}
	entries, err := os.ReadDir(resolved)
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		name := entry.Name()
		if name == legacyRunStoreMarker || name == legacyRunImportMarker ||
			strings.HasPrefix(name, ".durable-") && strings.HasSuffix(name, ".tmp") {
			continue
		}
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || filepath.Ext(name) != ".json" {
			return &LegacyImportError{Name: name, Err: errors.New("entry is not a trusted JSON record")}
		}
		runID := strings.TrimSuffix(name, ".json")
		if err := runid.Validate(runID); err != nil {
			return fmt.Errorf("legacy Run Store contains invalid record name %q", name)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > MaxRecordBytes {
			return &LegacyImportError{Name: name, Err: errors.New("record exceeds byte budget")}
		}
		raw, err := os.ReadFile(filepath.Join(resolved, name))
		if err != nil {
			return err
		}
		record, err := OpenRecord(raw, valueCatalog)
		if err != nil || record.Admission().RunID != runID {
			return &LegacyImportError{Name: name, Err: errors.Join(err, ErrRunIdentity)}
		}
		stored, err := ledgerRecord(record, valueCatalog)
		if err != nil {
			return err
		}
		if err := repository.Import(ctx, stored); err != nil {
			return &LegacyImportError{Name: name, Err: err}
		}
		count, err := repository.Count(ctx)
		if err != nil {
			return err
		}
		if count > maxRecords {
			return errors.New("legacy Run Store exceeds record limit")
		}
	}
	if err := durablefs.WriteFile(
		filepath.Join(resolved, legacyRunImportMarker),
		[]byte(legacyRunImportContents), 0o600,
	); err != nil {
		return fmt.Errorf("mark legacy Run import: %w", err)
	}
	return nil
}

func exactLegacyMarker(path, expected string) (bool, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return false, errors.New("run Store marker is not a trusted file")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	if string(raw) != expected {
		return false, errors.New("run Store marker has an unsupported version")
	}
	return true, nil
}
