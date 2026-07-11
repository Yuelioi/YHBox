package mcpserver

import (
	"encoding/json"
	"testing"

	"github.com/yottaapp/yotta/internal/node"

	_ "github.com/yottaapp/yotta/internal/nodes/all"
)

func FuzzMCPNodeParams(f *testing.F) {
	f.Add("Sleep", []byte(`{"DurationMs":10}`))
	f.Add("ClickAt", []byte(`{"Point":{"x":0.5,"y":0.5},"Button":"left"}`))
	f.Add("unknown", []byte(`{}`))
	f.Fuzz(func(t *testing.T, kind string, raw []byte) {
		if len(kind) > 256 || len(raw) > 64<<10 {
			t.Skip()
		}
		var params map[string]any
		if json.Unmarshal(raw, &params) != nil {
			return
		}
		container, _, err := buildMicroContainerWithRegistry(node.DefaultRegistrySnapshot(), kind, params)
		if err != nil {
			return
		}
		if _, err := json.Marshal(container); err != nil {
			t.Fatalf("micro-container cannot be encoded: %v", err)
		}
	})
}
