package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/pkg/locale"
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
	UI      UISettings      `json:"ui"`
	Locale  string          `json:"locale"`  // "zh" | "en"；i18n 口子，目前默认且仅 zh
	Capture CaptureSettings `json:"capture"` // 截屏后端选择
	AI      AISettings      `json:"ai"`
}

type AISettings struct {
	Profiles []AIModelSettings `json:"profiles"`
}

// AIModelSettings is installation metadata only. Credential material lives in
// AISecrets; model calls bind this logical slot through the trusted Host
// Profile frozen at application startup.
type AIModelSettings struct {
	Slot            string                 `json:"slot"`
	Label           string                 `json:"label"`
	Provider        ai.ProviderKind        `json:"provider"`
	Model           string                 `json:"model"`
	MaxOutputTokens int64                  `json:"maxOutputTokens"`
	Capabilities    ai.ProfileCapabilities `json:"capabilities"`
	Evaluation      ai.EvaluationStatus    `json:"evaluation"`
	EvaluationSuite artifact.Digest        `json:"evaluationSuite,omitempty"`
	WorkflowConsent artifact.Digest        `json:"workflowConsent,omitempty"`
}

func (p AIModelSettings) profileDraft() ai.ModelProfileDraft {
	return ai.ModelProfileDraft{
		Provider: p.Provider, Model: p.Model, Capabilities: p.Capabilities,
		MaxOutputTokens: p.MaxOutputTokens, Evaluation: p.Evaluation, EvaluationSuite: p.EvaluationSuite,
		ProviderMetadata: json.RawMessage(`{}`),
	}
}

func (p AIModelSettings) installationDraft() ai.InstallationDraft {
	return ai.InstallationDraft{Slot: p.Slot, Profile: p.profileDraft(), Consent: p.WorkflowConsent}
}

func (s AISettings) InstallationDrafts() []ai.InstallationDraft {
	result := make([]ai.InstallationDraft, 0, len(s.Profiles))
	for _, profile := range s.Profiles {
		result = append(result, profile.installationDraft())
	}
	return result
}

// CaptureSettings 选哪个截屏后端。Method 改这个要重启 exe 才生效；DumpDebug 即时生效。
//   - "auto": 启动期根据 OS build 自动选——Win11/Server 2022 (build>=20348) 走 WGC，
//     老系统走 GDI。新装默认就是这个。
//   - "gdi": PrintWindow + GDI，兼容性最好，但 D3D 游戏后台容易拿冻结帧
//   - "wgc": Windows Graphics Capture（Win10 1903+），后台抓帧稳定，
//     依赖 bin/capture_wgc.dll；Win10 上有黄框关不掉
//   - "mock": 从磁盘 PNG 序列回放（调试用），需把帧放进 bin/mock-frames/
//     或设置环境变量 YOTTA_MOCK_DIR 指向目录
type CaptureSettings struct {
	Method    string `json:"method"`    // "auto" | "gdi" | "wgc" | "mock"
	DumpDebug bool   `json:"dumpDebug"` // bot detect 关键路径 dump 带框 PNG 到 bin/captures/，调试用
}

type UISettings struct {
	Logger         LoggerSettings `json:"logger"`
	Window         WindowSettings `json:"window"`
	Autostart      bool           `json:"autostart"`      // 开机自启（HKCU Run 注册表）
	MinimizeToTray bool           `json:"minimizeToTray"` // 关闭按钮 → 隐藏到托盘
	// ActionStopHotkey 全局打断动作热键（默认 "Ctrl+Shift+F9"）。
	// 改动需重启生效（main.go 启动期注册一次，不监听 settings 变化）。
	ActionStopHotkey string `json:"actionStopHotkey"`
	// CalibrateHotkey DPI 校准启动/停止 toggle 热键（默认 "F8"）。
	// 「快捷键」页 rebind 经 onSystemHotkeyChange 写回, 启动期 main.go 读这字段注册。
	CalibrateHotkey string `json:"calibrateHotkey"`
	// WindowCaptureHotkey 窗口捕获键（默认 "F9"）—— NodeInspector「捕获目标窗口」按下它抓前台游戏窗口。
	// registry 值持有者条目 (tools.window-capture)，「快捷键」页可 rebind；捕获时临时 OS 注册它。
	WindowCaptureHotkey string `json:"windowCaptureHotkey"`
	// RecordingStopHotkey 录制停止热键 (LL hook 拦截, 不透传游戏). 默认 "F12".
	// 改动需重启生效 (Service 注入 adapter 启动期读一次).
	RecordingStopHotkey string `json:"recordingStopHotkey"`
	// RecordingPauseHotkey 录制暂停/继续切换热键 (LL hook 拦截, 不透传游戏也不进 clip). 默认 "F11".
	// 录制中按 → 暂停; 暂停中按 → 走 3s 倒计时后继续. 走热键避免点 HUD 按钮的点击被录进 clip.
	RecordingPauseHotkey string `json:"recordingPauseHotkey"`
	// RecordingMouseMode 录制时鼠标语义. "relative" (FPS 相机 RawDelta, 默认) / "absolute" (UI 点击 MouseMove screen px).
	// 决定 recorder drainLoop 是否窗口过滤. 改动需重启生效.
	RecordingMouseMode string `json:"recordingMouseMode"`
	// MouseProfiles 命名鼠标校准 profile 列表。同一台机器玩不同游戏 (异环 vs 原神) 内灵敏度不同 →
	// 360° counts 不同, 装不进单个全局值。每个 profile 存一个 {Label, Counts360}。
	// 跨电脑共享脚本时, counts360 是 (本机 × 游戏内灵敏度) 的函数, 本就 per-machine, 不跨机同步。
	MouseProfiles []MouseProfile `json:"mouseProfiles"`
	// ActiveMouseProfile 指向 MouseProfiles 里某个 Label, 标记"当前默认"。
	// 录制兜底 getter (无 MouseCalibration 节点时) + 新建节点默认值 + "同步所有容器" 用它。
	// 空 / 指向不存在的 label → ActiveMouseCounts360() 兜底 (见该方法)。
	ActiveMouseProfile string `json:"activeMouseProfile"`
	// LauncherItems 悬浮窗启动器的块序列（用户在设置里编排，有序，积木式）。空 = 空启动器。
	LauncherItems []LauncherBlock `json:"launcherItems"`
	// LauncherDisplay 按钮显示：""/"both"=图标+文字 | "icon"=仅图标 | "text"=仅文字。
	LauncherDisplay string `json:"launcherDisplay"`
	// LauncherToggleHotkey 呼出/隐藏悬浮窗的全局热键（空 = 未绑）。
	LauncherToggleHotkey string `json:"launcherToggleHotkey"`
}

