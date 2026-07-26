// Package workspacefs exposes a bounded, Run-scoped filesystem rooted in
// Yotta-managed workflow storage. It never accepts absolute host paths.
package workspacefs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/durablefs"
	"github.com/yottaapp/yotta/internal/resource"
)

const (
	ProviderID  = "yotta.workspace-files"
	ProviderABI = "https://schemas.yotta.dev/provider-abi/workspace-files/v1"
	TargetID    = "workflow-files"
	TargetKind  = "workspace-filesystem"
	Kind        = "workspace-files/session"

	OperationStat        = "stat"
	OperationRead        = "read"
	OperationReadRange   = "read-range"
	OperationWriteAppend = "write-append"
	OperationWriteCommit = "write-commit"
	OperationWriteCancel = "write-cancel"

	ScopeRoot = "workflow-files"

	CodeInvalidPath       = "filesystem.invalid_path"
	CodeNotFound          = "filesystem.not_found"
	CodeBudgetExceeded    = "filesystem.budget_exceeded"
	CodeIsDirectory       = "filesystem.is_directory"
	CodeReadFailed        = "filesystem.read_failed"
	CodeWriteFailed       = "filesystem.write_failed"
	CodeContractViolation = "filesystem.contract_violation"

	maxSafeInteger = int64(9_007_199_254_740_991)
)

type Failure struct {
	Code  string
	Cause error
}

func (f *Failure) Error() string {
	if f == nil || f.Cause == nil {
		return "workspace filesystem failed"
	}
	return f.Cause.Error()
}

func (f *Failure) Unwrap() error {
	if f == nil {
		return nil
	}
	return f.Cause
}

type Limits struct {
	MaxReadBytes  int
	MaxWriteBytes int
	MaxChunkBytes int
}

type Scope struct {
	Root string `json:"root"`
}

type ReadRequest struct {
	Path     string `json:"path"`
	MaxBytes int    `json:"maxBytes"`
}

type ReadRangeRequest struct {
	Path   string `json:"path"`
	Offset int64  `json:"offset"`
	Length int64  `json:"length"`
}

type WriteConfig struct {
	Path      string `json:"path"`
	Overwrite bool   `json:"overwrite"`
}

type StatRequest struct {
	Path string `json:"path"`
}

type Metadata struct {
	Path               string `json:"path"`
	Name               string `json:"name"`
	Extension          string `json:"extension"`
	MediaType          string `json:"mediaType"`
	Size               int64  `json:"size"`
	ModifiedUnixMillis int64  `json:"modifiedUnixMillis"`
	IsDirectory        bool   `json:"isDirectory"`
}

type ReadResponse struct {
	Data     []byte   `json:"data"`
	Metadata Metadata `json:"metadata"`
}

func OpenMetadata(raw []byte) (Metadata, error) {
	canonical, err := artifact.Canonicalize(raw)
	if err != nil || !bytes.Equal(canonical, raw) {
		return Metadata{}, failure(CodeContractViolation, errors.New("workspace file metadata is not canonical"))
	}
	var result Metadata
	if err := decodeExact(raw, &result); err != nil || result.Validate() != nil {
		return Metadata{}, failure(CodeContractViolation, errors.Join(err, result.Validate()))
	}
	return result, nil
}

func OpenReadResponse(raw []byte, maxBytes int) (ReadResponse, error) {
	canonical, err := artifact.Canonicalize(raw)
	if err != nil || !bytes.Equal(canonical, raw) {
		return ReadResponse{}, failure(CodeContractViolation, errors.New("workspace file response is not canonical"))
	}
	var result ReadResponse
	if err := decodeExact(raw, &result); err != nil || result.Metadata.Validate() != nil || result.Metadata.IsDirectory || maxBytes <= 0 || len(result.Data) > maxBytes {
		return ReadResponse{}, failure(CodeContractViolation, errors.Join(err, result.Metadata.Validate()))
	}
	return result, nil
}

func (m Metadata) Validate() error {
	if m.Path == "" || len(m.Path) > 4096 || m.Name == "" || len(m.Name) > 255 || len(m.Extension) > 255 ||
		m.MediaType == "" || len(m.MediaType) > 255 || m.Size < 0 || m.Size > maxSafeInteger ||
		m.ModifiedUnixMillis < -maxSafeInteger || m.ModifiedUnixMillis > maxSafeInteger ||
		filepath.ToSlash(filepath.Clean(filepath.FromSlash(m.Path))) != m.Path ||
		m.Path == "." || m.Path == ".." || strings.HasPrefix(m.Path, "../") || filepath.IsAbs(m.Path) ||
		filepath.Base(filepath.FromSlash(m.Path)) != m.Name || strings.ToLower(filepath.Ext(m.Name)) != m.Extension {
		return errors.New("invalid workspace file metadata")
	}
	return nil
}

