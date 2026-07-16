package nodepackage

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

const (
	// ArchiveManifestPath is the only manifest location accepted at the root
	// of a Node Package archive.
	ArchiveManifestPath     = "yotta-node-package.json"
	maxArchiveBytes         = int64(16 << 30)
	maxArchiveExpandedBytes = int64(16 << 30)
	maxArchiveEntries       = MaxDefinitions*2 + 1
	archiveEncryptionFlag   = 1 << 0
	archiveStagingFileMode  = 0o600
	archiveStagingExecMode  = 0o700
	archiveStagingDirectory = 0o700
)

type archivePayload struct {
	payload    Payload
	executable bool
}

// ExtractArchive verifies one complete Node Package archive and publishes its
// canonical manifest and exact payload set at a new destination directory.
// Any failure leaves destination absent and removes private staging bytes.
func ExtractArchive(ctx context.Context, archivePath, destination string) (Manifest, error) {
	if ctx == nil {
		return Manifest{}, errors.New("node package extraction requires a context")
	}
	if err := ctx.Err(); err != nil {
		return Manifest{}, err
	}
	destination, parent, err := prepareArchiveDestination(destination)
	if err != nil {
		return Manifest{}, err
	}
	archiveFile, archiveSize, err := openArchiveFile(archivePath)
	if err != nil {
		return Manifest{}, err
	}
	defer archiveFile.Close()

	reader, err := zip.NewReader(archiveFile, archiveSize)
	if err != nil {
		return Manifest{}, fmt.Errorf("open node package archive: %w", err)
	}
	entries, err := indexArchiveEntries(reader.File)
	if err != nil {
		return Manifest{}, err
	}
	manifestEntry, found := entries[ArchiveManifestPath]
	if !found {
		return Manifest{}, errors.New("node package archive is missing its manifest")
	}
	manifestBytes, err := readArchiveManifest(ctx, manifestEntry)
	if err != nil {
		return Manifest{}, err
	}
	manifest, err := Open(manifestBytes)
	if err != nil {
		return Manifest{}, fmt.Errorf("open archived node package manifest: %w", err)
	}
	payloads, err := archivePayloads(manifest)
	if err != nil {
		return Manifest{}, err
	}
	if err := validateArchiveEntrySet(entries, payloads); err != nil {
		return Manifest{}, err
	}

	staging, err := os.MkdirTemp(parent, "."+filepath.Base(destination)+".staging-*")
	if err != nil {
		return Manifest{}, fmt.Errorf("create node package staging directory: %w", err)
	}
	published := false
	defer func() {
		if !published {
			_ = os.RemoveAll(staging)
		}
	}()
	if err := os.Chmod(staging, archiveStagingDirectory); err != nil {
		return Manifest{}, fmt.Errorf("secure node package staging directory: %w", err)
	}
	if err := writeStagedBytes(filepath.Join(staging, filepath.FromSlash(ArchiveManifestPath)), manifestBytes, archiveStagingFileMode); err != nil {
		return Manifest{}, fmt.Errorf("stage node package manifest: %w", err)
	}

	paths := make([]string, 0, len(payloads))
	for payloadPath := range payloads {
		paths = append(paths, payloadPath)
	}
	sort.Strings(paths)
	for _, payloadPath := range paths {
		if err := ctx.Err(); err != nil {
			return Manifest{}, err
		}
		payload := payloads[payloadPath]
		mode := os.FileMode(archiveStagingFileMode)
		if payload.executable {
			mode = archiveStagingExecMode
		}
		target := filepath.Join(staging, filepath.FromSlash(payloadPath))
		if err := extractArchivePayload(ctx, entries[payloadPath], target, payload.payload, mode); err != nil {
			return Manifest{}, err
		}
	}
	if err := ctx.Err(); err != nil {
		return Manifest{}, err
	}
	if _, err := os.Lstat(destination); err == nil {
		return Manifest{}, errors.New("node package extraction destination already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return Manifest{}, fmt.Errorf("inspect node package extraction destination: %w", err)
	}
	if err := os.Rename(staging, destination); err != nil {
		return Manifest{}, fmt.Errorf("publish extracted node package: %w", err)
	}
	published = true
	return manifest, nil
}

func prepareArchiveDestination(destination string) (string, string, error) {
	if strings.TrimSpace(destination) == "" {
		return "", "", errors.New("node package extraction destination is empty")
	}
	absolute, err := filepath.Abs(destination)
	if err != nil {
		return "", "", fmt.Errorf("resolve node package extraction destination: %w", err)
	}
	absolute = filepath.Clean(absolute)
	parent := filepath.Dir(absolute)
	if absolute == parent || filepath.Base(absolute) == "." {
		return "", "", errors.New("node package extraction destination is invalid")
	}
	parentInfo, err := os.Lstat(parent)
	if err != nil {
		return "", "", fmt.Errorf("inspect node package extraction parent: %w", err)
	}
	if !parentInfo.IsDir() {
		return "", "", errors.New("node package extraction parent is not a directory")
	}
	if _, err := os.Lstat(absolute); err == nil {
		return "", "", errors.New("node package extraction destination already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", "", fmt.Errorf("inspect node package extraction destination: %w", err)
	}
	return absolute, parent, nil
}

func openArchiveFile(archivePath string) (*os.File, int64, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return nil, 0, fmt.Errorf("open node package archive: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, 0, fmt.Errorf("inspect node package archive: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxArchiveBytes {
		_ = file.Close()
		return nil, 0, errors.New("node package archive file is invalid or exceeds its byte budget")
	}
	return file, info.Size(), nil
}

func indexArchiveEntries(source []*zip.File) (map[string]*zip.File, error) {
	if len(source) == 0 || len(source) > maxArchiveEntries {
		return nil, errors.New("node package archive entry count is invalid")
	}
	entries := make(map[string]*zip.File, len(source))
	folded := make(map[string]string, len(source))
	for _, entry := range source {
		if !validPortablePath(entry.Name) || filepath.ToSlash(filepath.Clean(entry.Name)) != entry.Name {
			return nil, fmt.Errorf("node package archive entry path %q is invalid", entry.Name)
		}
		if entry.Flags&archiveEncryptionFlag != 0 {
			return nil, fmt.Errorf("node package archive entry %q is encrypted", entry.Name)
		}
		if entry.Method != zip.Store && entry.Method != zip.Deflate {
			return nil, fmt.Errorf("node package archive entry %q uses an unsupported compression method", entry.Name)
		}
		if !entry.FileInfo().Mode().IsRegular() {
			return nil, fmt.Errorf("node package archive entry %q is not a regular file", entry.Name)
		}
		foldedPath := strings.ToLower(entry.Name)
		if previous, exists := folded[foldedPath]; exists {
			return nil, fmt.Errorf("node package archive entries %q and %q collide", previous, entry.Name)
		}
		folded[foldedPath] = entry.Name
		entries[entry.Name] = entry
	}
	return entries, nil
}

func readArchiveManifest(ctx context.Context, entry *zip.File) ([]byte, error) {
	if entry.UncompressedSize64 == 0 || entry.UncompressedSize64 > uint64(MaxBytes) {
		return nil, errors.New("archived node package manifest exceeds its byte budget")
	}
	reader, err := entry.Open()
	if err != nil {
		return nil, fmt.Errorf("open archived node package manifest: %w", err)
	}
	defer reader.Close()
	limited := &io.LimitedReader{R: &contextReader{ctx: ctx, reader: reader}, N: MaxBytes + 1}
	raw, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read archived node package manifest: %w", err)
	}
	if int64(len(raw)) > MaxBytes || uint64(len(raw)) != entry.UncompressedSize64 {
		return nil, errors.New("archived node package manifest size is invalid")
	}
	return raw, nil
}

func archivePayloads(manifest Manifest) (map[string]archivePayload, error) {
	payloads := make(map[string]archivePayload)
	for _, node := range manifest.Nodes() {
		payload := node.Implementation.Payload
		candidate := archivePayload{payload: payload, executable: node.Implementation.ABI.Kind == nodecontract.ABIProcess}
		if err := mergeArchivePayload(payloads, candidate); err != nil {
			return nil, err
		}
	}
	for _, payload := range manifest.Documentation() {
		if err := mergeArchivePayload(payloads, archivePayload{payload: payload}); err != nil {
			return nil, err
		}
	}
	return payloads, nil
}

func mergeArchivePayload(target map[string]archivePayload, candidate archivePayload) error {
	if candidate.payload.Path == ArchiveManifestPath {
		return errors.New("node package payload uses the reserved archive manifest path")
	}
	if existing, found := target[candidate.payload.Path]; found {
		if existing.payload != candidate.payload {
			return fmt.Errorf("node package payload path %q has conflicting identities", candidate.payload.Path)
		}
		candidate.executable = candidate.executable || existing.executable
	}
	target[candidate.payload.Path] = candidate
	return nil
}

func validateArchiveEntrySet(entries map[string]*zip.File, payloads map[string]archivePayload) error {
	if len(entries) != len(payloads)+1 {
		return errors.New("node package archive file set does not match its manifest")
	}
	var expanded int64
	for payloadPath, expected := range payloads {
		entry, found := entries[payloadPath]
		if !found {
			return fmt.Errorf("node package archive is missing payload %q", payloadPath)
		}
		if entry.UncompressedSize64 != uint64(expected.payload.Size) {
			return fmt.Errorf("node package archive payload %q size does not match its manifest", payloadPath)
		}
		if expected.payload.Size > maxArchiveExpandedBytes-expanded {
			return errors.New("node package archive exceeds its expanded byte budget")
		}
		expanded += expected.payload.Size
	}
	for entryPath := range entries {
		if entryPath == ArchiveManifestPath {
			continue
		}
		if _, found := payloads[entryPath]; !found {
			return fmt.Errorf("node package archive contains undeclared payload %q", entryPath)
		}
	}
	return nil
}

func extractArchivePayload(ctx context.Context, entry *zip.File, target string, expected Payload, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(target), archiveStagingDirectory); err != nil {
		return fmt.Errorf("create payload directory for %q: %w", expected.Path, err)
	}
	if err := os.Chmod(filepath.Dir(target), archiveStagingDirectory); err != nil {
		return fmt.Errorf("secure payload directory for %q: %w", expected.Path, err)
	}
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return fmt.Errorf("create staged payload %q: %w", expected.Path, err)
	}
	closed := false
	defer func() {
		if !closed {
			_ = file.Close()
		}
	}()
	if err := file.Chmod(mode); err != nil {
		return fmt.Errorf("secure staged payload %q: %w", expected.Path, err)
	}
	reader, err := entry.Open()
	if err != nil {
		return fmt.Errorf("open archived payload %q: %w", expected.Path, err)
	}
	defer reader.Close()
	hash := sha256.New()
	limited := &io.LimitedReader{R: &contextReader{ctx: ctx, reader: reader}, N: expected.Size + 1}
	written, err := io.CopyBuffer(io.MultiWriter(file, hash), limited, make([]byte, 64<<10))
	if err != nil {
		return fmt.Errorf("extract archived payload %q: %w", expected.Path, err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if written != expected.Size {
		return fmt.Errorf("archived payload %q size integrity check failed", expected.Path)
	}
	digest := artifact.Digest("sha256:" + hex.EncodeToString(hash.Sum(nil)))
	if digest != expected.Digest {
		return fmt.Errorf("archived payload %q digest integrity check failed", expected.Path)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync staged payload %q: %w", expected.Path, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close staged payload %q: %w", expected.Path, err)
	}
	closed = true
	return nil
}

func writeStagedBytes(target string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(target), archiveStagingDirectory); err != nil {
		return err
	}
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	if err := file.Chmod(mode); err != nil {
		_ = file.Close()
		return err
	}
	if _, err := io.Copy(file, bytes.NewReader(data)); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(target []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(target)
}
