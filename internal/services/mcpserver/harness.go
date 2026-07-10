package mcpserver

import (
	"context"
	"errors"

	"github.com/yottaapp/yotta/internal/node"
	"github.com/yottaapp/yotta/internal/services/container"
	"github.com/yottaapp/yotta/internal/services/container/runtime"
)

type RunNodeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type RunNodeResult struct {
	Ok          bool           `json:"ok"`
	FiredOutput string         `json:"firedOutput,omitempty"`
	Data        map[string]any `json:"data,omitempty"`
	Error       *RunNodeError  `json:"error,omitempty"`
}

const microNodeID = "n"

// buildMicroContainer 把单动作节点包成 {Start → 节点} 微容器, 节点参数进 config.literal.
func buildMicroContainer(kind string, params map[string]any) (*container.Container, string, error) {
	rn, ok := node.Get(kind)
	if !ok {
		return nil, "", errors.New("unknown kind")
	}
	if !isRunnable(rn.Spec) {
		return nil, "", errors.New("kind not runnable")
	}
	execIn := execInPin(rn.Spec)
	if execIn == "" {
		return nil, "", errors.New("kind has no exec input")
	}
	if params == nil {
		params = map[string]any{}
	}
	c := &container.Container{
		SchemaVersion: container.CurrentSchemaVersion,
		Name:          "mcp-run-node",
		Graph: container.Graph{
			SchemaVersion: container.GraphSchemaVersion,
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: microNodeID, Kind: kind, Config: map[string]any{"literal": params}},
			},
			Edges: []container.GraphEdge{
				{From: "start.Done", To: microNodeID + "." + execIn},
			},
		},
	}
	c.Normalize()
	return c, microNodeID, nil
}

// runMicroContainer 跑微容器并从 held-output 缓存收割目标节点输出.
// rt 由调用方备好 (生产: NewRuntimeContext + SetActiveWindow; 测试: 预注入 mock).
func runMicroContainer(ctx context.Context, rt *runtime.RuntimeContext, c *container.Container, nodeID string) (RunNodeResult, *node.Image) {
	r := runtime.NewContainerRunner(rt)
	runErr := r.Run(ctx)

	// 收割: execOutputs 里属于本节点的字段.
	prefix := nodeID + "."
	data := map[string]any{}
	var img *node.Image
	for k, v := range r.ExecOutputs() {
		if len(k) <= len(prefix) || k[:len(prefix)] != prefix {
			continue
		}
		field := k[len(prefix):]
		if im, ok := v.(node.Image); ok {
			cp := im
			img = &cp
			continue
		}
		data[field] = v
	}

	if runErr != nil {
		code := "error"
		if errors.Is(runErr, context.DeadlineExceeded) {
			code = "TIMEOUT"
		} else {
			var coded node.Coded
			if errors.As(runErr, &coded) {
				code = string(coded.ErrCode())
			}
		}
		return RunNodeResult{Ok: false, Data: data, Error: &RunNodeError{Code: code, Message: runErr.Error()}}, img
	}
	return RunNodeResult{Ok: true, FiredOutput: firedOutput(c, nodeID, data), Data: data}, img
}

// firedOutput 按节点 spec 反推走了哪个 exec 出口: 收割到的字段属于哪个出口声明的 Data 集.
// best-effort: 无数据字段的出口 (如 NotFound) 推不出 → 返 "" (消费方从空 data 自行判断).
func firedOutput(c *container.Container, nodeID string, data map[string]any) string {
	var kind string
	for i := range c.Graph.Nodes {
		if c.Graph.Nodes[i].ID == nodeID {
			kind = c.Graph.Nodes[i].Kind
		}
	}
	rn, ok := node.Get(kind)
	if !ok {
		return ""
	}
	for _, o := range rn.Spec.Outputs {
		for _, d := range o.Data {
			if _, present := data[d.Name]; present {
				return o.Name
			}
		}
	}
	return ""
}
