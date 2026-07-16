package nodepackage

import (
	"archive/zip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/yottaapp/yotta/internal/nodecontract"
)

type archiveTestEntry struct {
	name string
	data []byte
	mode os.FileMode
}

func TestExtractArchiveVerifiesAndPublishesExactPackage(t *testing.T) {
	draft := testDraft(t, nodecontract.ABIProcess)
	draft.Documentation = []Payload{testPayload(t, "docs/node.md", "text/markdown", "documentation")}
	manifest, err := Seal(draft)
	if err != nil {
		t.Fatal(err)
	}
	archivePath := writeArchive(t, []archiveTestEntry{
		{name: ArchiveManifestPath, data: manifest.Bytes()},
		{name: "bin/plugin.exe", data: []byte("process")},
		{name: "docs/node.md", data: []byte("documentation")},
	})
	destination := filepath.Join(t.TempDir(), "unpacked")

	extracted, err := ExtractArchive(context.Background(), archivePath, destination)
	if err != nil {
		t.Fatal(err)
	}
	if extracted.Digest() != manifest.Digest() {
		t.Fatalf("manifest digest = %q, want %q", extracted.Digest(), manifest.Digest())
	}
	assertFileContent(t, filepath.Join(destination, filepath.FromSlash(ArchiveManifestPath)), manifest.Bytes())
	assertFileContent(t, filepath.Join(destination, "bin", "plugin.exe"), []byte("process"))
	assertFileContent(t, filepath.Join(destination, "docs", "node.md"), []byte("documentation"))
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(destination, "bin", "plugin.exe"))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o700 {
			t.Fatalf("process payload mode = %o, want 700", info.Mode().Perm())
		}
	}
}

func TestExtractArchiveRejectsInvalidEntrySetsWithoutPublishing(t *testing.T) {
	manifest, err := Seal(testDraft(t, nodecontract.ABIProcess))
	if err != nil {
		t.Fatal(err)
	}
	validManifest := archiveTestEntry{name: ArchiveManifestPath, data: manifest.Bytes()}
	validPayload := archiveTestEntry{name: "bin/plugin.exe", data: []byte("process")}
	tests := []struct {
		name    string
		entries []archiveTestEntry
	}{
		{name: "missing manifest", entries: []archiveTestEntry{validPayload}},
		{name: "missing payload", entries: []archiveTestEntry{validManifest}},
		{name: "extra payload", entries: []archiveTestEntry{validManifest, validPayload, {name: "extra.txt", data: []byte("extra")}}},
		{name: "tampered payload", entries: []archiveTestEntry{validManifest, {name: validPayload.name, data: []byte("PROCESS")}}},
		{name: "duplicate payload", entries: []archiveTestEntry{validManifest, validPayload, validPayload}},
		{name: "traversal entry", entries: []archiveTestEntry{validManifest, validPayload, {name: "../escape", data: []byte("escape")}}},
		{name: "directory entry", entries: []archiveTestEntry{validManifest, {name: "bin/", mode: os.ModeDir | 0o755}, validPayload}},
		{name: "symlink entry", entries: []archiveTestEntry{validManifest, {name: validPayload.name, data: []byte("process"), mode: os.ModeSymlink | 0o777}}},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			parent := t.TempDir()
			destination := filepath.Join(parent, "unpacked")
			if _, err := ExtractArchive(context.Background(), writeArchive(t, testCase.entries), destination); err == nil {
				t.Fatal("invalid archive was extracted")
			}
			assertNotPublished(t, parent, destination)
		})
	}
}

