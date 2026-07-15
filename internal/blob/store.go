// Package blob owns Yotta 3.1 immutable, content-addressed binary values.
package blob

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/durablefs"
)

const (
	ownershipMarker = ".yotta-blob-store"
	markerContents  = "yotta/blob-store/3.1\n"
)

type Ref struct {
	MediaType string          `json:"mediaType"`
	Digest    artifact.Digest `json:"digest"`
	Size      int64           `json:"size"`
}

func (r Ref) Validate() error {
	if !r.Digest.Valid() || r.Size < 0 {
		return errors.New("invalid blob identity")
	}
	mediaType, params, err := mime.ParseMediaType(r.MediaType)
	if err != nil || len(params) != 0 || mediaType != r.MediaType || mediaType != strings.ToLower(mediaType) {
		return errors.New("blob media type must be canonical and parameter-free")
	}
	return nil
}

type Limits struct {
	MaxBlobBytes  int64
	MaxTotalBytes int64
}

type Store struct {
	mu     sync.RWMutex
	putMu  sync.Mutex
	root   string
	limits Limits
	total  int64
}

func Open(root string, limits Limits) (*Store, error) {
	if limits.MaxBlobBytes <= 0 || limits.MaxTotalBytes < limits.MaxBlobBytes {
		return nil, errors.New("invalid blob store limits")
	}
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("blob store root is required")
	}
	resolved, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(resolved, 0o700); err != nil {
		return nil, fmt.Errorf("create blob store: %w", err)
	}
	entries, err := os.ReadDir(resolved)
	if err != nil {
		return nil, err
	}
	markerPath := filepath.Join(resolved, ownershipMarker)
	if len(entries) == 0 {
		if err := durablefs.WriteFile(markerPath, []byte(markerContents), 0o600); err != nil {
			return nil, fmt.Errorf("claim blob store directory: %w", err)
		}
		entries, err = os.ReadDir(resolved)
		if err != nil {
			return nil, err
		}
	} else {
		marker, err := os.ReadFile(markerPath)
		if err != nil || string(marker) != markerContents {
			return nil, errors.New("blob store directory is not owned by Yotta 3.1")
		}
	}
	var total int64
	for _, entry := range entries {
		if entry.Name() == ownershipMarker {
			continue
		}
		if entry.IsDir() {
			continue
		}
		if strings.HasPrefix(entry.Name(), ".tmp-") {
			if err := durablefs.Remove(filepath.Join(resolved, entry.Name())); err != nil {
				return nil, fmt.Errorf("remove abandoned blob staging file: %w", err)
			}
			continue
		}
		if !validObjectName(entry.Name()) || entry.Type()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("blob store contains invalid object %q", entry.Name())
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		total += info.Size()
		if total > limits.MaxTotalBytes {
			return nil, errors.New("blob store exceeds total byte quota")
		}
	}
	return &Store{root: resolved, limits: limits, total: total}, nil
}

