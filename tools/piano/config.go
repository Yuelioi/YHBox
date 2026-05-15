// Package piano 自动弹钢琴：解析 MIDI → 时间对齐 → 后台按键。
//
// 游戏键位分两种模式：
//   - Keys36：3 八度 × 12 半音，自然音 7 键 + Shift 得到升半音
//   - Keys21：3 八度 × 7 自然音（白键），无修饰符
//
// 键位机制（36 键，游戏同时支持 Ctrl+key = 低半音，但跟 Shift+下方自然音等价，
// 算法只用 Shift 让和弦能在同一 modifier 组里齐按）。
package piano

import "time"

// Mode 键位模式。
type Mode int

const (
	Keys36 Mode = iota // 全半音 (3 八度 × 12)
	Keys21             // 仅自然音 (3 八度 × 7)
)

func (m Mode) String() string {
	if m == Keys21 {
		return "21键"
	}
	return "36键"
}

// 轨道选择常量（Config.TrackPick 取值）。
const (
	TrackPickSmart = -1 // 智能选轨：丢鼓 + 挑分数最高的 track
	TrackPickAll   = -2 // 全部 track 合并
)

// Config 弹奏配置。
type Config struct {
	Mode          Mode          // 键位模式
	TrackPick     int           // -1=智能 -2=全部 >=0=具体 track 索引
	AutoTranspose bool          // 自动整曲八度平移让最多音落进游戏范围（保持音程关系）
	OctaveOffset  int           // 手动八度微调，叠加在自动偏移之上 (-2..+2)
	MelodyOnly    bool          // 同时音只保留最高音（过滤伴奏，旋律干净，无和声）
	KeyHold       time.Duration // 单键按下保持时长
	ChordWindow   time.Duration // 和弦聚合窗口：事件距首音 ≤ 此值算同和弦
	KeyStagger    time.Duration // 组内 KeyDown/Up 之间的微小延迟，防游戏在同帧丢键
}

func DefaultConfig() *Config {
	return &Config{
		Mode:          Keys36,
		TrackPick:     TrackPickSmart,
		AutoTranspose: true,
		OctaveOffset:  0,
		MelodyOnly:    false,
		KeyHold:       30 * time.Millisecond,
		ChordWindow:   15 * time.Millisecond,
		KeyStagger:    2 * time.Millisecond,
	}
}
