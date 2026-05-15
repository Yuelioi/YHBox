package bots

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/lxn/win"
	"github.com/rs/zerolog"
	"github.com/wailsapp/wails/v3/pkg/application"

	"yhbox/tools/cook"
	"yhbox/internal/services"
)

type CookService struct {
	app *services.App
	run *botRun

	cfg *cook.Config // 运行中 cfg 指针，SetIntervalMs 用
}

func NewCookService(app *services.App) *CookService {
	return &CookService{app: app, run: newBotRun("cook", app)}
}

func init() {
	services.RegisterBot(services.BotMeta{
		Kind: services.KindCook,
		Construct: func(a *services.App) (services.BotService, application.Service) {
			s := NewCookService(a)
			return s, application.NewService(s)
		},
	})
}

func (s *CookService) Start() error {
	s.run.opMu.Lock()
	defer s.run.opMu.Unlock()

	if s.run.state.Load() != bsIdle {
		return errors.New("cook 已在运行")
	}

	game := s.app.Game()
	if game == nil || !game.OK {
		return errors.New("未检测到异环窗口")
	}

	lease, err := s.app.AcquireBot("cook")
	if err != nil {
		return err
	}

	settings := s.app.Settings()
	cfg := cook.DefaultConfig()
	cfg.Interval = time.Duration(settings.Cook.IntervalMs) * time.Millisecond
	s.cfg = cfg

	ctx, cancel := context.WithCancel(context.Background())
	s.run.ctx = ctx
	s.run.cancel = cancel
	s.run.ctrl = NewBotControl()
	s.run.done = make(chan struct{})
	s.run.lease = lease
	s.run.exitOnce = sync.Once{}

	botLogger := zerolog.New(s.app.GetLogSink()).With().
		Timestamp().Str("bot", "cook").Logger()

	s.run.markRunning()

	hwnd := win.HWND(uintptr(game.HWND))
	go s.run.runWorker(botLogger, func() {
		cook.Run(ctx, hwnd, cfg, botLogger, s.run.ctrl)
	})
	return nil
}

func (s *CookService) Pause() error {
	s.run.opMu.Lock()
	defer s.run.opMu.Unlock()
	if s.run.state.Load() != bsRunning {
		return errors.New("cook 未在运行")
	}
	s.run.pause()
	return nil
}

func (s *CookService) Resume() error {
	s.run.opMu.Lock()
	defer s.run.opMu.Unlock()
	if s.run.state.Load() != bsPaused {
		return errors.New("cook 未暂停")
	}
	s.run.resume()
	return nil
}

func (s *CookService) Stop() error {
	s.run.opMu.Lock()
	defer s.run.opMu.Unlock()
	if s.run.state.Load() == bsIdle {
		return errors.New("cook 未运行")
	}
	s.run.stop()
	return nil
}

// SetIntervalMs 改点击间隔。运行中改：更新 cfg.Interval（cook.Run 每轮读它）。
// 持久化到 settings.json。Validate 确保 100..10000 范围。
func (s *CookService) SetIntervalMs(ms int) error {
	if ms < 100 || ms > 10000 {
		return fmt.Errorf("intervalMs out of range [100, 10000]: %d", ms)
	}
	if s.cfg != nil {
		s.cfg.Interval = time.Duration(ms) * time.Millisecond
	}
	s.app.UpdateSettings(func(st *services.Settings) { st.Cook.IntervalMs = ms })
	return s.app.SaveSettings()
}

func (s *CookService) StopSync() {
	s.run.opMu.Lock()
	defer s.run.opMu.Unlock()
	if s.run.state.Load() == bsIdle {
		return
	}
	s.run.stopSync()
}
