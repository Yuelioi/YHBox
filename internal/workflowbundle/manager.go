// Package workflowbundle owns portable Workflow Source archives.
package workflowbundle

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/durablefs"
	"github.com/yottaapp/yotta/internal/workflow/schema"
	"github.com/yottaapp/yotta/internal/workflowstore"
)

const (
	Format                = "yotta.workflow-bundle"
	Version               = 1
	ManifestPath          = "yotta-workflow-bundle.json"
	SourcePath            = "workflow.json"
	Extension             = ".yotta-workflow"
	maxArchiveBytes       = int64(1 << 30)
	maxExpandedBytes      = int64(1 << 30)
	maxSourceBytes        = int64(16 << 20)
	maxManifestBytes      = int64(1 << 20)
	maxEntries            = 4098
	privateFileMode       = 0o600
	archiveEncryptionFlag = 1 << 0
)

type SourceRepository interface {
	GetSource(string) (workflowstore.SourceSnapshot, error)
	PublishImportedSource(context.Context, []byte, int64, artifact.Digest) (workflowstore.SourceSnapshot, error)
}

type BlobStore interface {
	PutRetained(context.Context, string, io.Reader) (blob.BlobRef, blob.Retention, error)
	WriteTo(context.Context, blob.BlobRef, io.Writer) error
	Verify(context.Context, blob.BlobRef) error
}

type Manager struct {
	sources SourceRepository
	blobs   BlobStore
	newID   func() string
}

type Manifest struct {
	Format     string          `json:"format"`
	Version    int             `json:"version"`
	WorkflowID string          `json:"workflowId"`
	SourceHash artifact.Digest `json:"sourceHash"`
	Blobs      []blob.BlobRef  `json:"blobs"`
}

type Info struct {
	WorkflowID string          `json:"workflowId"`
	Name       string          `json:"name"`
	Revision   int64           `json:"revision"`
	SourceHash artifact.Digest `json:"sourceHash"`
	BlobCount  int             `json:"blobCount"`
	BlobBytes  int64           `json:"blobBytes"`
}

type ExportResult struct {
	Info Info   `json:"info"`
	Path string `json:"path"`
}

type ImportMode string

const (
	ImportCopy    ImportMode = "copy"
	ImportReplace ImportMode = "replace"
)

type ImportRequest struct {
	Path               string          `json:"path"`
	Mode               ImportMode      `json:"mode"`
	TargetWorkflowID   string          `json:"targetWorkflowId,omitempty"`
	ExpectedRevision   int64           `json:"expectedRevision,omitempty"`
	ExpectedSourceHash artifact.Digest `json:"expectedSourceHash,omitempty"`
}

type ImportResult struct {
	Original Info                         `json:"original"`
	Source   workflowstore.SourceSnapshot `json:"-"`
}

func New(sources SourceRepository, blobs BlobStore) (*Manager, error) {
	if sources == nil || blobs == nil {
		return nil, errors.New("workflow bundle manager requires source and blob stores")
	}
	return &Manager{sources: sources, blobs: blobs, newID: uuid.NewString}, nil
}

func (m *Manager) Export(ctx context.Context, workflowID, destination string) (ExportResult, error) {
	if ctx == nil || strings.TrimSpace(workflowID) == "" || strings.TrimSpace(destination) == "" {
		return ExportResult{}, errors.New("workflow bundle export requires context, workflow ID, and destination")
	}
	snapshot, err := m.sources.GetSource(workflowID)
	if err != nil {
		return ExportResult{}, err
	}
	document, canonical, digest, refs, err := inspectSource(snapshot.Artifact())
	if err != nil {
		return ExportResult{}, err
	}
	if digest != snapshot.Hash() || !bytes.Equal(canonical, snapshot.Artifact()) {
		return ExportResult{}, errors.New("stored Workflow Source identity changed before export")
	}
	for _, ref := range refs {
		if err := m.blobs.Verify(ctx, ref); err != nil {
			return ExportResult{}, fmt.Errorf("verify export blob %s: %w", ref.Digest, err)
		}
	}
	manifest := Manifest{Format: Format, Version: Version, WorkflowID: workflowID, SourceHash: digest, Blobs: refs}
	manifestRaw, err := json.Marshal(manifest)
	if err != nil {
		return ExportResult{}, err
	}
	if err := m.writeArchive(ctx, destination, manifestRaw, canonical, refs); err != nil {
		return ExportResult{}, err
	}
	return ExportResult{Info: sourceInfo(document, digest, refs), Path: destination}, nil
}

