// internal/services/asset/model.go
package asset

import (
	"fmt"
	"time"

	"github.com/yottaapp/yotta/internal/blob"
)

const (
	KindTemplate = "template"
	KindClip     = "clip"
)

// RecordSchemaVersion is an exact persisted contract. Other versions are
// rejected; Yotta 3.1 does not carry a compatibility reader.
const RecordSchemaVersion = 2

// Origin 描述资产来源。
type Origin struct {
	Kind     string `json:"kind"` // "user" | "imported" | "subgraph"
	SourceID string `json:"sourceID,omitempty"`
}

// Variant 一个模板的单分辨率变体（像素级定位数据）。
type Variant struct {
	Resolution [2]int   `json:"resolution"`        // [W,H] 录制帧尺寸
	BBox       [4]int   `json:"bbox"`              // [x1,y1,x2,y2] 源帧像素位置
	Regions    [][4]int `json:"regions,omitempty"` // 多槽检测, 空=单 BBox
	Blob       blob.Ref `json:"blob"`
}

// AssetRecord 全局资产库的一条记录。
type AssetRecord struct {
	SchemaVersion int       `json:"schemaVersion"` // 写入时由 store 统一盖 RecordSchemaVersion
	GUID          string    `json:"guid"`
	Kind          string    `json:"kind"`                  // KindTemplate | KindClip
	Name          string    `json:"name"`                  // 可变显示标签, 可重名
	Description   string    `json:"description,omitempty"` // 库管理用; 创建侧填值留后续
	Category      string    `json:"category,omitempty"`    // 库分组用; 同子图 Category 语义
	Tags          []string  `json:"tags,omitempty"`
	Origin        Origin    `json:"origin"`
	Variants      []Variant `json:"variants,omitempty"` // 仅 template; 按 Resolution 唯一
	Blob          *blob.Ref `json:"blob,omitempty"`     // 仅 clip
	CreatedAt     time.Time `json:"createdAt"`
}

func (r AssetRecord) validate() error {
	if r.GUID == "" {
		return fmt.Errorf("asset GUID is required")
	}
	if kindDir(r.Kind) == "" {
		return fmt.Errorf("unknown asset kind %q", r.Kind)
	}
	for i, variant := range r.Variants {
		if err := variant.Blob.Validate(); err != nil {
			return fmt.Errorf("variant %d blob: %w", i, err)
		}
	}
	if r.Blob != nil {
		if err := r.Blob.Validate(); err != nil {
			return fmt.Errorf("clip blob: %w", err)
		}
	}
	return nil
}
