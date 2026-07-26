package migrate

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yottaapp/yotta/internal/durablefs"
	"github.com/yottaapp/yotta/internal/storage"
	"github.com/yottaapp/yotta/internal/storage/catalog"
	_ "modernc.org/sqlite"
)

const (
	SnapshotFormat           = "yotta.storage-migration-snapshot"
	snapshotVersion          = 1
	snapshotManifestFilename = "manifest.json"
)

type SnapshotFile struct {
	Path    string `json:"path"`
	Present bool   `json:"present"`
	Bytes   int64  `json:"bytes"`
	SHA256  string `json:"sha256,omitempty"`
}

type SnapshotManifest struct {
	Format    string         `json:"format"`
	Version   int            `json:"version"`
	CreatedAt string         `json:"createdAt"`
	Files     []SnapshotFile `json:"files"`
}

func estimateSnapshotBytes(roots storage.Roots) (uint64, error) {
	candidates, err := snapshotCandidates(roots)
	if err != nil {
		return 0, err
	}
	var total uint64
	for _, path := range candidates {
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return 0, err
		}
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return 0, fmt.Errorf("snapshot authority %q is not a trusted file", filepath.Base(path))
		}
		if info.Size() > 0 {
			total += uint64(info.Size())
		}
	}
	for _, database := range []string{
		filepath.Join(roots.Catalog, catalog.ContentFilename),
		filepath.Join(roots.State, catalog.RunFilename),
	} {
		if info, err := os.Stat(database + "-wal"); err == nil && info.Size() > 0 {
			total += uint64(info.Size())
		}
	}
	return total, nil
}

func createSnapshot(
	ctx context.Context,
	roots storage.Roots,
	migrationDir string,
	now time.Time,
) (string, error) {
	snapshotDir := filepath.Join(migrationDir, "snapshot")
	manifestPath := filepath.Join(snapshotDir, snapshotManifestFilename)
	if manifest, err := readSnapshotManifest(manifestPath); err == nil {
		if err := validateSnapshotFiles(roots, snapshotDir, manifest); err != nil {
			return "", err
		}
		return filepath.Rel(migrationDir, manifestPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if err := os.RemoveAll(snapshotDir); err != nil {
		return "", fmt.Errorf("reset partial migration snapshot: %w", err)
	}
	if err := ensureTrustedSubdirectory(roots.Root, filepath.Join(snapshotDir, "files")); err != nil {
		return "", err
	}
	for _, path := range []string{
		filepath.Join(roots.Catalog, catalog.ContentFilename),
		filepath.Join(roots.State, catalog.RunFilename),
	} {
		if err := checkpointClosedDatabase(ctx, path); err != nil {
			return "", err
		}
	}
	candidates, err := snapshotCandidates(roots)
	if err != nil {
		return "", err
	}
	manifest := SnapshotManifest{
		Format: SnapshotFormat, Version: snapshotVersion,
		CreatedAt: now.UTC().Format(time.RFC3339Nano),
		Files:     make([]SnapshotFile, 0, len(candidates)),
	}
	for _, source := range candidates {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		relative, err := filepath.Rel(roots.Root, source)
		if err != nil || strings.HasPrefix(relative, "..") {
			return "", errors.New("snapshot source escaped storage root")
		}
		entry := SnapshotFile{Path: filepath.ToSlash(relative)}
		info, err := os.Lstat(source)
		if errors.Is(err, os.ErrNotExist) {
			manifest.Files = append(manifest.Files, entry)
			continue
		}
		if err != nil {
			return "", err
		}
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("snapshot source %q is not a trusted file", relative)
		}
		entry.Present = true
		destination := filepath.Join(snapshotDir, "files", relative)
		size, digest, err := copyFileDurable(source, destination)
		if err != nil {
			return "", err
		}
		entry.Bytes, entry.SHA256 = size, digest
		manifest.Files = append(manifest.Files, entry)
	}
	if err := writeJSON(manifestPath, manifest); err != nil {
		return "", err
	}
	return filepath.Rel(migrationDir, manifestPath)
}

func snapshotCandidates(roots storage.Roots) ([]string, error) {
	result := []string{
		roots.ManifestFile(),
		filepath.Join(roots.Catalog, catalog.ContentFilename),
		filepath.Join(roots.State, catalog.RunFilename),
		filepath.Join(roots.Data, "workspace", "runs", ".yotta-run-ledger-imported"),
	}
	info, err := os.Lstat(roots.Config)
	if errors.Is(err, os.ErrNotExist) {
		return result, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("storage config root is not a trusted directory")
	}
	err = filepath.WalkDir(roots.Config, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("config snapshot source %q is not a trusted file", entry.Name())
		}
		result = append(result, path)
		return nil
	})
	return result, err
}

func checkpointClosedDatabase(ctx context.Context, path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("migration database %q is not a trusted file", filepath.Base(path))
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		return fmt.Errorf("checkpoint %s: %w", filepath.Base(path), err)
	}
	var quick string
	if err := db.QueryRowContext(ctx, "PRAGMA quick_check").Scan(&quick); err != nil || quick != "ok" {
		return errors.Join(err, fmt.Errorf("%s quick-check failed", filepath.Base(path)))
	}
	return nil
}

