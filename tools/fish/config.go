package fish

import "sync/atomic"

// BarROI 是耐力条 ROI 在某个分辨率下的精确像素定位。
//   - Resolution: [W, H]
//   - BBox:       [x1, y1, x2, y2]，x2/y2 互斥（width=x2-x1, height=y2-y1）
type BarROI struct {
	Resolution [2]int
	BBox       [4]int
}

// Config 是 fish 模块的运行时配置。
// 早期阶段无大量可调参数 — 延迟/置信度/timeout 写死成 constants.go 包级常量。
// 当前仅 AutoSell 是用户偏好（鱼仓满后行为），通过 GUI 调度。
//
// AutoSell 是 atomic.Bool：GUI 线程写、bot goroutine 读，atomic 保证无 race。
type Config struct {
	AutoSell atomic.Bool // 鱼仓满 (1000) 时是否自动开商店卖鱼
}

func DefaultConfig() *Config {
	c := &Config{}
	c.AutoSell.Store(true)
	return c
}
