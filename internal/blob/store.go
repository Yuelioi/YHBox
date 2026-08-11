// Package blob owns immutable, content-addressed binary values.
package blob

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	stagingDirName  = ".staging"
	trashDirName    = ".trash"
	LayoutVersion   = "2"
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
	staging   string
	trash     string
	limits    Limits
	inventory Inventory
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

// Inventory is the durable object metadata seam. Production roots use the
// Content Catalog implementation so startup, quota, and GC never scan or hash
// the complete physical object tree.
type Inventory interface {
	Observe(context.Context, Object) error
	Forget(context.Context, artifact.Digest) error
	Objects(context.Context) ([]Object, error)
	Object(context.Context, artifact.Digest) (Object, bool, error)
	TotalBytes(context.Context) (int64, error)
}

type stagingIntent struct {
	Digest artifact.Digest `json:"digest"`
	Size   int64           `json:"size"`
}

func Open(root string, limits Limits, inventories ...Inventory) (*Store, error) {
	if limits.MaxBlobBytes <= 0 || limits.MaxTotalBytes < limits.MaxBlobBytes {
		return nil, errors.New("invalid blob store limits")
	}
	if len(inventories) > 1 || len(inventories) == 1 && inventories[0] == nil {
		return nil, errors.New("blob store accepts one non-nil inventory")
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
	for _, entry := range entries {
		if entry.Name() == ownershipMarker || entry.Name() == stagingDirName || entry.Name() == trashDirName {
			continue
		}
		if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || !validShardName(entry.Name()) {
			return nil, fmt.Errorf("blob store contains invalid top-level entry %q", entry.Name())
		}
		secondLevel, err := os.ReadDir(filepath.Join(resolved, entry.Name()))
		if err != nil {
			return nil, err
		}
		for _, second := range secondLevel {
			if !second.IsDir() || second.Type()&os.ModeSymlink != 0 || !validShardName(second.Name()) {
				return nil, fmt.Errorf("blob store contains invalid second-level shard %q", second.Name())
			}
		}
	}
	staging := filepath.Join(resolved, stagingDirName)
	trash := filepath.Join(resolved, trashDirName)
	if err := os.MkdirAll(staging, 0o700); err != nil {
		return nil, fmt.Errorf("create blob staging directory: %w", err)
	}
	if err := os.MkdirAll(trash, 0o700); err != nil {
		return nil, fmt.Errorf("create blob trash directory: %w", err)
	}
	var inventory Inventory
	if len(inventories) == 1 {
		inventory = inventories[0]
	}
	permit := make(chan struct{}, 1)
	permit <- struct{}{}
	store := &Store{
		root: resolved, staging: staging, trash: trash, limits: limits,
		inventory: inventory, putPermit: permit, pins: map[string]int{},
	}
	if err := store.recoverInterruptedOperations(context.Background()); err != nil {
		return nil, err
	}
	total, err := store.loadTotal(context.Background())
	if err != nil {
		return nil, err
	}
	if total > limits.MaxTotalBytes {
		return nil, errors.New("blob store exceeds total byte quota")
	}
	store.total = total
	return store, nil
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
	tmp, err := os.CreateTemp(s.staging, "put-*.data")
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
	if err := s.ensureObjectDirectory(digest); err != nil {
		return BlobRef{}, nil, err
	}
	destination := s.objectPath(digest)
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
		known, err := s.inventoryObject(ctx, digest)
		if err != nil {
			return BlobRef{}, nil, err
		}
		if !known {
			if s.total+written > s.limits.MaxTotalBytes {
				return BlobRef{}, nil, errors.New("blob store total byte quota exceeded")
			}
		}
		if err := s.observe(ctx, Object{Digest: digest, Size: written}); err != nil {
			return BlobRef{}, nil, err
		}
		if !known {
			s.total += written
		}
		return ref, s.pinLocked(ref, pinResult), nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return BlobRef{}, nil, err
	}
	if s.total+written > s.limits.MaxTotalBytes {
		return BlobRef{}, nil, errors.New("blob store total byte quota exceeded")
	}
	intentPath := strings.TrimSuffix(tmpPath, ".data") + ".json"
	intentRaw, err := json.Marshal(stagingIntent{Digest: digest, Size: written})
	if err != nil {
		return BlobRef{}, nil, err
	}
	if err := durablefs.WriteFile(intentPath, intentRaw, 0o600); err != nil {
		return BlobRef{}, nil, fmt.Errorf("write blob staging intent: %w", err)
	}
	if err := durablefs.Replace(tmpPath, destination); err != nil {
		return BlobRef{}, nil, fmt.Errorf("commit blob: %w", err)
	}
	if err := s.observe(ctx, Object{Digest: digest, Size: written}); err != nil {
		return BlobRef{}, nil, fmt.Errorf("index committed blob: %w", err)
	}
	s.total += written
	if err := durablefs.Remove(intentPath); err != nil {
		return BlobRef{}, nil, fmt.Errorf("finalize blob staging intent: %w", err)
	}
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
	if s.inventory != nil {
		return s.inventory.Objects(context.Background())
	}
	return s.scanObjects()
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
	path, err := s.checkedObjectPath(ref.Digest)
	if err != nil {
		return nil, err
	}
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
	path, err := s.checkedObjectPath(ref.Digest)
	if err != nil {
		return err
	}
	file, err := os.Open(path)
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
	path, err := s.checkedObjectPath(ref.Digest)
	if err != nil {
		return err
	}
	file, err := os.Open(path)
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
	objects, err := s.objectsLocked(context.Background())
	if err != nil {
		return 0, err
	}
	reclaimed := 0
	for _, object := range objects {
		name := objectName(object.Digest)
		if _, ok := liveObjects[name]; ok {
			continue
		}
		if s.pins[name] > 0 {
			continue
		}
		if err := s.trashObject(context.Background(), object); err != nil {
			return reclaimed, err
		}
		s.total -= object.Size
		reclaimed++
	}
	return reclaimed, nil
}

