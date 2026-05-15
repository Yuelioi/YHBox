package fish

import (
	"context"
	"errors"
	"fmt"
	"image"
	"time"

	"yhbox/pkg/capture"
	"yhbox/pkg/input"
)

// Step 是流程的单步执行函数。返回 error 表示步骤失败（流程终止）；返回 nil 继续下一步。
type Step func(ctx context.Context, m *machine) error

// MissPolicy 控制 clickIfSeen 在检测不到目标时的行为。
type MissPolicy int

const (
	missFailPause MissPolicy = iota // 默认：弹窗等用户介入（真异常）
	missSkip                        // 跳过本步骤（游戏 UI 条件分支，不是异常）
	missStop                        // 终止流程
)

// flow 是一个线性步骤序列。OnDone 指定流程完成后切到哪个 State。
type flow struct {
	Name   string
	Steps  []Step
	OnDone State
}

// errFlowAbort 内部 sentinel：表示 ctx 取消 / 用户 ESC，调用方应停止后续 step。
var errFlowAbort = errors.New("flow aborted by user/ctx")

// DetectFn 检测器方法签名（用于 clickIfSeen 等）。
type DetectFn func(*Detector, *image.RGBA) (Hit, bool)

// runFlow 顺序执行 flow.Steps，任一步返回非 nil error 即终止。
// 无论成功失败，结束时把 machine.state 切到 OnDone（统一收口）。
func runFlow(ctx context.Context, m *machine, f flow) {
	logSafe(m, "==== flow=%s 开始 (%d 步) ====", f.Name, len(f.Steps))
	for i, step := range f.Steps {
		if ctx.Err() != nil || m.shouldExit() {
			logSafe(m, "flow=%s 在第 %d 步被取消", f.Name, i+1)
			break
		}
		if err := step(ctx, m); err != nil {
			if !errors.Is(err, errFlowAbort) {
				logSafe(m, "flow=%s 第 %d 步失败: %v", f.Name, i+1, err)
			}
			break
		}
	}
	logSafe(m, "flow=%s 结束", f.Name)
	m.setState(f.OnDone)
}

// logSafe 记录一条 state 日志（zerolog.Nop 会静默丢弃，测试用）。
func logSafe(m *machine, format string, args ...any) {
	m.logState(format, args...)
}

// ----- 原语 -----

// tap 按一个键。
func tap(key string) Step {
	return func(ctx context.Context, m *machine) error {
		input.Tap(m.hwnd, key, delayMid, delayShort)
		return nil
	}
}

// waitDur 等指定时长。ctx 取消 → errFlowAbort。
func waitDur(d time.Duration) Step {
	return func(ctx context.Context, m *machine) error {
		if !m.sleep(ctx, d) {
			return errFlowAbort
		}
		return nil
	}
}

// waitLong 等 delayLong (~2s)，UI 加载/动画。
func waitLong() Step {
	return waitDur(delayLong)
}

// waitUntilOr 在 d 内每 interval 调一次 check；check 返 true 提前结束。
// d 用完都没 check=true 就正常返回（不算失败，让后续 step 决定怎么办）。
//
// 用途：fish 长等待期（钓鱼/卖鱼/买鱼饵等）能持续检测中断条件（鱼脱钩 /
// 突然弹窗 / 状态变化），不必傻睡满整段。
//
// check 函数从 m.hwnd 抓帧 + det 判断，注意每次都会真的截屏，interval 别太短
// （建议 100ms 起，重 detect 的 50ms 起）。
func waitUntilOr(d, interval time.Duration, check func(*machine) bool) Step {
	return func(ctx context.Context, m *machine) error {
		deadline := time.Now().Add(d)
		for time.Now().Before(deadline) {
			remain := time.Until(deadline)
			step := interval
			if remain < step {
				step = remain
			}
			if !m.sleep(ctx, step) {
				return errFlowAbort
			}
			if check(m) {
				return nil
			}
		}
		return nil
	}
}

// logf 记录一条信息（test 模式 nil log 静默）。
func logf(format string, args ...any) Step {
	return func(ctx context.Context, m *machine) error {
		logSafe(m, format, args...)
		return nil
	}
}

