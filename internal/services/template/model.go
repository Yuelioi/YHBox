// Package template 管理模板（PNG + meta）。
//
// v2 spec §1.5：top-level TemplateService 取消，改为 ContainerService（容器内 templates/）
// 和 LibraryService（库 templates/）各自托管。本包降级为基础库：PNG IO / 哈希 / meta 序列化。
package template

import "time"

// OriginKind 模板来源（v2 spec §1.5 / GPT 第三轮）。
// 解决 copy-on-use 后用户问"这张图哪来的"的溯源问题。
const (
	OriginKindScreenshot = "screenshot" // 本机 ScreenPicker 截的
	OriginKindLibrary    = "library"    // 从 library/templates/ 拖入
	OriginKindImported   = "imported"   // 从某个 library subgraph 的 staging 夹带过来
	OriginKindEmbedded   = "embedded"   // v1 预留，未来"嵌入到 subgraph json"用
)

// TemplateOrigin 来源追踪。新建空白时默认 Kind = screenshot, SourceID = ""。
type TemplateOrigin struct {
	Kind     string `json:"kind"`
	SourceID string `json:"sourceID,omitempty"`
}

// TemplateMeta 单个模板的元数据。key 由外层 map 持有。
// v2 在 v1 基础上加 Tags + Origin。
type TemplateMeta struct {
	Name               string         `json:"name"`
	Description        string         `json:"description,omitempty"`
	RecordedResolution [2]int         `json:"recordedResolution"`
	SHA256             string         `json:"sha256"`
	Width              int            `json:"width"`
	Height             int            `json:"height"`
	Region             [4]float32     `json:"region"`
	CreatedAt          time.Time      `json:"createdAt"`
	Tags               []string       `json:"tags,omitempty"` // 仅库 template 用；容器内 template 留空
	Origin             TemplateOrigin `json:"origin"`         // 必填；新截图默认 {screenshot,""}
}

// TemplateIndex 模板索引。容器内 templates 也用这个结构（每个容器一份索引）。
type TemplateIndex struct {
	SchemaVersion int                     `json:"schemaVersion"`
	Templates     map[string]TemplateMeta `json:"templates"`
}

// CurrentSchemaVersion 写入时填的值。
const CurrentSchemaVersion = 1
