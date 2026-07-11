package services

import (
	"errors"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// App 是顶层协调器（不暴露给 JS，不注册为 RPC service）。
// 持有所有 service 共享的 mutex / Settings / LogSink。
// service 通过反向引用 *App 拿这些共享资源。
type App struct {
	emitterMu        sync.RWMutex
	emitter          func(name string, data any)
	presentationLife presentationLifecycle

	settingsPath string // exe 同目录的 settings.json；NewApp 决定，SaveSettings 用
	settings     *Settings
	settingsMu   sync.RWMutex

	logSink   *LogSink
	rootLog   zerolog.Logger // app/service 层 logger
	logMerger *LogMerger     // 把 raw container:node-dump 合并成 batch + 写 file

	// node-enter batch: state_FISHING 30ms tick × 多节点 ≈ 数百/sec, 每次走 presentation Event
	// IPC + 前端 reactivity tick CPU 占大头. 改 batch: 1s 累积一次, emit
	// container:node-enter-batch (payload = []{nodeId, nodeKind} 顺序). 前端取 last 1 个
	// 作为 currentNode 覆盖, batch 内不丢任何 event (落 zerolog file 仍按个写不变, 这条是
	// 给 GUI 高亮的, 不影响 post-mortem). IPC 频率 数百/sec → 1/sec.
	nodeEnterMu    sync.Mutex
	nodeEnterBuf   []nodeEnterEntry
	nodeEnterTimer *time.Timer
}

type presentationLifecycle uint8

const (
	presentationNew presentationLifecycle = iota
	presentationAttached
	presentationClosed
)

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

// AttachEmitter atomically connects the presentation event transport and the
// node-dump merger. The transport is a single-assignment application resource.
func (a *App) AttachEmitter(emit func(name string, data any)) error {
	if emit == nil {
		return errors.New("event emitter is nil")
	}
	merger := NewLogMerger(
		emit,
		func(line string) {
			if a.logSink != nil {
				a.logSink.AppendDumpLine(line)
			}
		},
	)
	a.emitterMu.Lock()
	if a.presentationLife != presentationNew {
		a.emitterMu.Unlock()
		merger.Close()
		return errors.New("presentation transport is already attached or closed")
	}
	a.emitter = emit
	a.logMerger = merger
	a.presentationLife = presentationAttached
	a.emitterMu.Unlock()
	return nil
}

func (a *App) presentationSnapshot() (func(string, any), *LogMerger) {
	a.emitterMu.RLock()
	defer a.emitterMu.RUnlock()
	return a.emitter, a.logMerger
}

// Emit sends an application event through the attached presentation transport.
// Before attachment events are dropped, matching the GUI startup contract.
// container:* 事件镜像到 zerolog 方便 post-mortem; 但 container:node-enter 频率太高
// (state_FISHING 30ms tick × 多节点 ≈ 数百/sec) 不镜像 file IO. node-enter 也不直接发 IPC,
// 走 buf + 1s 定时 flush 成 container:node-enter-batch (payload = list), 前端取 last 1 个
// 作为 currentNode. IPC 数百/sec → 1/sec, 不丢 event.
func (a *App) Emit(name string, data any) {
	if shouldMirrorToRootLog(name) {
		a.rootLog.Info().Str("event", name).Interface("data", data).Msg("runtime event")
	}
	emit, merger := a.presentationSnapshot()
	if emit == nil {
		return
	}
	switch name {
	case "container:node-enter":
		a.bufferNodeEnter(data)
		return
	case "container:node-dump":
		// raw per-execution dump — 喂 merger 合并, 不直接转发前端 (merger 发 batch).
		m, _ := data.(map[string]any)
		if m != nil && merger != nil {
			merger.Add(str(m["containerId"]), str(m["nodeId"]), str(m["nodeKind"]), str(m["line"]), str(m["lineKey"]), boolOf(m["isError"]))
		}
		return
	case "container:node-dump-flush":
		// run 停止信号 — 让 merger 收尾该容器未刷的段, 不转发前端.
		m, _ := data.(map[string]any)
		if m != nil && merger != nil {
			merger.FlushContainer(str(m["containerId"]))
		}
		return
	case "container:action-trace":
		if a.logSink != nil {
			a.logSink.AppendActionTrace(data)
		}
	}
	emit(name, data)
}

func str(v any) string  { s, _ := v.(string); return s }
func boolOf(v any) bool { b, _ := v.(bool); return b }

// shouldMirrorToRootLog 决定一个事件名是否镜像到 rootLog (→ file + log:lines SYS 行) 做 post-mortem.
// 只镜像 container:* 事件, 但排除高频/内部 plumbing — 这些每次节点执行就来一发, 镜像 = 面板刷屏 + 重复:
//   - container:node-enter        (数百/sec, 已有 batch 路径)
//   - container:node-dump 家族    (每节点执行一次; file 走 LogMerger.AppendDumpLine, 面板走 node-dump-batch)
func shouldMirrorToRootLog(name string) bool {
	if len(name) < 10 || name[:10] != "container:" {
		return false
	}
	switch name {
	case "container:node-enter", "container:node-dump", "container:node-dump-batch", "container:node-dump-flush", "container:action-trace":
		return false
	}
	return true
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
	emit, _ := a.presentationSnapshot()
	if len(batch) == 0 || emit == nil {
		return
	}
	emit("container:node-enter-batch", map[string]any{"entries": batch})
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
	a.emitterMu.Lock()
	if a.presentationLife == presentationClosed {
		a.emitterMu.Unlock()
		return
	}
	a.presentationLife = presentationClosed
	merger := a.logMerger
	a.logMerger = nil
	a.emitter = nil
	a.emitterMu.Unlock()
	if merger != nil {
		merger.Close()
	}
	if a.logSink != nil {
		a.logSink.drain()
	}
}
