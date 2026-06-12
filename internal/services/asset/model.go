// internal/services/asset/model.go
package asset

import "time"

const (
	KindTemplate = "template"
	KindClip     = "clip"
)

// RecordSchemaVersion 资产记录文件格式版本。读取契约: 磁盘上版本 > 此值 → 拒载报错
// (版本号不是装饰品); 写入时统一盖当前值。演进时在此处升版 + 写一次性升级器。
const RecordSchemaVersion = 1

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
	Blob       string   `json:"blob"`              // 像素 PNG 的 sha256
}

// AssetRecord 全局资产库的一条记录。
type AssetRecord struct {
	SchemaVersion int       `json:"schemaVersion"` // 写入时由 store 统一盖 RecordSchemaVersion
	GUID          string    `json:"guid"`
	Kind          string    `json:"kind"` // KindTemplate | KindClip
	Name          string    `json:"name"` // 可变显示标签, 可重名
	Description   string    `json:"description,omitempty"` // 库管理用; 创建侧填值留后续
	Category      string    `json:"category,omitempty"`    // 库分组用; 同子图 Category 语义
	Tags          []string  `json:"tags,omitempty"`
	Origin        Origin    `json:"origin"`
	Variants      []Variant `json:"variants,omitempty"` // 仅 template; 按 Resolution 唯一
	Blob          string    `json:"blob,omitempty"`     // 仅 clip: 事件流序列化字节的 sha256
	CreatedAt     time.Time `json:"createdAt"`
}
