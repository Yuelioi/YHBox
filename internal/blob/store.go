// Package blob owns immutable, content-addressed binary values.
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
	"sort"
	"strings"
	"sync"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/durablefs"
)

const (
	ownershipMarker = ".yotta-blob-store"
	LayoutVersion   = "1"
	markerContents  = "yotta/blob-store/" + LayoutVersion + "\n"
)

type BlobRef struct {
	MediaType string          `json:"mediaType" jsonschema:"required,minLength=3,maxLength=255,pattern=^[a-z0-9][a-z0-9!#$&^_.+-]+/[a-z0-9][a-z0-9!#$&^_.+-]+$"`
	Digest    artifact.Digest `json:"digest" jsonschema:"required,pattern=^sha256:[a-f0-9]{64}$"`
	Size      int64           `json:"size" jsonschema:"required,minimum=0"`
}

func (r BlobRef) Validate() error {
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
	mu        sync.RWMutex
	putPermit chan struct{}
	root      string
	limits    Limits
	total     int64
	pins      map[string]int
}

type blobPin struct {
	store  *Store
	object string
	once   sync.Once
}

type Retention interface{ Release() }

type Object struct {
	Digest artifact.Digest `json:"digest"`
	Size   int64           `json:"size"`
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
			return nil, errors.New("blob store directory has an unsupported ownership marker")
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
	permit := make(chan struct{}, 1)
	permit <- struct{}{}
	return &Store{root: resolved, limits: limits, total: total, putPermit: permit, pins: map[string]int{}}, nil
}

func (s *Store) Put(ctx context.Context, mediaType string, source io.Reader) (BlobRef, error) {
	ref, _, err := s.put(ctx, mediaType, source, false)
	return ref, err
}

func (s *Store) PutRetained(ctx context.Context, mediaType string, source io.Reader) (BlobRef, Retention, error) {
	return s.putPinned(ctx, mediaType, source)
}

func (s *Store) putPinned(ctx context.Context, mediaType string, source io.Reader) (BlobRef, *blobPin, error) {
	return s.put(ctx, mediaType, source, true)
}

func (s *Store) put(ctx context.Context, mediaType string, source io.Reader, pinResult bool) (BlobRef, *blobPin, error) {
	probe := BlobRef{MediaType: mediaType, Digest: artifact.Digest("sha256:" + strings.Repeat("0", 64)), Size: 0}
	if err := probe.Validate(); err != nil {
		return BlobRef{}, nil, err
	}
	if err := ctx.Err(); err != nil {
		return BlobRef{}, nil, err
	}
	select {
	case <-ctx.Done():
		return BlobRef{}, nil, ctx.Err()
	case <-s.putPermit:
	}
	defer func() { s.putPermit <- struct{}{} }()
	tmp, err := os.CreateTemp(s.root, ".tmp-")
	if err != nil {
		return BlobRef{}, nil, fmt.Errorf("create blob temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	hash := sha256.New()
	written, copyErr := copyBounded(ctx, io.MultiWriter(tmp, hash), source, s.limits.MaxBlobBytes)
	syncErr := tmp.Sync()
	closeErr := tmp.Close()
	if copyErr != nil {
		return BlobRef{}, nil, copyErr
	}
	if syncErr != nil {
		return BlobRef{}, nil, syncErr
	}
	if closeErr != nil {
		return BlobRef{}, nil, closeErr
	}
	digest := artifact.Digest("sha256:" + hex.EncodeToString(hash.Sum(nil)))
	ref := BlobRef{MediaType: mediaType, Digest: digest, Size: written}

	s.mu.Lock()
	defer s.mu.Unlock()
	destination := filepath.Join(s.root, strings.TrimPrefix(digest.String(), "sha256:"))
	if info, err := os.Lstat(destination); err == nil {
		if !info.Mode().IsRegular() {
			return BlobRef{}, nil, errors.New("existing blob object is not a regular file")
		}
		if info.Size() != written {
			return BlobRef{}, nil, errors.New("existing blob size does not match its digest")
		}
		file, err := os.Open(destination)
		if err != nil {
			return BlobRef{}, nil, err
		}
		verifyErr := verifyOpenFile(ctx, file, ref)
		closeErr := file.Close()
		if verifyErr != nil {
			return BlobRef{}, nil, verifyErr
		}
		if closeErr != nil {
			return BlobRef{}, nil, closeErr
		}
		return ref, s.pinLocked(ref, pinResult), nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return BlobRef{}, nil, err
	}
	if s.total+written > s.limits.MaxTotalBytes {
		return BlobRef{}, nil, errors.New("blob store total byte quota exceeded")
	}
	if err := durablefs.Replace(tmpPath, destination); err != nil {
		if durablefs.Committed(err) {
			s.total += written
		}
		return BlobRef{}, nil, fmt.Errorf("commit blob: %w", err)
	}
	s.total += written
	return ref, s.pinLocked(ref, pinResult), nil
}

func (s *Store) pinLocked(ref BlobRef, requested bool) *blobPin {
	if !requested {
		return nil
	}
	object := objectName(ref.Digest)
	s.pins[object]++
	return &blobPin{store: s, object: object}
}

func (p *blobPin) Release() {
	if p == nil || p.store == nil {
		return
	}
	p.once.Do(func() {
		p.store.mu.Lock()
		defer p.store.mu.Unlock()
		if p.store.pins[p.object] <= 1 {
			delete(p.store.pins, p.object)
			return
		}
		p.store.pins[p.object]--
	})
}

func (s *Store) Objects() ([]Object, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return nil, err
	}
	result := make([]Object, 0, len(entries))
	for _, entry := range entries {
		if entry.Name() == ownershipMarker || strings.HasPrefix(entry.Name(), ".tmp-") {
			continue
		}
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || !validObjectName(entry.Name()) {
			return nil, fmt.Errorf("blob store contains invalid object %q", entry.Name())
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		result = append(result, Object{Digest: artifact.Digest("sha256:" + entry.Name()), Size: info.Size()})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Digest < result[j].Digest })
	return result, nil
}

func (s *Store) ReadRange(ctx context.Context, ref BlobRef, offset, length int64) ([]byte, error) {
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
func (s *Store) Verify(ctx context.Context, ref BlobRef) error {
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

func (s *Store) WriteTo(ctx context.Context, ref BlobRef, destination io.Writer) error {
	if destination == nil {
		return errors.New("blob destination is required")
	}
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
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.Size() != ref.Size {
		return errors.New("blob size integrity check failed")
	}
	hash := sha256.New()
	written, err := copyBounded(ctx, io.MultiWriter(destination, hash), file, ref.Size)
	if err != nil {
		return err
	}
	if written != ref.Size || "sha256:"+hex.EncodeToString(hash.Sum(nil)) != ref.Digest.String() {
		return errors.New("blob digest integrity check failed")
	}
	return nil
}

// Sweep deletes committed objects that are not present in the complete live set.
// The caller must provide a stop-the-world snapshot of every durable reference.
func (s *Store) Sweep(live []BlobRef) (int, error) {
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
		if s.pins[entry.Name()] > 0 {
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

func verifyOpenFile(ctx context.Context, file *os.File, ref BlobRef) error {
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
