// internal/services/asset/blobstore.go
package asset

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// BlobStore 内容寻址字节池。
// 布局：<root>/blobs/<sha256hex>（裸 sha，无扩展名）。
type BlobStore struct {
	root string // 指向 <assetRoot>/blobs/
}

// NewBlobStore 初始化 blob 池目录并返回 BlobStore。
func NewBlobStore(root string) (*BlobStore, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("blob mkdir %s: %w", root, err)
	}
	return &BlobStore{root: root}, nil
}

// Put 把 data 写入 blob 池，返回其 sha256 hex。幂等：同内容只存一份。
func (b *BlobStore) Put(data []byte) (string, error) {
	h := sha256.Sum256(data)
	sha := hex.EncodeToString(h[:])

	dst := filepath.Join(b.root, sha)
	// 已存在直接返回（内容寻址，同 sha = 同内容）。
	if _, err := os.Stat(dst); err == nil {
		return sha, nil
	}

	tmp := dst + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return "", fmt.Errorf("blob write tmp: %w", err)
	}
	if err := os.Rename(tmp, dst); err != nil {
		// rename 目标已存在（并发竞争）视为成功。
		if errors.Is(err, os.ErrExist) {
			_ = os.Remove(tmp)
			return sha, nil
		}
		// Windows: Rename 在目标已存在时会返回 ERROR_ALREADY_EXISTS，
		// 而不是 os.ErrExist，需要额外检查目标是否已落盘。
		if _, statErr := os.Stat(dst); statErr == nil {
			_ = os.Remove(tmp)
			return sha, nil
		}
		return "", fmt.Errorf("blob rename: %w", err)
	}
	return sha, nil
}

// Has 报告 blob 是否存在。
func (b *BlobStore) Has(sha string) bool {
	_, err := os.Stat(filepath.Join(b.root, sha))
	return err == nil
}

// Read 返回 blob 的原始字节。
func (b *BlobStore) Read(sha string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Join(b.root, sha))
	if err != nil {
		return nil, fmt.Errorf("blob read %s: %w", sha, err)
	}
	return data, nil
}
