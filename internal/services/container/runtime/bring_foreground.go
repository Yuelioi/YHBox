package runtime

import (
	"context"
	"fmt"
	"time"

	"yhbox/internal/services/container"
)

// bringToForegroundWithRetry 调 GameProvider.BringToForeground，失败重试。
// SetForegroundWindow 在全屏独占 / 反作弊场景常被 OS 拒绝，重试覆盖大多数失败。
//   - tries: 最多尝试次数（含首次）
//   - interval: 两次之间 sleep
//   - 任一次成功 → 立即返 true
//   - 全 fail / ctx 取消 → false
func bringToForegroundWithRetry(ctx context.Context, g GameProvider, tries int, interval time.Duration) bool {
	if g == nil {
		return false
	}
	hwnd, ok := g.HWND()
	if !ok || hwnd == 0 {
		return false
	}
	for i := 0; i < tries; i++ {
		if g.BringToForeground(hwnd) {
			return true
		}
		if i == tries-1 {
			break
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(interval):
		}
	}
	return false
}

// execBringGameForegroundImpl 真正实现。3 次 × 50ms 重试。失败仅日志，继续走 out。
func (r *ContainerRunner) execBringGameForegroundImpl(ctx context.Context, node *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	g := r.rt.GameProvider()
	ok := bringToForegroundWithRetry(ctx, g, 3, 50*time.Millisecond)
	if !ok {
		if r.rt.Emit != nil {
			r.rt.Emit("log:line", map[string]any{
				"level":   "warn",
				"tag":     "RUNTIME",
				"message": fmt.Sprintf("BringGameForeground 节点 %s 重试 3 次仍失败（游戏窗口可能是全屏独占）", node.ID),
			})
		}
	}
	return r.edges.next(node.ID+".out", tok.LoopStack), nil
}
