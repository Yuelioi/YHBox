package blob

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/durablefs"
)

const legacyMarkerContents = "yotta/blob-store/1\n"

// LayoutInspection is the read-only physical state used by the root-layout
// migration coordinator. An empty Version means the Blob Store is unclaimed.
type LayoutInspection struct {
	Version string
	Objects int
	Bytes   int64
}

// InspectLayout validates one legacy or current Blob Store without changing it.
func InspectLayout(ctx context.Context, root string) (LayoutInspection, error) {
	if ctx == nil {
		return LayoutInspection{}, errors.New("inspect Blob Store layout requires a context")
	}
	resolved, entries, marker, err := inspectLayoutRoot(root, false)
	if err != nil {
		return LayoutInspection{}, err
	}
	if len(entries) == 0 {
		return LayoutInspection{}, nil
	}
	switch marker {
	case legacyMarkerContents:
		objects, bytes, err := inspectLegacyEntries(ctx, resolved, entries)
		return LayoutInspection{Version: "1", Objects: objects, Bytes: bytes}, err
	case markerContents:
		objects, bytes, err := inspectCurrentEntries(resolved, entries)
		return LayoutInspection{Version: LayoutVersion, Objects: objects, Bytes: bytes}, err
	default:
		return LayoutInspection{}, errors.New("blob store directory has an unsupported ownership marker")
	}
}

// MigrateLayoutOneToTwo durably and idempotently converts flat v1 objects to
// the v2 sharded layout. The legacy marker remains the authority until every
// object has moved and the optional durable inventory has been reconciled.
func MigrateLayoutOneToTwo(
	ctx context.Context,
	root string,
	inventory Inventory,
) (LayoutInspection, error) {
	if ctx == nil {
		return LayoutInspection{}, errors.New("migrate Blob Store requires a context")
	}
	resolved, entries, marker, err := inspectLayoutRoot(root, true)
	if err != nil {
		return LayoutInspection{}, err
	}
	if len(entries) == 0 {
		if err := durablefs.WriteFile(
			filepath.Join(resolved, ownershipMarker), []byte(markerContents), 0o600,
		); err != nil {
			return LayoutInspection{}, fmt.Errorf("claim migrated Blob Store: %w", err)
		}
		return LayoutInspection{Version: LayoutVersion}, nil
	}
	if marker == markerContents {
		store := &Store{root: resolved}
		objects, err := store.scanObjects()
		if err != nil {
			return LayoutInspection{}, err
		}
		return reconcileMigratedObjects(ctx, store, objects, inventory)
	}
	if marker != legacyMarkerContents {
		return LayoutInspection{}, errors.New("blob store directory has an unsupported ownership marker")
	}
	if _, _, err := inspectLegacyEntries(ctx, resolved, entries); err != nil {
		return LayoutInspection{}, err
	}

	store := &Store{root: resolved}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return LayoutInspection{}, err
		}
		name := entry.Name()
		if name == ownershipMarker || name == stagingDirName || name == trashDirName ||
			entry.IsDir() {
			continue
		}
		path := filepath.Join(resolved, name)
		if strings.HasPrefix(name, ".tmp-") {
			if err := durablefs.Remove(path); err != nil {
				return LayoutInspection{}, fmt.Errorf("remove abandoned legacy Blob staging file: %w", err)
			}
			continue
		}
		digest := artifact.Digest("sha256:" + name)
		if err := store.ensureObjectDirectory(digest); err != nil {
			return LayoutInspection{}, err
		}
		destination := store.objectPath(digest)
		if destinationInfo, destinationErr := os.Lstat(destination); destinationErr == nil {
			sourceInfo, err := trustedObjectInfo(path, name)
			if err != nil {
				return LayoutInspection{}, err
			}
			if !destinationInfo.Mode().IsRegular() ||
				destinationInfo.Mode()&os.ModeSymlink != 0 ||
				destinationInfo.Size() != sourceInfo.Size() {
				return LayoutInspection{}, fmt.Errorf(
					"legacy Blob object %s conflicts with migrated destination", digest,
				)
			}
			if err := verifyObjectPath(ctx, destination, name, destinationInfo.Size()); err != nil {
				return LayoutInspection{}, err
			}
			if err := durablefs.Remove(path); err != nil {
				return LayoutInspection{}, fmt.Errorf("remove duplicate legacy Blob object: %w", err)
			}
		} else if !errors.Is(destinationErr, os.ErrNotExist) {
			return LayoutInspection{}, destinationErr
		} else if err := durablefs.Replace(path, destination); err != nil {
			return LayoutInspection{}, fmt.Errorf("migrate legacy Blob object: %w", err)
		}
	}
	for _, directory := range []string{stagingDirName, trashDirName} {
		if err := os.MkdirAll(filepath.Join(resolved, directory), 0o700); err != nil {
			return LayoutInspection{}, err
		}
	}
	objects, err := store.scanObjects()
	if err != nil {
		return LayoutInspection{}, err
	}
	report, err := reconcileMigratedObjects(ctx, store, objects, inventory)
	if err != nil {
		return LayoutInspection{}, err
	}
	if err := durablefs.WriteFile(
		filepath.Join(resolved, ownershipMarker), []byte(markerContents), 0o600,
	); err != nil {
		return LayoutInspection{}, fmt.Errorf("publish Blob Store layout 2: %w", err)
	}
	return report, nil
}

