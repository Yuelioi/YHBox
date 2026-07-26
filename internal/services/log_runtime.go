package services

import (
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/rs/zerolog"
)

// LogRuntime owns the process-wide logging policy. It is intentionally the
// only place that changes zerolog's global minimum level or LogSink outputs.
// This lets copied zerolog.Logger values stop work before field/message
// serialization when logging is disabled or has no destination.
type LogRuntime struct {
	sink    *LogSink
	enabled atomic.Bool
	live    atomic.Bool
	persist atomic.Bool
	minimum atomic.Int32

	defaultFileDir string
}

func NewLogRuntime(sink *LogSink, defaultFileDir ...string) *LogRuntime {
	runtime := &LogRuntime{sink: sink}
	if len(defaultFileDir) != 0 {
		runtime.defaultFileDir = strings.TrimSpace(defaultFileDir[0])
	}
	return runtime
}

func (r *LogRuntime) Configure(settings LoggerSettings) {
	r.configure(settings, true)
}

// ConfigurePolicy initializes source/stream behavior without taking ownership
// of a file configured by a test or embedding host. OpenConfiguredApp promotes
// the same settings to persisted-output ownership during construction.
func (r *LogRuntime) ConfigurePolicy(settings LoggerSettings) {
	r.configure(settings, false)
}

func (r *LogRuntime) configure(settings LoggerSettings, configureFile bool) {
	r.enabled.Store(settings.Enabled)
	r.live.Store(settings.Enabled && settings.LiveView)
	r.persist.Store(settings.Enabled && settings.WriteFile)
	r.minimum.Store(int32(parseLogLevel(settings.Level)))

	if r.sink != nil {
		r.sink.SetStreamEnabled(r.live.Load())
		if configureFile {
			dir := ""
			if r.persist.Load() {
				dir = strings.TrimSpace(settings.FileDir)
				if dir == "" {
					dir = r.defaultFileDir
				} else if r.defaultFileDir != "" && !filepath.IsAbs(dir) {
					dir = filepath.Join(filepath.Dir(r.defaultFileDir), dir)
				}
			}
			r.sink.SetFileWriter(dir)
		}
	}

	// Keep Fatal enabled for process-control semantics (zerolog.Fatal performs
	// os.Exit after Msg). With both sink destinations off it still emits no I/O.
	level := zerolog.FatalLevel
	if r.Enabled() {
		level = parseLogLevel(settings.Level)
	}
	zerolog.SetGlobalLevel(level)
}

// Enabled means at least one diagnostic destination is active.
func (r *LogRuntime) Enabled() bool {
	return r.enabled.Load() && (r.live.Load() || r.persist.Load())
}

func (r *LogRuntime) LiveEnabled() bool    { return r.live.Load() }
func (r *LogRuntime) PersistEnabled() bool { return r.persist.Load() }

func (r *LogRuntime) Allows(level string) bool {
	if !r.Enabled() {
		return false
	}
	return int32(parseEntryLevel(level)) >= r.minimum.Load()
}

func parseLogLevel(level string) zerolog.Level {
	switch level {
	case "debug":
		return zerolog.DebugLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

func parseEntryLevel(level string) zerolog.Level {
	switch level {
	case "debug", "trace":
		return zerolog.DebugLevel
	case "warn":
		return zerolog.WarnLevel
	case "error", "fatal":
		return zerolog.ErrorLevel
	default: // info, log, node, dump, action
		return zerolog.InfoLevel
	}
}
