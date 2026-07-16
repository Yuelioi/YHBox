// Package dependency 定义统一的依赖描述类型, 给 scanner 跟 export/import 用.
package dependency

// Kind 依赖种类. 加新 kind 时只加这里 + 各 node spec 自己声明 extractor, 不动 scanner.
type Kind string

const (
	KindTemplate Kind = "template"
	KindSubgraph Kind = "subgraph"
	// 未来扩: KindOCRModel, KindSoundProfile, KindScript ...
)

// Dependency 单个依赖项. Kind + Key 唯一标识.
type Dependency struct {
	Kind Kind   `json:"kind"`
	Key  string `json:"key"`
}

// String 返 "kind:key" 用于 dedup 跟 log.
func (d Dependency) String() string { return string(d.Kind) + ":" + d.Key }
