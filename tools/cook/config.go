package cook

import "time"

// HammerPoint 一个分辨率下锤子的客户区像素坐标 (x, y)。
type HammerPoint struct {
	X, Y int
}

// ResolutionKey Configs map 的 key。
type ResolutionKey struct {
	W, H int
}

// Configs 各支持分辨率的锤子坐标；由 LoadConfig 从 yaml 加载。
var Configs map[ResolutionKey]HammerPoint

// PickHammer 按窗口客户区尺寸查锤子坐标。未配置的分辨率 ok=false。
func PickHammer(w, h int) (HammerPoint, bool) {
	p, ok := Configs[ResolutionKey{W: w, H: h}]
	return p, ok
}

// Config 是运行时配置（GUI 可调字段 + click 时序）。
// 锤子坐标走 Configs 不进这里。
type Config struct {
	Interval          time.Duration // 两次 click 之间的间隔
	ClickHold         time.Duration // DOWN+UP 后的 sleep，让游戏 tick 处理 click
	ActivateDelay     time.Duration // FakeActivate 后等多久让 Slate 翻 IsActive
	CursorSettleDelay time.Duration // SetCursorPos 后等 OS 发自然 MOUSEMOVE
}

func DefaultConfig() *Config {
	return &Config{
		Interval:          1500 * time.Millisecond,
		ClickHold:         30 * time.Millisecond,
		ActivateDelay:     30 * time.Millisecond,
		CursorSettleDelay: 30 * time.Millisecond,
	}
}
