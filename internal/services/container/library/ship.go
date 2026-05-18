package library

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed all:builtin_library
var builtinFS embed.FS

// builtinVersionHash returns SHA256 of all embedded paths + contents.
func builtinVersionHash() (string, error) {
	h := sha256.New()
	err := fs.WalkDir(builtinFS, "builtin_library", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		b, err := builtinFS.ReadFile(path)
		if err != nil {
			return err
		}
		h.Write([]byte(path))
		h.Write(b)
		return nil
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// EnsureBuiltinLibrary copies embedded builtin assets to libraryDir on first
// launch or when the embedded content changes (version hash mismatch).
// User-created files (those not present in the embed) are never touched.
// Builtin-named files are overwritten only on hash change; users who want to
// customise a builtin should fork it as a _user.json file (Task 17).
func EnsureBuiltinLibrary(libraryDir string) error {
	versionFile := filepath.Join(libraryDir, ".builtin_version")
	existing, _ := os.ReadFile(versionFile)
	fresh, err := builtinVersionHash()
	if err != nil {
		return err
	}
	if string(existing) == fresh {
		return nil
	}

	err = fs.WalkDir(builtinFS, "builtin_library", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		// rel is the path under builtin_library/, e.g. "subgraphs/foo.json"
		rel := path[len("builtin_library/"):]
		// skip git placeholder files
		if rel == ".keep" || filepath.Base(rel) == ".keep" {
			return nil
		}
		dst := filepath.Join(libraryDir, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		b, err := builtinFS.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, b, 0o644)
	})
	if err != nil {
		return err
	}

	if err := os.MkdirAll(libraryDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(versionFile, []byte(fresh), 0o644)
}

// IsBuiltinPath reports whether the given relative path (e.g. "subgraphs/foo.json")
// exists in the embedded builtin library FS.
// Task 17 will call this before save to prompt the user to fork as _user.json.
func IsBuiltinPath(rel string) bool {
	_, err := builtinFS.Open(filepath.Join("builtin_library", rel))
	return err == nil
}
