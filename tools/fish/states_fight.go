package fish

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
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
					m.recordOutcome(false, fmt.Sprintf("（鱼儿溜走了 conf=%.2f）", hit.Conf), frame)
					m.fishingBarMissingStart = time.Time{}
					m.fc.reset()
					_ = m.sleep(ctx, delayLong)
					m.setState(IDLE)
					return
				}
				if hit, ok := m.det.Result(frame); ok {
					m.recordOutcome(true, fmt.Sprintf("（结算 conf=%.2f）", hit.Conf), frame)
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
//
// frame 是当前结算画面（可为 nil，兜底路径调时没有）。如果分类是 golden 且
// cfg.CaptureGolden 开启，把 frame 存成 PNG —— atomic 读让 GUI 改"立刻生效"。
func (m *machine) recordOutcome(success bool, suffix string, frame *image.RGBA) {
	if success {
		m.stats.Success++
		if !m.fishingStart.IsZero() && !m.fishingBarMissingStart.IsZero() {
			kind := m.classifyFish(m.fishingBarMissingStart.Sub(m.fishingStart))
			if kind == "golden" && m.cfg.CaptureGolden.Load() && frame != nil {
				m.captureGoldenFrame(frame)
			}
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
// 返回 "common"|"purple"|"golden" 给上层（决定是否截图等）。
func (m *machine) classifyFish(fightDur time.Duration) string {
	switch {
	case fightDur <= 6*time.Second:
		m.stats.CommonCount++
		return "common"
	case fightDur <= 9*time.Second:
		m.stats.PurpleCount++
		return "purple"
	default:
		m.stats.GoldenCount++
		return "golden"
	}
}

// captureGoldenFrame 把结算瞬间的 frame 写到 captures/golden_YYYYMMDD_HHMMSS.png。
// 失败仅 warn 不影响 bot 继续。运行在 bot goroutine，PNG encode 是 CPU 密集但单张
// 1080p 大约 < 50 ms，每条金色鱼之间至少 30 s（钓鱼周期），不会拖累实时性。
func (m *machine) captureGoldenFrame(frame *image.RGBA) {
	const dir = "captures"
	if err := os.MkdirAll(dir, 0755); err != nil {
		m.log.Warn().Err(err).Str("tag", "RESULT").Msg("金色鱼截图：创建 captures/ 失败")
		return
	}
	path := filepath.Join(dir, "golden_"+time.Now().Format("20060102_150405")+".png")
	f, err := os.Create(path)
	if err != nil {
		m.log.Warn().Err(err).Str("tag", "RESULT").Msgf("金色鱼截图：打开 %s 失败", path)
		return
	}
	defer f.Close()
	if err := png.Encode(f, frame); err != nil {
		m.log.Warn().Err(err).Str("tag", "RESULT").Msg("金色鱼截图：PNG 编码失败")
		return
	}
	m.log.Info().Str("tag", "RESULT").Msgf("✨ 金色鱼截图已保存: %s", path)
}

// ---------------- RESULT ----------------

func (m *machine) handleResult(ctx context.Context, frame *image.RGBA) {
	if m.resultEnteredAt.IsZero() {
		m.resultEnteredAt = time.Now()
	}

	if _, ok := m.det.Result(frame); ok {
		m.recordOutcome(true, "", frame)
		m.pressEscUntilClear(ctx)
		m.resultEnteredAt = time.Time{}
		m.setState(IDLE)
		return
	}

	if _, ok := m.det.FishEscape(frame); ok {
		m.recordOutcome(false, "", frame)
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
