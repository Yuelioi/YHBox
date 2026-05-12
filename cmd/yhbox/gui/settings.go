package gui

import (
	"encoding/json"
	"os"
)

// Settings 是 GUI 持久化层 — 跟 fish.Config/cook.Config/piano.Config 解耦。
// 启动时 LoadSettings → 把字段写入各 tool 的 Config；运行时 SaveSettings 全量写回 JSON。
type Settings struct {
	Fish  FishSettings  `json:"fish"`
	Cook  CookSettings  `json:"cook"`
	Piano PianoSettings `json:"piano"`
	UI    UISettings    `json:"ui"`
}

type FishSettings struct {
	AutoSell bool `json:"auto_sell"`
}

type CookSettings struct {
	IntervalMs int `json:"interval_ms"`
}

type PianoSettings struct {
	Mode          int    `json:"mode"`           // 0=36键 1=21键
	TrackPick     int    `json:"track_pick"`     // -1=智能 -2=全部 >=0=具体 track 索引
	AutoTranspose bool   `json:"auto_transpose"` // 自动整曲八度对齐
	OctaveOffset  int    `json:"octave_offset"`  // 手动八度微调 -2..+2，叠加在自动偏移上
	MelodyOnly    bool   `json:"melody_only"`    // 同时音只保留最高音
	LastMidiPath  string `json:"last_midi_path"`
}

type UISettings struct {
	Window WindowSettings `json:"window"`
	Logger LoggerSettings `json:"logger"`
	Battle BattleSettings `json:"battle"`
}

type WindowSettings struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type LoggerSettings struct {
	Show       bool `json:"show"`
	AutoScroll bool `json:"auto_scroll"`
	ShowTime   bool `json:"show_time"`  // 显示 "HH:MM:SS" 前缀
	ShowTag    bool `json:"show_tag"`   // 显示 "[TAG]" 前缀
	WrapText   bool `json:"wrap_text"`  // true=自动换行 false=超宽 clip
	WriteFile  bool `json:"write_file"` // 是否把日志写到 logs/ 下的本地文件
}

// BattleSettings 战斗 tab 配置。
//   HotkeyEnabled: 启动时自动注册热键
//   HotkeyMods:    修饰键组合字符串，见 tab_battle.go 的 modCombos 列表
//                  合法值: "Ctrl" / "Ctrl+Shift" / "Ctrl+Alt" / "Shift+Alt" / "Ctrl+Shift+Alt"
//                  数字键固定 1~6，不可配
type BattleSettings struct {
	HotkeyEnabled bool   `json:"hotkey_enabled"`
	HotkeyMods    string `json:"hotkey_mods"`
}

// defaultSettings 返回内置默认值（缺失/损坏 JSON 用）。
func defaultSettings() *Settings {
	return &Settings{
		Fish:  FishSettings{AutoSell: true},
		Cook:  CookSettings{IntervalMs: 1500},
		Piano: PianoSettings{Mode: 0, TrackPick: -1, AutoTranspose: true, OctaveOffset: 0},
		UI: UISettings{
			// X=-1, Y=-1 是哨兵值，表示"从未保存过位置" → 启动时居中
			Window: WindowSettings{X: -1, Y: -1, Width: 600, Height: 650},
			Logger: LoggerSettings{Show: true, AutoScroll: true, ShowTime: true, ShowTag: true, WrapText: false, WriteFile: true},
			Battle: BattleSettings{HotkeyEnabled: false, HotkeyMods: "Ctrl+Shift"},
		},
	}
}

// LoadSettings 从 path 读 JSON。文件不存在或损坏都用默认值（不报错）。
func LoadSettings(path string) *Settings {
	s := defaultSettings()
	data, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	// 用 default 占位，再 unmarshal 覆盖 — 缺失字段保留默认
	_ = json.Unmarshal(data, s)
	// 归一：用户/旧版 settings 可能写出空 HotkeyMods，下游 modsByLabel 会 fallback，
	// 但 spec.Name / 日志会丑成 "+1"。直接在 load 边界修好。
	if s.UI.Battle.HotkeyMods == "" {
		s.UI.Battle.HotkeyMods = "Ctrl+Shift"
	}
	return s
}

// SaveSettings 把 s 全量序列化写到 path（覆盖）。
// 失败返回 error 由调用方 log，不阻塞 GUI。
func SaveSettings(path string, s *Settings) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
