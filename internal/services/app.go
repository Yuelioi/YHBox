package services

import (
	"context"
	"errors"
	"fmt"
	"sync"

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

	logSink *LogSink
	logs    *LogRuntime
	rootLog zerolog.Logger // app/service 层 logger

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

// AttachEmitter atomically connects the single-assignment presentation transport.
func (a *App) AttachEmitter(emit func(name string, data any)) error {
	if emit == nil {
		return errors.New("event emitter is nil")
	}
	a.emitterMu.Lock()
	defer a.emitterMu.Unlock()
	if a.presentationLife != presentationNew {
		return errors.New("presentation transport is already attached or closed")
	}
	a.emitter = emit
	a.presentationLife = presentationAttached
	return nil
}

func (a *App) presentationSnapshot() func(string, any) {
	a.emitterMu.RLock()
	defer a.emitterMu.RUnlock()
	return a.emitter
}

// Emit sends an application event through the attached presentation transport.
// Before attachment events are dropped, matching the GUI startup contract.
func (a *App) Emit(name string, data any) {
	emit := a.presentationSnapshot()
	if emit == nil {
		return
	}
	emit(name, data)
}

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
		a.emitter = nil
		a.emitterMu.Unlock()

		go func() {
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