func (s *Store) loadTotal(ctx context.Context) (int64, error) {
	if s.inventory != nil {
		return s.inventory.TotalBytes(ctx)
	}
	objects, err := s.scanObjects()
	if err != nil {
		return 0, err
	}
	var total int64
	for _, object := range objects {
		total += object.Size
	}
	return total, nil
}

func (s *Store) inventoryObject(
	ctx context.Context,
	digest artifact.Digest,
) (bool, error) {
	if s.inventory == nil {
		return false, nil
	}
	_, found, err := s.inventory.Object(ctx, digest)
	return found, err
}

func (s *Store) observe(ctx context.Context, object Object) error {
	if s.inventory == nil {
		return nil
	}
	return s.inventory.Observe(ctx, object)
}

func (s *Store) objectsLocked(ctx context.Context) ([]Object, error) {
	if s.inventory != nil {
		return s.inventory.Objects(ctx)
	}
	return s.scanObjects()
}

func (s *Store) scanObjects() ([]Object, error) {
	firstLevel, err := os.ReadDir(s.root)
	if err != nil {
		return nil, err
	}
	var result []Object
	for _, first := range firstLevel {
		if first.Name() == ownershipMarker || first.Name() == stagingDirName || first.Name() == trashDirName {
			continue
		}
		if !first.IsDir() || first.Type()&os.ModeSymlink != 0 || !validShardName(first.Name()) {
			return nil, fmt.Errorf("blob store contains invalid shard %q", first.Name())
		}
		firstPath := filepath.Join(s.root, first.Name())
		secondLevel, err := os.ReadDir(firstPath)
		if err != nil {
			return nil, err
		}
		for _, second := range secondLevel {
			if !second.IsDir() || second.Type()&os.ModeSymlink != 0 || !validShardName(second.Name()) {
				return nil, fmt.Errorf("blob store contains invalid shard %q", second.Name())
			}
			secondPath := filepath.Join(firstPath, second.Name())
			entries, err := os.ReadDir(secondPath)
			if err != nil {
				return nil, err
			}
			for _, entry := range entries {
				if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 ||
					!validObjectName(entry.Name()) ||
					!strings.HasPrefix(entry.Name(), first.Name()+second.Name()) {
					return nil, fmt.Errorf("blob store contains invalid object %q", entry.Name())
				}
				info, err := entry.Info()
				if err != nil {
					return nil, err
				}
				result = append(result, Object{
					Digest: artifact.Digest("sha256:" + entry.Name()),
					Size:   info.Size(),
				})
			}
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Digest < result[j].Digest })
	return result, nil
}

