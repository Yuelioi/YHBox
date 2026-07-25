// Package storage owns the application profile root, its lifecycle projection,
// layout identity, and the single-writer lease shared by every durable store.
package storage

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const (
	EnvironmentRoot  = "YOTTA_ROOT"
	vendorDirectory  = "Yotta"
	productDirectory = "Yotta"
)

// Roots is the complete physical projection of one Yotta application profile.
// Callers receive paths; they do not derive lifecycle directories themselves.
type Roots struct {
	Root        string
	Config      string
	Data        string
	Catalog     string
	Objects     string
	Documents   string
	Exports     string
	Packages    string
	Cache       string
	State       string
	Diagnostics string
	Logs        string
	Crashes     string
	Captures    string
	Backups     string
	Migrations  string
	Runtime     string
	Temp        string
}

// Resolve projects either an explicit root, YOTTA_ROOT, or the platform's
// per-user local application data location. Explicit input always wins.
func Resolve(explicit string) (Roots, error) {
	root := strings.TrimSpace(explicit)
	if root == "" {
		root = strings.TrimSpace(os.Getenv(EnvironmentRoot))
	}
	if root == "" {
		local, err := localDataRoot()
		if err != nil {
			return Roots{}, err
		}
		root = filepath.Join(local, vendorDirectory, productDirectory)
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return Roots{}, err
	}
	absolute = filepath.Clean(absolute)
	if absolute == "" || filepath.Dir(absolute) == absolute {
		return Roots{}, errors.New("storage root must be an application-specific directory")
	}
	diagnostics := filepath.Join(absolute, "diagnostics")
	backups := filepath.Join(absolute, "backups")
	documents := filepath.Join(absolute, "documents")
	return Roots{
		Root: absolute, Config: filepath.Join(absolute, "config"), Data: filepath.Join(absolute, "data"),
		Catalog: filepath.Join(absolute, "catalog"), Objects: filepath.Join(absolute, "objects", "sha256"),
		Documents: documents, Exports: filepath.Join(documents, "exports"), Packages: filepath.Join(absolute, "packages"),
		Cache: filepath.Join(absolute, "cache"), State: filepath.Join(absolute, "state"),
		Diagnostics: diagnostics, Logs: filepath.Join(diagnostics, "logs"), Crashes: filepath.Join(diagnostics, "crashes"),
		Captures: filepath.Join(diagnostics, "captures"),
		Backups: backups, Migrations: filepath.Join(backups, "migrations"),
		Runtime: filepath.Join(absolute, "runtime"), Temp: filepath.Join(absolute, "tmp"),
	}, nil
}

func (r Roots) directories() []string {
	return []string{
		r.Config, r.Data, r.Catalog, r.Objects, r.Documents, r.Exports, r.Packages, r.Cache, r.State,
		r.Diagnostics, r.Logs, r.Crashes, r.Captures, r.Backups, r.Migrations, r.Runtime, r.Temp,
	}
}

func (r Roots) SettingsFile() string { return filepath.Join(r.Config, "settings.json") }
