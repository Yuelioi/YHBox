package storage

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const RedactedRoot = "<redacted>"

type InspectOptions struct {
	Root             string
	ShowPhysicalPath bool
}

type CategoryHealth struct {
	Name    string `json:"name"`
	Files   uint64 `json:"files"`
	Bytes   uint64 `json:"bytes"`
	Present bool   `json:"present"`
}

type HealthReport struct {
	Root           string           `json:"root"`
	Format         string           `json:"format"`
	LayoutVersion  string           `json:"layoutVersion"`
	Supported      bool             `json:"supported"`
	Categories     []CategoryHealth `json:"categories"`
	StagingFiles   uint64           `json:"stagingFiles"`
	RecoveryFiles  uint64           `json:"recoveryFiles"`
	UnknownEntries uint64           `json:"unknownEntries"`
}

// Inspect reads one profile without claiming it, creating directories, or
// acquiring the writer lease. Physical paths are redacted unless explicitly
// requested by a local operator.
func Inspect(ctx context.Context, options InspectOptions) (HealthReport, error) {
	if ctx == nil {
		return HealthReport{}, errors.New("inspect storage profile requires a context")
	}
	roots, err := Resolve(options.Root)
	if err != nil {
		return HealthReport{}, err
	}
	raw, err := os.ReadFile(filepath.Join(roots.Root, rootManifestFilename))
	if err != nil {
		return HealthReport{}, fmt.Errorf("read storage root manifest: %w", err)
	}
	manifest, err := decodeRootManifest(raw)
	if err != nil {
		return HealthReport{}, fmt.Errorf("decode storage root manifest: %w", err)
	}
	if manifest.Format != RootFormat {
		return HealthReport{}, fmt.Errorf("%w: found application identity %q", ErrUnsupportedLayout, manifest.Format)
	}
	report := HealthReport{
		Root:          RedactedRoot,
		Format:        manifest.Format,
		LayoutVersion: manifest.Version,
		Supported:     manifest.Version == LayoutVersion,
	}
	if options.ShowPhysicalPath {
		report.Root = roots.Root
	}
	categories := []struct {
		name string
		path string
	}{
		{"config", roots.Config},
		{"data", roots.Data},
		{"catalog", roots.Catalog},
		{"objects", roots.Objects},
		{"documents", roots.Documents},
		{"packages", roots.Packages},
		{"cache", roots.Cache},
		{"state", roots.State},
		{"diagnostics", roots.Diagnostics},
		{"backups", roots.Backups},
		{"runtime", roots.Runtime},
		{"temp", roots.Temp},
	}
	report.Categories = make([]CategoryHealth, 0, len(categories))
	for _, category := range categories {
		health, staging, recovery, err := inspectCategory(ctx, roots.Root, category.name, category.path)
		if err != nil {
			return HealthReport{}, err
		}
		report.Categories = append(report.Categories, health)
		report.StagingFiles += staging
		report.RecoveryFiles += recovery
	}
	known := map[string]struct{}{
		rootManifestFilename: {}, "config": {}, "data": {}, "catalog": {}, "objects": {}, "documents": {},
		"packages": {}, "cache": {}, "state": {}, "diagnostics": {}, "backups": {}, "runtime": {}, "tmp": {},
	}
	entries, err := os.ReadDir(roots.Root)
	if err != nil {
		return HealthReport{}, fmt.Errorf("list storage root: %w", err)
	}
	for _, entry := range entries {
		if _, exists := known[entry.Name()]; !exists {
			report.UnknownEntries++
		}
	}
	return report, nil
}

func inspectCategory(ctx context.Context, root, name, path string) (CategoryHealth, uint64, uint64, error) {
	health := CategoryHealth{Name: name}
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return health, 0, 0, nil
	}
	if err != nil {
		return health, 0, 0, fmt.Errorf("inspect storage category %s: %w", name, err)
	}
	if !info.IsDir() {
		return health, 0, 0, fmt.Errorf("storage category %s is not a directory", name)
	}
	health.Present = true
	var staging, recovery uint64
	err = filepath.WalkDir(path, func(itemPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		health.Files++
		if info.Size() > 0 {
			health.Bytes += uint64(info.Size())
		}
		base := entry.Name()
		if strings.Contains(base, ".staging-") {
			staging++
		}
		relative, err := filepath.Rel(root, itemPath)
		if err != nil {
			return err
		}
		for _, part := range strings.Split(filepath.ToSlash(relative), "/") {
			if part == "recovery" || part == "quarantine" {
				recovery++
				break
			}
		}
		return nil
	})
	if err != nil {
		return health, 0, 0, fmt.Errorf("inspect storage category %s: %w", name, err)
	}
	return health, staging, recovery, nil
}