// RollbackLayoutTwoToOne reverses a not-yet-committed root migration. It is
// idempotent for partially migrated stores and publishes the legacy marker last.
func RollbackLayoutTwoToOne(ctx context.Context, root string) error {
	if ctx == nil {
		return errors.New("rollback Blob Store migration requires a context")
	}
	resolved, entries, marker, err := inspectLayoutRoot(root, false)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil
	}
	if marker != legacyMarkerContents && marker != markerContents {
		return errors.New("blob store directory has an unsupported ownership marker")
	}
	if _, _, err := inspectLegacyEntries(ctx, resolved, entries); err != nil {
		return err
	}
	sharded, _, err := inspectShardedObjects(resolved, entries)
	if err != nil {
		return err
	}
	for _, object := range sharded {
		if err := ctx.Err(); err != nil {
			return err
		}
		name := objectName(object.Digest)
		source := filepath.Join(resolved, name[:2], name[2:4], name)
		destination := filepath.Join(resolved, name)
		if destinationInfo, destinationErr := os.Lstat(destination); destinationErr == nil {
			if !destinationInfo.Mode().IsRegular() ||
				destinationInfo.Mode()&os.ModeSymlink != 0 ||
				destinationInfo.Size() != object.Size {
				return fmt.Errorf("legacy Blob rollback destination %s conflicts", object.Digest)
			}
			if err := verifyObjectPath(ctx, destination, name, object.Size); err != nil {
				return err
			}
			if err := durablefs.Remove(source); err != nil {
				return err
			}
		} else if !errors.Is(destinationErr, os.ErrNotExist) {
			return destinationErr
		} else if err := durablefs.Replace(source, destination); err != nil {
			return fmt.Errorf("rollback Blob object %s: %w", object.Digest, err)
		}
	}
	if err := removeEmptyMigrationDirectories(resolved); err != nil {
		return err
	}
	if err := durablefs.WriteFile(
		filepath.Join(resolved, ownershipMarker), []byte(legacyMarkerContents), 0o600,
	); err != nil {
		return fmt.Errorf("publish Blob Store layout 1 rollback: %w", err)
	}
	return nil
}

// RollbackLayoutTwoToUnclaimed restores an empty directory that had no Blob
// ownership marker before a root migration.
func RollbackLayoutTwoToUnclaimed(ctx context.Context, root string) error {
	inspection, err := InspectLayout(ctx, root)
	if err != nil {
		return err
	}
	if inspection.Version == "" {
		return nil
	}
	if inspection.Version != LayoutVersion || inspection.Objects != 0 {
		return errors.New("cannot restore a non-empty Blob Store to an unclaimed directory")
	}
	resolved, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	if err := removeEmptyMigrationDirectories(resolved); err != nil {
		return err
	}
	if err := durablefs.Remove(filepath.Join(resolved, ownershipMarker)); err != nil {
		return fmt.Errorf("remove migrated Blob Store marker: %w", err)
	}
	return nil
}

