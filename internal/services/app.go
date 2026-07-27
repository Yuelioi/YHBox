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

	settingsPath      string
	settings          *Settings
	settingsMu        sync.RWMutex
	settingsUpdateMu  sync.Mutex
	settingsSaver     func(string, *Settings) error
	settingsActivator SettingsActivationPreparer

	logSink *LogSink
	logs    *LogRuntime
	rootLog zerolog.Logger // app/service 层 logger

	shutdownOnce sync.Once
	shutdownDone chan struct{}
	shutdownErr  error
}

// SettingsActivationPlan is prepared from a validated candidate before it is
// persisted. Commit publishes the matching runtime generation after the disk
// commit; Abort releases an unpublished candidate.
type SettingsActivationPlan struct {
	Commit func() error
	Abort  func()
}

type SettingsActivationPreparer func(before, after *Settings) (*SettingsActivationPlan, error)

// AttachSettingsActivator installs the composition-owned runtime activation
// seam. It must be attached once during startup before presentation commands.
func (a *App) AttachSettingsActivator(preparer SettingsActivationPreparer) error {
	if a == nil || preparer == nil {
		return errors.New("settings activator is required")
	}
	a.settingsUpdateMu.Lock()
	defer a.settingsUpdateMu.Unlock()
	if a.settingsActivator != nil {
		return errors.New("settings activator is already attached")
	}
	a.settingsActivator = preparer
	return nil
}

type presentationLifecycle uint8

const (
	presentationNew presentationLifecycle = iota
	presentationAttached
	presentationClosed
)

func OpenApp(settingsPath, defaultLogsDir string, sink *LogSink, rootLog zerolog.Logger) (*App, error) {
	store, settings, err := OpenSettingsStore(settingsPath)
	if err != nil {
		return nil, err
	}
	app := &App{
		settingsPath:  settingsPath,
		settings:      settings,
		settingsSaver: func(_ string, next *Settings) error { return store.Save(next) },
		logSink:       sink,
		rootLog:       rootLog,
		shutdownDone:  make(chan struct{}),
	}
	app.logs = NewLogRuntime(sink, defaultLogsDir)
	app.logs.ConfigurePolicy(settings.UI.Logger)
	return app, nil
}

// OpenConfiguredApp constructs the production owner and immediately applies
// persisted file-output policy. Call OpenApp when the caller owns an
// already-configured sink.
func OpenConfiguredApp(settingsPath, defaultLogsDir string, sink *LogSink, rootLog zerolog.Logger) (*App, error) {
	app, err := OpenApp(settingsPath, defaultLogsDir, sink, rootLog)
	if err != nil {
		return nil, err
	}
	app.logs.Configure(app.Settings().UI.Logger)
	return app, nil
}

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
	var activation *SettingsActivationPlan
	if a.settingsActivator != nil {
		activation, err = a.settingsActivator(before.Clone(), after.Clone())
		if err != nil {
			return before, nil, fmt.Errorf("prepare settings activation: %w", err)
		}
	}
	saveErr := a.settingsSaver(a.settingsPath, after)
	if saveErr != nil && !settingsSaveCommitted(saveErr) {
		if activation != nil && activation.Abort != nil {
			activation.Abort()
		}
		return before, nil, fmt.Errorf("save settings: %w", saveErr)
	}
	a.settingsMu.Lock()
	a.settings = after.Clone()
	a.settingsMu.Unlock()
	if activation != nil && activation.Commit != nil {
		if activationErr := activation.Commit(); activationErr != nil {
			return before, after.Clone(), &settingsCommittedError{err: fmt.Errorf("activate settings: %w", activationErr)}
		}
	}
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

// RootLogger 暴露给 service 使用 app 级别 logger（默认仅写 LogSink）。
func (a *App) RootLogger() zerolog.Logger { return a.rootLog }

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