func (s *Store) Put(ctx context.Context, mediaType string, source io.Reader) (Ref, error) {
	probe := Ref{MediaType: mediaType, Digest: artifact.Digest("sha256:" + strings.Repeat("0", 64)), Size: 0}
	if err := probe.Validate(); err != nil {
		return Ref{}, err
	}
	if err := ctx.Err(); err != nil {
		return Ref{}, err
	}
	s.putMu.Lock()
	defer s.putMu.Unlock()
	tmp, err := os.CreateTemp(s.root, ".tmp-")
	if err != nil {
		return Ref{}, fmt.Errorf("create blob temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	hash := sha256.New()
	written, copyErr := copyBounded(ctx, io.MultiWriter(tmp, hash), source, s.limits.MaxBlobBytes)
	syncErr := tmp.Sync()
	closeErr := tmp.Close()
	if copyErr != nil {
		return Ref{}, copyErr
	}
	if syncErr != nil {
		return Ref{}, syncErr
	}
	if closeErr != nil {
		return Ref{}, closeErr
	}
	digest := artifact.Digest("sha256:" + hex.EncodeToString(hash.Sum(nil)))
	ref := Ref{MediaType: mediaType, Digest: digest, Size: written}

	s.mu.Lock()
	defer s.mu.Unlock()
	destination := filepath.Join(s.root, strings.TrimPrefix(digest.String(), "sha256:"))
	if info, err := os.Lstat(destination); err == nil {
		if !info.Mode().IsRegular() {
			return Ref{}, errors.New("existing blob object is not a regular file")
		}
		if info.Size() != written {
			return Ref{}, errors.New("existing blob size does not match its digest")
		}
		file, err := os.Open(destination)
		if err != nil {
			return Ref{}, err
		}
		verifyErr := verifyOpenFile(ctx, file, ref)
		closeErr := file.Close()
		if verifyErr != nil {
			return Ref{}, verifyErr
		}
		if closeErr != nil {
			return Ref{}, closeErr
		}
		return ref, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return Ref{}, err
	}
	if s.total+written > s.limits.MaxTotalBytes {
		return Ref{}, errors.New("blob store total byte quota exceeded")
	}
	if err := durablefs.Replace(tmpPath, destination); err != nil {
		if durablefs.Committed(err) {
			s.total += written
		}
		return Ref{}, fmt.Errorf("commit blob: %w", err)
	}
	s.total += written
	return ref, nil
}

func (s *Store) ReadRange(ctx context.Context, ref Ref, offset, length int64) ([]byte, error) {
	if err := ref.Validate(); err != nil {
		return nil, err
	}
	if offset < 0 || length < 0 || offset > ref.Size || length > ref.Size-offset {
		return nil, errors.New("blob range is outside the object")
	}
	if length > int64(int(^uint(0)>>1)) {
		return nil, errors.New("blob range exceeds addressable memory")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	path := filepath.Join(s.root, objectName(ref.Digest))
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open blob: %w", err)
	}
	defer file.Close()
	if err := verifyOpenFile(ctx, file, ref); err != nil {
		return nil, err
	}
	result := make([]byte, int(length))
	if _, err := file.ReadAt(result, offset); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	return result, nil
}

// Verify streams the full object through its size and digest checks without
// materializing its bytes.
func (s *Store) Verify(ctx context.Context, ref Ref) error {
	if err := ref.Validate(); err != nil {
		return err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	file, err := os.Open(filepath.Join(s.root, objectName(ref.Digest)))
	if err != nil {
		return fmt.Errorf("open blob: %w", err)
	}
	defer file.Close()
	return verifyOpenFile(ctx, file, ref)
}

// Sweep deletes committed objects that are not present in the complete live set.
// The caller must provide a stop-the-world snapshot of every durable reference.
func (s *Store) Sweep(live []Ref) (int, error) {
	liveObjects := make(map[string]struct{}, len(live))
	for _, ref := range live {
		if err := ref.Validate(); err != nil {
			return 0, fmt.Errorf("invalid live blob reference: %w", err)
		}
		liveObjects[objectName(ref.Digest)] = struct{}{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return 0, err
	}
	reclaimed := 0
	for _, entry := range entries {
		if entry.Name() == ownershipMarker {
			continue
		}
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".tmp-") {
			continue
		}
		if !validObjectName(entry.Name()) || entry.Type()&os.ModeSymlink != 0 {
			return reclaimed, fmt.Errorf("blob store contains invalid object %q", entry.Name())
		}
		if _, ok := liveObjects[entry.Name()]; ok {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return reclaimed, err
		}
		if err := durablefs.Remove(filepath.Join(s.root, entry.Name())); err != nil {
			if durablefs.Committed(err) {
				s.total -= info.Size()
				reclaimed++
			}
			return reclaimed, err
		}
		s.total -= info.Size()
		reclaimed++
	}
	return reclaimed, nil
}

func verifyOpenFile(ctx context.Context, file *os.File, ref Ref) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.Size() != ref.Size {
		return errors.New("blob size integrity check failed")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	hash := sha256.New()
	if _, err := copyBounded(ctx, hash, file, ref.Size); err != nil {
		return err
	}
	if got := "sha256:" + hex.EncodeToString(hash.Sum(nil)); got != ref.Digest.String() {
		return errors.New("blob digest integrity check failed")
	}
	return nil
}

func objectName(digest artifact.Digest) string {
	return strings.TrimPrefix(digest.String(), "sha256:")
}

func validObjectName(name string) bool {
	if len(name) != 64 || name != strings.ToLower(name) {
		return false
	}
	_, err := hex.DecodeString(name)
	return err == nil
}

func copyBounded(ctx context.Context, destination io.Writer, source io.Reader, maximum int64) (int64, error) {
	buffer := make([]byte, 64<<10)
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		read, readErr := source.Read(buffer)
		if read > 0 {
			if total+int64(read) > maximum {
				return total, errors.New("blob exceeds byte quota")
			}
			written, writeErr := destination.Write(buffer[:read])
			total += int64(written)
			if writeErr != nil {
				return total, writeErr
			}
			if written != read {
				return total, io.ErrShortWrite
			}
		}
		if errors.Is(readErr, io.EOF) {
			return total, nil
		}
		if readErr != nil {
			return total, readErr
		}
		if read == 0 {
			return total, io.ErrNoProgress
		}
	}
}