func (m *Manager) Inspect(ctx context.Context, archivePath string) (Info, error) {
	bundle, err := m.openArchive(ctx, archivePath)
	if err != nil {
		return Info{}, err
	}
	defer bundle.close()
	return bundle.info, nil
}

func (m *Manager) Import(ctx context.Context, request ImportRequest) (ImportResult, error) {
	if ctx == nil {
		return ImportResult{}, errors.New("workflow bundle import requires context")
	}
	bundle, err := m.openArchive(ctx, request.Path)
	if err != nil {
		return ImportResult{}, err
	}
	defer bundle.close()
	document := bundle.document
	baseRevision := int64(-1)
	expectedHash := artifact.Digest("")
	switch request.Mode {
	case "", ImportCopy:
		document.Workflow.ID = m.newID()
		document.Revision = 0
	case ImportReplace:
		if strings.TrimSpace(request.TargetWorkflowID) == "" || request.ExpectedRevision < 0 || !request.ExpectedSourceHash.Valid() {
			return ImportResult{}, errors.New("replace import requires exact target revision and source hash")
		}
		document.Workflow.ID = request.TargetWorkflowID
		document.Revision = request.ExpectedRevision + 1
		baseRevision = request.ExpectedRevision
		expectedHash = request.ExpectedSourceHash
	default:
		return ImportResult{}, fmt.Errorf("unsupported workflow bundle import mode %q", request.Mode)
	}
	raw, err := json.Marshal(document)
	if err != nil {
		return ImportResult{}, err
	}
	_, canonical, _, diagnostics, err := schema.CanonicalSource(raw)
	if err != nil {
		return ImportResult{}, err
	}
	if len(diagnostics) != 0 {
		return ImportResult{}, errors.New("rewritten Workflow Source is invalid")
	}
	retentions := make([]blob.Retention, 0, len(bundle.manifest.Blobs))
	defer func() {
		for _, retention := range retentions {
			retention.Release()
		}
	}()
	for _, ref := range bundle.manifest.Blobs {
		entry := bundle.blobEntries[ref.Digest]
		reader, err := entry.Open()
		if err != nil {
			return ImportResult{}, err
		}
		stored, retention, putErr := m.blobs.PutRetained(ctx, ref.MediaType, reader)
		closeErr := reader.Close()
		if putErr != nil {
			return ImportResult{}, fmt.Errorf("import blob %s: %w", ref.Digest, putErr)
		}
		if closeErr != nil {
			return ImportResult{}, closeErr
		}
		if stored.Digest != ref.Digest || stored.Size != ref.Size {
			return ImportResult{}, fmt.Errorf("import blob %s changed identity", ref.Digest)
		}
		retentions = append(retentions, retention)
	}
	published, err := m.sources.PublishImportedSource(ctx, canonical, baseRevision, expectedHash)
	if err != nil {
		return ImportResult{}, err
	}
	return ImportResult{Original: bundle.info, Source: published}, nil
}

type openedBundle struct {
	file        *os.File
	manifest    Manifest
	document    schema.WorkflowSource
	info        Info
	blobEntries map[artifact.Digest]*zip.File
}

func (b *openedBundle) close() { _ = b.file.Close() }

func (m *Manager) openArchive(ctx context.Context, archivePath string) (*openedBundle, error) {
	if ctx == nil || strings.TrimSpace(archivePath) == "" {
		return nil, errors.New("workflow bundle path and context are required")
	}
	file, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*openedBundle, error) {
		_ = file.Close()
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		return fail(err)
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxArchiveBytes {
		return fail(errors.New("workflow bundle archive size is invalid"))
	}
	reader, err := zip.NewReader(file, info.Size())
	if err != nil {
		return fail(fmt.Errorf("open workflow bundle: %w", err))
	}
	entries, err := indexEntries(reader.File)
	if err != nil {
		return fail(err)
	}
	manifestEntry, ok := entries[ManifestPath]
	if !ok {
		return fail(errors.New("workflow bundle manifest is missing"))
	}
	manifestRaw, err := readEntry(ctx, manifestEntry, maxManifestBytes)
	if err != nil {
		return fail(err)
	}
	var manifest Manifest
	if err := decodeStrict(manifestRaw, &manifest); err != nil {
		return fail(fmt.Errorf("decode workflow bundle manifest: %w", err))
	}
	if err := validateManifest(manifest); err != nil {
		return fail(err)
	}
	sourceEntry, ok := entries[SourcePath]
	if !ok {
		return fail(errors.New("workflow bundle Source is missing"))
	}
	sourceRaw, err := readEntry(ctx, sourceEntry, maxSourceBytes)
	if err != nil {
		return fail(err)
	}
	document, canonical, digest, refs, err := inspectSource(sourceRaw)
	if err != nil {
		return fail(err)
	}
	if !bytes.Equal(sourceRaw, canonical) || digest != manifest.SourceHash || document.Workflow.ID != manifest.WorkflowID || !equalRefs(refs, manifest.Blobs) {
		return fail(errors.New("workflow bundle Source identity does not match its manifest"))
	}
	blobEntries := make(map[artifact.Digest]*zip.File)
	for _, ref := range manifest.Blobs {
		entryPath := blobEntryPath(ref.Digest)
		entry, ok := entries[entryPath]
		if !ok {
			return fail(fmt.Errorf("workflow bundle blob %s is missing", ref.Digest))
		}
		if entry.UncompressedSize64 != uint64(ref.Size) {
			return fail(fmt.Errorf("workflow bundle blob %s size does not match", ref.Digest))
		}
		if existing := blobEntries[ref.Digest]; existing != nil && existing != entry {
			return fail(fmt.Errorf("workflow bundle blob %s has conflicting entries", ref.Digest))
		}
		blobEntries[ref.Digest] = entry
	}
	if len(entries) != len(blobEntries)+2 {
		return fail(errors.New("workflow bundle archive contains undeclared entries"))
	}
	for digest, entry := range blobEntries {
		if err := verifyEntry(ctx, entry, digest); err != nil {
			return fail(err)
		}
	}
	return &openedBundle{file: file, manifest: manifest, document: document, info: sourceInfo(document, digest, refs), blobEntries: blobEntries}, nil
}

