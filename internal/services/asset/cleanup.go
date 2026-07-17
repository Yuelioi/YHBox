package asset

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
)

var ErrCleanupPreviewStale = errors.New("asset cleanup preview is stale")

type DurableReferenceSource interface {
	WithDurableBlobReferences(context.Context, func([]blob.BlobRef) error) error
}

type CleanupPreview struct {
	Token          artifact.Digest `json:"token"`
	CandidateCount int             `json:"candidateCount"`
	CandidateBytes int64           `json:"candidateBytes"`
	LiveCount      int             `json:"liveCount"`
	ObjectCount    int             `json:"objectCount"`
}

type CleanupResult struct {
	Token     artifact.Digest `json:"token"`
	Reclaimed int             `json:"reclaimed"`
}

type cleanupSnapshot struct {
	Token   artifact.Digest
	Live    []blob.BlobRef
	Preview CleanupPreview
}

func (s *Service) PreviewCleanup() (CleanupPreview, error) {
	var preview CleanupPreview
	err := s.withCleanupSnapshot(context.Background(), func(snapshot cleanupSnapshot) error {
		preview = snapshot.Preview
		return nil
	})
	return preview, err
}

func (s *Service) CommitCleanup(expected artifact.Digest) (CleanupResult, error) {
	if !expected.Valid() {
		return CleanupResult{}, errors.New("asset cleanup requires a valid preview token")
	}
	var result CleanupResult
	err := s.withCleanupSnapshot(context.Background(), func(snapshot cleanupSnapshot) error {
		if snapshot.Token != expected {
			return ErrCleanupPreviewStale
		}
		reclaimed, err := s.store.blobs.Sweep(snapshot.Live)
		if err != nil {
			return err
		}
		result = CleanupResult{Token: snapshot.Token, Reclaimed: reclaimed}
		return nil
	})
	return result, err
}

func (s *Service) withCleanupSnapshot(ctx context.Context, visit func(cleanupSnapshot) error) error {
	if s.store == nil || visit == nil {
		return errors.New("asset cleanup requires store and visitor")
	}
	s.store.gcMu.Lock()
	defer s.store.gcMu.Unlock()
	assetRefs := s.store.assetBlobReferences()
	complete := func(external []blob.BlobRef) error {
		live, err := normalizeLiveReferences(append(assetRefs, external...))
		if err != nil {
			return err
		}
		objects, err := s.store.blobs.Objects()
		if err != nil {
			return err
		}
		snapshot, err := buildCleanupSnapshot(live, objects)
		if err != nil {
			return err
		}
		return visit(snapshot)
	}
	if s.references == nil {
		return complete(nil)
	}
	return s.references.WithDurableBlobReferences(ctx, complete)
}

func (s *Store) assetBlobReferences() []blob.BlobRef {
	s.mu.RLock()
	defer s.mu.RUnlock()
	refs := make([]blob.BlobRef, 0)
	for _, record := range s.recs {
		if record.Blob != nil {
			refs = append(refs, *record.Blob)
		}
		for _, variant := range record.Variants {
			refs = append(refs, variant.Blob)
		}
	}
	return refs
}

func normalizeLiveReferences(source []blob.BlobRef) ([]blob.BlobRef, error) {
	byDigest := make(map[artifact.Digest]blob.BlobRef, len(source))
	for _, ref := range source {
		if err := ref.Validate(); err != nil {
			return nil, err
		}
		if previous, ok := byDigest[ref.Digest]; ok && previous.Size != ref.Size {
			return nil, fmt.Errorf("blob %s has conflicting durable sizes", ref.Digest)
		}
		if _, exists := byDigest[ref.Digest]; !exists {
			byDigest[ref.Digest] = ref
		}
	}
	result := make([]blob.BlobRef, 0, len(byDigest))
	for _, ref := range byDigest {
		result = append(result, ref)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Digest < result[j].Digest })
	return result, nil
}

func buildCleanupSnapshot(live []blob.BlobRef, objects []blob.Object) (cleanupSnapshot, error) {
	document := struct {
		Live    []blob.BlobRef `json:"live"`
		Objects []blob.Object  `json:"objects"`
	}{Live: live, Objects: objects}
	raw, err := artifact.Marshal(document)
	if err != nil {
		return cleanupSnapshot{}, err
	}
	token, err := artifact.Sum("yotta/blob-cleanup-preview/v1", raw)
	if err != nil {
		return cleanupSnapshot{}, err
	}
	liveDigests := make(map[artifact.Digest]struct{}, len(live))
	for _, ref := range live {
		liveDigests[ref.Digest] = struct{}{}
	}
	preview := CleanupPreview{Token: token, LiveCount: len(live), ObjectCount: len(objects)}
	for _, object := range objects {
		if _, ok := liveDigests[object.Digest]; ok {
			continue
		}
		preview.CandidateCount++
		preview.CandidateBytes += object.Size
	}
	return cleanupSnapshot{Token: token, Live: live, Preview: preview}, nil
}
