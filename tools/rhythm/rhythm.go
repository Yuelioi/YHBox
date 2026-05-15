package rhythm

import (
	"context"
	"time"

	"github.com/lxn/win"
	"github.com/rs/zerolog"

	"yhbox/pkg/capture"
	"yhbox/pkg/input"
	"yhbox/pkg/platform"
	"yhbox/pkg/runctl"
)

// keyer 抽象 pkg/input 的 KeyDown/KeyUp。生产用 hwndKeyer 包装 hwnd；
// 测试用 mockKeyer。
type keyer interface {
	KeyDown(key string) bool
	KeyUp(key string) bool
}

type hwndKeyer struct{ hwnd win.HWND }

func (h *hwndKeyer) KeyDown(k string) bool { return input.KeyDown(h.hwnd, k) }
func (h *hwndKeyer) KeyUp(k string) bool   { return input.KeyUp(h.hwnd, k) }

// pressWorker 单轨按键执行者（异步队列，不阻塞 detect 主循环）。
//
//   - trigger(): non-blocking send 进 buffered channel；满则 drop（状态机已防合法连续触发）
//   - loop(): 持续消费 channel → KeyDown → sleep KeyHoldMs → KeyUp
//
// 跟旧版（同步阻塞）的关键区别：detect 主循环 trigger() 立即返回，按键由独立
// goroutine 处理。多轨同时触发时不会因为 KeyDown→sleep→KeyUp 卡 30ms 错过下一帧。
//
// keyup-on-exit fix：内层 select 让 ctx cancel 在 KeyDown 后也能保证 KeyUp 防卡键。
type pressWorker struct {
	key string
	k   keyer
	ch  chan struct{}
}

func newPressWorker(key string, k keyer) *pressWorker {
	// buffered cap=4：detect 主循环可能在 worker 还没消费完上一次 trigger 时就再 trigger
	// （retrigger 85ms 周期 vs key hold 5ms 一般够），缓冲一下让 trigger 不容易 drop。
	return &pressWorker{key: key, k: k, ch: make(chan struct{}, 4)}
}

func (w *pressWorker) trigger() {
	select {
	case w.ch <- struct{}{}:
	default:
		// 满了，drop（防溢出，状态机自带 retrigger 节流）
	}
}

func (w *pressWorker) start(ctx context.Context) {
	go w.loop(ctx)
}

func (w *pressWorker) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.ch:
			w.k.KeyDown(w.key)
			// 内层 select：ctx 在 KeyDown 后立刻 cancel 也要保证 KeyUp 发出，防卡键。
			select {
			case <-ctx.Done():
				w.k.KeyUp(w.key)
				return
			case <-time.After(KeyHoldMs):
				w.k.KeyUp(w.key)
			}
		}
	}
}

