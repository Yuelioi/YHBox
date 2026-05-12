package fish

import (
	"context"
	"image"
	"time"

	"yhbox/pkg/capture"
	"yhbox/pkg/input"
	"yhbox/pkg/log"
)

// ---------------- IDLE ----------------

func (m *machine) handleIdle(ctx context.Context, frame *image.RGBA) {
	sfHit, sfOK := m.det.StartFish(frame)
	if sfOK {
		m.idleLogNext = time.Time{}
		m.logState("发现钓鱼准备界面 (conf=%.2f)", sfHit.Conf)
		m.setState(SETUP)
		return
	}

	hookHit, hookOK := m.det.HookIconDim(frame)
	if hookOK {
		m.idleLogNext = time.Time{}
		m.stats.CastCount++
		m.publishStats()
		m.logState("发现钓鱼点 (icon conf=%.2f)，抛竿 #%d", hookHit.Conf, m.stats.CastCount)
		input.Tap(m.hwnd, "f", delayMid, delayShort)
		// 鱼饵警告 ~0.25s 即时弹，先短等 300ms 检一次
		const baitProbeDelay = 300 * time.Millisecond
		if !m.sleep(ctx, baitProbeDelay) {
			return
		}
		if frame2, err := capture.Frame(m.hwnd); err == nil {
			if hit, ok := m.det.NeedBait(frame2); ok {
				m.logState("鱼饵不足 (conf=%.2f)，进入自动购买流程", hit.Conf)
				m.stats.CastCount--
				m.publishStats()
				input.ReleaseAll(m.hwnd)
				m.setState(BUYBAIT)
				return
			}
			if hit, ok := m.det.WarehouseFull(frame2); ok {
				m.stats.CastCount--
				m.publishStats()
				input.ReleaseAll(m.hwnd)
				if m.cfg.AutoSell.Load() {
					m.logState("鱼仓已满 (conf=%.2f)，进入售卖流程", hit.Conf)
					m.setState(SHOPSELL)
				} else {
					m.logState("鱼仓已满 (conf=%.2f)，AutoSell=false → 自动暂停等用户手动卖", hit.Conf)
					if m.ctrl != nil {
						m.ctrl.Pause()
						m.log.Log(log.SYSTEM, "鱼仓已满，自动出售已禁用，已自动暂停。手动卖完后在 GUI 点[继续]，或点[停止]退出。")
						if !m.ctrl.WaitUnpause() {
							return
						}
						m.log.Log(log.SYSTEM, "已恢复（state 留在 IDLE 继续扫描）")
					}
					// state 留 IDLE，下次循环重新检测画面
				}
				return
			}
			// F 被忽略时游戏可能直接回准备界面
			if _, ok := m.det.StartFish(frame2); ok {
				m.logState("抛竿后回到准备界面，重新走准备流程")
				m.stats.CastCount--
				m.publishStats()
				m.setState(SETUP)
				return
			}
		}
		// 鱼饵 OK + F 已被游戏接受，等剩余的 cast 动画
		if remaining := delayLong - baitProbeDelay; remaining > 0 {
			if !m.sleep(ctx, remaining) {
				return
			}
		}
		m.waitingStart = time.Now()
		m.hookStreak = 0
		m.hookStreakStart = time.Time{}
		m.setState(WAITING)
		return
	}

	if _, ok := m.det.Result(frame); ok {
		m.logState("检测到残留的结算界面，关闭中")
		m.pressEscUntilClear(ctx)
		return
	}

	if m.debugEnabled() && (m.idleLogNext.IsZero() || time.Now().After(m.idleLogNext)) {
		m.logState("扫描钓鱼点中  start_fish=%.2f hook_icon=%.2f",
			sfHit.Conf, hookHit.Conf)
		m.idleLogNext = time.Now().Add(10 * time.Second)
	}
}

// ---------------- SETUP ----------------

func (m *machine) handleSetup(ctx context.Context, frame *image.RGBA) {
	if hit, ok := m.det.StartFish(frame); ok {
		m.logState("点击开始钓鱼 (%d, %d, conf=%.2f)", hit.ClientX, hit.ClientY, hit.Conf)
		input.Click(m.hwnd, hit.ClientX, hit.ClientY, delayShort, delayShort, delayShort)
		_ = m.sleep(ctx, 1*time.Second)

		if frame2, err := capture.Frame(m.hwnd); err == nil {
			if hit, ok := m.det.NeedBait(frame2); ok {
				m.logState("鱼饵不足 (conf=%.2f)，进入自动购买流程", hit.Conf)
				input.ReleaseAll(m.hwnd)
				m.setState(BUYBAIT)
				return
			}
		}

		m.setState(IDLE)
		return
	}
	m.setState(IDLE)
}

// ---------------- WAITING ----------------