func (s *Store) recoverInterruptedOperations(ctx context.Context) error {
	if err := s.recoverStaging(ctx); err != nil {
		return err
	}
	return s.recoverTrash(ctx)
}

func (s *Store) recoverStaging(ctx context.Context) error {
	entries, err := os.ReadDir(s.staging)
	if err != nil {
		return err
	}
	intents := make(map[string]struct{})
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("blob staging contains invalid entry %q", entry.Name())
		}
		if strings.HasSuffix(entry.Name(), ".json") {
			intents[strings.TrimSuffix(entry.Name(), ".json")] = struct{}{}
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".data") {
			return fmt.Errorf("blob staging contains invalid entry %q", entry.Name())
		}
	}
	for base := range intents {
		intentPath := filepath.Join(s.staging, base+".json")
		raw, err := os.ReadFile(intentPath)
		if err != nil {
			return err
		}
		var intent stagingIntent
		if err := json.Unmarshal(raw, &intent); err != nil ||
			!intent.Digest.Valid() || intent.Size < 0 {
			return fmt.Errorf("blob staging intent %q is invalid", filepath.Base(intentPath))
		}
		dataPath := filepath.Join(s.staging, base+".data")
		destination := s.objectPath(intent.Digest)
		if err := s.ensureObjectDirectory(intent.Digest); err != nil {
			return err
		}
		dataInfo, dataErr := os.Lstat(dataPath)
		destInfo, destErr := os.Lstat(destination)
		if dataErr != nil && !errors.Is(dataErr, os.ErrNotExist) {
			return dataErr
		}
		if destErr != nil && !errors.Is(destErr, os.ErrNotExist) {
			return destErr
		}
		if destErr != nil {
			if dataErr != nil {
				return fmt.Errorf("blob staging intent %q lost both staged and published bytes", base)
			}
			if !dataInfo.Mode().IsRegular() || dataInfo.Size() != intent.Size {
				return fmt.Errorf("blob staging data %q is invalid", base)
			}
			if err := verifyPath(ctx, dataPath, intent); err != nil {
				return err
			}
			if err := durablefs.Replace(dataPath, destination); err != nil {
				return fmt.Errorf("recover staged blob: %w", err)
			}
		} else {
			if !destInfo.Mode().IsRegular() || destInfo.Size() != intent.Size {
				return fmt.Errorf("published blob %s conflicts with staging intent", intent.Digest)
			}
			if err := verifyPath(ctx, destination, intent); err != nil {
				return err
			}
			if dataErr == nil {
				if err := durablefs.Remove(dataPath); err != nil {
					return err
				}
			}
		}
		if err := s.observe(ctx, Object(intent)); err != nil {
			return err
		}
		if err := durablefs.Remove(intentPath); err != nil {
			return err
		}
	}
	entries, err = os.ReadDir(s.staging)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".data") {
			continue
		}
		if err := durablefs.Remove(filepath.Join(s.staging, entry.Name())); err != nil {
			return fmt.Errorf("remove abandoned blob staging data: %w", err)
		}
	}
	return nil
}