// failPauseWaitOrAbort 是 missFailPause 路径的统一收尾：ctrl.Pause() 让
// bot 阻塞在 cond.Wait()，GUI 显示"已暂停"，用户可点[继续]/[停止]。
func failPauseWaitOrAbort(m *machine, label string) error {
	input.ReleaseAll(m.hwnd)
	if m.ctrl != nil {
		m.ctrl.Pause()
		m.log.Warn().Str("tag", "SYSTEM").Msgf("%s 检测失败，已自动暂停。手动处理后在 GUI 点[继续]，或点[停止]退出。", label)
		if !m.ctrl.WaitUnpause() {
			return errFlowAbort
		}
		m.log.Info().Str("tag", "SYSTEM").Msg("已恢复")
	}
	return fmt.Errorf("%s 失败暂停", label)
}

// clickIfSeen 抓帧 → detect → 命中则 click，未命中按 MissPolicy 处理。
func clickIfSeen(label string, det DetectFn, onMiss MissPolicy) Step {
	return func(ctx context.Context, m *machine) error {
		frame, err := capture.Frame(m.hwnd)
		if err != nil {
			return fmt.Errorf("抓帧失败: %w", err)
		}
		hit, ok := det(m.det, frame)
		if !ok {
			switch onMiss {
			case missSkip:
				logSafe(m, "%s 未出现 (conf=%.2f)，跳过（条件分支）", label, hit.Conf)
				return nil
			case missStop:
				logSafe(m, "%s 未识别 (conf=%.2f)，终止流程", label, hit.Conf)
				return fmt.Errorf("%s 未识别", label)
			case missFailPause:
				fallthrough
			default:
				logSafe(m, "%s 未识别 (conf=%.2f)", label, hit.Conf)
				return failPauseWaitOrAbort(m, label)
			}
		}
		logSafe(m, "点击 %s (%d,%d, conf=%.2f)", label, hit.ClientX, hit.ClientY, hit.Conf)
		input.Click(m.hwnd, hit.ClientX, hit.ClientY, delayShort, delayShort, delayShort)
		return nil
	}
}

// clickUntilGone 点目标 → 等 verifyDelay → 检测目标是否还在 → 还在就再点。
// 用于点击可能被游戏丢的情况（PostMessage 动画期被忽略、未获焦点等）。
//
// 行为：
//   - 首次检测 MISS：与 clickIfSeen 一致，走 onMiss 策略。
//   - 首次 HIT：点击 → sleep verifyDelay → 再次 detect。
//     若 MISS → 成功返回 nil；若 HIT → attempt++，未达 maxAttempts 则再点。
//   - 全部 maxAttempts 次点击后仍 HIT → 走 onMiss 策略（卡死）。
//
// maxAttempts 含首次点击，例：maxAttempts=3 表示首次 + 最多 2 次重试。
func clickUntilGone(label string, det DetectFn, onMiss MissPolicy, maxAttempts int, verifyDelay time.Duration) Step {
	return func(ctx context.Context, m *machine) error {
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			frame, err := capture.Frame(m.hwnd)
			if err != nil {
				return fmt.Errorf("抓帧失败: %w", err)
			}
			hit, ok := det(m.det, frame)
			if !ok {
				if attempt == 1 {
					// 首次就没看到 → 走 clickIfSeen 等价 miss 逻辑
					switch onMiss {
					case missSkip:
						logSafe(m, "%s 未出现 (conf=%.2f)，跳过（条件分支）", label, hit.Conf)
						return nil
					case missStop:
						logSafe(m, "%s 未识别 (conf=%.2f)，终止流程", label, hit.Conf)
						return fmt.Errorf("%s 未识别", label)
					case missFailPause:
						fallthrough
					default:
						logSafe(m, "%s 未识别 (conf=%.2f)", label, hit.Conf)
						return failPauseWaitOrAbort(m, label)
					}
				}
				// 重试中突然消失 → 点击生效了，成功
				logSafe(m, "%s 已消失（点击第 %d 次后生效）", label, attempt-1)
				return nil
			}
			// HIT — 点击
			if attempt == 1 {
				logSafe(m, "点击 %s (%d,%d, conf=%.2f)", label, hit.ClientX, hit.ClientY, hit.Conf)
			} else {
				logSafe(m, "%s 仍可见 conf=%.2f，重试 %d/%d", label, hit.Conf, attempt, maxAttempts)
			}
			input.Click(m.hwnd, hit.ClientX, hit.ClientY, delayShort, delayShort, delayShort)
			if !m.sleep(ctx, verifyDelay) {
				return errFlowAbort
			}
		}
		// 用光全部 attempts 仍 HIT
		logSafe(m, "%s 点击 %d 次仍未消失", label, maxAttempts)
		switch onMiss {
		case missSkip:
			return nil
		case missStop:
			return fmt.Errorf("%s 点击未生效", label)
		case missFailPause:
			fallthrough
		default:
			return failPauseWaitOrAbort(m, label)
		}
	}
}

