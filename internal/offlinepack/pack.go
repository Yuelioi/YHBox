// Package offlinepack owns the transport container for one exact Installation
// Plan and its original release artifact bytes. The container adds no trust,
// signature, installation, configuration, or execution authority.
package offlinepack

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/durablefs"
	"github.com/yottaapp/yotta/internal/installationplan"
)

const (
	Extension        = ".yotta-offline-pack"
	PlanPath         = "installation-plan.json"
	maxPlanBytes     = int64(8 << 20)
	maxArchiveBytes  = int64(4 << 30)
	maxArtifactCount = 4097
	privateFileMode  = 0o600
	encryptionFlag   = 1 << 0
)

type ArtifactSource func(
	context.Context,
	installationplan.ArtifactDescriptor,
) (io.ReadCloser, error)

func Write(
	ctx context.Context,
	destination string,
	plan installationplan.Plan,
	open ArtifactSource,
) error {
	if ctx == nil || !plan.Valid() || open == nil || strings.TrimSpace(destination) == "" {
		return errors.New("offline pack write request is invalid")
	}
	artifacts, err := planArtifacts(plan)
	if err != nil {
		return err
	}
	destination, err = filepath.Abs(destination)
	if err != nil {
		return err
	}
	directory := filepath.Dir(destination)
	info, err := os.Stat(directory)
	if err != nil || !info.IsDir() {
		return errors.New("offline pack destination directory does not exist")
	}
	if current, statErr := os.Lstat(destination); statErr == nil &&
		(!current.Mode().IsRegular() || current.Mode()&os.ModeSymlink != 0) {
		return errors.New("offline pack destination is not a regular file")
	} else if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return statErr
	}
	temp, err := os.CreateTemp(directory, ".yotta-offline-pack-")
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
	if err := writeBytes(archive, PlanPath, plan.Bytes()); err != nil {
		_ = archive.Close()
		_ = temp.Close()
		return err
	}
	for _, descriptor := range artifacts {
		if err := ctx.Err(); err != nil {
			_ = archive.Close()
			_ = temp.Close()
			return err
		}
		reader, err := open(ctx, descriptor)
		if err != nil || reader == nil {
			_ = archive.Close()
			_ = temp.Close()
			if err == nil {
				err = errors.New("artifact source returned a nil reader")
			}
			return fmt.Errorf("open offline artifact %s: %w", descriptor.Digest, err)
		}
		writer, createErr := archive.CreateHeader(archiveHeader(artifactPath(descriptor.Digest)))
		if createErr != nil {
			_ = reader.Close()
			_ = archive.Close()
			_ = temp.Close()
			return createErr
		}
		hash := sha256.New()
		written, copyErr := io.Copy(io.MultiWriter(writer, hash), io.LimitReader(reader, descriptor.Size+1))
		closeErr := reader.Close()
		if copyErr != nil || closeErr != nil || written != descriptor.Size ||
			rawDigest(hash.Sum(nil)) != descriptor.Digest {
			_ = archive.Close()
			_ = temp.Close()
			return fmt.Errorf("offline artifact %s changed identity", descriptor.Digest)
		}
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
		return fmt.Errorf("publish offline pack: %w", err)
	}
	return nil
}