type Provider struct {
	root   string
	limits Limits
}

type session struct {
	root   string
	limits Limits
	writer *writerState
}

type writerState struct {
	file      *os.File
	temporary string
	target    string
	relative  string
	written   int
	overwrite bool
	committed bool
}

func ProviderArtifactDigest() (artifact.Digest, error) {
	manifest, err := artifact.Marshal(map[string]any{
		"providerId": ProviderID, "providerAbi": ProviderABI, "implementationVersion": "v2",
		"resourceKinds": []string{Kind}, "operations": []string{
			OperationRead, OperationReadRange, OperationStat, OperationWriteAppend, OperationWriteCancel, OperationWriteCommit,
		},
	})
	if err != nil {
		return "", err
	}
	return artifact.Sum("yotta/provider-implementation-manifest/v1", manifest)
}

func NewProvider(root string, limits Limits) (*Provider, error) {
	if strings.TrimSpace(root) == "" || limits.MaxReadBytes <= 0 {
		return nil, errors.New("workspace filesystem requires a root and positive read budget")
	}
	if limits.MaxWriteBytes <= 0 {
		limits.MaxWriteBytes = limits.MaxReadBytes
	}
	if limits.MaxChunkBytes <= 0 {
		limits.MaxChunkBytes = 64 << 10
	}
	if limits.MaxWriteBytes <= 0 || limits.MaxChunkBytes <= 0 || limits.MaxChunkBytes > 4<<20 {
		return nil, errors.New("workspace filesystem write and chunk budgets are invalid")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace filesystem root: %w", err)
	}
	if err := os.MkdirAll(abs, 0o700); err != nil {
		return nil, fmt.Errorf("create workspace filesystem root: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace filesystem root links: %w", err)
	}
	return &Provider{root: filepath.Clean(resolved), limits: limits}, nil
}

func (p *Provider) Open(_ context.Context, request resource.ProviderOpenRequest) (any, error) {
	if p == nil || request.ProviderID != ProviderID || request.TargetID != TargetID || request.Kind != Kind || request.CredentialBindingID != "" {
		return nil, errors.New("workspace filesystem open identity is invalid")
	}
	if err := requireOperations(request.Operations); err != nil {
		return nil, err
	}
	var scope Scope
	if err := decodeExact(request.CapabilityScope, &scope); err != nil || scope.Root != ScopeRoot {
		return nil, errors.New("workspace filesystem capability scope is invalid")
	}
	result := &session{root: p.root, limits: p.limits}
	if writeOperations(request.Operations) {
		var config WriteConfig
		if err := decodeExact(request.Config, &config); err != nil {
			return nil, errors.New("workspace filesystem write config is invalid")
		}
		writer, err := openWriter(p.root, config)
		if err != nil {
			return nil, err
		}
		result.writer = writer
	} else {
		var config struct{}
		if err := decodeExact(request.Config, &config); err != nil {
			return nil, errors.New("workspace filesystem config must be an empty object")
		}
	}
	return result, nil
}