// MouseProfile 一个命名校准档。Counts360 = 原地转身 360° 鼠标硬件累积的 |dx| (用户在游戏里实测)。
type MouseProfile struct {
	Label     string `json:"label"`
	Counts360 int    `json:"counts360"`
}

// LauncherBlock 悬浮窗启动器的一个块（积木式编排）。Type 决定形态：
//   - "container": 一个容器按钮（ContainerID + 可选 Icon/Label）
//   - "label":     文字标题（Label = 标题文字），占整行
//   - "hsep":      水平分隔符（占整行的横线，把后面的块挤到下一排）
//   - "vsep":      垂直分隔符（同排按钮之间的竖线）
//
// 只存编排数据；容器名/状态/热键运行时拉。
type LauncherBlock struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	ContainerID string `json:"containerId,omitempty"` // type=container
	Icon        string `json:"icon,omitempty"`        // type=container 自定义图标（完整 tabler 名）
	Label       string `json:"label,omitempty"`       // container 自定义名 / label 标题文字
}

// ActiveMouseCounts360 返回当前生效 profile 的 counts360。
//   - ActiveMouseProfile 命中某个 profile.Label → 返该值
//   - 没命中但恰好只有一个 profile → 返那个 (单 profile 场景免设 active 的便利, 非兼容垫片)
//   - 否则 (空列表 / 多 profile 没选 active 没命中) → 0 (= 未校准, 回放不缩放)
func (s *Settings) ActiveMouseCounts360() int {
	for _, p := range s.UI.MouseProfiles {
		if p.Label == s.UI.ActiveMouseProfile {
			return p.Counts360
		}
	}
	if len(s.UI.MouseProfiles) == 1 {
		return s.UI.MouseProfiles[0].Counts360
	}
	return 0
}

// WindowSettings 记录用户调到的窗口尺寸。main.go 启动时读取，
// WindowEndResize 事件触发时写回 settings.json。
// 不存 x/y：多屏拔插容易把窗口扔到屏幕外，用户自己点 Center() 更省事。
type WindowSettings struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type LoggerSettings struct {
	Enabled       bool   `json:"enabled"`   // master diagnostic switch; false stops production at source
	LiveView      bool   `json:"liveView"`  // stream normalized batches to the main webview
	Level         string `json:"level"`     // debug | info | warn | error
	PanelOpen     bool   `json:"panelOpen"` // 面板折叠态 (false=折叠)
	AutoScroll    bool   `json:"autoScroll"`
	ShowTime      bool   `json:"showTime"`
	ShowTag       bool   `json:"showTag"`
	WrapText      bool   `json:"wrapText"`
	WriteFile     bool   `json:"writeFile"`
	ShowNodeEnter bool   `json:"showNodeEnter"` // 面板显示节点切换 trace (默认 false 隐藏)
	FileDir       string `json:"fileDir"`       // 写文件目录, 空 = 默认 "logs"
}

// defaultSettings 返回内置默认值。
func defaultSettings() *Settings {
	return &Settings{
		UI: UISettings{
			Logger: LoggerSettings{
				Enabled: true, LiveView: true, Level: "info",
				PanelOpen: true, AutoScroll: true, ShowTime: true, ShowTag: true,
				WrapText: false, WriteFile: true, FileDir: "logs",
			},
			Window:               WindowSettings{Width: 1100, Height: 720},
			ActionStopHotkey:     "Ctrl+Shift+F9",
			CalibrateHotkey:      "F8",
			WindowCaptureHotkey:  "F9",
			RecordingStopHotkey:  "F12",
			RecordingPauseHotkey: "F11",
			RecordingMouseMode:   "relative",
		},
		Locale: "zh",
		// 默认 auto：启动期 main.go 看 OS build 决定 Win10 走 GDI / Win11+ 走 WGC。
		// 用户嫌弃自动选的可以去设置硬切 gdi/wgc/mock。
		Capture: CaptureSettings{Method: "auto", DumpDebug: false},
		AI:      AISettings{Profiles: []AIModelSettings{}},
	}
}