func (m *Manager) writeArchive(ctx context.Context, destination string, manifest, source []byte, refs []blob.BlobRef) error {
	destination, err := filepath.Abs(destination)
	if err != nil {
		return err
	}
	directory := filepath.Dir(destination)
	info, err := os.Stat(directory)
	if err != nil || !info.IsDir() {
		return errors.New("workflow bundle destination directory does not exist")
	}
	if current, statErr := os.Lstat(destination); statErr == nil && (!current.Mode().IsRegular() || current.Mode()&os.ModeSymlink != 0) {
		return errors.New("workflow bundle destination is not a regular file")
	} else if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return statErr
	}
	temp, err := os.CreateTemp(directory, ".yotta-workflow-")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(privateFileMode); err != nil {
		_ = temp.Close()
		return err
	}
	archive := zip.NewWriter(temp)
	writeBytes := func(name string, raw []byte) error {
		writer, err := archive.CreateHeader(archiveHeader(name))
		if err != nil {
			return err
		}
		_, err = writer.Write(raw)
		return err
	}
	if err := writeBytes(ManifestPath, manifest); err != nil {
		_ = archive.Close()
		_ = temp.Close()
		return err
	}
	if err := writeBytes(SourcePath, source); err != nil {
		_ = archive.Close()
		_ = temp.Close()
		return err
	}
	written := make(map[artifact.Digest]struct{})
	for _, ref := range refs {
		if _, ok := written[ref.Digest]; ok {
			continue
		}
		writer, err := archive.CreateHeader(archiveHeader(blobEntryPath(ref.Digest)))
		if err != nil {
			_ = archive.Close()
			_ = temp.Close()
			return err
		}
		if err := m.blobs.WriteTo(ctx, ref, writer); err != nil {
			_ = archive.Close()
			_ = temp.Close()
			return err
		}
		written[ref.Digest] = struct{}{}
	}
	if err := archive.Close(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := durablefs.Replace(tempPath, destination); err != nil {
		return fmt.Errorf("publish workflow bundle: %w", err)
	}
	return nil
}

func inspectSource(raw []byte) (schema.WorkflowSource, []byte, artifact.Digest, []blob.BlobRef, error) {
	document, canonical, digest, diagnostics, err := schema.CanonicalSource(raw)
	if err != nil {
		return schema.WorkflowSource{}, nil, "", nil, err
	}
	if len(diagnostics) != 0 {
		return schema.WorkflowSource{}, nil, "", nil, errors.New("workflow bundle Source is invalid")
	}
	refs := make([]blob.BlobRef, 0)
	seen := make(map[blob.BlobRef]struct{})
	for _, graph := range document.Graphs {
		for _, node := range graph.Nodes {
			for _, binding := range node.Bindings {
				if binding.Kind != schema.BindingBlob || binding.Blob == nil {
					continue
				}
				if err := binding.Blob.Validate(); err != nil {
					return schema.WorkflowSource{}, nil, "", nil, err
				}
				if _, ok := seen[*binding.Blob]; ok {
					continue
				}
				seen[*binding.Blob] = struct{}{}
				refs = append(refs, *binding.Blob)
			}
		}
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].Digest == refs[j].Digest {
			if refs[i].MediaType == refs[j].MediaType {
				return refs[i].Size < refs[j].Size
			}
			return refs[i].MediaType < refs[j].MediaType
		}
		return refs[i].Digest < refs[j].Digest
	})
	return document, canonical, digest, refs, nil
}

