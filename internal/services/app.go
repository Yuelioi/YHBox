package services

import (
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// App 是顶层协调器（不暴露给 JS，不进 application.Options.Services）。
// 持有所有 service 共享的 mutex / Settings / LogSink。
// service 通过反向引用 *App 拿这些共享资源。
type App struct {
	wailsApp *application.App

	settingsPath string // exe 同目录的 settings.json；NewApp 决定，SaveSettings 用
	settings     *Settings
	settingsMu   sync.RWMutex

	logSink *LogSink
	rootLog zerolog.Logger // app/service 层 logger

	// node-enter batch: state_FISHING 30ms tick × 多节点 ≈ 数百/sec, 每次走 wails Event
	// IPC + 前端 reactivity tick CPU 占大头. 改 batch: 200ms 累积一次, emit
	// container:node-enter-batch (payload = []{nodeId, nodeKind} 顺序). 前端取 last 1 个
	// 作为 currentNode 覆盖, batch 内不丢任何 event (落 zerolog file 仍按个写不变, 这条是
	// 给 GUI 高亮的, 不影响 post-mortem). IPC 频率 数百/sec → 5/sec.
	nodeEnterMu      sync.Mutex
	nodeEnterBuf     []nodeEnterEntry
	nodeEnterTimer   *time.Timer
}

type nodeEnterEntry struct {
	NodeID   string `json:"nodeId"`
	NodeKind string `json:"nodeKind"`
	Count    int    `json:"count"` // 连续重复进同 node 折叠 (console.log × N 风格)
}

// nodeEnterBatchInterval: 1s 一批刷前端. 节点高亮 GUI 不需要更高频率 (1Hz 人眼能看清),
// 5Hz/200ms 时 IPC + frontend reactivity 占大量 CPU. v1 没节点高亮所以 baseline 低很多.
const nodeEnterBatchInterval = 1000 * time.Millisecond

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
// container:* 事件镜像到 zerolog 方便 post-mortem; 但 container:node-enter 频率太高
// (state_FISHING 30ms tick × 多节点 ≈ 数百/sec) 不镜像 file IO. node-enter 也不直接发 IPC,
// 走 buf + 200ms 定时 flush 成 container:node-enter-batch (payload = list), 前端取 last 1 个
// 作为 currentNode. IPC 数百/sec → 5/sec, 不丢 event.
func (a *App) Emit(name string, data any) {
	if name != "container:node-enter" && len(name) >= 10 && name[:10] == "container:" {
		a.rootLog.Info().Str("event", name).Interface("data", data).Msg("runtime event")
	}
	if a.wailsApp == nil {
		return
	}
	if name == "container:node-enter" {
		a.bufferNodeEnter(data)
		return
	}
	a.wailsApp.Event.Emit(name, data)
}

// bufferNodeEnter 把 node-enter event 累积进 buf, 启动定时 flush.
// 连续同 nodeId 折叠 (Loop body 反复进同节点是常态, e.g. state_FISHING.barTrack 30Hz).
func (a *App) bufferNodeEnter(data any) {
	m, _ := data.(map[string]any)
	if m == nil {
		return
	}
	id, _ := m["nodeId"].(string)
	kind, _ := m["nodeKind"].(string)
	a.nodeEnterMu.Lock()
	if n := len(a.nodeEnterBuf); n > 0 && a.nodeEnterBuf[n-1].NodeID == id {
		a.nodeEnterBuf[n-1].Count++
	} else {
		a.nodeEnterBuf = append(a.nodeEnterBuf, nodeEnterEntry{NodeID: id, NodeKind: kind, Count: 1})
	}
	if a.nodeEnterTimer == nil {
		a.nodeEnterTimer = time.AfterFunc(nodeEnterBatchInterval, a.flushNodeEnter)
	}
	a.nodeEnterMu.Unlock()
}

// flushNodeEnter 由定时器触发, 发出累积的 batch.
func (a *App) flushNodeEnter() {
	a.nodeEnterMu.Lock()
	batch := a.nodeEnterBuf
	a.nodeEnterBuf = nil
	a.nodeEnterTimer = nil
	a.nodeEnterMu.Unlock()
	if len(batch) == 0 || a.wailsApp == nil {
		return
	}
	a.wailsApp.Event.Emit("container:node-enter-batch", map[string]any{"entries": batch})
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

// UpdateSettings 持锁应用 mutator 修改 live settings；给跨包用户
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

// LogSink 暴露给跨包构造 zerolog MultiWriter 时用。
func (a *App) GetLogSink() *LogSink { return a.logSink }

// RootLogger 暴露给 service 使用 app 级别 logger（默认仅写 LogSink）。
func (a *App) RootLogger() zerolog.Logger { return a.rootLog }

// Shutdown 集中退出钩子。挂在 wails3 window close 钩子上。
func (a *App) Shutdown() {
	if a.logSink != nil {
		a.logSink.Flush()
	}
}