func Inspect(ctx context.Context, archivePath string) (installationplan.Plan, error) {
	if ctx == nil || strings.TrimSpace(archivePath) == "" {
		return installationplan.Plan{}, errors.New("offline pack inspect request is invalid")
	}
	file, err := os.Open(archivePath)
	if err != nil {
		return installationplan.Plan{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxArchiveBytes {
		return installationplan.Plan{}, errors.New("offline pack archive size is invalid")
	}
	reader, err := zip.NewReader(file, info.Size())
	if err != nil {
		return installationplan.Plan{}, fmt.Errorf("open offline pack: %w", err)
	}
	entries, err := indexEntries(reader.File)
	if err != nil {
		return installationplan.Plan{}, err
	}
	planEntry, found := entries[PlanPath]
	if !found {
		return installationplan.Plan{}, errors.New("offline pack Installation Plan is missing")
	}
	planBytes, err := readEntry(ctx, planEntry, maxPlanBytes)
	if err != nil {
		return installationplan.Plan{}, err
	}
	plan, err := installationplan.Open(planBytes)
	if err != nil {
		return installationplan.Plan{}, err
	}
	artifacts, err := planArtifacts(plan)
	if err != nil {
		return installationplan.Plan{}, err
	}
	if len(entries) != len(artifacts)+1 {
		return installationplan.Plan{}, errors.New("offline pack contains undeclared or missing artifacts")
	}
	for _, descriptor := range artifacts {
		entry, found := entries[artifactPath(descriptor.Digest)]
		if !found || entry.UncompressedSize64 != uint64(descriptor.Size) {
			return installationplan.Plan{}, fmt.Errorf("offline artifact %s is missing or has the wrong size", descriptor.Digest)
		}
		if err := verifyEntry(ctx, entry, descriptor); err != nil {
			return installationplan.Plan{}, err
		}
	}
	return plan, nil
}

func planArtifacts(plan installationplan.Plan) ([]installationplan.ArtifactDescriptor, error) {
	draft := plan.Machine()
	descriptors := make([]installationplan.ArtifactDescriptor, 0, len(draft.Packages)+1)
	descriptors = append(descriptors, draft.Workflow.Artifact)
	for _, packageRelease := range draft.Packages {
		descriptors = append(descriptors, packageRelease.Artifact)
	}
	sort.Slice(descriptors, func(i, j int) bool { return descriptors[i].Digest < descriptors[j].Digest })
	unique := descriptors[:0]
	var total int64
	for _, descriptor := range descriptors {
		if len(unique) != 0 && unique[len(unique)-1].Digest == descriptor.Digest {
			previous := unique[len(unique)-1]
			if previous.Size != descriptor.Size || previous.MediaType != descriptor.MediaType {
				return nil, errors.New("offline pack plan has a conflicting artifact digest")
			}
			continue
		}
		if descriptor.Size > maxArchiveBytes-maxPlanBytes-total {
			return nil, errors.New("offline pack expanded bytes exceed their budget")
		}
		total += descriptor.Size
		unique = append(unique, descriptor)
	}
	if len(unique) == 0 || len(unique) > maxArtifactCount {
		return nil, errors.New("offline pack artifact count is invalid")
	}
	return unique, nil
}

func indexEntries(files []*zip.File) (map[string]*zip.File, error) {
	if len(files) < 2 || len(files) > maxArtifactCount+1 {
		return nil, errors.New("offline pack entry count is invalid")
	}
	entries := make(map[string]*zip.File, len(files))
	var expanded uint64
	for _, entry := range files {
		name := entry.Name
		if name == "" || name == ".." || strings.HasPrefix(name, "../") ||
			strings.Contains(name, "\\") || strings.HasPrefix(name, "/") ||
			path.Clean(name) != name || strings.Contains(name, ":") ||
			entry.FileInfo().IsDir() || entry.Mode()&os.ModeSymlink != 0 ||
			entry.Flags&encryptionFlag != 0 {
			return nil, fmt.Errorf("offline pack contains unsafe entry %q", name)
		}
		if _, exists := entries[name]; exists {
			return nil, fmt.Errorf("offline pack contains duplicate entry %q", name)
		}
		if entry.UncompressedSize64 > uint64(maxArchiveBytes)-expanded {
			return nil, errors.New("offline pack expanded bytes exceed their budget")
		}
		expanded += entry.UncompressedSize64
		entries[name] = entry
	}
	return entries, nil
}

func readEntry(ctx context.Context, entry *zip.File, maximum int64) ([]byte, error) {
	if entry.UncompressedSize64 > uint64(maximum) {
		return nil, fmt.Errorf("offline pack entry %q exceeds its byte budget", entry.Name)
	}
	reader, err := entry.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	raw, err := io.ReadAll(io.LimitReader(&contextReader{ctx: ctx, reader: reader}, maximum+1))
	if err != nil || int64(len(raw)) > maximum || uint64(len(raw)) != entry.UncompressedSize64 {
		return nil, fmt.Errorf("offline pack entry %q size is invalid", entry.Name)
	}
	return raw, nil
}

func verifyEntry(
	ctx context.Context,
	entry *zip.File,
	descriptor installationplan.ArtifactDescriptor,
) error {
	reader, err := entry.Open()
	if err != nil {
		return err
	}
	defer reader.Close()
	hash := sha256.New()
	written, err := io.Copy(hash, &contextReader{ctx: ctx, reader: reader})
	if err != nil || written != descriptor.Size || rawDigest(hash.Sum(nil)) != descriptor.Digest {
		return fmt.Errorf("offline artifact %s failed its integrity check", descriptor.Digest)
	}
	return nil
}

func writeBytes(archive *zip.Writer, name string, raw []byte) error {
	writer, err := archive.CreateHeader(archiveHeader(name))
	if err != nil {
		return err
	}
	_, err = writer.Write(raw)
	return err
}

func archiveHeader(name string) *zip.FileHeader {
	header := &zip.FileHeader{Name: name, Method: zip.Deflate}
	header.SetMode(privateFileMode)
	header.Modified = time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)
	return header
}

func artifactPath(digest artifact.Digest) string {
	return "artifacts/" + strings.TrimPrefix(digest.String(), "sha256:")
}

func rawDigest(sum []byte) artifact.Digest {
	return artifact.Digest("sha256:" + hex.EncodeToString(sum))
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
