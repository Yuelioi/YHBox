package bots

import (
	"context"
	"errors"
	"sync"

	"github.com/lxn/win"
	"github.com/rs/zerolog"
	"github.com/wailsapp/wails/v3/pkg/application"

	"yhbox/tools/rhythm"
	"yhbox/internal/services"
)

// RhythmService 超强音自动音游 RPC。
// 跟 fish/cook/piano 一致用 botRun 骨架。无 settings 持久化 (spec v5 §D)。
type RhythmService struct {
	app *services.App
	run *botRun

	cfgMu      sync.Mutex
	debugFlags uint
}

func NewRhythmService(app *services.App) *RhythmService {
	return &RhythmService{app: app, run: newBotRun("rhythm", app)}
}

func init() {
	services.RegisterBot(services.BotMeta{
		Kind: services.KindRhythm,
		Construct: func(a *services.App) (services.BotService, application.Service) {
			s := NewRhythmService(a)
			return s, application.NewService(s)
		},
	})
}

// Start 起 bot worker。前置：游戏存在 + 未占用。
func (s *RhythmService) Start() error {
	s.run.opMu.Lock()
	defer s.run.opMu.Unlock()

	if s.run.state.Load() != bsIdle {
		return errors.New("rhythm 已在运行")
	}

	game := s.app.Game()
	if game == nil || !game.OK {
		return errors.New("未检测到异环窗口")
	}

	lease, err := s.app.AcquireBot("rhythm")
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.run.ctx = ctx
	s.run.cancel = cancel
	s.run.ctrl = NewBotControl()
	s.run.done = make(chan struct{})
	s.run.lease = lease
	s.run.exitOnce = sync.Once{}

	botLogger := zerolog.New(s.app.GetLogSink()).With().
		Timestamp().Str("bot", "rhythm").Logger()

	s.cfgMu.Lock()
	flags := s.debugFlags
	s.cfgMu.Unlock()

	s.run.markRunning()

	hwnd := win.HWND(uintptr(game.HWND))
	go s.run.runWorker(botLogger, func() {
		rhythm.Run(ctx, hwnd, botLogger, s.run.ctrl, flags)
	})

	return nil
}

func (s *RhythmService) Pause() error {
	s.run.opMu.Lock()
	defer s.run.opMu.Unlock()
	if s.run.state.Load() != bsRunning {
		return errors.New("rhythm 未在运行")
	}
	s.run.pause()
	return nil
}

func (s *RhythmService) Resume() error {
	s.run.opMu.Lock()
	defer s.run.opMu.Unlock()
	if s.run.state.Load() != bsPaused {
		return errors.New("rhythm 未暂停")
	}
	s.run.resume()
	return nil
}

func (s *RhythmService) Stop() error {
	s.run.opMu.Lock()
	defer s.run.opMu.Unlock()
	if s.run.state.Load() == bsIdle {
		return errors.New("rhythm 未运行")
	}
	s.run.stop()
	return nil
}

// SetDebugFlags 配置 debug 位掩码（位定义见 tools/rhythm/debug.go）。
// 仅影响下次 Start；运行中改不生效（rhythm.Run 启动时一次性读）。
func (s *RhythmService) SetDebugFlags(v uint) error {
	s.cfgMu.Lock()
	s.debugFlags = v
	s.cfgMu.Unlock()
	return nil
}

// StopSync App.Shutdown 调；阻塞到 worker 退出。
func (s *RhythmService) StopSync() {
	s.run.opMu.Lock()
	defer s.run.opMu.Unlock()
	if s.run.state.Load() == bsIdle {
		return
	}
	s.run.stopSync()
}
