// Package rhythm 超强音自动音游：识别 4 轨命中并 PostMessage 按 D/F/J/K。
//
// 当前支持 1920×1080 + 1280×720。跨分辨率行为一致（异环 UI 等比缩放）。
//
// 检测原理：每轨 ROI 内"暗像素占比" ≥ DarkRatioThreshold 即视为音符到达
// （背景≈245、音符≈28，亮度差大，对 anti-aliasing 不敏感）。
// 配置外置：tools/rhythm/configs/<locale>/config.yaml；用户可在 exe 同目录
// 放 configs/rhythm/<locale>/config.yaml 覆盖部分字段（merge 语义）。
package rhythm

import (
	"image"
	"time"
)

// ResolutionKey 用作 Configs map 的 key。客户区像素 (W, H)。
type ResolutionKey struct {
	W, H int
}

// ResConfig 一个分辨率的全套配置。
// 当前只有 ROI 一组数据；亮度阈值/retrigger 间隔/key hold 都是包级常量
// （改这些就是改 bot 行为，应该走代码 review，不外置到 yaml）。
type ResConfig struct {
	ROIs [4]image.Rectangle // 4 轨命中圈在客户区的像素 ROI（11×21 居中命中点）
}

// Configs 各支持分辨率的 ROI 配置；由 config.go LoadConfig 从 yaml 加载。
var Configs map[ResolutionKey]ResConfig

// PickConfig 按窗口客户区尺寸查找对应配置。
// 未配置的分辨率 ok=false，caller 应该报错给用户而不是用默认值。
func PickConfig(w, h int) (ResConfig, bool) {
	cfg, ok := Configs[ResolutionKey{W: w, H: h}]
	return cfg, ok
}

// Keys 4 轨对应的键名（喂给 pkg/input.KeyDown/KeyUp）。
var Keys = [4]string{"d", "f", "j", "k"}

// TrackNames 用于日志标签。
var TrackNames = [4]string{"T1", "T2", "T3", "T4"}

// 检测 + 时序常量（行为参数，不外置到 YAML）。改这些就是改 bot 行为，走 code review。
const (
	// 主循环 tick 间隔。8ms = ~120Hz；Windows scheduler 抖动后实际 80~100Hz。
	TickInterval = 8 * time.Millisecond

	// 亮度阈值：像素 (R+G+B)/3 < BrightThresh 视为"暗"。背景亮 245，音符暗 28，
	// 取 100 留充足缓冲：阴影 / AA 边缘也会判暗，但不会把背景判暗。
	BrightThresh = 100

	// 暗像素占比阈值：ROI 内暗像素 ≥ DarkRatioThreshold 视为音符到达。
	// 11×21 ROI 共 231 像素；6% ≈ 14 像素，音符进入 ROI ~2 帧就够触发。
	DarkRatioThreshold float32 = 0.06

	// 长按 retrigger 间隔：音符持续占 ROI 时每隔 RetriggerInterval 重发一次按键，
	// 应付长按音符。85ms 实测对异环音游接受度合适。
	RetriggerInterval = 85 * time.Millisecond

	// KeyDown → KeyUp 间隔。ok-nte 实测 5ms 异环稳定接受。
	// 旧版用 30ms 阻塞主循环；现在异步 worker 不阻塞，hold 也短。
	KeyHoldMs = 5 * time.Millisecond
)
