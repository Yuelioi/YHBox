package runtime

import (
	"context"
	"errors"
	"fmt"

	"yhbox/internal/services/container"
	"yhbox/internal/services/container/nodekind"
	_ "yhbox/internal/services/container/nodekind/specs" // ensure registry is populated
)

// execNode 单节点执行入口. atomic #3 cutover: 老 964 行 switch 已拆, 改走 dispatchInRegion
// (Phase 5.5b 新 framework, 内部 route Loop/Subgraph/Try 等 RegionRunner 或普通节点).
//
// IsPureData / IsVisualOnly / Disabled 这 3 个 gatekeep 仍走 nodekind registry. atomic #5
// 完成后改走 nodepkg.Get(kind).Spec.
func (r *ContainerRunner) execNode(ctx context.Context, node *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	// Gatekeep: kind must be registered. Catches typos and stale switch cases.
	spec, ok := nodekind.Get(node.Kind)
	if !ok {
		return nil, fmt.Errorf("execNode: unknown kind %q (not in nodekind registry)", node.Kind)
	}
	// IsPureData kinds should never reach execNode (no exec edge points to them).
	if spec.IsPureData {
		return nil, fmt.Errorf("execNode: kind %q is pure-data, cannot be executed (validator drift!)", node.Kind)
	}
	// IsVisualOnly = render-only nodes (CommentBox). No-op execution.
	if spec.IsVisualOnly {
		return nil, nil
	}

	// Editor v2 C — Disable Node: skip kind-specific logic, route through configured exit pin.
	if node.Disabled {
		return r.passthroughDisabled(node, tok)
	}

	return r.dispatchInRegion(ctx, node, tok)
}

// runSubFlow OnEvent listener 子分支用的迷你 dispatch (与主 dispatch 同语义).
// listener.go 持 ContainerRunner 调它跑 OnEvent.out 出口下游.
//
// !! 已知 goroutine race !! — r.currentTick 跨 goroutine 共享. listener 单 goroutine 内
// 顺序执行, 但跟主 runner.go::Run 并行会撞 snapshot. backend-backlog.md B1 跟进.
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
		// Per-exec-tick snapshot (sequential within this goroutine).
		// !! NOT goroutine-safe — see backend-backlog.md B1 注释.
		r.currentTick = CaptureSnapshot(r.rt.Vars(), r.rt.Sys())
		out, err := r.execNode(ctx, node, tok)
		r.currentTick = nil
		if err != nil {
			if errors.Is(err, errStopRun) {
				return nil
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
