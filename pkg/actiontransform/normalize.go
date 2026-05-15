package actiontransform

import "sort"

// Normalize 按 TimestampMs 排序事件。
// 抖动剔除目前只在 PairEvents 出 step 时按 gap 处理；这里仅排序，保留所有 raw events 给后续 pass 决策。
func Normalize(events []Event) []Event {
	out := make([]Event, len(events))
	copy(out, events)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].TimestampMs < out[j].TimestampMs
	})
	return out
}
