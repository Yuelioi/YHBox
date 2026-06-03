package main

import (
	"encoding/json"

	"github.com/google/uuid"

	"yotta/internal/catalog"
	"yotta/internal/services/container"
)

// listNodesJSON 返节点目录 JSON (catalog.BuildWithI18n 含展示文案, 已按 category→kind 稳定排序)。
func listNodesJSON() []byte {
	b, _ := json.MarshalIndent(catalog.BuildWithI18n(), "", "  ")
	return b
}

// SaveResult is returned by saveContainer on success.
type SaveResult struct {
	ID       string                      `json:"id"`
	Path     string                      `json:"path"`
	Warnings []container.ValidationError `json:"warnings,omitempty"`
}

// saveContainer: 全量校验 → 有 error 级拒存(返 []ValidationError JSON) → 否则自分配新 UUID、
// Store.Save 落盘, 返 {id, path, warnings}。warning 不阻塞。
func saveContainer(st *container.Store, raw []byte) (SaveResult, []byte) {
	var c container.Container
	if err := json.Unmarshal(raw, &c); err != nil {
		b, _ := json.MarshalIndent([]container.ValidationError{{
			Severity: container.SeverityError, Code: "INVALID_JSON",
			Params: map[string]any{"err": err.Error()},
		}}, "", "  ")
		return SaveResult{}, b
	}
	c.Normalize()
	all := container.ValidateContainer(&c)
	var warnings []container.ValidationError
	hasErr := false
	for _, e := range all {
		if e.Severity == container.SeverityError {
			hasErr = true
		} else {
			warnings = append(warnings, e)
		}
	}
	if hasErr {
		b, _ := json.MarshalIndent(all, "", "  ")
		return SaveResult{}, b
	}
	// 服务端分配 ID，忽略入参里的 id
	c.ID = uuid.NewString()
	if c.Graph.ID == "" {
		c.Graph.ID = uuid.NewString()
	}
	if err := st.Save(&c); err != nil {
		b, _ := json.MarshalIndent([]container.ValidationError{{
			Severity: container.SeverityError, Code: "SAVE_FAILED",
			Params: map[string]any{"err": err.Error()},
		}}, "", "  ")
		return SaveResult{}, b
	}
	return SaveResult{ID: c.ID, Path: c.ID + "/container.json", Warnings: warnings}, nil
}

// validateContainerJSON: 解析 → Normalize → ValidateContainer → 返 []ValidationError JSON
// (含 error+warning 两级)。第二返回值 = 是否含 error 级。
func validateContainerJSON(raw []byte) ([]byte, bool) {
	var c container.Container
	if err := json.Unmarshal(raw, &c); err != nil {
		errs := []container.ValidationError{{
			Severity: container.SeverityError, Code: "INVALID_JSON",
			Params: map[string]any{"err": err.Error()},
		}}
		b, _ := json.MarshalIndent(errs, "", "  ")
		return b, true
	}
	c.Normalize()
	errs := container.ValidateContainer(&c)
	hasErr := false
	for _, e := range errs {
		if e.Severity == container.SeverityError {
			hasErr = true
		}
	}
	b, _ := json.MarshalIndent(errs, "", "  ")
	return b, hasErr
}
