package main

import (
	"encoding/json"

	"yotta/internal/catalog"
)

// listNodesJSON 返节点目录 JSON (catalog.Build 已按 category→kind 稳定排序)。
func listNodesJSON() []byte {
	b, _ := json.MarshalIndent(catalog.Build(), "", "  ")
	return b
}