// tapUntilGone 跟 clickUntilGone 同构，但用按键代替点击。
// 用于"按 ESC 关 UI"这类场景：ESC 在 UI 动画/聚焦切换期间有概率被游戏丢弃，
// 单次 tap + 单次 retryIfStillSeen 不够顽强 —— 改用 tap → verify → 还在就再 tap 循环。
//
// 行为：
//   - 首次 detect MISS：UI 不在，跳过（return nil，不走 onMiss——按键不像点击有"必须命中"
//     的语义，目标已经不在就当成功）。
//   - 首次 HIT：tap → sleep verifyDelay → 再次 detect。
//     若 MISS → 成功；若 HIT → attempt++，未达 maxAttempts 再 tap。
//   - 全部 maxAttempts 次 tap 后仍 HIT → 走 onMiss 策略（卡死，多半是 detect 误报或游戏挂死）。
//
// maxAttempts 含首次 tap。
func tapUntilGone(label, key string, det DetectFn, onMiss MissPolicy, maxAttempts int, verifyDelay time.Duration) Step {
	return func(ctx context.Context, m *machine) error {
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			frame, err := capture.Frame(m.hwnd)
			if err != nil {
				return fmt.Errorf("抓帧失败: %w", err)
			}
			hit, ok := det(m.det, frame)
			if !ok {
				if attempt == 1 {
					return nil // 目标本就不在，按键也不需要按
				}
				logSafe(m, "%s 已消失（%s 按第 %d 次后生效）", label, key, attempt-1)
				return nil
			}
			if attempt == 1 {
				logSafe(m, "%s 仍可见 conf=%.2f，按 %s 关闭", label, hit.Conf, key)
			} else {
				logSafe(m, "%s 仍可见 conf=%.2f，重试 %s %d/%d", label, hit.Conf, key, attempt, maxAttempts)
			}
			input.Tap(m.hwnd, key, delayMid, delayShort)
			if !m.sleep(ctx, verifyDelay) {
				return errFlowAbort
			}
		}
		logSafe(m, "%s %s %d 次仍未消失", label, key, maxAttempts)
		switch onMiss {
		case missSkip:
			return nil
		case missStop:
			return fmt.Errorf("%s 按键未生效", label)
		case missFailPause:
			fallthrough
		default:
			return failPauseWaitOrAbort(m, label)
		}
	}
}

// multiSlotClick 用 Detector.BaitInShop 这种"多候选选末位"的检测点击。
// 行为等价 clickIfSeen，但通过命名表达"这是多 slot 检测"。
func multiSlotClick(label string, det DetectFn) Step {
	return clickIfSeen(label, det, missFailPause)
}

// retryIfStillSeen 检测到 X 仍可见时执行 then 序列。常用于 ESC 关 UI 后再确认一次。
func retryIfStillSeen(label string, det DetectFn, then ...Step) Step {
	return func(ctx context.Context, m *machine) error {
		frame, err := capture.Frame(m.hwnd)
		if err != nil {
			return nil // 抓帧失败当 X 不见了，跳过 retry
		}
		if _, ok := det(m.det, frame); !ok {
			return nil
		}
		logSafe(m, "%s 仍可见，执行 retry %d 步", label, len(then))
		for _, s := range then {
			if err := s(ctx, m); err != nil {
				return err
			}
		}
		return nil
	}
}
