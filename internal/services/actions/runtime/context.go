package runtime

import (
	"context"
	"time"

	"yhbox/internal/services/actions"
)

// RuntimeContext 一次 RunOnce 期间的所有 mutable runtime state。
//
// 约定：actions.Action 是 immutable 视图，runtime 不写回 action.json；
// 所有 pressedKeys 等可变状态住这里；Stop / 失败 / 完成 → context 销毁。
type RuntimeContext struct {
	// 不变：从 action 加载来的引用（read-only）
	action *actions.Action

	// 标识 + 生命周期
	ActionID  string
	RunID     uint64 // Start 分配的 epoch token，防 stale defer
	Cancel    context.CancelFunc
	StartedAt time.Time

	// 输入 cleanup contract
	pressedKeys map[string]int
}

// newRuntimeContext 由 Runner.Start 调用，分配一个新 ctx。
func newRuntimeContext(a *actions.Action, runID uint64, cancel context.CancelFunc) *RuntimeContext {
	return &RuntimeContext{
		action:      a,
		ActionID:    a.ID,
		RunID:       runID,
		Cancel:      cancel,
		StartedAt:   time.Now(),
		pressedKeys: map[string]int{},
	}
}
