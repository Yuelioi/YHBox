package container

import (
	"testing"

	_ "yotta/internal/nodes/ai"      // 注册 AI
	_ "yotta/internal/nodes/control" // 注册 Start
)

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