// Run rhythm 主循环。
//
// 启动序：PickConfig → ApplyConfig → BuildLUTs → HighResTimer → NewCapturer →
// 起 4 个 pressWorker → 120Hz ticker（c.Frame → Counts → state.Tick → trigger）。
// 退出：ctx.Done / ctrl.Stop / capture 连续失败超阈值。
//
// debugFlags 是 debug.go 的位掩码；0 → 无 debug 输出。
func Run(ctx context.Context, hwnd win.HWND, logger zerolog.Logger, ctrl runctl.Control, debugFlags uint) {
	w, h, err := capture.ClientSize(hwnd)
	if err != nil {
		logger.Error().Err(err).Str("tag", "SYSTEM").Msg("获取客户区尺寸失败")
		return
	}
	cfg, ok := PickConfig(w, h)
	if !ok {
		logger.Error().Str("tag", "SYSTEM").Msgf("不支持的分辨率 %dx%d（仅 1920×1080 / 1280×720）", w, h)
		return
	}
	logger.Info().Str("tag", "SYSTEM").Msgf("rhythm 启用分辨率 %dx%d  dark_ratio≥%.2f  retrigger=%v",
		w, h, DarkRatioThreshold, RetriggerInterval)

	hires := platform.NewHighResTimer()
	if err := hires.Begin(); err != nil {
		logger.Warn().Err(err).Str("tag", "SYSTEM").Msg("HighResTimer 启用失败，120Hz tick 可能不准")
	}
	defer hires.End()

	cp, err := capture.NewCapturer(hwnd, cfg.ROIs[:])
	if err != nil {
		logger.Error().Err(err).Str("tag", "SYSTEM").Msg("创建 Capturer 失败")
		return
	}
	defer cp.Close()

	// 异环 IMC 在窗口 IsActive=false 时丢弃 PostMessage 键盘消息（pkg/input 注释）。
	// YHBox 占前台时异环 IsActive 会翻 false → rhythm 按键全 miss。FakeActivate 发
	// WM_ACTIVATE WA_ACTIVE 翻 IsActive=true，不抢前台焦点。
	// 启动一次 + 后台周期保活（异环 tick 可能在某些情况再翻回 false）。
	input.FakeActivate(hwnd)
	go func() {
		t := time.NewTicker(500 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				input.FakeActivate(hwnd)
			}
		}
	}()

	k := &hwndKeyer{hwnd: hwnd}
	workers := [4]*pressWorker{}
	for i := range workers {
		workers[i] = newPressWorker(Keys[i], k)
		workers[i].start(ctx)
	}

	var csvOut *CSVWriter
	if debugFlags&DebugExportCSV != 0 {
		if cw, err := NewCSVWriter(platform.DevOutputDir); err == nil {
			csvOut = cw
			logger.Info().Str("tag", "SYSTEM").Msgf("CSV 输出: %s", csvOut.Path())
			defer csvOut.Close()
		} else {
			logger.Warn().Err(err).Str("tag", "SYSTEM").Msg("CSVWriter 创建失败")
		}
	}

	tel := NewTelemetry(120)
	logTicker := time.NewTicker(1 * time.Second)
	defer logTicker.Stop()

	ticker := time.NewTicker(TickInterval)
	defer ticker.Stop()

	var states [4]TrackState
	var ratios [4]float32
	captureFails := 0
	const maxConsecutiveCaptureFails = 10 // ~80~300ms 容忍区间，超过就退出（窗口 resize / hwnd 失效）

	logger.Info().Str("tag", "SYSTEM").Msg("rhythm 已启动")

	for {
		select {
		case <-ctx.Done():
			logger.Info().Str("tag", "STOP").Msg("ctx done，rhythm 退出")
			return
		case <-logTicker.C:
			s := tel.Snapshot()
			logger.Info().Str("tag", "TELEMETRY").Msgf(
				"tick p50=%v p95=%v p99=%v late=%d  cap=%v det=%v",
				s.TickP50, s.TickP95, s.TickP99, s.LateTicks, s.CaptureP50, s.DetectP50,
			)
		case tickNow := <-ticker.C:
			if ctrl.IsPaused() {
				if !ctrl.WaitUnpause() {
					return
				}
				continue
			}

			t0 := time.Now()
			imgs, err := cp.Frame()
			capDur := time.Since(t0)
			if err != nil {
				captureFails++
				if captureFails >= maxConsecutiveCaptureFails {
					logger.Error().Err(err).Str("tag", "STOP").
						Msgf("capture 连续失败 %d 次，rhythm 退出（窗口 resize / hwnd 失效等永久错误）", captureFails)
					return
				}
				logger.Warn().Str("tag", "MISS").Msgf("capture 失败 (%d/%d): %v", captureFails, maxConsecutiveCaptureFails, err)
				continue
			}
			captureFails = 0

			t1 := time.Now()
			DarkRatios(imgs, &ratios)
			detDur := time.Since(t1)

			for i := 0; i < 4; i++ {
				if states[i].Tick(ratios[i], tickNow) {
					workers[i].trigger()
					if debugFlags&DebugLogCnts != 0 {
						logger.Info().Str("tag", "PRESS").Msgf("%s ratio=%.3f", TrackNames[i], ratios[i])
					}
				}
			}

			tel.RecordTick(capDur, detDur, time.Since(t0))
			if csvOut != nil {
				pressed := [4]bool{states[0].prev, states[1].prev, states[2].prev, states[3].prev}
				_ = csvOut.Write(tickNow, ratios, pressed)
			}
		}
	}
}
