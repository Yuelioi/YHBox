package mcpserver

import (
	"encoding/json"

	"github.com/google/uuid"

	"github.com/yottaapp/yotta/internal/catalog"
	"github.com/yottaapp/yotta/internal/node"
	"github.com/yottaapp/yotta/internal/services/container"
)

func listNodesJSONWithRegistry(registry node.RegistryReader) []byte {
	b, _ := json.MarshalIndent(catalog.BuildWithI18nFrom(registry), "", "  ")
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
	return saveContainerWithRegistry(st, node.DefaultRegistrySnapshot(), raw)
}

func saveContainerWithRegistry(st *container.Store, registry node.RegistryReader, raw []byte) (SaveResult, []byte) {
	var c container.Container
	if err := json.Unmarshal(raw, &c); err != nil {
		b, _ := json.MarshalIndent([]container.ValidationError{{
			Severity: container.SeverityError, Code: "INVALID_JSON",
			Params: map[string]any{"err": err.Error()},
		}}, "", "  ")
		return SaveResult{}, b
	}
	c.Normalize()
	// MCP 工具语境无全局子图池 — 引用子图会报 MISSING_SUBGRAPH (已知限制, 真需要再接池).
	all := container.ValidateContainerWithRegistry(&c, nil, registry)
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
	return SaveResult{ID: c.ID, Path: c.ID + "/", Warnings: warnings}, nil
}

func validateContainerJSONWithRegistry(registry node.RegistryReader, raw []byte) ([]byte, bool) {
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
	errs := container.ValidateContainerWithRegistry(&c, nil, registry)
	hasErr := false
	for _, e := range errs {
		if e.Severity == container.SeverityError {
			hasErr = true
		}
	}
	b, _ := json.MarshalIndent(errs, "", "  ")
	return b, hasErr
}
