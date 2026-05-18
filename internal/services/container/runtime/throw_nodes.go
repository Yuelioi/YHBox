package runtime

import (
	"context"

	"yhbox/internal/services/container"
)

// ---- Throw (v3 Phase C) ----
//
// Throw 节点显式抛错, 走 errThrow sentinel 路径冒泡:
//   - 被 Try (Task 7) 包裹: Try 截获, 走 error 出口
//   - 没 Try 包: 冒泡到主 runner, container:error emit (跟 panic 合流)
//
// message 当前是字面量, 不做占位符解析. v3 后续可加 r.resolveTemplate 支持
// "${var}" 注入运行时 var. errThrow sentinel 本体在 nodes.go (跟 errStopRun 一起).

// execThrow 立刻 return errThrow, 不碰 edges/nodesByID — runSubFlow 接住后冒泡.
// v4: message via data-in pin (string).
func (r *ContainerRunner) execThrow(_ context.Context, n *container.GraphNode, _ ExecToken) ([]ExecToken, error) {
	msg := r.pullString(n, "message")
	return nil, &errThrow{message: msg}
}