func (s *Store) recoverTrash(ctx context.Context) error {
	entries, err := os.ReadDir(s.trash)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || !validObjectName(entry.Name()) {
			return fmt.Errorf("blob trash contains invalid object %q", entry.Name())
		}
		digest := artifact.Digest("sha256:" + entry.Name())
		trashPath := filepath.Join(s.trash, entry.Name())
		_, indexed, err := func() (Object, bool, error) {
			if s.inventory == nil {
				info, infoErr := entry.Info()
				if infoErr != nil {
					return Object{}, false, infoErr
				}
				return Object{Digest: digest, Size: info.Size()}, true, nil
			}
			return s.inventory.Object(ctx, digest)
		}()
		if err != nil {
			return err
		}
		if !indexed {
			if err := durablefs.Remove(trashPath); err != nil {
				return fmt.Errorf("finalize forgotten blob trash: %w", err)
			}
			continue
		}
		destination := s.objectPath(digest)
		if err := s.ensureObjectDirectory(digest); err != nil {
			return err
		}
		if _, err := os.Lstat(destination); errors.Is(err, os.ErrNotExist) {
			if err := durablefs.Replace(trashPath, destination); err != nil {
				return fmt.Errorf("restore indexed blob from trash: %w", err)
			}
		} else if err != nil {
			return err
		} else if err := durablefs.Remove(trashPath); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) trashObject(ctx context.Context, object Object) error {
	if !object.Digest.Valid() || object.Size < 0 {
		return errors.New("GC candidate identity is invalid")
	}
	source, err := s.checkedObjectPath(object.Digest)
	if err != nil {
		return err
	}
	info, err := os.Lstat(source)
	if err != nil {
		return fmt.Errorf("open GC candidate %s: %w", object.Digest, err)
	}
	if !info.Mode().IsRegular() || info.Size() != object.Size {
		return fmt.Errorf("GC candidate %s failed physical identity check", object.Digest)
	}
	trashPath := filepath.Join(s.trash, objectName(object.Digest))
	if _, err := os.Lstat(trashPath); err == nil {
		return fmt.Errorf("blob trash already contains %s", object.Digest)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := durablefs.Replace(source, trashPath); err != nil {
		return fmt.Errorf("move blob to trash: %w", err)
	}
	if s.inventory != nil {
		if err := s.inventory.Forget(ctx, object.Digest); err != nil {
			restoreErr := s.ensureObjectDirectory(object.Digest)
			if restoreErr == nil {
				restoreErr = durablefs.Replace(trashPath, source)
			}
			return errors.Join(err, restoreErr)
		}
	}
	if err := durablefs.Remove(trashPath); err != nil {
		return fmt.Errorf("finalize blob trash: %w", err)
	}
	return nil
}

func verifyPath(ctx context.Context, path string, intent stagingIntent) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return verifyOpenFile(ctx, file, BlobRef{
		MediaType: "application/octet-stream",
		Digest:    intent.Digest,
		Size:      intent.Size,
	})
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

func (s *Store) objectPath(digest artifact.Digest) string {
	name := objectName(digest)
	if len(name) != 64 {
		return filepath.Join(s.root, "invalid")
	}
	return filepath.Join(s.root, name[:2], name[2:4], name)
}

func (s *Store) ensureObjectDirectory(digest artifact.Digest) error {
	name := objectName(digest)
	if len(name) != 64 {
		return errors.New("blob object digest is invalid")
	}
	for _, path := range []string{
		filepath.Join(s.root, name[:2]),
		filepath.Join(s.root, name[:2], name[2:4]),
	} {
		if err := os.Mkdir(path, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("create blob object shard: %w", err)
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("blob object shard %q is not a real directory", filepath.Base(path))
		}
	}
	return nil
}

func (s *Store) checkedObjectPath(digest artifact.Digest) (string, error) {
	name := objectName(digest)
	if len(name) != 64 {
		return "", errors.New("blob object digest is invalid")
	}
	for _, path := range []string{
		filepath.Join(s.root, name[:2]),
		filepath.Join(s.root, name[:2], name[2:4]),
	} {
		info, err := os.Lstat(path)
		if err != nil {
			return "", fmt.Errorf("inspect blob object shard: %w", err)
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("blob object shard %q is not a real directory", filepath.Base(path))
		}
	}
	return s.objectPath(digest), nil
}

func validShardName(name string) bool {
	if len(name) != 2 || name != strings.ToLower(name) {
		return false
	}
	_, err := hex.DecodeString(name)
	return err == nil
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