// LoadSettings 从 path 读 JSON。文件不存在/损坏/Validate 失败都用默认值（不报错）。
// 直接 unmarshal 进 defaultSettings() 基底：缺失字段天然保留默认，无需逐字段空值回落。
func LoadSettings(path string) *Settings {
	data, err := os.ReadFile(path)
	if err != nil {
		return defaultSettings()
	}
	s := defaultSettings()
	if err := json.Unmarshal(data, s); err != nil {
		return defaultSettings() // 损坏 → 全新默认（s 已被部分改写，不能返回）
	}
	// Window 显式 0/负值会覆盖默认 → 兜底（非空值垫片，保留）
	if s.UI.Window.Width <= 0 || s.UI.Window.Height <= 0 {
		s.UI.Window.Width = 1100
		s.UI.Window.Height = 720
	}
	// RecordingMouseMode 枚举校验：非 relative/absolute → relative（非空值垫片，保留）
	if s.UI.RecordingMouseMode != "relative" && s.UI.RecordingMouseMode != "absolute" {
		s.UI.RecordingMouseMode = "relative"
	}
	if err := s.Validate(); err != nil {
		return defaultSettings()
	}
	return s
}

// SaveSettings writes a complete snapshot through a same-directory temporary
// file, fsync, atomic replace, and host-specific directory durability barrier.
func SaveSettings(path string, s *Settings) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return saveSettingsBytes(path, data, replaceSettingsFile, syncSettingsDir)
}

type settingsCommittedError struct{ err error }

func (e *settingsCommittedError) Error() string   { return e.err.Error() }
func (e *settingsCommittedError) Unwrap() error   { return e.err }
func (e *settingsCommittedError) Committed() bool { return true }

func settingsSaveCommitted(err error) bool {
	var committed *settingsCommittedError
	return errors.As(err, &committed)
}

func saveSettingsBytes(
	path string,
	data []byte,
	replace func(oldPath, newPath string) error,
	syncDir func(string) error,
) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create settings directory: %w", err)
	}
	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat settings file: %w", err)
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create settings temp file: %w", err)
	}
	tmpPath := tmp.Name()
	committed := false
	defer func() {
		_ = tmp.Close()
		if !committed {
			_ = os.Remove(tmpPath)
		}
	}()
	if err := tmp.Chmod(mode); err != nil {
		return fmt.Errorf("chmod settings temp file: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("write settings temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync settings temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close settings temp file: %w", err)
	}
	if err := replace(tmpPath, path); err != nil {
		return fmt.Errorf("replace settings file: %w", err)
	}
	committed = true
	if err := syncDir(dir); err != nil {
		return &settingsCommittedError{err: fmt.Errorf("sync settings directory: %w", err)}
	}
	return nil
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
	switch s.UI.Logger.Level {
	case "debug", "info", "warn", "error":
		// ok
	default:
		return fmt.Errorf("ui.logger.level 必须是 debug/info/warn/error，got %q", s.UI.Logger.Level)
	}
	if err := s.AI.validate(); err != nil {
		return err
	}
	return nil
}

func (settings *AISettings) validate() error {
	if len(settings.Profiles) > 64 {
		return errors.New("ai.profiles exceeds installation budget")
	}
	seenSlot := make(map[string]bool, len(settings.Profiles))
	seenLabel := make(map[string]bool, len(settings.Profiles))
	for _, configured := range settings.Profiles {
		if len(configured.Label) == 0 || len(configured.Label) > 128 || seenLabel[configured.Label] {
			return fmt.Errorf("ai.profiles has an invalid or duplicate label %q", configured.Label)
		}
		seenLabel[configured.Label] = true
		if err := ai.ValidateInstallationSlot(configured.Slot); err != nil || seenSlot[configured.Slot] {
			return fmt.Errorf("ai.profiles has an invalid or duplicate slot %q", configured.Slot)
		}
		seenSlot[configured.Slot] = true
		profile, err := ai.SealModelProfile(configured.profileDraft())
		if err != nil {
			return fmt.Errorf("ai.profiles[%s]: %w", configured.Slot, err)
		}
		if configured.WorkflowConsent != "" {
			expected, err := ai.WorkflowConsentDigest(configured.Slot, profile)
			if err != nil || configured.WorkflowConsent != expected {
				return fmt.Errorf("ai.profiles[%s] has stale workflow consent", configured.Slot)
			}
		}
	}
	return nil
}

// ApplyMergePatch 把 RFC7386 patch 应用到 s 上。语义：
//   - object 字段递归 merge
//   - array 整体替换
//   - scalar 覆盖
//   - null 删除字段（RFC7386 标准）
//
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
