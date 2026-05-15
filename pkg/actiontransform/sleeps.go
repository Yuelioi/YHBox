package actiontransform

import "yhbox/internal/services/actions"

// MergeSleeps 合并相邻 step_sleep；剔除 durationMs<50 的 sleep。
// PairEvents 已经按 50ms 阈值发 sleep，这里二次保险 + 合并连续 sleep。
func MergeSleeps(steps []actions.Step) []actions.Step {
	out := make([]actions.Step, 0, len(steps))
	for _, s := range steps {
		if s.Kind == actions.StepSleep {
			if s.DurationMs < 50 {
				continue
			}
			if n := len(out); n > 0 && out[n-1].Kind == actions.StepSleep {
				out[n-1].DurationMs += s.DurationMs
				continue
			}
		}
		out = append(out, s)
	}
	return out
}
