package runtime

import (
	"context"
	"fmt"

	"yhbox/internal/services/container"
)

// ---- Stopwatch (v3 Phase C) ----
//
// 三节点共用 ContainerRunner.stopwatches (per-run scope). key 命名空间独立于 $vars.*,
// 同名 key/var 不冲突. Start 已存在 key → reset; Stop 不存在 key → 静默 (validator 已
// static-warn); Read 不存在 key → 0. 详见 spec v3 Phase C.

func (r *ContainerRunner) execStopwatchStart(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	key := configString(n, "key")
	if key == "" {
		return nil, fmt.Errorf("StopwatchStart %s: empty key", n.ID)
	}
	r.stopwatches.start(key)
	return r.edges.next(n.ID+".out", tok.LoopStack), nil
}

func (r *ContainerRunner) execStopwatchStop(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	key := configString(n, "key")
	if key == "" {
		return nil, fmt.Errorf("StopwatchStop %s: empty key", n.ID)
	}
	if !r.stopwatches.stop(key) {
		// 静默. validator 已 static-warn key 不匹配的情况.
	}
	return r.edges.next(n.ID+".out", tok.LoopStack), nil
}

func (r *ContainerRunner) execStopwatchRead(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	key := configString(n, "key")
	if key == "" {
		return nil, fmt.Errorf("StopwatchRead %s: empty key", n.ID)
	}
	elapsedMs := r.stopwatches.read(key)
	// 此 runtime 没有通用 dataOut 总线, 跟 WaitTemplate/DetectColor 一样把结果
	// 推 SysState, 下游通过 $sys.lastStopwatch.elapsedMs 读取.
	r.rt.UpdateSys(func(s *SysState) {
		s.LastStopwatchElapsedMs = elapsedMs
	})
	return r.edges.next(n.ID+".out", tok.LoopStack), nil
}
