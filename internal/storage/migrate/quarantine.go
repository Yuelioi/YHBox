package migrate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yottaapp/yotta/internal/durablefs"
	"github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/storage"
)

type QuarantineRecord struct {
	Name   string `json:"name"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

func ListQuarantine(options Options) ([]QuarantineRecord, error) {
	roots, err := storage.Resolve(options.Root)
	if err != nil {
		return nil, err
	}
	root := filepath.Join(roots.Migrations, layoutOneToTwoID, "quarantine")
	info, err := os.Lstat(root)
	if errors.Is(err, os.ErrNotExist) {
		return []QuarantineRecord{}, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("migration quarantine is not a trusted directory")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	result := make([]QuarantineRecord, 0, len(entries))
	for _, entry := range entries {
		record, err := inspectQuarantineFile(root, entry)
		if err != nil {
			return nil, err
		}
		result = append(result, record)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func QuarantineLegacyRun(ctx context.Context, options Options, name string) (QuarantineRecord, error) {
	if err := validateLegacyRunName(name); err != nil {
		return QuarantineRecord{}, err
	}
	roots, journal, profile, err := openQuarantineProfile(ctx, options)
	if err != nil {
		return QuarantineRecord{}, err
	}
	defer profile.Close()
	if journal.BlockedEntry != "" && journal.BlockedEntry != name {
		return QuarantineRecord{}, errors.New("requested record is not the migration blocker")
	}
	source := filepath.Join(roots.Data, "workspace", "runs", name)
	if err := requireTrustedRegularFile(source, run.MaxRecordBytes); err != nil {
		return QuarantineRecord{}, err
	}
	destinationRoot := filepath.Join(roots.Migrations, layoutOneToTwoID, "quarantine")
	if err := ensureTrustedSubdirectory(roots.Root, destinationRoot); err != nil {
		return QuarantineRecord{}, err
	}
	destination := filepath.Join(destinationRoot, name)
	if _, err := os.Lstat(destination); !errors.Is(err, os.ErrNotExist) {
		if err == nil {
			return QuarantineRecord{}, errors.New("migration quarantine record already exists")
		}
		return QuarantineRecord{}, err
	}
	bytes, digest, err := copyFileDurable(source, destination)
	if err != nil {
		return QuarantineRecord{}, err
	}
	if err := durablefs.Remove(source); err != nil {
		return QuarantineRecord{}, err
	}
	return QuarantineRecord{Name: name, Bytes: bytes, SHA256: digest}, nil
}

func RestoreLegacyRun(ctx context.Context, options Options, name string) (QuarantineRecord, error) {
	if err := validateLegacyRunName(name); err != nil {
		return QuarantineRecord{}, err
	}
	roots, _, profile, err := openQuarantineProfile(ctx, options)
	if err != nil {
		return QuarantineRecord{}, err
	}
	defer profile.Close()
	sourceRoot := filepath.Join(roots.Migrations, layoutOneToTwoID, "quarantine")
	source := filepath.Join(sourceRoot, name)
	if err := requireTrustedRegularFile(source, run.MaxRecordBytes); err != nil {
		return QuarantineRecord{}, err
	}
	destination := filepath.Join(roots.Data, "workspace", "runs", name)
	if _, err := os.Lstat(destination); !errors.Is(err, os.ErrNotExist) {
		if err == nil {
			return QuarantineRecord{}, errors.New("legacy Run record already exists")
		}
		return QuarantineRecord{}, err
	}
	bytes, digest, err := copyFileDurable(source, destination)
	if err != nil {
		return QuarantineRecord{}, err
	}
	if err := durablefs.Remove(source); err != nil {
		return QuarantineRecord{}, err
	}
	return QuarantineRecord{Name: name, Bytes: bytes, SHA256: digest}, nil
}

func openQuarantineProfile(
	ctx context.Context,
	options Options,
) (storage.Roots, Journal, *storage.Profile, error) {
	roots, err := storage.Resolve(options.Root)
	if err != nil {
		return storage.Roots{}, Journal{}, nil, err
	}
	journal, found, err := readJournal(
		filepath.Join(roots.Migrations, layoutOneToTwoID, journalFilename),
	)
	if err != nil {
		return storage.Roots{}, Journal{}, nil, err
	}
	if !found || journal.State != StateRecoveryRequired {
		return storage.Roots{}, Journal{}, nil, errors.New("migration quarantine requires recovery state")
	}
	profile, err := storage.OpenForMigration(
		ctx, storage.OpenOptions{Root: roots.Root}, journal.From,
	)
	if err != nil {
		return storage.Roots{}, Journal{}, nil, err
	}
	return roots, journal, profile, nil
}

func inspectQuarantineFile(root string, entry os.DirEntry) (QuarantineRecord, error) {
	if err := validateLegacyRunName(entry.Name()); err != nil {
		return QuarantineRecord{}, err
	}
	info, err := entry.Info()
	if err != nil {
		return QuarantineRecord{}, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return QuarantineRecord{}, errors.New("migration quarantine contains an untrusted entry")
	}
	file, err := os.Open(filepath.Join(root, entry.Name()))
	if err != nil {
		return QuarantineRecord{}, err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return QuarantineRecord{}, err
	}
	return QuarantineRecord{
		Name: entry.Name(), Bytes: info.Size(), SHA256: hex.EncodeToString(hash.Sum(nil)),
	}, nil
}

func validateLegacyRunName(name string) error {
	if strings.TrimSpace(name) == "" || filepath.Base(name) != name ||
		filepath.Ext(name) != ".json" {
		return errors.New("legacy Run quarantine requires one JSON basename")
	}
	return nil
}

func requireTrustedRegularFile(path string, maxBytes int64) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 ||
		info.Size() < 0 || info.Size() > maxBytes {
		return errors.New("migration record is not a trusted bounded file")
	}
	return nil
}
