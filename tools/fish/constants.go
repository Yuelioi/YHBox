package fish

import "time"

// 全部曾经的 fish.Config 字段。早期阶段无用户，写死成包级常量，不再可配置。
const (
	// 主循环节奏
	loopInterval   = 30 * time.Millisecond
	keepAliveEvery = 500 * time.Millisecond

	// 通用延迟（3 档）
	delayShort = 30 * time.Millisecond  // UI post-settle: click hold / activate / cursor settle
	delayMid   = 150 * time.Millisecond // 按键 hold
	delayLong  = 2 * time.Second        // UI 加载 / 动画等待

	// 置信度（3 档）
	confHigh   = float32(0.85) // 强信号 icon/上钩文字
	confNormal = float32(0.75) // 默认所有模板
	confBar    = 0.50          // bar 颜色分析（语义不同，单独）

	// 特殊命名 timeout
	baitWarningTimeout  = 75 * time.Second
	fishingTimeout      = 40 * time.Second
	barMissingTimeout   = 10 * time.Second
	resultDetectTimeout = 9 * time.Second
	minIconLatency      = 500 * time.Millisecond

	// 上钩检测
	hookStreakCount  = 2
	hookStreakWindow = 100 * time.Millisecond
	hookFMaxRetries  = 30
	hookFRetryDelay  = 2 * time.Second

	// 控制
	deadzoneRatio = 0.08

	// detect ROI 扩边
	roiPaddingPx = 30
)
