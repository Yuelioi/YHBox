// internal/node/widget.go
package node

import "encoding/json"

// WidgetSpec UI widget 配置, 跟 InputSpec.Type/Semantic 解耦.
type WidgetSpec struct {
	Kind  string         `json:"kind,omitempty"`
	Props map[string]any `json:"props,omitempty"`
}

// 推荐: 每种 widget 提供 typed config struct, MarshalProps 转 map (避免裸 map typo).

type SliderProps struct {
	Min  float64 `json:"min"`
	Max  float64 `json:"max"`
	Step float64 `json:"step"`
}

type DropdownProps struct {
	Options  []EnumOption `json:"options"`
	Multiple bool         `json:"multiple,omitempty"`
}

type AsyncDropdownProps struct {
	AsyncSource string         `json:"asyncSource"`
	AsyncParams map[string]any `json:"asyncParams,omitempty"`
}

type JSONProps struct {
	Rows   int    `json:"rows,omitempty"`
	Schema string `json:"schema,omitempty"`
}

type TextareaProps struct {
	Rows      int `json:"rows,omitempty"`
	MaxLength int `json:"maxLength,omitempty"`
}

type RadioProps struct {
	Options []EnumOption `json:"options"`
}

type DurationProps struct {
	Unit string `json:"unit,omitempty"` // "ms" / "s" / "auto"
}

// MarshalProps typed struct → map[string]any (json 中转, 保字段名).
func MarshalProps(props any) map[string]any {
	b, _ := json.Marshal(props)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	return m
}
