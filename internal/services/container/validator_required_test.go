package container

import "testing"

func reqContainer(nodes []GraphNode, edges []GraphEdge) *Container {
	return &Container{
		SchemaVersion: CurrentSchemaVersion,
		ID:            "req-test",
		Graph:         Graph{Version: GraphSchemaVersion, Nodes: nodes, Edges: edges},
	}
}

func hasCodeForNode(errs []ValidationError, code, nodeID string) bool {
	for _, e := range errs {
		if e.Code == code && e.NodeID == nodeID {
			return true
		}
	}
	return false
}

// GetVar.VarName 是 Required:true 且无 Default。无 edge / 无 literal → 必须报 MISSING_REQUIRED_PIN。
func TestRequired_MissingNoSourceFlags(t *testing.T) {
	c := reqContainer([]GraphNode{
		{ID: "g", Kind: "GetVar", Config: map[string]any{}},
	}, nil)
	errs := validateRequiredPins(c)
	if !hasCodeForNode(errs, CodeMissingRequiredPin, "g") {
		t.Fatalf("expected MISSING_REQUIRED_PIN for GetVar.VarName, got %+v", errs)
	}
}

// config.literal 有 VarName → 满足,不报。
func TestRequired_LiteralSatisfies(t *testing.T) {
	c := reqContainer([]GraphNode{
		{ID: "g", Kind: "GetVar", Config: map[string]any{"literal": map[string]any{"VarName": "myVar"}}},
	}, nil)
	if errs := validateRequiredPins(c); hasCodeForNode(errs, CodeMissingRequiredPin, "g") {
		t.Fatalf("literal VarName present should satisfy required, got %+v", errs)
	}
}

// 连入 data 边 → 满足,不报。(用 GetVar 当数据源喂 Switch.Value)
// Switch.Value 是 Required:true 且无 Default。
func TestRequired_EdgeSatisfies(t *testing.T) {
	c := reqContainer([]GraphNode{
		{ID: "s", Kind: "Switch", Config: map[string]any{"cases": []any{"A", "B"}}},
		{ID: "g", Kind: "GetVar", Config: map[string]any{"literal": map[string]any{"VarName": "x"}}},
	}, []GraphEdge{{From: "g.Value", To: "s.Value"}})
	if errs := validateRequiredPins(c); hasCodeForNode(errs, CodeMissingRequiredPin, "s") {
		t.Fatalf("incoming data edge should satisfy required, got %+v", errs)
	}
}

// KeyPress.VK 是 Required:true 且 Default:"W"。无 edge/literal 也不报(有 Default 兜底)。
func TestRequired_DefaultSuppresses(t *testing.T) {
	c := reqContainer([]GraphNode{
		{ID: "k", Kind: "KeyPress", Config: map[string]any{}},
	}, nil)
	if errs := validateRequiredPins(c); hasCodeForNode(errs, CodeMissingRequiredPin, "k") {
		t.Fatalf("KeyPress.VK has Default='W', must not flag, got %+v", errs)
	}
}
