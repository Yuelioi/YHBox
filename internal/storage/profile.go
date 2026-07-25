package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/yottaapp/yotta/internal/durablefs"
)

const (
	RootFormat           = "yotta.storage-root"
	LayoutVersion        = "2"
	rootManifestFilename = "root.json"
	rootLeaseFilename    = "writer.lock"
)

var (
	ErrUnclaimedRoot     = errors.New("storage root is non-empty and not owned by Yotta")
	ErrUnsupportedLayout = errors.New("storage root layout is unsupported")
	ErrRootInUse         = errors.New("storage root is already open for writing")
)

type RootManifest struct {
	Format  string `json:"format"`
	Version string `json:"version"`
}

type OpenOptions struct {
	Root string
}

// Profile is the process-owned handle to one durable Yotta profile. Close
// releases its writer lease; stores only receive Profile.Roots.
type Profile struct {
	Roots Roots
	lease *lease
}

func Open(ctx context.Context, options OpenOptions) (*Profile, error) {
	if ctx == nil {
		return nil, errors.New("open storage profile requires a context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	roots, err := Resolve(options.Root)
	if err != nil {
		return nil, err
	}
	claimable, err := prepareRoot(roots.Root)
	if err != nil {
		return nil, err
	}
	// Claim the empty root before creating runtime/lease files. A crash between
	// claim and directory projection leaves a valid, reopenable partial Yotta
	// root instead of a non-empty unclaimed directory.
	if err := openOrClaimRoot(roots, claimable); err != nil {
		return nil, err
	}
	return openProfileDirectories(roots)
}

// OpenForMigration acquires the normal profile writer lease for one exact
// released older layout without claiming or advancing it. Only the root
// migration Module should use this entry point.
func OpenForMigration(ctx context.Context, options OpenOptions, from string) (*Profile, error) {
	if ctx == nil {
		return nil, errors.New("open storage migration profile requires a context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	roots, err := Resolve(options.Root)
	if err != nil {
		return nil, err
	}
	if _, err := prepareRoot(roots.Root); err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(roots.ManifestFile())
	if err != nil {
		return nil, fmt.Errorf("read storage root manifest for migration: %w", err)
	}
	manifest, err := decodeRootManifest(raw)
	if err != nil || manifest.Format != RootFormat || manifest.Version != from {
		return nil, fmt.Errorf("%w: migration requires %s/%s", ErrUnsupportedLayout, RootFormat, from)
	}
	return openProfileDirectories(roots)
}

func (p *Profile) Close() error {
	if p == nil || p.lease == nil {
		return nil
	}
	held := p.lease
	p.lease = nil
	return held.close()
}

// PublishCurrentLayout is the migration commit point. It advances only an
// exact older manifest while the caller still owns the profile writer lease.
func (p *Profile) PublishCurrentLayout(from string) error {
	if p == nil || p.lease == nil {
		return errors.New("publish storage layout requires an open migration profile")
	}
	raw, err := os.ReadFile(p.Roots.ManifestFile())
	if err != nil {
		return err
	}
	manifest, err := decodeRootManifest(raw)
	if err != nil || manifest.Format != RootFormat || manifest.Version != from {
		return fmt.Errorf("%w: storage layout changed during migration", ErrUnsupportedLayout)
	}
	encoded, err := json.MarshalIndent(RootManifest{Format: RootFormat, Version: LayoutVersion}, "", "  ")
	if err != nil {
		return err
	}
	if err := durablefs.WriteFile(p.Roots.ManifestFile(), encoded, 0o600); err != nil {
		return fmt.Errorf("publish storage layout: %w", err)
	}
	return nil
}

func openProfileDirectories(roots Roots) (*Profile, error) {
	if err := os.MkdirAll(roots.Runtime, 0o700); err != nil {
		return nil, fmt.Errorf("create storage runtime directory: %w", err)
	}
	held, err := acquireLease(filepath.Join(roots.Runtime, rootLeaseFilename))
	if err != nil {
		return nil, err
	}
	profile := &Profile{Roots: roots, lease: held}
	fail := func(err error) (*Profile, error) {
		return nil, errors.Join(err, profile.Close())
	}
	for _, directory := range roots.directories() {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return fail(fmt.Errorf("create storage directory %q: %w", directory, err))
		}
	}
	return profile, nil
}

func prepareRoot(root string) (bool, error) {
	info, err := os.Lstat(root)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(root, 0o700); err != nil {
			return false, fmt.Errorf("create storage root: %w", err)
		}
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect storage root: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return false, errors.New("storage root must be a real directory")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return false, fmt.Errorf("list storage root: %w", err)
	}
	if len(entries) == 0 {
		return true, nil
	}
	for _, entry := range entries {
		if entry.Name() == rootManifestFilename {
			return false, nil
		}
	}
	return false, ErrUnclaimedRoot
}

func openOrClaimRoot(roots Roots, claimable bool) error {
	path := filepath.Join(roots.Root, rootManifestFilename)
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		if !claimable {
			return ErrUnclaimedRoot
		}
		encoded, err := json.MarshalIndent(RootManifest{Format: RootFormat, Version: LayoutVersion}, "", "  ")
		if err != nil {
			return err
		}
		if err := durablefs.WriteFile(path, encoded, 0o600); err != nil {
			return fmt.Errorf("claim storage root: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("read storage root manifest: %w", err)
	}
	manifest, err := decodeRootManifest(raw)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnsupportedLayout, err)
	}
	if manifest.Format != RootFormat || manifest.Version != LayoutVersion {
		return fmt.Errorf("%w: found %s/%s, require %s/%s",
			ErrUnsupportedLayout, manifest.Format, manifest.Version, RootFormat, LayoutVersion)
	}
	return nil
}

func decodeRootManifest(raw []byte) (RootManifest, error) {
	if len(raw) == 0 || len(raw) > 4096 {
		return RootManifest{}, errors.New("root manifest exceeds byte budget")
	}
	var manifest RootManifest
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return RootManifest{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return RootManifest{}, errors.New("root manifest must contain exactly one JSON value")
	}
	if manifest.Format == "" || manifest.Version == "" {
		return RootManifest{}, errors.New("root manifest identity is missing")
	}
	return manifest, nil
}
