package runtime

import (
	"context"
	"errors"
	"fmt"

	nodepkg "yhbox/internal/node"
	"yhbox/internal/services/container"
)

// execNode 单节点执行入口. atomic #3 cutover: 老 964 行 switch 已拆, 改走 dispatchInRegion
// (Phase 5.5b 新 framework, 内部 route Loop/Subgraph/Try 等 RegionRunner 或普通节点).
//
// IsPureData / IsVisualOnly / Disabled 3 个 gatekeep 走 nodepkg.Get(kind).Spec.
func (r *ContainerRunner) execNode(ctx context.Context, node *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	rn, ok := nodepkg.Get(node.Kind)
	if !ok {
		return nil, fmt.Errorf("execNode: unknown kind %q (not in nodepkg registry)", node.Kind)
	}
	// IsPureData kinds should never reach execNode (no exec edge points to them).
	if rn.Spec.IsPureData {
		return nil, fmt.Errorf("execNode: kind %q is pure-data, cannot be executed (validator drift!)", node.Kind)
	}
	// IsVisualOnly = render-only nodes (CommentBox). No-op execution.
	if rn.Spec.IsVisualOnly {
		return nil, nil
	}

	// Editor v2 C — Disable Node: skip kind-specific logic, route through configured exit pin.
	if node.Disabled {
		return r.passthroughDisabled(node, tok)
	}

	return r.dispatchInRegion(ctx, node, tok)
}

// runSubFlow OnEvent listener 子分支用的迷你 dispatch (与主 dispatch 同语义).
// listener.go 持独立 ContainerRunner subRunner (makeSubRunner) 调它跑 OnEvent.out 下游.
// per-exec-tick snapshot 由 dispatchInRegion 入口写到 ctx (tickCtxKey, B1), per-goroutine
// 独立, 跟主 runner.go::Run 不撞.
func (r *ContainerRunner) runSubFlow(ctx context.Context, seeds []ExecToken) error {
	queue := append([]ExecToken{}, seeds...)
	for len(queue) > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		tok := queue[0]
		queue = queue[1:]
		node, ok := r.nodesByID[tok.NodeID]
		if !ok {
			return fmt.Errorf("subflow: unknown node %q", tok.NodeID)
		}
		out, err := r.execNode(ctx, node, tok)
		if err != nil {
			if errors.Is(err, errStopRun) {
				return nil
			}
			// P1.2: 同 ContainerRunner.Run, listener subflow 顶层也防 sentinel leak.
			if _, wrapped := r.checkSentinelLeak(node, err); wrapped != err {
				return wrapped
			}
			return err
		}
		queue = append(queue, out...)
	}
	return nil
}

// asFloat 把 JSON 解出来的 any 转 float64. int / float64 / int64 都兼容.
// data_pull.go 用 (parsePoint 等).
func asFloat(v any) float64 {
	switch x := v.(type) {
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case float64:
		return x
	case float32:
		return float64(x)
	}
	return 0
}
