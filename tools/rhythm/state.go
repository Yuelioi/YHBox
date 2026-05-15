package rhythm

import "time"

// TrackState 单轨状态机（亮度版）。
//
// 转换：
//   - prev=false + darkRatio >= DarkRatioThreshold       → emit press (上升沿)
//   - prev=true  + darkRatio >= DarkRatioThreshold + 距上次按下 ≥ RetriggerInterval
//                                                          → emit press (长按 retrigger)
//   - 其它                                                → 不触发
//
// 跟旧版 Idle/Pressed + cnt 死区的对比：
//   - 删 LowThresh/HighThresh，单一 DarkRatioThreshold 阈值 (亮度检测稳定，不需要死区抗抖)
//   - 加 RetriggerInterval：长按音符 prev 一直 true，按 85ms 一次重发按键
//   - 删 StuckReset 兜底：边沿模型不会卡死（音符离开 ROI 自然 prev=false）
//
// Tick 返回 true 表示这一 tick 触发了按键事件。
type TrackState struct {
	prev      bool
	lastPress time.Time
}

// Tick 接受当前 darkRatio 和当前时间，更新状态机并返回是否触发按键。
func (s *TrackState) Tick(darkRatio float32, now time.Time) bool {
	has := darkRatio >= DarkRatioThreshold
	if !has {
		s.prev = false
		return false
	}
	// has=true
	if !s.prev {
		// 上升沿
		s.prev = true
		s.lastPress = now
		return true
	}
	// 持续 has=true：超过 retrigger 间隔再按一次
	if now.Sub(s.lastPress) >= RetriggerInterval {
		s.lastPress = now
		return true
	}
	return false
}
