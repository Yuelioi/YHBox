package fish

import (
	"context"
	"fmt"
	"image"
	"math"
	"time"

	"yhbox/pkg/capture"
	"yhbox/pkg/input"
)

// ---------------- FISHING ----------------

func (m *machine) handleFishing(ctx context.Context) {
	elapsed := time.Since(m.fishingStart)

	if elapsed > fishingTimeout {
		m.logState("溜鱼超时 %.0fs，进入恢复", elapsed.Seconds())
		input.ReleaseAll(m.hwnd)
		m.fishingBarMissingStart = time.Time{}
		m.fc.reset()
		m.enterRecover("溜鱼超时", true)
		return
	}

	// 用 FrameROI 只抓耐力条区域
	barX, barY, barW, barH := m.det.PickBarROI(m.clientW, m.clientH)
	if barW <= 0 || barH <= 0 {
		return // 资源缺失
	}
	barFrame, err := capture.FrameROI(m.hwnd, barX, barY, barW, barH)
	if err != nil {
		return
	}

	bar := m.det.FishingBarDirect(barFrame)
	barVisible := bar.CursorX >= 0 && bar.TargetX >= 0 && bar.Confidence >= confBar

	if !barVisible {
		// F 键到耐力条出现有动画延迟，最弱的鱼也要 5s 才会脱钩，
		// 前 4s 不进结算判定，避免把动画过渡期误判成结算
		if elapsed < 4*time.Second {
			return
		}
		if m.fishingBarMissingStart.IsZero() {
			m.fishingBarMissingStart = time.Now()
			input.ReleaseAll(m.hwnd)
			m.logState("耐力条消失，开始检测结算信号")
		}

		// bar 不可见时降频检测结算信号（每 500ms 一次）
		now := time.Now()
		if now.Sub(m.fishingFullFrameAt) >= 500*time.Millisecond {
			m.fishingFullFrameAt = now
			frame, err := capture.Frame(m.hwnd)
			if err == nil {
				if hit, ok := m.det.FishEscape(frame); ok {
					m.recordOutcome(false, fmt.Sprintf("（鱼儿溜走了 conf=%.2f）", hit.Conf))
					m.fishingBarMissingStart = time.Time{}
					m.fc.reset()
					_ = m.sleep(ctx, delayLong)
					m.setState(IDLE)
					return
				}
				if hit, ok := m.det.Result(frame); ok {
					m.recordOutcome(true, fmt.Sprintf("（结算 conf=%.2f）", hit.Conf))
					m.fishingBarMissingStart = time.Time{}
					m.fc.reset()
					m.pressEscUntilClear(ctx)
					m.setState(IDLE)
					return
				}
			}
		}

		if time.Since(m.fishingBarMissingStart) > barMissingTimeout {
			m.logState("耐力条消失 %.0fs 无结算信号，进入恢复",
				time.Since(m.fishingBarMissingStart).Seconds())
			m.fishingBarMissingStart = time.Time{}
			m.fc.reset()
			m.enterRecover("耐力条消失后无结算信号", true)
		}
		return
	}

	m.fishingBarMissingStart = time.Time{}

	errVal := float64(bar.TargetX - bar.CursorX)
	deadzonePx := math.Max(2, float64(bar.TargetW)*deadzoneRatio)

	dir := chooseDirection(errVal, deadzonePx)
	m.fc.applyDirection(m.hwnd, dir)

	if m.fishingLogNext.IsZero() || time.Now().After(m.fishingLogNext) {
		dirStr := "—"
		if dir > 0 {
			dirStr = "D"
		} else if dir < 0 {
			dirStr = "A"
		}
		m.logState("溜鱼 %.0fs  err=%.0f dir=%s dead=%.0f conf=%.2f",
			elapsed.Seconds(), errVal, dirStr, deadzonePx, bar.Confidence)
		if m.debugEnabled() {
			saveDebugFrame(fmt.Sprintf("bar_%.2f", bar.Confidence), barFrame)
		}
		m.fishingLogNext = time.Now().Add(1 * time.Second)
	}
}

// recordOutcome 累加成功/失败统计；成功时按耐力条可见时长分类鱼种。
// 分类只在 fishingStart 和 fishingBarMissingStart 都有效时执行——兜底恢复路径
// (routePhase / handleResult) 可能缺这两个时间戳之一，那条路径只计总数不分类。
func (m *machine) recordOutcome(success bool, suffix string) {
	if success {
		m.stats.Success++
		if !m.fishingStart.IsZero() && !m.fishingBarMissingStart.IsZero() {
			m.classifyFish(m.fishingBarMissingStart.Sub(m.fishingStart))
		}
		m.logState("钓鱼成功%s ✓ %d ✗ %d", suffix, m.stats.Success, m.stats.Fail)
	} else {
		m.stats.Fail++
		m.logState("钓鱼失败%s ✓ %d ✗ %d", suffix, m.stats.Success, m.stats.Fail)
	}
	m.publishStats()
}

// classifyFish 按耐力条可见时长分鱼种。阈值：<=6s 普通；7-9s 紫色；>=10s 金色。
// 用 fishingBarMissingStart - fishingStart（耐力条真正可见的时长），不能用
// time.Since(fishingStart)——那是 Result 检测时刻，包含 1-3s 结算动画，会错位分类。
func (m *machine) classifyFish(fightDur time.Duration) {
	switch {
	case fightDur <= 6*time.Second:
		m.stats.CommonCount++
	case fightDur <= 9*time.Second:
		m.stats.PurpleCount++
	default:
		m.stats.GoldenCount++
	}
}

// ---------------- RESULT ----------------

func (m *machine) handleResult(ctx context.Context, frame *image.RGBA) {
	if m.resultEnteredAt.IsZero() {
		m.resultEnteredAt = time.Now()
	}

	if _, ok := m.det.Result(frame); ok {
		m.recordOutcome(true, "")
		m.pressEscUntilClear(ctx)
		m.resultEnteredAt = time.Time{}
		m.setState(IDLE)
		return
	}

	if _, ok := m.det.FishEscape(frame); ok {
		m.recordOutcome(false, "")
		_ = m.sleep(ctx, delayLong)
		m.resultEnteredAt = time.Time{}
		m.setState(IDLE)
		return
	}

	if time.Since(m.resultEnteredAt) > resultDetectTimeout {
		m.logState("结算判定超时，进入恢复")
		m.resultEnteredAt = time.Time{}
		m.enterRecover("结算判定超时", true)
	}
}

func (m *machine) pressEscUntilClear(ctx context.Context) {
	for i := 0; i < 5; i++ {
		if i > 0 {
			m.logState("结算界面仍在，重试 ESC #%d", i+1)
		}
		input.Tap(m.hwnd, "esc", delayMid, delayShort)
		deadline := time.Now().Add(delayLong)
		for time.Now().Before(deadline) {
			if !m.sleep(ctx, 150*time.Millisecond) {
				return
			}
			frame, err := capture.Frame(m.hwnd)
			if err != nil {
				continue
			}
			// 关闭成功的唯一条件：result 文字消失。
			// 不强求 hook icon 同时可见——玩家可能此时移动了，只要结算面板没了就 OK。
			if _, hasResult := m.det.Result(frame); !hasResult {
				return
			}
		}
	}
	m.logState("结算界面关闭失败，5 次 ESC 后仍未清除")
}
