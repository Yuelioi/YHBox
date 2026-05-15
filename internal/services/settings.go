package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"yhbox/pkg/locale"
)

// settingsFileName / settingsFilePath 跟 walk 时代保持兼容：exe 同目录的 settings.json。
const settingsFileName = "settings.json"

var settingsFilePath = func() string {
	exe, err := os.Executable()
	if err != nil {
		return settingsFileName
	}
	return filepath.Join(filepath.Dir(exe), settingsFileName)
}()

// Settings 是持久化层 schema。
type Settings struct {
	Fish    FishSettings    `json:"fish"`
	Cook    CookSettings    `json:"cook"`
	Piano   PianoSettings   `json:"piano"`
	UI      UISettings      `json:"ui"`
	Locale  string          `json:"locale"`  // "zh" | "en"；i18n 口子，目前默认且仅 zh
	Capture CaptureSettings `json:"capture"` // 截屏后端选择
}

// CaptureSettings 选哪个截屏后端。Method 改这个要重启 exe 才生效；DumpDebug 即时生效。
//   - "auto": 启动期根据 OS build 自动选——Win11/Server 2022 (build>=20348) 走 WGC，
//     老系统走 GDI。新装默认就是这个。
//   - "gdi": PrintWindow + GDI，兼容性最好，但 D3D 游戏后台容易拿冻结帧
//   - "wgc": Windows Graphics Capture（Win10 1903+），后台抓帧稳定，
//     依赖 bin/capture_wgc.dll；Win10 上有黄框关不掉
//   - "mock": 从磁盘 PNG 序列回放（调试用），需把帧放进 bin/mock-frames/
//     或设置环境变量 YHBOX_MOCK_DIR 指向目录
type CaptureSettings struct {
	Method    string `json:"method"`    // "auto" | "gdi" | "wgc" | "mock"
	DumpDebug bool   `json:"dumpDebug"` // bot detect 关键路径 dump 带框 PNG 到 bin/captures/，调试用
}

type FishSettings struct {
	AutoSell      bool `json:"autoSell"`
	CaptureGolden bool `json:"captureGolden"` // 钓上金色鱼时把结算画面截图保存到 captures/
}

type CookSettings struct {
	IntervalMs int `json:"intervalMs"`
}

type PianoSettings struct {
	Mode          int    `json:"mode"`          // 0=36键 1=21键
	TrackPick     int    `json:"trackPick"`     // -1=智能 -2=全部 >=0=具体 track index
	AutoTranspose bool   `json:"autoTranspose"` // 自动整曲八度对齐
	OctaveOffset  int    `json:"octaveOffset"`  // -2..+2，叠加在自动偏移上
	MelodyOnly    bool   `json:"melodyOnly"`    // 同时音只留最高
	LastMidiPath  string `json:"lastMidiPath"`
}

type UISettings struct {
	Logger         LoggerSettings `json:"logger"`
	Battle         BattleSettings `json:"battle"`
	Window         WindowSettings `json:"window"`
	Autostart      bool           `json:"autostart"`      // 开机自启（HKCU Run 注册表）
	MinimizeToTray bool           `json:"minimizeToTray"` // 关闭按钮 → 隐藏到托盘
	// ActionStopHotkey 全局打断动作热键（默认 "Ctrl+Shift+F9"）。
	// 改动需重启生效（main.go 启动期注册一次，不监听 settings 变化）。
	ActionStopHotkey string `json:"actionStopHotkey"`
	// MouseCounts360 鼠标转身 360° 累积的 HID counts（用户在游戏里实测得到）。
	// 跨电脑共享脚本时用：录制时存源机基准，回放时 dx *= localCounts / sourceCounts。
	// 0 = 未校准（回放不做缩放）。
	MouseCounts360 int `json:"mouseCounts360"`
}

// WindowSettings 记录用户调到的窗口尺寸。main.go 启动时读取，
// WindowEndResize 事件触发时写回 settings.json。
// 不存 x/y：多屏拔插容易把窗口扔到屏幕外，用户自己点 Center() 更省事。
type WindowSettings struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type LoggerSettings struct {
	Show       bool `json:"show"`
	AutoScroll bool `json:"autoScroll"`
	ShowTime   bool `json:"showTime"`
	ShowTag    bool `json:"showTag"`
	WrapText   bool `json:"wrapText"`
	WriteFile  bool `json:"writeFile"`
}

type BattleSettings struct {
	HotkeyEnabled bool   `json:"hotkeyEnabled"`
	HotkeyMods    string `json:"hotkeyMods"`
}

// defaultSettings 返回内置默认值。
func defaultSettings() *Settings {
	return &Settings{
		Fish:  FishSettings{AutoSell: true, CaptureGolden: false},
		Cook:  CookSettings{IntervalMs: 1500},
		Piano: PianoSettings{Mode: 0, TrackPick: -1, AutoTranspose: true, OctaveOffset: 0},
		UI: UISettings{
			Logger:           LoggerSettings{Show: true, AutoScroll: true, ShowTime: true, ShowTag: true, WrapText: false, WriteFile: true},
			Battle:           BattleSettings{HotkeyEnabled: false, HotkeyMods: "Ctrl+Shift"},
			Window:           WindowSettings{Width: 1100, Height: 720},
			ActionStopHotkey: "Ctrl+Shift+F9",
		},
		Locale: "zh",
		// 默认 auto：启动期 main.go 看 OS build 决定 Win10 走 GDI / Win11+ 走 WGC。
		// 用户嫌弃自动选的可以去设置硬切 gdi/wgc/mock。
		Capture: CaptureSettings{Method: "auto", DumpDebug: false},
	}
}

