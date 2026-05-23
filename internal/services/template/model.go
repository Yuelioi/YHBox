// internal/services/template/model.go
// Package template 管理模板（PNG + meta）。
//
// v2.2 schema: 目录-per-key, 每 key 含 _meta.json + N 个 variant (<WxH>.{png,json}).
package template

import "time"

type TemplateOrigin struct {
	Kind     string `json:"kind"`               // "user" | "imported" | "subgraph"
	SourceID string `json:"sourceID,omitempty"`
}

// KeyMeta 模板 key 级元数据 (跨 variant 共享). 文件 = <root>/<key>/_meta.json.
type KeyMeta struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	Origin      TemplateOrigin `json:"origin"`
}

// VariantMeta 单 variant 元数据 (跟 PNG 同分辨率). 文件 = <root>/<key>/<W>x<H>.json.
type VariantMeta struct {
	Resolution [2]int    `json:"resolution"` // [W, H], 录制时 frame size
	BBox       [4]int    `json:"bbox"`       // [x1, y1, x2, y2] 源帧像素位置. runtime → ratio+30px padding ROI (1:1 fish bot), GUI repaint 也用.
	SHA256     string    `json:"sha256"`
	Width      int       `json:"width"`     // bbox[2]-bbox[0]
	Height     int       `json:"height"`    // bbox[3]-bbox[1]
	CreatedAt  time.Time `json:"createdAt"`
	Note       string    `json:"note,omitempty"`
}

// TemplateMeta 兼容 shape, Service.List() 仍返这个给 FE. 内部从 KeyMeta + first VariantMeta 拼.
type TemplateMeta struct {
	Name               string         `json:"name"`
	Description        string         `json:"description,omitempty"`
	RecordedResolution [2]int         `json:"recordedResolution"`
	SHA256             string         `json:"sha256"`
	Width              int            `json:"width"`
	Height             int            `json:"height"`
	Region             [4]float32     `json:"region"`
	CreatedAt          time.Time      `json:"createdAt"`
	Tags               []string       `json:"tags,omitempty"`
	Origin             TemplateOrigin `json:"origin"`
	VariantCount       int            `json:"variantCount"` // 新加, FE 知道有几个 variant
}