func (p *Provider) Invoke(ctx context.Context, object any, operation string, payload []byte) ([]byte, error) {
	state, ok := object.(*session)
	if !ok || state == nil {
		return nil, errors.New("invalid workspace filesystem session")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	switch operation {
	case OperationStat:
		var request StatRequest
		if err := decodeExact(payload, &request); err != nil {
			return nil, errors.New("invalid workspace filesystem stat request")
		}
		_, relative, info, err := resolveExisting(state.root, request.Path)
		if err != nil {
			return nil, err
		}
		return artifact.Marshal(metadata(relative, info))
	case OperationRead:
		var request ReadRequest
		if err := decodeExact(payload, &request); err != nil || request.MaxBytes <= 0 || request.MaxBytes > state.limits.MaxReadBytes {
			return nil, errors.New("invalid workspace filesystem read request")
		}
		resolved, relative, info, err := resolveExisting(state.root, request.Path)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			return nil, failure(CodeIsDirectory, errors.New("workspace filesystem cannot read a directory"))
		}
		if info.Size() < 0 || info.Size() > int64(request.MaxBytes) {
			return nil, failure(CodeBudgetExceeded, errors.New("workspace filesystem file exceeds the read budget"))
		}
		file, err := os.Open(resolved)
		if err != nil {
			return nil, failure(CodeReadFailed, fmt.Errorf("open workspace file: %w", err))
		}
		defer file.Close()
		data, err := io.ReadAll(io.LimitReader(file, int64(request.MaxBytes)+1))
		if err != nil {
			return nil, failure(CodeReadFailed, fmt.Errorf("read workspace file: %w", err))
		}
		if len(data) > request.MaxBytes {
			return nil, failure(CodeBudgetExceeded, errors.New("workspace filesystem file exceeds the read budget"))
		}
		return artifact.Marshal(ReadResponse{Data: data, Metadata: metadata(relative, info)})
	case OperationReadRange:
		var request ReadRangeRequest
		if err := decodeExact(payload, &request); err != nil || request.Offset < 0 || request.Length <= 0 || request.Length > int64(state.limits.MaxChunkBytes) {
			return nil, errors.New("invalid workspace filesystem range request")
		}
		resolved, _, info, err := resolveExisting(state.root, request.Path)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			return nil, failure(CodeIsDirectory, errors.New("workspace filesystem cannot read a directory"))
		}
		if info.Size() < 0 || info.Size() > int64(state.limits.MaxReadBytes) || request.Offset > info.Size() || request.Length > info.Size()-request.Offset {
			return nil, failure(CodeBudgetExceeded, errors.New("workspace filesystem range exceeds the read budget"))
		}
		file, err := os.Open(resolved)
		if err != nil {
			return nil, failure(CodeReadFailed, fmt.Errorf("open workspace file range: %w", err))
		}
		defer file.Close()
		result := make([]byte, int(request.Length))
		if _, err := file.ReadAt(result, request.Offset); err != nil && !errors.Is(err, io.EOF) {
			return nil, failure(CodeReadFailed, fmt.Errorf("read workspace file range: %w", err))
		}
		return result, nil
	case OperationWriteAppend:
		if state.writer == nil || len(payload) == 0 || len(payload) > state.limits.MaxChunkBytes || state.writer.written > state.limits.MaxWriteBytes-len(payload) {
			return nil, failure(CodeBudgetExceeded, errors.New("workspace filesystem write exceeds its budget"))
		}
		if _, err := state.writer.file.Write(payload); err != nil {
			return nil, failure(CodeWriteFailed, fmt.Errorf("append workspace file: %w", err))
		}
		state.writer.written += len(payload)
		return nil, nil
	case OperationWriteCommit:
		if state.writer == nil || len(payload) != 0 {
			return nil, errors.New("invalid workspace filesystem write commit")
		}
		result, err := state.writer.commit()
		if err != nil {
			return nil, err
		}
		return artifact.Marshal(result)
	case OperationWriteCancel:
		if state.writer == nil || len(payload) != 0 {
			return nil, errors.New("invalid workspace filesystem write cancel")
		}
		return nil, state.writer.cancel()
	default:
		return nil, fmt.Errorf("workspace filesystem operation %q is unsupported", operation)
	}
}

func (p *Provider) Close(_ context.Context, object any) error {
	state, ok := object.(*session)
	if !ok {
		return errors.New("invalid workspace filesystem session")
	}
	if state.writer != nil && !state.writer.committed {
		return state.writer.cancel()
	}
	return nil
}

func requireOperations(operations []string) error {
	copyOf := append([]string(nil), operations...)
	sort.Strings(copyOf)
	if len(copyOf) == 0 || len(copyOf) > 3 {
		return errors.New("workspace filesystem requires bounded file operations")
	}
	if writeOperations(copyOf) {
		return nil
	}
	for index, operation := range copyOf {
		if operation != OperationRead && operation != OperationReadRange && operation != OperationStat || index > 0 && copyOf[index-1] == operation {
			return errors.New("workspace filesystem operation set is invalid")
		}
	}
	return nil
}

func writeOperations(operations []string) bool {
	copyOf := append([]string(nil), operations...)
	sort.Strings(copyOf)
	want := []string{OperationWriteAppend, OperationWriteCancel, OperationWriteCommit}
	for index := range want {
		if index >= len(copyOf) || copyOf[index] != want[index] {
			return false
		}
	}
	return len(copyOf) == len(want)
}

func openWriter(root string, config WriteConfig) (*writerState, error) {
	clean, err := cleanRelativePath(config.Path)
	if err != nil {
		return nil, err
	}
	parent := filepath.Join(root, filepath.Dir(clean))
	resolvedParent, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return nil, failure(CodeInvalidPath, fmt.Errorf("resolve workspace file parent: %w", err))
	}
	relativeParent, err := filepath.Rel(root, resolvedParent)
	if err != nil || relativeParent == ".." || strings.HasPrefix(relativeParent, ".."+string(filepath.Separator)) || filepath.IsAbs(relativeParent) {
		return nil, failure(CodeInvalidPath, errors.New("workspace file parent escapes its root"))
	}
	info, err := os.Stat(resolvedParent)
	if err != nil || !info.IsDir() {
		return nil, failure(CodeInvalidPath, errors.Join(errors.New("workspace file parent must be an existing directory"), err))
	}
	target := filepath.Join(resolvedParent, filepath.Base(clean))
	if existing, err := os.Lstat(target); err == nil {
		if existing.IsDir() || existing.Mode()&os.ModeSymlink != 0 || !config.Overwrite {
			return nil, failure(CodeWriteFailed, errors.New("workspace file already exists or is not replaceable"))
		}
	} else if !os.IsNotExist(err) {
		return nil, failure(CodeWriteFailed, fmt.Errorf("inspect workspace file destination: %w", err))
	}
	temporary, err := os.CreateTemp(resolvedParent, ".yotta-write-*")
	if err != nil {
		return nil, failure(CodeWriteFailed, fmt.Errorf("create workspace file temporary: %w", err))
	}
	relative, err := filepath.Rel(root, target)
	if err != nil {
		_ = temporary.Close()
		_ = os.Remove(temporary.Name())
		return nil, failure(CodeInvalidPath, err)
	}
	return &writerState{file: temporary, temporary: temporary.Name(), target: target, relative: filepath.ToSlash(relative), overwrite: config.Overwrite}, nil
}

