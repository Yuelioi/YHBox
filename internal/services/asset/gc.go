// internal/services/asset/gc.go
package asset

import (
	"os"
	"path/filepath"
	"strings"
)

// Referrer 一处引用某资产 GUID 的位置。由 wire 层（有 container/dependency 访问权）填充。
type Referrer struct {
	ContainerID string `json:"containerID"`
	SubgraphID  string `json:"subgraphID,omitempty"`
	NodeID      string `json:"nodeID"`
	NodeKind    string `json:"nodeKind"`
}

// GCBlobs 删除 blobs/ 目录中不再被任何记录引用的孤立 blob 文件。
// live set = 所有记录 Variants[].Blob ∪ clip 记录 Blob（非空）。
// 跳过 .tmp 残留文件，返回回收数量。锁内计算 live set，锁释放后执行删除。
func (s *Store) GCBlobs() (reclaimed int, err error) {
	// 锁内快照 live set。
	s.mu.RLock()
	live := make(map[string]struct{})
	for _, rec := range s.recs {
		for _, v := range rec.Variants {
			if v.Blob != "" {
				live[v.Blob] = struct{}{}
			}
		}
		if rec.Blob != "" {
			live[rec.Blob] = struct{}{}
		}
	}
	blobDir := s.blobs.root
	s.mu.RUnlock()

	// 扫 blobs/ 目录，删不在 live set 的文件（跳过 .tmp）。
	entries, err := os.ReadDir(blobDir)
	if err != nil {
		return 0, err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".tmp") {
			continue
		}
		if _, ok := live[name]; !ok {
			if rmErr := os.Remove(filepath.Join(blobDir, name)); rmErr == nil {
				reclaimed++
			}
		}
	}
	return reclaimed, nil
}
