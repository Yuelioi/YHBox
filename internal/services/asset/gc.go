// internal/services/asset/gc.go
package asset

import "github.com/yottaapp/yotta/internal/blob"

// Referrer 一处引用某资产 GUID 的位置。由 wire 层（有 container/dependency 访问权）填充。
type Referrer struct {
	ContainerID string `json:"containerID"`
	SubgraphID  string `json:"subgraphID,omitempty"`
	NodeID      string `json:"nodeID"`
	NodeKind    string `json:"nodeKind"`
}

// GCBlobs gives the blob store a complete live-reference snapshot. Asset code
// never reaches into the object directory or deletes store-owned files.
func (s *Store) GCBlobs() (reclaimed int, err error) {
	s.blobLifecycle.Lock()
	defer s.blobLifecycle.Unlock()
	s.mu.RLock()
	live := make([]blob.BlobRef, 0)
	for _, rec := range s.recs {
		for _, v := range rec.Variants {
			live = append(live, v.Blob)
		}
		if rec.Blob != nil {
			live = append(live, *rec.Blob)
		}
	}
	s.mu.RUnlock()
	return s.blobs.Sweep(live)
}
