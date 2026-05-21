// Package template 管理模板（PNG + meta）。
//
// Top-level TemplateService 取消, 改为 ContainerService (容器内 templates/) 和
// LibraryService (库 templates/) 各自托管. 本包降级为基础库: PNG IO / 哈希 / meta 序列化.
package template

import "time"

type TemplateOrigin struct {
	Kind     string `json:"kind"`
	SourceID string `json:"sourceID,omitempty"`
}

// TemplateMeta 单模板的元数据. 跟 PNG 同 key, 文件 = <key>.json.
type TemplateMeta struct {
	Name               string         `json:"name"`
	Description        string         `json:"description,omitempty"`
	RecordedResolution [2]int         `json:"recordedResolution"`
	SHA256             string         `json:"sha256"`
	Width              int            `json:"width"`
	Height             int            `json:"height"`
	Region             [4]float32     `json:"region"`
	Regions            [][4]float32   `json:"regions,omitempty"` // multi-slot
	CreatedAt          time.Time      `json:"createdAt"`
	Tags               []string       `json:"tags,omitempty"`
	Origin             TemplateOrigin `json:"origin"`
}

// TemplateIndex / CurrentSchemaVersion 删除 — 新 schema 每模板独立 JSON 文件, 无集中索引