func (w *writerState) commit() (Metadata, error) {
	if w.committed || w.file == nil {
		return Metadata{}, failure(CodeWriteFailed, errors.New("workspace file writer is already closed"))
	}
	if err := w.file.Sync(); err != nil {
		return Metadata{}, failure(CodeWriteFailed, fmt.Errorf("sync workspace file: %w", err))
	}
	if err := w.file.Close(); err != nil {
		return Metadata{}, failure(CodeWriteFailed, fmt.Errorf("close workspace file: %w", err))
	}
	w.file = nil
	publish := durablefs.PublishNew
	if w.overwrite {
		publish = durablefs.Replace
	}
	if err := publish(w.temporary, w.target); err != nil {
		if durablefs.Committed(err) {
			w.committed = true
		}
		return Metadata{}, failure(CodeWriteFailed, fmt.Errorf("publish workspace file: %w", err))
	}
	w.committed = true
	info, err := os.Stat(w.target)
	if err != nil {
		return Metadata{}, failure(CodeWriteFailed, fmt.Errorf("stat written workspace file: %w", err))
	}
	return metadata(w.relative, info), nil
}

func (w *writerState) cancel() error {
	if w.committed {
		return nil
	}
	var closeErr error
	if w.file != nil {
		closeErr = w.file.Close()
		w.file = nil
	}
	removeErr := os.Remove(w.temporary)
	if os.IsNotExist(removeErr) {
		removeErr = nil
	}
	return errors.Join(closeErr, removeErr)
}

func cleanRelativePath(path string) (string, error) {
	if path == "" || strings.TrimSpace(path) != path || !utf8.ValidString(path) || strings.ContainsRune(path, 0) || filepath.IsAbs(path) || filepath.VolumeName(path) != "" {
		return "", failure(CodeInvalidPath, errors.New("workspace file path must be a non-empty relative path"))
	}
	clean := filepath.Clean(filepath.FromSlash(path))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", failure(CodeInvalidPath, errors.New("workspace file path escapes its root"))
	}
	return clean, nil
}

func resolveExisting(root, path string) (string, string, os.FileInfo, error) {
	clean, err := cleanRelativePath(path)
	if err != nil {
		return "", "", nil, err
	}
	candidate := filepath.Join(root, clean)
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		code := CodeReadFailed
		if os.IsNotExist(err) {
			code = CodeNotFound
		}
		return "", "", nil, failure(code, fmt.Errorf("resolve workspace file: %w", err))
	}
	relative, err := filepath.Rel(root, resolved)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return "", "", nil, failure(CodeInvalidPath, errors.New("workspace file link escapes its root"))
	}
	info, err := os.Stat(resolved)
	if err != nil {
		code := CodeReadFailed
		if os.IsNotExist(err) {
			code = CodeNotFound
		}
		return "", "", nil, failure(code, fmt.Errorf("stat workspace file: %w", err))
	}
	return resolved, filepath.ToSlash(relative), info, nil
}

func failure(code string, cause error) error {
	return &Failure{Code: code, Cause: cause}
}

func metadata(relative string, info os.FileInfo) Metadata {
	extension := strings.ToLower(filepath.Ext(info.Name()))
	mediaType := mime.TypeByExtension(extension)
	if mediaType == "" {
		mediaType = "application/octet-stream"
	}
	return Metadata{
		Path: relative, Name: info.Name(), Extension: extension, MediaType: mediaType,
		Size: info.Size(), ModifiedUnixMillis: info.ModTime().UTC().UnixMilli(), IsDirectory: info.IsDir(),
	}
}

func decodeExact(raw []byte, destination any) error {
	if len(raw) == 0 || len(raw) > 4<<20 {
		return errors.New("JSON payload exceeds budget")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("JSON contains trailing values")
	}
	return nil
}