func TestExtractArchiveRejectsManifestSizeMismatchAndExpandedBudget(t *testing.T) {
	t.Run("manifest size mismatch", func(t *testing.T) {
		draft := testDraft(t, nodecontract.ABIProcess)
		draft.Nodes[0].Implementation.Payload.Size++
		manifest, err := Seal(draft)
		if err != nil {
			t.Fatal(err)
		}
		archivePath := writeArchive(t, []archiveTestEntry{
			{name: ArchiveManifestPath, data: manifest.Bytes()},
			{name: "bin/plugin.exe", data: []byte("process")},
		})
		destination := filepath.Join(t.TempDir(), "unpacked")
		if _, err := ExtractArchive(context.Background(), archivePath, destination); err == nil {
			t.Fatal("archive with mismatched declared size was extracted")
		}
	})

	t.Run("expanded budget", func(t *testing.T) {
		draft := testDraft(t, nodecontract.ABIProcess)
		draft.Nodes[0].Implementation.Payload.Size = maxArchiveExpandedBytes/2 + 1
		documentation := testPayload(t, "docs/node.md", "text/markdown", "documentation")
		documentation.Size = maxArchiveExpandedBytes / 2
		draft.Documentation = []Payload{documentation}
		manifest, err := Seal(draft)
		if err != nil {
			t.Fatal(err)
		}
		archivePath := writeArchive(t, []archiveTestEntry{
			{name: ArchiveManifestPath, data: manifest.Bytes()},
			{name: "bin/plugin.exe", data: []byte("process")},
		})
		destination := filepath.Join(t.TempDir(), "unpacked")
		if _, err := ExtractArchive(context.Background(), archivePath, destination); err == nil {
			t.Fatal("archive exceeding expanded byte budget was extracted")
		}
	})
}

func TestExtractArchiveEnforcesArchiveAndEntryBudgets(t *testing.T) {
	t.Run("archive bytes", func(t *testing.T) {
		archivePath := filepath.Join(t.TempDir(), "oversized.ynp")
		file, err := os.Create(archivePath)
		if err != nil {
			t.Fatal(err)
		}
		if err := file.Truncate(maxArchiveBytes + 1); err != nil {
			_ = file.Close()
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err := ExtractArchive(context.Background(), archivePath, filepath.Join(t.TempDir(), "unpacked")); err == nil {
			t.Fatal("oversized archive was extracted")
		}
	})

	t.Run("entry count", func(t *testing.T) {
		entries := make([]archiveTestEntry, maxArchiveEntries+1)
		for index := range entries {
			entries[index] = archiveTestEntry{name: fmt.Sprintf("extra/%05d.bin", index)}
		}
		if _, err := ExtractArchive(context.Background(), writeArchive(t, entries), filepath.Join(t.TempDir(), "unpacked")); err == nil {
			t.Fatal("archive exceeding entry count budget was extracted")
		}
	})
}

func TestExtractArchiveHonoursCancellationAndExistingDestination(t *testing.T) {
	manifest, err := Seal(testDraft(t, nodecontract.ABIWIT))
	if err != nil {
		t.Fatal(err)
	}
	archivePath := writeArchive(t, []archiveTestEntry{
		{name: ArchiveManifestPath, data: manifest.Bytes()},
		{name: "bin/plugin.wasm", data: []byte("wasm")},
	})

	t.Run("cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		parent := t.TempDir()
		destination := filepath.Join(parent, "unpacked")
		if _, err := ExtractArchive(ctx, archivePath, destination); err == nil {
			t.Fatal("cancelled extraction succeeded")
		}
		assertNotPublished(t, parent, destination)
	})

	t.Run("existing destination", func(t *testing.T) {
		parent := t.TempDir()
		destination := filepath.Join(parent, "unpacked")
		if err := os.Mkdir(destination, 0o700); err != nil {
			t.Fatal(err)
		}
		marker := filepath.Join(destination, "owned.txt")
		if err := os.WriteFile(marker, []byte("keep"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := ExtractArchive(context.Background(), archivePath, destination); err == nil {
			t.Fatal("existing destination was overwritten")
		}
		assertFileContent(t, marker, []byte("keep"))
	})
}

func writeArchive(t *testing.T, entries []archiveTestEntry) string {
	t.Helper()
	archivePath := filepath.Join(t.TempDir(), "package.ynp")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.name, Method: zip.Deflate}
		if entry.mode != 0 {
			header.SetMode(entry.mode)
		}
		entryWriter, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entryWriter.Write(entry.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return archivePath
}

func assertFileContent(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("%s = %q, want %q", path, got, want)
	}
}

func assertNotPublished(t *testing.T, parent, destination string) {
	t.Helper()
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatalf("destination exists after rejected archive: %v", err)
	}
	staging, err := filepath.Glob(filepath.Join(parent, ".unpacked.staging-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(staging) != 0 {
		t.Fatalf("staging directories leaked: %v", staging)
	}
}