// LoadSettings 从 path 读 JSON。文件不存在或损坏都用默认值（不报错）。
// Validate 失败也用默认值。
func LoadSettings(path string) *Settings {
	s := defaultSettings()
	data, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	loaded := &Settings{}
	if err := json.Unmarshal(data, loaded); err != nil {
		return s
	}
	// 归一：空 HotkeyMods 落到 default
	if loaded.UI.Battle.HotkeyMods == "" {
		loaded.UI.Battle.HotkeyMods = "Ctrl+Shift"
	}
	// 归一：空 Locale 落到 default "zh"（旧 settings.json 没这字段）
	if loaded.Locale == "" {
		loaded.Locale = "zh"
	}
	// 归一：空 Capture.Method 落到 default "auto"（旧 settings.json 没这字段）
	if loaded.Capture.Method == "" {
		loaded.Capture.Method = "auto"
	}
	// 归一：旧 settings 没 window 字段或被改成 0，回落到默认尺寸
	if loaded.UI.Window.Width <= 0 || loaded.UI.Window.Height <= 0 {
		loaded.UI.Window.Width = 1100
		loaded.UI.Window.Height = 720
	}
	// 归一：旧 settings 没 actionStopHotkey 字段，回落默认 F9
	if loaded.UI.ActionStopHotkey == "" {
		loaded.UI.ActionStopHotkey = "Ctrl+Shift+F9"
	}
	if err := loaded.Validate(); err != nil {
		return s
	}
	return loaded
}

// SaveSettings 把 s 全量序列化写到 path（覆盖）。
func SaveSettings(path string, s *Settings) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Clone deep copy。SettingsService.Update 的 clone→merge→Validate→swap→save 流程用。
func (s *Settings) Clone() *Settings {
	data, _ := json.Marshal(s)
	var out Settings
	_ = json.Unmarshal(data, &out)
	return &out
}

// Validate 字段范围检查。malformed patch 应用后失败 → SettingsService.Update 拒绝 swap。
func (s *Settings) Validate() error {
	if s.Cook.IntervalMs < 100 || s.Cook.IntervalMs > 10000 {
		return fmt.Errorf("cook.intervalMs out of range [100, 10000]: got %d", s.Cook.IntervalMs)
	}
	if s.Piano.Mode != 0 && s.Piano.Mode != 1 {
		return fmt.Errorf("piano.mode must be 0 or 1: got %d", s.Piano.Mode)
	}
	if s.Piano.TrackPick < -2 {
		return fmt.Errorf("piano.trackPick must be >= -2: got %d", s.Piano.TrackPick)
	}
	if s.Piano.OctaveOffset < -2 || s.Piano.OctaveOffset > 2 {
		return fmt.Errorf("piano.octaveOffset out of range [-2, 2]: got %d", s.Piano.OctaveOffset)
	}
	if !validHotkeyMods(s.UI.Battle.HotkeyMods) {
		return fmt.Errorf("ui.battle.hotkeyMods invalid: %q", s.UI.Battle.HotkeyMods)
	}
	if !locale.Valid(s.Locale) {
		return fmt.Errorf("locale 不在支持列表内: got %q", s.Locale)
	}
	switch s.Capture.Method {
	case "auto", "gdi", "wgc", "mock":
		// ok
	default:
		return fmt.Errorf("capture.method 必须是 \"auto\" / \"gdi\" / \"wgc\" / \"mock\"，got %q", s.Capture.Method)
	}
	// 窗口尺寸不强制下限（main.go 用 SetMinSize 兜底），但拦住明显无效值
	if s.UI.Window.Width < 100 || s.UI.Window.Height < 100 {
		return fmt.Errorf("ui.window 尺寸过小: %dx%d", s.UI.Window.Width, s.UI.Window.Height)
	}
	return nil
}

// ValidHotkeyMods 公开版本给 internal/bots 复用。
func ValidHotkeyMods(label string) bool { return validHotkeyMods(label) }

func validHotkeyMods(label string) bool {
	switch label {
	case "Ctrl", "Ctrl+Shift", "Ctrl+Alt", "Shift+Alt", "Ctrl+Shift+Alt":
		return true
	}
	return false
}

// ApplyMergePatch 把 RFC7386 patch 应用到 s 上。语义：
//   - object 字段递归 merge
//   - array 整体替换
//   - scalar 覆盖
//   - null 删除字段（RFC7386 标准）
// 流程：把 s 序列化 → MergePatch(jsonBytes, patch) → 反序列化回 s。
// 用 DisallowUnknownFields 防 patch 写错字段名。
func ApplyMergePatch(s *Settings, patch json.RawMessage) error {
	cur, err := json.Marshal(s)
	if err != nil {
		return err
	}
	merged, err := jsonpatch.MergePatch(cur, patch)
	if err != nil {
		return fmt.Errorf("RFC7386 merge failed: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(merged))
	dec.DisallowUnknownFields()
	*s = Settings{}
	if err := dec.Decode(s); err != nil {
		return fmt.Errorf("merged settings unmarshal failed: %w", err)
	}
	return nil
}
