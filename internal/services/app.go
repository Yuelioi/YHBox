package services

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// App 是顶层协调器（不暴露给 JS，不进 application.Options.Services）。
// 持有所有 service 共享的 mutex / Settings / LogSink / Game 缓存。
// service 通过反向引用 *App 拿这些共享资源。
type App struct {
	wailsApp *application.App

	settingsPath string // exe 同目录的 settings.json；NewApp 决定，SaveSettings 用
	settings     *Settings
	settingsMu   sync.RWMutex

	botMu    sync.Mutex
	botOwner string

	game    atomic.Pointer[GameStatusEvent]
	logSink *LogSink
	rootLog zerolog.Logger // 不挂 bot 的 logger；用于 app/service 层事件

	// 长跑型 bot service 走 registry（fish/cook/piano/rhythm）。
	// botServices 在 RegisterBotServices 期间被填满；Shutdown 时统一调用 StopSync。
	botServices []BotService
	// battle 是 hotkey-driven 形态不同，单独引用。
	battleService BattleHotkeyService
}

// NewApp 构造。settingsPath="" 走全局 default（settings.go 里 exe 同目录的 settings.json）。
// LoadSettings 失败也返回（fallback default）。
func NewApp(settingsPath string, sink *LogSink, rootLog zerolog.Logger) *App {
	if settingsPath == "" {
		settingsPath = settingsFilePath
	}
	return &App{
		settingsPath: settingsPath,
		settings:     LoadSettings(settingsPath),
		logSink:      sink,
		rootLog:      rootLog,
	}
}

// AttachWailsApp main.go 创建完 application.App 后调，让 App 能 Emit。
func (a *App) AttachWailsApp(w *application.App) { a.wailsApp = w }

// Emit 包装 wailsApp.Event.Emit。a.wailsApp == nil 时（启动前）静默丢弃。
// 同时把所有 container:* 事件 (除高频 node-enter 用 Debug 之外) 镜像到 zerolog
// 让 logs/yhfish-YYYYMMDD.log 含完整 trace, 方便 post-mortem 调试.
func (a *App) Emit(name string, data any) {
	if name == "container:node-enter" {
		a.rootLog.Debug().Str("event", name).Interface("data", data).Msg("runtime event")
	} else if len(name) >= 10 && name[:10] == "container:" {
		a.rootLog.Info().Str("event", name).Interface("data", data).Msg("runtime event")
	}
	if a.wailsApp == nil {
		return
	}
	a.wailsApp.Event.Emit(name, data)
}

// Settings 返回当前 settings 的指针快照。读多写少，加 RLock 即可。
// 调用方不应修改返回值；要改走 SettingsService.Update 走 patch 流程。
func (a *App) Settings() *Settings {
	a.settingsMu.RLock()
	defer a.settingsMu.RUnlock()
	return a.settings
}

// SnapshotSettings deep clone 当前 settings；给 SettingsService.Update 的 "clone → merge" 用。
func (a *App) SnapshotSettings() *Settings {
	a.settingsMu.RLock()
	defer a.settingsMu.RUnlock()
	return a.settings.Clone()
}

// SwapSettings 原子替换 live settings 指针（SettingsService.Update 验证通过后调）。
func (a *App) SwapSettings(s *Settings) {
	a.settingsMu.Lock()
	a.settings = s
	a.settingsMu.Unlock()
}

// UpdateSettings 持锁应用 mutator 修改 live settings；给 internal/bots 等跨包用户
// 避免直接 poke 私有 settingsMu。注意 mutator 不能再调任何持锁 App 方法（自死锁）。
func (a *App) UpdateSettings(mutator func(*Settings)) {
	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()
	mutator(a.settings)
}

// UpdateWindowSize 写窗口尺寸到 settings 并持久化。WindowEndResize 事件用。
// 不 emit、不走 patch 流程：纯 UI 状态，前端不订阅，写盘失败也不致命。
func (a *App) UpdateWindowSize(w, h int) {
	a.settingsMu.Lock()
	if a.settings.UI.Window.Width == w && a.settings.UI.Window.Height == h {
		a.settingsMu.Unlock()
		return
	}
	a.settings.UI.Window.Width = w
	a.settings.UI.Window.Height = h
	a.settingsMu.Unlock()
	_ = a.SaveSettings()
}

