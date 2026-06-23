package container

import (
	"testing"

	_ "yotta/internal/nodes/ai"       // 注册 AI
	_ "yotta/internal/nodes/control"  // 注册 Start
	_ "yotta/internal/nodes/io"       // 注册 Log
	_ "yotta/internal/nodes/variable" // 注册 SetVar (直连消费者)
)

// 用户场景: AI → log1 → log2 串联, ai.red→log1(紧邻 OK), ai.white→log2(跨跳)。
// 跨跳的 white→log2 应报 EXEC_DATA_NOT_ADJACENT 警告(编辑期提前暴露, 否则运行才 REQUIRED_FIELD_MISSING)。
func TestAINode_ExecDataNonAdjacentWarns(t *testing.T) {
	c := &Container{
		SchemaVersion: 1, ID: "t", Name: "t",
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "ai", Kind: "AI", Config: map[string]any{
					"literal": map[string]any{"User": "hi"},
					"Outputs": []any{
						map[string]any{"Name": "red", "Type": "Number"},
						map[string]any{"Name": "white", "Type": "Number"},
					},
				}},
				{ID: "log1", Kind: "Log", Config: map[string]any{"literal": map[string]any{"Level": "info"}}},
				{ID: "log2", Kind: "Log", Config: map[string]any{"literal": map[string]any{"Level": "info"}}},
			},
			Edges: []GraphEdge{
				{From: "start.Done", To: "ai.In"},
				{From: "ai.Done", To: "log1.In"},
				{From: "ai.red", To: "log1.Message"},
				{From: "log1.Done", To: "log2.In"},
				{From: "ai.white", To: "log2.Message"},
			},
		},
	}
	var warns []ValidationError
	for _, e := range ValidateContainer(c, nil) {
		if e.Code == "EXEC_DATA_NOT_ADJACENT" {
			warns = append(warns, e)
		}
	}
	if len(warns) != 1 || warns[0].NodeID != "log2" {
		t.Fatalf("应只对 log2(white 跨跳)报 1 个 EXEC_DATA_NOT_ADJACENT, got %+v", warns)
	}
}

// AI 声明的动态输出字段应算 data-out / exec-output data field(config-aware)→ 可直连下游。
func TestAINode_DynamicOutputIsDataOutPin(t *testing.T) {
	n := &GraphNode{ID: "ai", Kind: "AI", Config: map[string]any{
		"Outputs": []any{map[string]any{"Name": "red", "Type": "Number"}},
	}}
	if !IsDataOutPinNode(n, "red") {
		t.Error("声明输出 red 应算 data-out pin(config-aware)")
	}
	if !IsExecOutputDataFieldNode(n, "red") {
		t.Error("声明输出 red 应算 exec-output data field(走 exec-data 直连)")
	}
	// 未声明的字段不算。
	if IsDataOutPinNode(n, "ghost") {
		t.Error("未声明的 ghost 不该算 data-out")
	}
}

// AI.red 数据线直连下游 data-in pin(配合紧邻 exec 边)应校验通过, 不再 INVALID_PIN。
func TestAINode_DirectWireOutputNoInvalidPin(t *testing.T) {
	c := &Container{
		SchemaVersion: 1, ID: "t", Name: "t",
		Vars: []VarDecl{{Name: "v"}},
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "ai", Kind: "AI", Config: map[string]any{
					"literal": map[string]any{"User": "hi"},
					"Outputs": []any{map[string]any{"Name": "red", "Type": "Number"}},
				}},
				{ID: "sv", Kind: "SetVar", Config: map[string]any{
					"literal": map[string]any{"VarName": "v"},
				}},
			},
			Edges: []GraphEdge{
				{From: "start.Done", To: "ai.In"},
				{From: "ai.Done", To: "sv.In"},      // exec 边 — 带 exec-data 下发
				{From: "ai.red", To: "sv.Value"},    // data 边 — 动态输出字段直连
			},
		},
	}
	for _, e := range ValidateContainer(c, nil) {
		if e.Code == CodeInvalidPin {
			t.Errorf("AI.red 直连下游不该 INVALID_PIN: %+v", e)
		}
	}
}

// 复现真机 INVALID_PIN "Count": AI 节点声明 config.Outputs[Count] + 把 Count 绑到变量,
// validateCaptureRefs 应认 Count 可绑(BindableFieldsForNode 含 config.Outputs)。
func TestAINode_DeclaredOutputIsBindable(t *testing.T) {
	c := &Container{
		SchemaVersion: 1,
		ID:            "t",
		Name:          "t",
		Vars:          []VarDecl{{Name: "cnt"}},
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "ai", Kind: "AI", Config: map[string]any{
					"literal": map[string]any{"User": "hi"},
					"Outputs": []any{map[string]any{"Name": "Count", "Type": "Integer"}},
					"capture": map[string]any{"Count": "cnt"},
				}},
			},
		},
	}
	for _, e := range ValidateContainer(c, nil) {
		if e.Code == CodeInvalidPin {
			t.Errorf("declared output Count 应可绑, 却 INVALID_PIN: %+v", e)
		}
	}
}
