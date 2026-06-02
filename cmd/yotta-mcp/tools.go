package main

import (
	"encoding/json"

	"yotta/internal/catalog"
	"yotta/internal/services/container"
)

// listNodesJSON 返节点目录 JSON (catalog.Build 已按 category→kind 稳定排序)。
func listNodesJSON() []byte {
	b, _ := json.MarshalIndent(catalog.Build(), "", "  ")
	return b
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
