package battle

import (
	"context"
	"fmt"
	"time"

	"github.com/lxn/win"

	"yhbox/pkg/capture"
	"yhbox/pkg/input"
	"yhbox/pkg/log"
)

// SwitchTeam 把上阵队伍切换到 target（1..6）。一次性流程，跑完即返。
//
// 流程：
//  1. tap L 进编队 UI
//  2. team_tag 检测确认 UI 已出现（未出现 → 报错返回；通常是玩家在弹窗里 L 被吞）
//  3. 按 teamSlotROIs[res][target-1] 中心点 click（不做模板匹配，纯坐标）
//  4. team_enabled 已可见 → 该队伍已是当前队伍，跳到 ESC
//  5. 否则 click team_select 触发切换
//  6. **必须** 看到 team_switch_success toast 才能 ESC —— 用户实测过，提前 ESC 会被吞，
//     切换不生效。轮询直到看到或超 switchToastTimeout
//  7. tap ESC + team_tag 兜底验证，仍可见再 ESC 一次
//
// logger / ctrl 可为 nil（独立调用场景）。返回 error 时调用方记日志即可。
func SwitchTeam(ctx context.Context, hwnd win.HWND, det *Detector, logger *log.Logger, target int) error {
	if target < 1 || target > TeamCount {
		return fmt.Errorf("队伍号 %d 越界 [1, %d]", target, TeamCount)
	}

	cw, ch, err := capture.ClientSize(hwnd)
	if err != nil {
		return fmt.Errorf("拿不到客户区大小: %w", err)
	}
	clickX, clickY, ok := TeamClickPos(cw, ch, target)
	if !ok {
		return fmt.Errorf("不支持的分辨率 %dx%d（teamSlotROIs 未配，1080p 模板待补）", cw, ch)
	}

	logf := func(format string, args ...any) {
		if logger != nil {
			logger.Log(log.SYSTEM, "[battle] "+format, args...)
		}
	}

	// 1. 进编队 UI
	logf("切换到队伍 %d：按 L 进编队", target)
	input.Tap(hwnd, "L", delayMid, delayShort)
	if !sleep(ctx, delayLong) {
		return ctx.Err()
	}

	// 2. 确认在 UI 内
	frame, err := capture.Frame(hwnd)
	if err != nil {
		return fmt.Errorf("抓帧失败: %w", err)
	}
	if _, ok := det.TeamTag(frame); !ok {
		input.ReleaseAll(hwnd)
		return fmt.Errorf("编队 UI 未出现（检查游戏是否在主界面、有无弹窗占用 L 键）")
	}

	// 3. 点目标队伍编号
	input.Click(hwnd, clickX, clickY, delayShort, delayShort, delayShort)
	if !sleep(ctx, teamClickSettle) {
		return ctx.Err()
	}

	// 4. 看是不是已经启用
	frame, err = capture.Frame(hwnd)
	if err != nil {
		return fmt.Errorf("抓帧失败: %w", err)
	}
	if _, alreadyEnabled := det.TeamEnabled(frame); alreadyEnabled {
		logf("队伍 %d 已是当前上阵队伍，无需切换", target)
	} else {
		// 5. 点选择队伍
		sel, selOK := det.TeamSelect(frame)
		if !selOK {
			return fmt.Errorf("找不到「选择队伍」按钮（页面状态异常）")
		}
		input.Click(hwnd, sel.ClientX, sel.ClientY, delayShort, delayShort, delayShort)

		// 6. 等 toast，超时返回错误（不 ESC，保留现场给用户看）
		// 连续抓帧失败 → 游戏窗口可能被切到后台/被遮挡，提前 return 给用户准确错误
		// 而不是耗满 3s 后报「没看到 toast」
		deadline := time.Now().Add(switchToastTimeout)
		seen := false
		captureFailStreak := 0
		const maxCaptureFail = 3
		for time.Now().Before(deadline) {
			if !sleep(ctx, switchToastPoll) {
				return ctx.Err()
			}
			f, err := capture.Frame(hwnd)
			if err != nil {
				captureFailStreak++
				if captureFailStreak >= maxCaptureFail {
					return fmt.Errorf("连续 %d 次抓帧失败（游戏窗口可能被切到后台/最小化）: %w", maxCaptureFail, err)
				}
				continue
			}
			captureFailStreak = 0
			if _, ok := det.TeamSwitchSuccess(f); ok {
				seen = true
				break
			}
		}
		if !seen {
			return fmt.Errorf("未检测到「上阵队伍切换成功」toast（不 ESC，请检查游戏画面）")
		}
		logf("队伍 %d 切换成功", target)
	}

	// 7. ESC 退出 + 兜底
	input.Tap(hwnd, "esc", delayMid, delayShort)
	if !sleep(ctx, 800*time.Millisecond) {
		return ctx.Err()
	}
	if f, err := capture.Frame(hwnd); err == nil {
		if _, ok := det.TeamTag(f); ok {
			input.Tap(hwnd, "esc", delayMid, delayShort)
			sleep(ctx, 800*time.Millisecond)
		}
	}
	return nil
}

func sleep(ctx context.Context, d time.Duration) bool {
	select {
	case <-time.After(d):
		return true
	case <-ctx.Done():
		return false
	}
}
