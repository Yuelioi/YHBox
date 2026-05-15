package fish

import (
	"context"
	"image"
	"time"

	"yhbox/pkg/input"
)

// ---------------- RECOVERING ----------------

// enterRecover 切到 RECOVERING 状态。
//   - pressEsc=true:  在恢复过程中允许发一次 ESC（探不出画面时的最后手段）
//   - pressEsc=false: 跳过 ESC，仅靠 inspectPhase 路由（用于 F 收线已发但耐力条没出来这种"已动过手"的场景）
func (m *machine) enterRecover(reason string, pressEsc bool) {
	m.logState("进入恢复：%s", reason)
	// recoveryEscDone 起始 = !pressEsc：true 表示 ESC 已"消费"过、不再发
	m.recoveryEscDone = !pressEsc
	m.setState(RECOVERING)
}

func (m *machine) handleRecovering(ctx context.Context, frame *image.RGBA) {
	phase, evidence := m.inspectPhaseFrame(frame)
	if phase != PhaseUnknown {
		m.logState("恢复探测：%s [%s]", phase, evidence)
	}
	if m.routePhase(ctx, phase) {
		return
	}

	if !m.recoveryEscDone {
		input.Tap(m.hwnd, "esc", delayMid, delayShort)
		m.recoveryEscDone = true
		_ = m.sleep(ctx, 400*time.Millisecond)
		return
	}

	// 探不到任何 phase 且 ESC 已发：循环等下一帧，让用户在 GUI 点[停止]退出或[暂停]介入
	_ = m.sleep(ctx, 500*time.Millisecond)
}
