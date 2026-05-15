package actiontransform

import "yhbox/internal/services/actions"

// mouseMoveRelMergeSleepCapMs 合并 mouse_move_rel 时容忍的相邻 sleep 上限。
// 80ms：只合并真正相邻的 raw 桶（50ms 桶 + 偶尔 30ms 调度抖动）。
// 早期设 300ms 太激进 —— 战斗连续转视角的多个 burst 被合成一条均匀插值的巨型
// step，回放节奏全丢。80ms 既保 burst 节奏，又避免每帧一条 step 爆炸。
const mouseMoveRelMergeSleepCapMs = 80

// MergeMouseMoveRel 把连续 mouse_move_rel（可被 < 300ms sleep 间隔）合并成单条。
// PairEvents 的 50ms 桶切得细碎；这一 pass 在 step 层面做粗合并。
//
// 累加：dx/dy 求和，durationMs 包含中间所有 sleep 的总时长（保留 swipe 节奏）。
//
// 示例：
//   mouse_move_rel(dx=3, 16ms)
//   sleep(80ms)
//   mouse_move_rel(dx=1, dy=1, 16ms)
//   sleep(297ms)        ← 超 300ms 不合并
//   mouse_move_rel(dx=-8, dy=1, 16ms)
//
//   → mouse_move_rel(dx=4, dy=1, 112ms) / sleep(297ms) / mouse_move_rel(dx=-8, dy=1, 16ms)
func MergeMouseMoveRel(steps []actions.Step) []actions.Step {
	out := make([]actions.Step, 0, len(steps))
	i := 0
	for i < len(steps) {
		if steps[i].Kind != actions.StepMouseMoveRel {
			out = append(out, steps[i])
			i++
			continue
		}
		// 从 i 起扫连续的 mouse_move_rel（可被短 sleep 隔开）
		sumDx := steps[i].Dx
		sumDy := steps[i].Dy
		sumDur := steps[i].DurationMs
		merged := 1
		j := i + 1
		for j < len(steps) {
			if steps[j].Kind == actions.StepMouseMoveRel {
				sumDx += steps[j].Dx
				sumDy += steps[j].Dy
				sumDur += steps[j].DurationMs
				merged++
				j++
				continue
			}
			// 短 sleep + 后面是 mouse_move_rel → 吞进来
			if steps[j].Kind == actions.StepSleep &&
				steps[j].DurationMs <= mouseMoveRelMergeSleepCapMs &&
				j+1 < len(steps) && steps[j+1].Kind == actions.StepMouseMoveRel {
				sumDur += steps[j].DurationMs
				j++
				continue
			}
			break
		}
		if merged == 1 {
			out = append(out, steps[i])
		} else {
			out = append(out, actions.Step{
				Kind:       actions.StepMouseMoveRel,
				Dx:         sumDx,
				Dy:         sumDy,
				DurationMs: sumDur,
			})
		}
		i = j
	}
	return out
}
