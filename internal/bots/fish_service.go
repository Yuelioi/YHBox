package bots

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/lxn/win"
	"github.com/rs/zerolog"
	"github.com/wailsapp/wails/v3/pkg/application"

	"yhbox/tools/fish"
	"yhbox/internal/services"
)

// FishService Phase 2 完整实现 Start/Pause/Resume/Stop/SetAutoSell。
type FishService struct {
	app *services.App
	run *botRun

	// cfg 持有同一个 fish.Config：bot worker 读 + GUI 改 AutoSell。
	cfg *fish.Config

	// 最近一次 stats 快照（前端可能错过事件时用 GetStats 拉一次）。
	statsMu sync.RWMutex
	latest  fish.Stats
}

func NewFishService(app *services.App) *FishService {
	return &FishService{app: app, run: newBotRun("fish", app)}
}

func init() {
	services.RegisterBot(services.BotMeta{
		Kind: services.KindFish,
		Construct: func(a *services.App) (services.BotService, application.Service) {
			s := NewFishService(a)
			return s, application.NewService(s)
		},
	})
}

// Start 起 bot worker。前置检查：游戏窗口存在 + bot 互斥未占用。
func (s *FishService) Start() error {
	s.run.opMu.Lock()
	defer s.run.opMu.Unlock()

	if s.run.state.Load() != bsIdle {
		return errors.New("fish 已在运行")
	}

	game := s.app.Game()
	if game == nil || !game.OK {
		return errors.New("未检测到异环窗口")
	}

	lease, err := s.app.AcquireBot("fish")
	if err != nil {
		return err
	}

	// 用 settings 里的 AutoSell / CaptureGolden 作为初始值
	settings := s.app.Settings()
	cfg := fish.DefaultConfig()
	cfg.AutoSell.Store(settings.Fish.AutoSell)
	cfg.CaptureGolden.Store(settings.Fish.CaptureGolden)
	s.cfg = cfg

	// 准备 ctx + ctrl + done
	ctx, cancel := context.WithCancel(context.Background())
	s.run.ctx = ctx
	s.run.cancel = cancel
	s.run.ctrl = NewBotControl()
	s.run.done = make(chan struct{})
	s.run.lease = lease
	s.run.exitOnce = sync.Once{} // 每次 Start 重置

	// 重置 stats latest
	s.statsMu.Lock()
	s.latest = fish.Stats{}
	s.statsMu.Unlock()
	s.app.Emit(services.EventFishStats, services.FishStatsEvent{}) // 让前端清零

	// bot logger：zerolog 写到 logSink；附 bot=fish 字段
	botLogger := zerolog.New(s.app.GetLogSink()).With().
		Timestamp().Str("bot", "fish").Logger()

	// stats hook：bot goroutine 同步调；写到 latest + emit
	onStats := func(st fish.Stats) {
		s.statsMu.Lock()
		s.latest = st
		s.statsMu.Unlock()
		s.app.Emit(services.EventFishStats, fishStatsToEvent(st))
	}

	// Mark running + emit fish:state（在 launch goroutine 之前，确保前端先收到 running）
	s.run.markRunning()

	hwnd := win.HWND(uintptr(game.HWND))
	go s.run.runWorker(botLogger, func() {
		fish.Run(ctx, hwnd, cfg, botLogger, s.run.ctrl, onStats)
	})

	return nil
}

// Pause 暂停正在运行的 bot。
func (s *FishService) Pause() error {
	s.run.opMu.Lock()
	defer s.run.opMu.Unlock()
	if s.run.state.Load() != bsRunning {
		return errors.New("fish 未在运行")
	}
	s.run.pause()
	return nil
}

// Resume 恢复暂停中的 bot。
func (s *FishService) Resume() error {
	s.run.opMu.Lock()
	defer s.run.opMu.Unlock()
	if s.run.state.Load() != bsPaused {
		return errors.New("fish 未暂停")
	}
	s.run.resume()
	return nil
}

// Stop 通知 bot 停止（不阻塞等待退出）。
func (s *FishService) Stop() error {
	s.run.opMu.Lock()
	defer s.run.opMu.Unlock()
	if s.run.state.Load() == bsIdle {
		return errors.New("fish 未运行")
	}
	s.run.stop()
	return nil
}

// SetAutoSell 同时改运行中 cfg + 持久化到 settings.json。
// 直接 atomic + save，不走 SettingsService.Update 的 RFC7386/Validate 流程（单字段不需要）。
func (s *FishService) SetAutoSell(v bool) error {
	if s.cfg != nil {
		s.cfg.AutoSell.Store(v)
	}
	s.app.UpdateSettings(func(st *services.Settings) { st.Fish.AutoSell = v })
	if err := s.app.SaveSettings(); err != nil {
		return fmt.Errorf("save settings: %w", err)
	}
	return nil
}

// SetCaptureGolden 同 SetAutoSell 模式：atomic.Bool 写运行中 cfg + 持久化。
// bot 在 recordOutcome 里 Load() 读，开关立刻生效（无需重启 bot）。
func (s *FishService) SetCaptureGolden(v bool) error {
	if s.cfg != nil {
		s.cfg.CaptureGolden.Store(v)
	}
	s.app.UpdateSettings(func(st *services.Settings) { st.Fish.CaptureGolden = v })
	if err := s.app.SaveSettings(); err != nil {
		return fmt.Errorf("save settings: %w", err)
	}
	return nil
}

// GetStats 给前端可选拉一次 latest stats（避免漏事件时显示 0）。
func (s *FishService) GetStats() services.FishStatsEvent {
	s.statsMu.RLock()
	defer s.statsMu.RUnlock()
	return fishStatsToEvent(s.latest)
}

// StopSync App.Shutdown 调；阻塞直到 worker 真正退出。
func (s *FishService) StopSync() {
	s.run.opMu.Lock()
	defer s.run.opMu.Unlock()
	if s.run.state.Load() == bsIdle {
		return
	}
	s.run.stopSync()
}

// fishStatsToEvent 把 fish.Stats 转换为 wire payload。
// 总数 = Common + Purple + Golden；Success/Fail 不暴露给前端。
func fishStatsToEvent(s fish.Stats) services.FishStatsEvent {
	return services.FishStatsEvent{
		CommonCount: s.CommonCount,
		PurpleCount: s.PurpleCount,
		GoldenCount: s.GoldenCount,
		StartedAt:   s.StartedAt,
	}
}
