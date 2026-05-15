// Package template 管理全局模板库 data/assets/templates/。
//
// 模板按 path/name 命名空间组织：assets/templates/<path>/<name>.png + _index.json。
// 与之前 Phase 1 的 per-action 模板（actions/<id>/templates/）相比：
//   - 跨 Container 共享同一份模板
//   - 元数据集中在 _index.json
//   - 文件 key = path/name（斜杠分目录）
package template

import "time"

// TemplateMeta 单个模板的元数据。key 在 TemplateIndex.Templates 里。
//
// recordedResolution: 截图时屏幕分辨率，给将来的 auto-scale 用；v1 仅显示。
// region: 截图源 ROI（ratio xywh），UI 大图预览用做高亮框。
type TemplateMeta struct {
	Name               string     `json:"name"`
	Description        string     `json:"description,omitempty"`
	RecordedResolution [2]int     `json:"recordedResolution"`
	SHA256             string     `json:"sha256"`
	Width              int        `json:"width"`
	Height             int        `json:"height"`
	Region             [4]float32 `json:"region"`
	CreatedAt          time.Time  `json:"createdAt"`
}

// TemplateIndex 全部模板的索引。持久化到 data/assets/templates/_index.json。
type TemplateIndex struct {
	SchemaVersion int                     `json:"schemaVersion"`
	Templates     map[string]TemplateMeta `json:"templates"`
}

// CurrentSchemaVersion 写入时填的值。
const CurrentSchemaVersion = 1