func inspectLayoutRoot(root string, create bool) (string, []os.DirEntry, string, error) {
	if strings.TrimSpace(root) == "" {
		return "", nil, "", errors.New("blob store root is required")
	}
	resolved, err := filepath.Abs(root)
	if err != nil {
		return "", nil, "", err
	}
	if create {
		if err := os.MkdirAll(resolved, 0o700); err != nil {
			return "", nil, "", err
		}
	}
	entries, err := os.ReadDir(resolved)
	if errors.Is(err, os.ErrNotExist) && !create {
		return resolved, nil, "", nil
	}
	if err != nil || len(entries) == 0 {
		return resolved, entries, "", err
	}
	markerPath := filepath.Join(resolved, ownershipMarker)
	if _, err := trustedRegularInfo(markerPath); err != nil {
		return "", nil, "", errors.New("blob store directory has an unsupported ownership marker")
	}
	raw, err := os.ReadFile(markerPath)
	if err != nil {
		return "", nil, "", errors.New("blob store directory has an unsupported ownership marker")
	}
	return resolved, entries, string(raw), nil
}

func reconcileMigratedObjects(
	ctx context.Context,
	store *Store,
	objects []Object,
	inventory Inventory,
) (LayoutInspection, error) {
	var total int64
	for _, object := range objects {
		if err := ctx.Err(); err != nil {
			return LayoutInspection{}, err
		}
		name := objectName(object.Digest)
		if err := verifyObjectPath(
			ctx, store.objectPath(object.Digest), name, object.Size,
		); err != nil {
			return LayoutInspection{}, err
		}
		if inventory != nil {
			if err := inventory.Observe(ctx, object); err != nil {
				return LayoutInspection{}, fmt.Errorf("index migrated Blob object %s: %w", object.Digest, err)
			}
		}
		total += object.Size
	}
	return LayoutInspection{Version: LayoutVersion, Objects: len(objects), Bytes: total}, nil
}

func inspectLegacyEntries(
	ctx context.Context,
	root string,
	entries []os.DirEntry,
) (int, int64, error) {
	sharded, bytes, err := inspectShardedObjects(root, entries)
	if err != nil {
		return 0, 0, err
	}
	seen := make(map[string]struct{}, len(sharded))
	for _, object := range sharded {
		name := objectName(object.Digest)
		path := filepath.Join(root, name[:2], name[2:4], name)
		if err := verifyObjectPath(ctx, path, name, object.Size); err != nil {
			return 0, 0, err
		}
		seen[name] = struct{}{}
	}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return 0, 0, err
		}
		name := entry.Name()
		if name == ownershipMarker {
			continue
		}
		if name == stagingDirName || name == trashDirName {
			if err := trustedDirectoryEntry(entry); err != nil {
				return 0, 0, err
			}
			continue
		}
		if entry.IsDir() {
			continue
		}
		if strings.HasPrefix(name, ".tmp-") {
			if _, err := trustedRegularInfo(filepath.Join(root, name)); err != nil {
				return 0, 0, err
			}
			continue
		}
		info, err := trustedObjectInfo(filepath.Join(root, name), name)
		if err != nil {
			return 0, 0, err
		}
		if err := verifyObjectPath(ctx, filepath.Join(root, name), name, info.Size()); err != nil {
			return 0, 0, err
		}
		if _, duplicate := seen[name]; !duplicate {
			seen[name] = struct{}{}
			bytes += info.Size()
		}
	}
	return len(seen), bytes, nil
}

