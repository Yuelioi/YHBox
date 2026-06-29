package container

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestLegacyEdgeKindIgnored 老存档里 edge 带 kind:"data" / kind:"exec",
// 反序列化后 GraphEdge 应该只剩 From/To (Go encoding/json 默认 ignore 未知字段),
// 重新 marshal 不应再出现 kind 字段. 此 smoke 保证 C1+C2 删 GraphEdge.Kind
// 后, 用户老容器加载仍兼容.
func TestLegacyEdgeKindIgnored(t *testing.T) {
	legacyJSON := `{
		"id": "c1",
		"schemaVersion": 4,
		"label": "test",
		"graph": {
			"id": "g1",
			"version": 1,
			"nodes": [
				{"id": "start", "kind": "Start", "x": 0, "y": 0},
				{"id": "wt", "kind": "Win32WindowTarget", "x": 100, "y": 0},
				{"id": "log", "kind": "Log", "x": 200, "y": 0, "config": {"literal": {"Message": "hi", "Level": "info"}}}
			],
			"edges": [
				{"from": "start.Done", "to": "log.in", "kind": "exec"}
			]
		},
		"subgraphs": []
	}`
	var c Container
	if err := json.Unmarshal([]byte(legacyJSON), &c); err != nil {
		t.Fatalf("unmarshal legacy JSON: %v", err)
	}
	if len(c.Graph.Edges) != 1 {
		t.Fatalf("expected 1 edge after load, got %d", len(c.Graph.Edges))
	}
	got := c.Graph.Edges[0]
	if got.From != "start.Done" || got.To != "log.in" {
		t.Errorf("edge from/to corrupted: %+v", got)
	}

	// Re-marshal — kind field should NOT reappear (it's not in the struct).
	out, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(out)
	// Generic "kind" matches Container fields too (e.g. node kind). Look for
	// the specific edge-kind values that were in legacy data.
	if strings.Contains(s, `"kind":"exec"`) || strings.Contains(s, `"kind":"data"`) {
		t.Errorf("re-marshal contains edge kind literal that should be dropped:\n%s", s)
	}
}

// TestLegacyDataEdgeValidatesCleanly is the C1+C2 cross-cut: a legacy container
// with edge.kind="data" on a GetVar→Log data-pin connection used to flip to
// the exec-pin check via validator's switch e.Kind default and emit
// INVALID_PIN. After C1+C2 the edge type is derived from (kind, pin) so
// the legacy kind field is irrelevant — load + validate must be clean.
func TestLegacyDataEdgeValidatesCleanly(t *testing.T) {
	legacyJSON := `{
		"id": "c1",
		"schemaVersion": 4,
		"label": "test",
		"vars": [{"name": "x", "type": "any"}],
		"graph": {
			"id": "g1",
			"version": 1,
			"nodes": [
				{"id": "start", "kind": "Start", "x": 0, "y": 0},
				{"id": "wt", "kind": "Win32WindowTarget", "x": 100, "y": 0},
				{"id": "gv", "kind": "GetVar", "x": 150, "y": 100, "config": {"VarName": "x", "Scope": "local"}},
				{"id": "log", "kind": "Log", "x": 300, "y": 0, "config": {"literal": {"Message": "", "Level": "info"}}}
			],
			"edges": [
				{"from": "start.Done", "to": "log.In", "kind": "exec"},
				{"from": "gv.Value", "to": "log.Message", "kind": "data"}
			]
		},
		"subgraphs": []
	}`
	var c Container
	if err := json.Unmarshal([]byte(legacyJSON), &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	errs := ValidateContainer(&c, nil)
	for _, e := range errs {
		if e.Code == CodeInvalidPin {
			t.Errorf("regression: legacy data edge with kind:\"data\" still triggered INVALID_PIN: %+v", e)
		}
	}
}
