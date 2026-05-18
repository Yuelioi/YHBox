package library

import (
	"encoding/json"
	"fmt"
	"io/fs"
)

// BuiltinTemplateEntry 描述一张 builtin template 的 metadata.
type BuiltinTemplateEntry struct {
	File       string `json:"file"`
	Resolution [2]int `json:"resolution"`
	Bbox       [4]int `json:"bbox"` // [x1, y1, x2, y2]
	Note       string `json:"note,omitempty"`
}

// BuiltinTemplatesIndex 是 templates/index.json 解析结果.
type BuiltinTemplatesIndex struct {
	Templates map[string]BuiltinTemplateEntry `json:"templates"`
}

// LoadTemplatesIndex 从 builtin embedded FS 读 templates/index.json.
func LoadTemplatesIndex() (*BuiltinTemplatesIndex, error) {
	b, err := fs.ReadFile(builtinFS, "builtin_library/templates/index.json")
	if err != nil {
		return nil, fmt.Errorf("read templates index: %w", err)
	}
	var idx BuiltinTemplatesIndex
	if err := json.Unmarshal(b, &idx); err != nil {
		return nil, fmt.Errorf("parse templates index: %w", err)
	}
	return &idx, nil
}

// MustLoadTemplatesIndex panics on error. 适合 init / startup gate.
func MustLoadTemplatesIndex() *BuiltinTemplatesIndex {
	idx, err := LoadTemplatesIndex()
	if err != nil {
		panic(err)
	}
	return idx
}