func (m *machine) handleWaiting(ctx context.Context) {
	elapsed := time.Since(m.waitingStart)

	// 检测"鱼上钩了..."文字提示。文字模板自带 bbox，MatchTextROI 在固定区域内搜，
	// 比 hook icon 更稳——文字 vs 暗 bar 反差大，跨场景 conf 都 ≥ 0.99。
	// 直接靠 streak（连续 N 帧 textOK）过滤抖动，无需额外 conf 突变检测。
	var textHit Hit
	textHit.Conf = -1
	if elapsed >= minIconLatency {
		frame, err := capture.Frame(m.hwnd)
		if err == nil {
			var textOK bool
			textHit, textOK = m.det.HookText(frame)

			if textOK {
				if m.hookStreak == 0 {
					m.hookStreak = 1
					m.hookStreakStart = time.Now()
					m.logState("发现上钩文字 (conf=%.2f)，确认中", textHit.Conf)
				} else {
					m.hookStreak++
				}
				if m.hookStreak >= hookStreakCount && time.Since(m.hookStreakStart) >= hookStreakWindow {
					m.hookStreak = 0
					m.hookStreakStart = time.Time{}
					m.logState("上钩！(text conf=%.2f, %.1fs) 按 F 收线", textHit.Conf, elapsed.Seconds())
					if !m.tryHookF(ctx) {
						m.logState("F 收线 %d 次重试均未见耐力条，进入恢复", hookFMaxRetries)
						m.enterRecover("F 收线重试失败", false)
						return
					}
					m.fishingStart = time.Now()
					m.fishingLogNext = time.Time{}
					m.emptyCastStreak = 0 // 上钩 = 真的有进展，重置空抛计数
					m.setState(FISHING)
					return
				}
			} else if m.hookStreak > 0 {
				// 确认中文字消失：可能是误检，重置
				m.hookStreak = 0
				m.hookStreakStart = time.Time{}
			}

			// StartFish 误入检测（上一轮恢复后误进 WAITING）
			if m.waitingLogNext.IsZero() || time.Now().After(m.waitingLogNext) {
				if _, ok := m.det.StartFish(frame); ok {
					m.logState("等待中检测到准备界面（前一轮恢复后误进 WAITING），回 SETUP")
					m.setState(SETUP)
					return
				}
			}
		}
	}

	if m.debugEnabled() && (m.waitingLogNext.IsZero() || time.Now().After(m.waitingLogNext)) {
		m.logState("等待上钩 %.1fs  text=%.3f (阈值 %.2f)",
			elapsed.Seconds(), textHit.Conf, confHigh)
		m.waitingLogNext = time.Now().Add(5 * time.Second)
	}

	// WAITING 超时：探一次画面，能路由就路由（鱼饵不足/鱼跑了/回到准备界面），
	// 探不出 → 进 RECOVERING，让用户介入（不再静默回 IDLE 重抛）。
	if elapsed > baitWarningTimeout {
		input.ReleaseAll(m.hwnd)
		phase, evidence := m.inspectPhase()
		m.logState("WAITING %.0fs 未上钩，画面探测：%s [%s]", elapsed.Seconds(), phase, evidence)
		if m.routePhase(ctx, phase) {
			// PhaseReady = F 按钮仍可见 = 这次抛竿啥也没发生（鱼仓满/鱼饵不足等关键提示
			// 没识别就会卡这）。其他 phase（SettleWin/Fail/Setup/Shop/Bait）都算有进展。
			if phase == PhaseReady {
				m.emptyCastStreak++
				if m.emptyCastStreak >= maxEmptyCastStreak {
					m.logState("连续 %d 次空抛回 IDLE，可能鱼仓满/鱼饵不足等关键提示未识别，自动暂停", m.emptyCastStreak)
					m.emptyCastStreak = 0
					if m.ctrl != nil {
						m.ctrl.Pause()
						m.log.Log(log.SYSTEM, "连续空抛超限，已自动暂停。手动处理后在 GUI 点[继续]，或点[停止]退出。")
						if !m.ctrl.WaitUnpause() {
							return
						}
						m.log.Log(log.SYSTEM, "已恢复")
					}
				}
			} else {
				m.emptyCastStreak = 0
			}
			return
		}
		m.enterRecover("WAITING 超时且无识别", false)
	}
}

// tryHookF 按 F 收线，每次按完在 HookFRetryDelay 窗口内每 300ms 轮询耐力条 ROI，
// 见到耐力条立刻返回 true（可进入 FISHING）。窗口超时还没见到就再按一次，
// 重试 HookFMaxRetries 次都失败返回 false。
// 闭环验证对付 F 偶发丢键；快速轮询避免成功时白等一整窗口、UI 卡顿感。
func (m *machine) tryHookF(ctx context.Context) bool {
	const pollInterval = 300 * time.Millisecond
	for i := 0; i < hookFMaxRetries; i++ {
		if m.shouldExit() {
			return false
		}
		input.Tap(m.hwnd, "f", delayMid, delayShort)
		deadline := time.Now().Add(hookFRetryDelay)
		for time.Now().Before(deadline) {
			if !m.sleep(ctx, pollInterval) {
				return false
			}
			if m.shouldExit() {
				return false
			}
			barX, barY, barW, barH := m.det.PickBarROI(m.clientW, m.clientH)
			if barW <= 0 || barH <= 0 {
				continue // 资源缺失，跳过本次轮询
			}
			barFrame, err := capture.FrameROI(m.hwnd, barX, barY, barW, barH)
			if err != nil {
				continue
			}
			bar := m.det.FishingBarDirect(barFrame)
			if bar.CursorX >= 0 && bar.TargetX >= 0 && bar.Confidence >= confBar {
				if i > 0 {
					m.logState("F 收线成功（重试 %d 次后见耐力条 conf=%.2f）", i, bar.Confidence)
				}
				return true
			}
		}
		m.logState("F 收线 %s 内未见耐力条，重试 #%d", hookFRetryDelay, i+1)
	}
	return false
}
