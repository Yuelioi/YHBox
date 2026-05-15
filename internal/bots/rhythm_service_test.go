package bots

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"yhbox/internal/services"
)

func newRhythmTestApp(t *testing.T) *services.App {
	t.Helper()
	sink := services.NewLogSink(nil)
	rootLog := zerolog.New(sink).With().Timestamp().Logger()
	return services.NewApp(filepath.Join(t.TempDir(), "settings.json"), sink, rootLog)
}

func TestRhythmService_StartFailsWithoutGame(t *testing.T) {
	app := newRhythmTestApp(t)
	s := NewRhythmService(app)
	err := s.Start()
	if err == nil || !strings.Contains(err.Error(), "未检测到异环窗口") {
		t.Errorf("expected '未检测到异环窗口' error, got %v", err)
	}
}

func TestRhythmService_StartFailsWhenBotBusy(t *testing.T) {
	app := newRhythmTestApp(t)
	app.SetGame(services.GameStatusEvent{OK: true, HWND: 0xdead, W: 1920, H: 1080})
	lease, err := app.AcquireBot("fish")
	if err != nil {
		t.Fatalf("AcquireBot: %v", err)
	}
	defer lease.Release()

	s := NewRhythmService(app)
	if err := s.Start(); err == nil || !strings.Contains(err.Error(), "fish") {
		t.Errorf("expected bot-busy error mentioning 'fish', got %v", err)
	}
}

func TestRhythmService_StopWhenIdle(t *testing.T) {
	app := newRhythmTestApp(t)
	s := NewRhythmService(app)
	err := s.Stop()
	if err == nil || !strings.Contains(err.Error(), "未运行") {
		t.Errorf("expected '未运行' error from idle Stop, got %v", err)
	}
}

func TestRhythmService_PauseWhenIdle(t *testing.T) {
	app := newRhythmTestApp(t)
	s := NewRhythmService(app)
	if err := s.Pause(); err == nil {
		t.Errorf("Pause on idle should error")
	}
}
