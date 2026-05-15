package fish

import "sync/atomic"

// BarROI 是耐力条 ROI 在某个分辨率下的精确像素定位。
//   - Resolution: [W, H]
//   - BBox:       [x1, y1, x2, y2]
type BarROI struct {
	Resolution [2]int
	BBox       [4]int
}

// BarROIs 由 LoadConfig 从 configs/<loc>/config.yaml 填充；未调 LoadConfig 时为空。
var BarROIs []BarROI

// Config 是 fish 模块的运行时**用户偏好**（不是 yaml 配置）。
// 早期阶段无大量可调参数 — 延迟/置信度/timeout 写死成 constants.go 包级常量。
// 当前用户偏好通过 GUI 调度，全部用 atomic 让 GUI 写、bot 读 race-free。
type Config struct {
	AutoSell      atomic.Bool // 鱼仓满 (1000) 时是否自动开商店卖鱼
	CaptureGolden atomic.Bool // 钓上金色鱼时把结算画面 frame 存成 PNG 到 captures/
}

func DefaultConfig() *Config {
	c := &Config{}
	c.AutoSell.Store(true)
	c.CaptureGolden.Store(false)
	return c
}