func copyFileDurable(source, destination string) (int64, string, error) {
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return 0, "", err
	}
	input, err := os.Open(source)
	if err != nil {
		return 0, "", err
	}
	defer input.Close()
	staging, err := os.CreateTemp(filepath.Dir(destination), ".snapshot-*.tmp")
	if err != nil {
		return 0, "", err
	}
	stagingPath := staging.Name()
	committed := false
	defer func() {
		_ = staging.Close()
		if !committed {
			_ = os.Remove(stagingPath)
		}
	}()
	if err := staging.Chmod(0o600); err != nil {
		return 0, "", err
	}
	hash := sha256.New()
	size, err := io.Copy(io.MultiWriter(staging, hash), input)
	if err != nil {
		return 0, "", err
	}
	if err := staging.Sync(); err != nil {
		return 0, "", err
	}
	if err := staging.Close(); err != nil {
		return 0, "", err
	}
	if err := durablefs.Replace(stagingPath, destination); err != nil {
		if durablefs.Committed(err) {
			committed = true
		}
		return 0, "", err
	}
	committed = true
	return size, hex.EncodeToString(hash.Sum(nil)), nil
}

func readSnapshotManifest(path string) (SnapshotManifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return SnapshotManifest{}, err
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, (1<<20)+1))
	if err != nil {
		return SnapshotManifest{}, err
	}
	if len(raw) > 1<<20 {
		return SnapshotManifest{}, errors.New("migration snapshot manifest exceeds byte budget")
	}
	var manifest SnapshotManifest
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return SnapshotManifest{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return SnapshotManifest{}, errors.New("migration snapshot manifest must contain one JSON value")
	}
	if manifest.Format != SnapshotFormat || manifest.Version != snapshotVersion {
		return SnapshotManifest{}, errors.New("migration snapshot manifest is invalid")
	}
	return manifest, nil
}

func validateSnapshotFiles(roots storage.Roots, snapshotDir string, manifest SnapshotManifest) error {
	candidates, err := snapshotCandidates(roots)
	if err != nil {
		return err
	}
	expected := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		relative, err := filepath.Rel(roots.Root, candidate)
		if err != nil || strings.HasPrefix(relative, "..") {
			return errors.New("migration snapshot candidate escaped storage root")
		}
		expected[filepath.ToSlash(relative)] = struct{}{}
	}
	if len(manifest.Files) != len(expected) {
		return errors.New("migration snapshot manifest does not cover the fixed authority set")
	}
	seen := make(map[string]struct{}, len(manifest.Files))
	for _, entry := range manifest.Files {
		if _, exists := expected[entry.Path]; !exists {
			return errors.New("migration snapshot manifest contains an unexpected authority")
		}
		if _, duplicate := seen[entry.Path]; duplicate {
			return errors.New("migration snapshot manifest contains a duplicate authority")
		}
		seen[entry.Path] = struct{}{}
		if !entry.Present {
			if entry.Bytes != 0 || entry.SHA256 != "" {
				return errors.New("absent migration snapshot authority has file metadata")
			}
			continue
		}
		if entry.Bytes < 0 || len(entry.SHA256) != sha256.Size*2 {
			return errors.New("migration snapshot authority metadata is invalid")
		}
		if _, err := hex.DecodeString(entry.SHA256); err != nil {
			return errors.New("migration snapshot digest is invalid")
		}
		relative := filepath.FromSlash(entry.Path)
		if filepath.IsAbs(relative) || strings.HasPrefix(relative, "..") {
			return errors.New("migration snapshot path is invalid")
		}
		path := filepath.Join(snapshotDir, "files", relative)
		info, err := os.Stat(path)
		if err != nil || info.Size() != entry.Bytes {
			return errors.Join(err, errors.New("migration snapshot size mismatch"))
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		hash := sha256.New()
		_, copyErr := io.Copy(hash, file)
		closeErr := file.Close()
		if copyErr != nil || closeErr != nil ||
			hex.EncodeToString(hash.Sum(nil)) != entry.SHA256 {
			return errors.Join(copyErr, closeErr, errors.New("migration snapshot digest mismatch"))
		}
		target := filepath.Clean(filepath.Join(roots.Root, relative))
		if !strings.HasPrefix(target, filepath.Clean(roots.Root)+string(filepath.Separator)) &&
			target != filepath.Clean(roots.Root) {
			return errors.New("migration snapshot target escaped storage root")
		}
	}
	return nil
}

func inspectLegacyRuns(roots storage.Roots) (int, uint64, error) {
	root := filepath.Join(roots.Data, "workspace", "runs")
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}
	var count int
	var bytes uint64
	for _, entry := range entries {
		if entry.Name() == ".yotta-run-store" ||
			entry.Name() == ".yotta-run-ledger-imported" ||
			strings.HasPrefix(entry.Name(), ".durable-") {
			continue
		}
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 ||
			filepath.Ext(entry.Name()) != ".json" {
			return 0, 0, fmt.Errorf("legacy Run Store contains invalid entry %q", entry.Name())
		}
		info, err := entry.Info()
		if err != nil {
			return 0, 0, err
		}
		count++
		if info.Size() > 0 {
			bytes += uint64(info.Size())
		}
	}
	return count, bytes, nil
}