func inspectCurrentEntries(root string, entries []os.DirEntry) (int, int64, error) {
	for _, entry := range entries {
		if entry.Name() == ownershipMarker {
			continue
		}
		if entry.Name() == stagingDirName || entry.Name() == trashDirName {
			if err := trustedDirectoryEntry(entry); err != nil {
				return 0, 0, err
			}
			continue
		}
		if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || !validShardName(entry.Name()) {
			return 0, 0, fmt.Errorf("blob store contains invalid top-level entry %q", entry.Name())
		}
	}
	objects, bytes, err := inspectShardedObjects(root, entries)
	return len(objects), bytes, err
}

func inspectShardedObjects(
	root string,
	entries []os.DirEntry,
) ([]Object, int64, error) {
	var result []Object
	var total int64
	for _, first := range entries {
		if first.Name() == stagingDirName || first.Name() == trashDirName {
			if err := trustedDirectoryEntry(first); err != nil {
				return nil, 0, err
			}
			continue
		}
		if !first.IsDir() {
			continue
		}
		if first.Type()&os.ModeSymlink != 0 || !validShardName(first.Name()) {
			return nil, 0, fmt.Errorf("blob store contains invalid shard %q", first.Name())
		}
		firstPath := filepath.Join(root, first.Name())
		secondLevel, err := os.ReadDir(firstPath)
		if err != nil {
			return nil, 0, err
		}
		for _, second := range secondLevel {
			if !second.IsDir() || second.Type()&os.ModeSymlink != 0 ||
				!validShardName(second.Name()) {
				return nil, 0, fmt.Errorf("blob store contains invalid shard %q", second.Name())
			}
			secondPath := filepath.Join(firstPath, second.Name())
			objects, err := os.ReadDir(secondPath)
			if err != nil {
				return nil, 0, err
			}
			for _, entry := range objects {
				name := entry.Name()
				info, err := trustedObjectInfo(filepath.Join(secondPath, name), name)
				if err != nil || !strings.HasPrefix(name, first.Name()+second.Name()) {
					return nil, 0, errors.Join(err, fmt.Errorf("blob store contains invalid object %q", name))
				}
				result = append(result, Object{
					Digest: artifact.Digest("sha256:" + name),
					Size:   info.Size(),
				})
				total += info.Size()
			}
		}
	}
	return result, total, nil
}

func trustedObjectInfo(path, name string) (os.FileInfo, error) {
	if !validObjectName(name) {
		return nil, fmt.Errorf("blob store contains invalid object %q", name)
	}
	return trustedRegularInfo(path)
}

func trustedRegularInfo(path string) (os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("blob store contains untrusted file %q", filepath.Base(path))
	}
	return info, nil
}

func trustedDirectoryEntry(entry os.DirEntry) error {
	if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
		return fmt.Errorf("blob store contains untrusted directory %q", entry.Name())
	}
	return nil
}

func verifyObjectPath(ctx context.Context, path, name string, size int64) error {
	return verifyPath(ctx, path, stagingIntent{
		Digest: artifact.Digest("sha256:" + name),
		Size:   size,
	})
}

func removeEmptyMigrationDirectories(root string) error {
	for _, name := range []string{stagingDirName, trashDirName} {
		path := filepath.Join(root, name)
		entries, err := os.ReadDir(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if len(entries) != 0 {
			return fmt.Errorf("blob store migration directory %q is not empty", name)
		}
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	firstLevel, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, first := range firstLevel {
		if !first.IsDir() || !validShardName(first.Name()) {
			continue
		}
		firstPath := filepath.Join(root, first.Name())
		secondLevel, err := os.ReadDir(firstPath)
		if err != nil {
			return err
		}
		for _, second := range secondLevel {
			secondPath := filepath.Join(firstPath, second.Name())
			entries, err := os.ReadDir(secondPath)
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				if err := os.Remove(secondPath); err != nil {
					return err
				}
			}
		}
		remaining, err := os.ReadDir(firstPath)
		if err != nil {
			return err
		}
		if len(remaining) == 0 {
			if err := os.Remove(firstPath); err != nil {
				return err
			}
		}
	}
	return nil
}
