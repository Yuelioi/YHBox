package storage

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveUsesExplicitRootAndProjectsAllLifecycles(t *testing.T) {
	base := filepath.Join(t.TempDir(), "profile")
	roots, err := Resolve(base)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	absolute, err := filepath.Abs(base)
	if err != nil {
		t.Fatal(err)
	}
	if roots.Root != absolute {
		t.Fatalf("root = %q, want %q", roots.Root, absolute)
	}
	want := map[string]string{
		"config":      filepath.Join(absolute, "config"),
		"data":        filepath.Join(absolute, "data"),
		"catalog":     filepath.Join(absolute, "catalog"),
		"objects":     filepath.Join(absolute, "objects", "sha256"),
		"documents":   filepath.Join(absolute, "documents"),
		"exports":     filepath.Join(absolute, "documents", "exports"),
		"packages":    filepath.Join(absolute, "packages"),
		"cache":       filepath.Join(absolute, "cache"),
		"state":       filepath.Join(absolute, "state"),
		"diagnostics": filepath.Join(absolute, "diagnostics"),
		"logs":        filepath.Join(absolute, "diagnostics", "logs"),
		"crashes":     filepath.Join(absolute, "diagnostics", "crashes"),
		"captures":    filepath.Join(absolute, "diagnostics", "captures"),
		"backups":     filepath.Join(absolute, "backups"),
		"runtime":     filepath.Join(absolute, "runtime"),
		"temp":        filepath.Join(absolute, "tmp"),
	}
	got := map[string]string{
		"config": roots.Config, "data": roots.Data, "catalog": roots.Catalog, "objects": roots.Objects,
		"documents": roots.Documents, "exports": roots.Exports,
		"packages": roots.Packages, "cache": roots.Cache, "state": roots.State,
		"diagnostics": roots.Diagnostics, "logs": roots.Logs, "crashes": roots.Crashes, "captures": roots.Captures,
		"backups": roots.Backups, "runtime": roots.Runtime, "temp": roots.Temp,
	}
	for name, expected := range want {
		if got[name] != expected {
			t.Errorf("%s root = %q, want %q", name, got[name], expected)
		}
	}
}

func TestResolveUsesEnvironmentOnlyWhenNoExplicitRoot(t *testing.T) {
	fromEnvironment := filepath.Join(t.TempDir(), "environment")
	explicit := filepath.Join(t.TempDir(), "explicit")
	t.Setenv(EnvironmentRoot, fromEnvironment)

	roots, err := Resolve("")
	if err != nil {
		t.Fatalf("Resolve environment: %v", err)
	}
	wantEnvironment, _ := filepath.Abs(fromEnvironment)
	if roots.Root != wantEnvironment {
		t.Fatalf("environment root = %q, want %q", roots.Root, wantEnvironment)
	}

	roots, err = Resolve(explicit)
	if err != nil {
		t.Fatalf("Resolve explicit: %v", err)
	}
	wantExplicit, _ := filepath.Abs(explicit)
	if roots.Root != wantExplicit {
		t.Fatalf("explicit root = %q, want %q", roots.Root, wantExplicit)
	}
}

func TestOpenCreatesClaimedRootAndAllDirectories(t *testing.T) {
	root := filepath.Join(t.TempDir(), "profile")
	profile, err := Open(context.Background(), OpenOptions{Root: root})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer profile.Close()

	raw, err := os.ReadFile(filepath.Join(root, rootManifestFilename))
	if err != nil {
		t.Fatalf("read root manifest: %v", err)
	}
	var manifest RootManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("decode root manifest: %v", err)
	}
	if manifest.Format != RootFormat || manifest.Version != LayoutVersion {
		t.Fatalf("manifest = %#v", manifest)
	}
	for _, directory := range profile.Roots.directories() {
		info, err := os.Stat(directory)
		if err != nil || !info.IsDir() {
			t.Errorf("directory %q is unavailable: info=%v err=%v", directory, info, err)
		}
	}
}

func TestOpenRejectsUnclaimedAndUnsupportedRoots(t *testing.T) {
	t.Run("unclaimed non-empty", func(t *testing.T) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "mine.txt"), []byte("keep"), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := Open(context.Background(), OpenOptions{Root: root})
		if !errors.Is(err, ErrUnclaimedRoot) {
			t.Fatalf("Open error = %v, want ErrUnclaimedRoot", err)
		}
		if _, statErr := os.Stat(filepath.Join(root, "mine.txt")); statErr != nil {
			t.Fatalf("foreign content changed: %v", statErr)
		}
	})

	t.Run("unsupported layout", func(t *testing.T) {
		root := t.TempDir()
		raw, err := json.Marshal(RootManifest{Format: RootFormat, Version: "99"})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, rootManifestFilename), raw, 0o600); err != nil {
			t.Fatal(err)
		}
		_, err = Open(context.Background(), OpenOptions{Root: root})
		if !errors.Is(err, ErrUnsupportedLayout) {
			t.Fatalf("Open error = %v, want ErrUnsupportedLayout", err)
		}
	})
}

func TestOpenHoldsSingleWriterLease(t *testing.T) {
	root := filepath.Join(t.TempDir(), "profile")
	first, err := Open(context.Background(), OpenOptions{Root: root})
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	second, err := Open(context.Background(), OpenOptions{Root: root})
	if second != nil {
		_ = second.Close()
		t.Fatal("second Open unexpectedly succeeded")
	}
	if !errors.Is(err, ErrRootInUse) {
		t.Fatalf("second Open error = %v, want ErrRootInUse", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	reopened, err := Open(context.Background(), OpenOptions{Root: root})
	if err != nil {
		t.Fatalf("reopen after Close: %v", err)
	}
	if err := reopened.Close(); err != nil {
		t.Fatalf("reopened Close: %v", err)
	}
}
