package services

import (
	"context"
	"errors"
	"fmt"
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

	settingsPath     string // exe 同目录的 settings.json；NewApp 决定，SaveSettings 用
	settings         *Settings
	settingsMu       sync.RWMutex
	settingsUpdateMu sync.Mutex
	settingsSaver    func(string, *Settings) error

	logSink   *LogSink
	logs      *LogRuntime
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

	shutdownOnce sync.Once
	shutdownDone chan struct{}
	shutdownErr  error
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
	settings := LoadSettings(settingsPath)
	app := &App{
		settingsPath:  settingsPath,
		settings:      settings,
		settingsSaver: SaveSettings,
		logSink:       sink,
		rootLog:       rootLog,
		shutdownDone:  make(chan struct{}),
	}
	app.logs = NewLogRuntime(sink)
	app.logs.ConfigurePolicy(settings.UI.Logger)
	return app
}

// ConfigureLogging applies persisted output ownership. The desktop composition
// root calls this once after NewApp; settings updates call Configure directly.
func (a *App) ConfigureLogging() { a.logs.Configure(a.Settings().UI.Logger) }

// AttachEmitter atomically connects the presentation event transport and the
// node-dump merger. The transport is a single-assignment application resource.
func (a *App) AttachEmitter(emit func(name string, data any)) error {
	if emit == nil {
		return errors.New("event emitter is nil")
	}
	merger := NewLogMerger(
		func(name string, data any) { a.appendDiagnosticEvent(name, data) },
		func(line string, isError bool) {
			level := "info"
			if isError {
				level = "error"
			}
			if a.logSink != nil && a.logs.PersistEnabled() && a.logs.Allows(level) {
				a.logSink.AppendDumpLine(line)
				if isError {
					a.logSink.Flush()
				}
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
	emit, merger := a.presentationSnapshot()
	switch name {
	case "container:node-enter":
		if emit == nil {
			return
		}
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
		if a.logs.Allows("info") && a.logSink != nil {
			a.logSink.AppendActionTrace(data)
		}
		return
	case "container:log":
		a.appendContainerLog(data)
		return
	}
	if shouldMirrorToRootLog(name) {
		event := a.rootLog.Info()
		if name == "container:warning" || name == "container:node-error" {
			event = a.rootLog.Warn()
		}
		event.Str("event", name).Interface("data", data).Msg("runtime event")
	}
	if emit == nil {
		return
	}
	emit(name, data)
}

func (a *App) appendContainerLog(data any) {
	if a.logSink == nil {
		return
	}
	payload, _ := data.(map[string]any)
	level := stringOr(payload["level"], "info")
	if a.logs.Allows(level) {
		a.logSink.AppendRuntimeLog(level, stringOr(payload["message"], ""))
	}
}

func (a *App) appendDiagnosticEvent(name string, data any) {
	if name != "container:node-dump-batch" || !a.logs.LiveEnabled() || a.logSink == nil {
		return
	}
	payload, _ := data.(map[string]any)
	rawEntries, _ := payload["entries"].([]map[string]any)
	entries := make([]LogEntry, 0, len(rawEntries))
	for _, raw := range rawEntries {
		level := "dump"
		if boolOf(raw["isError"]) {
			level = "error"
		}
		if !a.logs.Allows(level) {
			continue
		}
		entries = append(entries, LogEntry{
			Level: level, Source: "CTR", Kind: "dump", Message: stringOr(raw["line"], ""),
			NodeID: stringOr(raw["nodeId"], ""), LineKey: stringOr(raw["lineKey"], ""),
			Count: intOf(raw["count"], 1), Final: boolOf(raw["final"]),
		})
	}
	a.logSink.AppendEntries(entries...)
}

func stringOr(value any, fallback string) string {
	if text, ok := value.(string); ok {
		return text
	}
	return fallback
}

func intOf(value any, fallback int) int {
	switch number := value.(type) {
	case int:
		return number
	case float64:
		return int(number)
	default:
		return fallback
	}
}

func str(v any) string  { s, _ := v.(string); return s }
func boolOf(v any) bool { b, _ := v.(bool); return b }

// shouldMirrorToRootLog 决定一个事件名是否镜像到 rootLog (→ file + log:batch SYS entry) 做 post-mortem.
// 只镜像 container:* 事件, 但排除高频/内部 plumbing — 这些每次节点执行就来一发, 镜像 = 面板刷屏 + 重复:
//   - container:node-enter        (数百/sec, 已有 batch 路径)
//   - container:node-dump 家族    (每节点执行一次; file 走 LogMerger.AppendDumpLine, 面板走 node-dump-batch)
func shouldMirrorToRootLog(name string) bool {
	if len(name) < 10 || name[:10] != "container:" {
		return false
	}
	switch name {
	case "container:log", "container:node-enter", "container:node-dump", "container:node-dump-batch", "container:node-dump-flush", "container:action-trace":
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
	if a.logs.NodeEnterEnabled() && a.logs.Allows("info") && a.logSink != nil {
		entries := make([]LogEntry, 0, len(batch))
		for _, item := range batch {
			message := fmt.Sprintf("→ %s (%s)", item.NodeKind, item.NodeID)
			if item.Count > 1 {
				message += fmt.Sprintf(" × %d", item.Count)
			}
			entries = append(entries, LogEntry{Level: "node", Source: "CTR", Kind: "node", Message: message, Count: item.Count})
		}
		a.logSink.AppendEntries(entries...)
	}
}

// DiagnosticsEnabled is safe to pass into a live container runtime. Detailed
// dump/action diagnostics are info-level, so the producer skips their input
// snapshots and payloads when logging is off or the minimum is warn/error.
func (a *App) DiagnosticsEnabled() bool { return a.logs != nil && a.logs.Allows("info") }

// LogStreamingEnabled guards the final Wails callback so an in-flight batch
// queued just before the user pauses logging cannot cross the UI boundary.
func (a *App) LogStreamingEnabled() bool { return a.logs != nil && a.logs.LiveEnabled() }

// Settings returns a deep snapshot. Callers can mutate it without changing
// live application state; writes must use MutateSettings.
func (a *App) Settings() *Settings {
	a.settingsMu.RLock()
	defer a.settingsMu.RUnlock()
	return a.settings.Clone()
}

// MutateSettings serializes writers and publishes the new immutable snapshot
// only after validation and an atomic disk commit succeed.
func (a *App) MutateSettings(
	mutator func(*Settings) error,
	sideEffects ...func(before, after *Settings),
) (before, after *Settings, err error) {
	if mutator == nil {
		return nil, nil, errors.New("settings mutator is nil")
	}
	a.settingsUpdateMu.Lock()
	defer a.settingsUpdateMu.Unlock()

	before = a.Settings()
	after = before.Clone()
	if err := mutator(after); err != nil {
		return before, nil, err
	}
	if err := after.Validate(); err != nil {
		return before, nil, fmt.Errorf("validate settings: %w", err)
	}
	saveErr := a.settingsSaver(a.settingsPath, after)
	if saveErr != nil && !settingsSaveCommitted(saveErr) {
		return before, nil, fmt.Errorf("save settings: %w", saveErr)
	}
	a.settingsMu.Lock()
	a.settings = after.Clone()
	a.settingsMu.Unlock()
	for _, sideEffect := range sideEffects {
		if sideEffect != nil {
			sideEffect(before.Clone(), after.Clone())
		}
	}
	if saveErr != nil {
		return before, after.Clone(), fmt.Errorf("save settings: %w", saveErr)
	}
	return before, after.Clone(), nil
}

// UpdateWindowSize 写窗口尺寸到 settings 并持久化。WindowEndResize 事件用。
// 不 emit、不走 patch 流程：纯 UI 状态，前端不订阅，写盘失败也不致命。
func (a *App) UpdateWindowSize(w, h int) {
	_, _, err := a.MutateSettings(func(settings *Settings) error {
		settings.UI.Window.Width = w
		settings.UI.Window.Height = h
		return nil
	})
	if err != nil {
		a.rootLog.Warn().Err(err).Str("tag", "SETTINGS").
			Int("width", w).Int("height", h).
			Msg("persist window size")
	}
}

// LogSink 暴露给跨包构造 zerolog MultiWriter 时用。
func (a *App) GetLogSink() *LogSink { return a.logSink }

// RootLogger 暴露给 service 使用 app 级别 logger（默认仅写 LogSink）。
func (a *App) RootLogger() zerolog.Logger { return a.rootLog }

// Shutdown is the presentation/log finalization fallback. Application-wide
// worker/server/daemon ownership and ordering live in appruntime.Runtime.
func (a *App) Shutdown() { _ = a.ShutdownContext(context.Background()) }

// ShutdownContext detaches presentation synchronously, then finalizes log
// resources once. A caller may stop waiting while cleanup continues.
func (a *App) ShutdownContext(ctx context.Context) error {
	a.shutdownOnce.Do(func() {
		a.emitterMu.Lock()
		a.presentationLife = presentationClosed
		merger := a.logMerger
		a.logMerger = nil
		a.emitter = nil
		a.emitterMu.Unlock()

		a.nodeEnterMu.Lock()
		if a.nodeEnterTimer != nil {
			a.nodeEnterTimer.Stop()
			a.nodeEnterTimer = nil
		}
		a.nodeEnterBuf = nil
		a.nodeEnterMu.Unlock()

		go func() {
			if merger != nil {
				merger.detachEmit()
				merger.Close()
			}
			if a.logSink != nil {
				a.shutdownErr = a.logSink.Close()
				a.logSink.drain()
			}
			close(a.shutdownDone)
		}()
	})
	select {
	case <-a.shutdownDone:
		return a.shutdownErr
	case <-ctx.Done():
		// Presentation cleanup may be blocked in a third-party callback. Close
		// the file-owning sink independently before honoring the deadline.
		if a.logSink != nil {
			return errors.Join(ctx.Err(), a.logSink.Close())
		}
		return ctx.Err()
	}
}