func sourceInfo(document schema.WorkflowSource, digest artifact.Digest, refs []blob.BlobRef) Info {
	var bytes int64
	seen := make(map[artifact.Digest]struct{})
	for _, ref := range refs {
		if _, ok := seen[ref.Digest]; ok {
			continue
		}
		seen[ref.Digest] = struct{}{}
		bytes += ref.Size
	}
	return Info{WorkflowID: document.Workflow.ID, Name: document.Workflow.Name, Revision: document.Revision, SourceHash: digest, BlobCount: len(seen), BlobBytes: bytes}
}

func validateManifest(manifest Manifest) error {
	if manifest.Format != Format || manifest.Version != Version || strings.TrimSpace(manifest.WorkflowID) == "" || !manifest.SourceHash.Valid() || len(manifest.Blobs) > maxEntries-2 {
		return errors.New("workflow bundle manifest identity is invalid")
	}
	for i, ref := range manifest.Blobs {
		if err := ref.Validate(); err != nil {
			return fmt.Errorf("workflow bundle blob %d is invalid: %w", i, err)
		}
		if i > 0 {
			previous := manifest.Blobs[i-1]
			if previous.Digest > ref.Digest || previous.Digest == ref.Digest && previous.MediaType >= ref.MediaType {
				return errors.New("workflow bundle blob manifest is not uniquely sorted")
			}
		}
	}
	return nil
}

func equalRefs(left, right []blob.BlobRef) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func indexEntries(files []*zip.File) (map[string]*zip.File, error) {
	if len(files) < 2 || len(files) > maxEntries {
		return nil, errors.New("workflow bundle entry count is invalid")
	}
	entries := make(map[string]*zip.File, len(files))
	var expanded uint64
	for _, entry := range files {
		name := entry.Name
		if name == "" || name == ".." || strings.HasPrefix(name, "../") || strings.Contains(name, "\\") || strings.HasPrefix(name, "/") || path.Clean(name) != name || strings.Contains(name, ":") || entry.FileInfo().IsDir() || entry.Mode()&os.ModeSymlink != 0 || entry.Flags&archiveEncryptionFlag != 0 {
			return nil, fmt.Errorf("workflow bundle contains unsafe entry %q", name)
		}
		if _, duplicate := entries[name]; duplicate {
			return nil, fmt.Errorf("workflow bundle contains duplicate entry %q", name)
		}
		if entry.UncompressedSize64 > uint64(maxExpandedBytes)-expanded {
			return nil, errors.New("workflow bundle exceeds expanded byte budget")
		}
		expanded += entry.UncompressedSize64
		entries[name] = entry
	}
	return entries, nil
}

func readEntry(ctx context.Context, entry *zip.File, maximum int64) ([]byte, error) {
	if entry.UncompressedSize64 > uint64(maximum) {
		return nil, fmt.Errorf("workflow bundle entry %q exceeds byte budget", entry.Name)
	}
	reader, err := entry.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	result, err := io.ReadAll(io.LimitReader(&contextReader{ctx: ctx, reader: reader}, maximum+1))
	if err != nil {
		return nil, err
	}
	if int64(len(result)) > maximum || uint64(len(result)) != entry.UncompressedSize64 {
		return nil, fmt.Errorf("workflow bundle entry %q size is invalid", entry.Name)
	}
	return result, nil
}

func verifyEntry(ctx context.Context, entry *zip.File, expected artifact.Digest) error {
	reader, err := entry.Open()
	if err != nil {
		return err
	}
	defer reader.Close()
	hash := sha256.New()
	written, err := io.Copy(hash, &contextReader{ctx: ctx, reader: reader})
	if err != nil {
		return err
	}
	if uint64(written) != entry.UncompressedSize64 || artifact.Digest("sha256:"+hex.EncodeToString(hash.Sum(nil))) != expected {
		return fmt.Errorf("workflow bundle blob %s failed integrity check", expected)
	}
	return nil
}

func decodeStrict(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("JSON document contains trailing data")
	}
	return nil
}

func archiveHeader(name string) *zip.FileHeader {
	header := &zip.FileHeader{Name: name, Method: zip.Deflate}
	header.SetMode(privateFileMode)
	header.Modified = time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)
	return header
}

func blobEntryPath(digest artifact.Digest) string {
	return "blobs/" + strings.TrimPrefix(digest.String(), "sha256:")
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
