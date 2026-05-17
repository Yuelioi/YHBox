// Package runtime 提供 QPC 时钟和 WaitUntil 调度原语.
package runtime

import "time"

// PlaybackPolicy 可调度容忍度. 不同 backend 可独立配置.
type PlaybackPolicy struct {
	EarlyToleranceUs        uint64
	LateToleranceUs         uint64
	SpinWaitThresholdUs     uint64
	CoarseSleepUnderShootUs uint64
}

func DefaultPlaybackPolicy() PlaybackPolicy {
	return PlaybackPolicy{
		EarlyToleranceUs:        0,
		LateToleranceUs:         2000,
		SpinWaitThresholdUs:     3000,
		CoarseSleepUnderShootUs: 2000,
	}
}

// WaitUntil 等到 QPC 达到 targetUs.
// 距离 > spinWaitThreshold 时 time.Sleep(remaining - underShoot); 否则 busy-wait.
func WaitUntil(targetUs uint64, policy PlaybackPolicy) {
	for {
		now := QPCMicros()
		if now >= targetUs {
			return
		}
		remaining := targetUs - now
		if remaining > policy.SpinWaitThresholdUs {
			sleepUs := remaining - policy.CoarseSleepUnderShootUs
			time.Sleep(time.Duration(sleepUs) * time.Microsecond)
		} else {
			for QPCMicros() < targetUs {
				// busy-wait, 不调 runtime.Gosched 避免调度回来延迟
			}
			return
		}
	}
}