// SaveSettings 把当前 settings 写到 settings.json。失败仅 log warning。
func (a *App) SaveSettings() error {
	a.settingsMu.RLock()
	s := a.settings
	a.settingsMu.RUnlock()
	return SaveSettings(a.settingsPath, s)
}

// BotLease 是 RAII-style 互斥句柄。
// service.Start() 拿到 lease，defer lease.Release()。Release 幂等，panic 也不会泄漏 owner。
type BotLease struct {
	app      *App
	who      string
	released atomic.Bool
}

// NewBotLease 构造（给 internal/bots 测试 + 极个别非 AcquireBot 路径用）。
// 生产路径走 App.AcquireBot 拿 lease，这里只是 escape hatch。
func NewBotLease(app *App, who string) *BotLease {
	return &BotLease{app: app, who: who}
}

// IsReleased 给测试断言用；幂等查 release 状态。
func (l *BotLease) IsReleased() bool { return l.released.Load() }

// Release 幂等释放。多次调用、panic 路径都安全。
func (l *BotLease) Release() {
	if l.released.Swap(true) {
		return
	}
	l.app.botMu.Lock()
	if l.app.botOwner == l.who {
		l.app.botOwner = ""
	}
	l.app.botMu.Unlock()
}

// AcquireBot 取得 bot 互斥。已被占用返回 error，error 文本必含 owner 名（前端 toast 直接展示）。
func (a *App) AcquireBot(who string) (*BotLease, error) {
	a.botMu.Lock()
	defer a.botMu.Unlock()
	if a.botOwner != "" {
		return nil, fmt.Errorf("bot %q is already running, stop it first", a.botOwner)
	}
	a.botOwner = who
	return &BotLease{app: a, who: who}, nil
}

// BotOwner 返回当前持有者名（"" 表示空闲）。给设置面板 / 调试用。
func (a *App) BotOwner() string {
	a.botMu.Lock()
	defer a.botMu.Unlock()
	return a.botOwner
}

// SetGame 缓存最新的游戏窗口检测结果。GameService.Detect 调。
func (a *App) SetGame(g GameStatusEvent) { a.game.Store(&g) }

// Game 返回最新缓存的游戏窗口状态（nil 表示从未检测过）。
func (a *App) Game() *GameStatusEvent { return a.game.Load() }

// LogSink 暴露给 bot service 构造 zerolog MultiWriter 时用。
func (a *App) GetLogSink() *LogSink { return a.logSink }

// RootLogger 暴露给 service 使用 app 级别 logger（默认仅写 LogSink）。
func (a *App) RootLogger() zerolog.Logger { return a.rootLog }

// RegisterBotServices 把长跑型 bot service 切片存进 App，方便 Shutdown 统一停。
// main.go 通过 RegisteredBots() 构造完所有 service 后调一次。
func (a *App) RegisterBotServices(svcs []BotService) {
	a.botServices = svcs
}

// BattleHotkeyService Shutdown 时需要反注册热键的 hotkey-driven service。
// 实现在 internal/bots（BattleService），用接口避免 services → bots 反向 import。
type BattleHotkeyService interface {
	ShutdownUnregister()
}

// RegisterBattleService 单独把 battle service 引用存进来（hotkey-driven 形态特殊）。
func (a *App) RegisterBattleService(s BattleHotkeyService) {
	a.battleService = s
}

// Shutdown 集中退出钩子。挂在 wails3 window close 钩子上。
// 顺序敏感：先停长跑 bot（释放游戏窗口输入）→ 反注册 battle 热键 → flush 日志。
func (a *App) Shutdown() {
	for _, svc := range a.botServices {
		svc.StopSync()
	}
	if a.battleService != nil {
		a.battleService.ShutdownUnregister()
	}
	if a.logSink != nil {
		a.logSink.Flush()
	}
}

// ErrBotBusy 哨兵 error（service 层可断言）
var ErrBotBusy = errors.New("bot busy")
