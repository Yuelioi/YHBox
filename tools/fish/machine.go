package fish

import (
	"context"
	"fmt"
	"time"

	"github.com/lxn/win"
	"github.com/rs/zerolog"

	"yhbox/pkg/capture"
	"yhbox/pkg/input"
	"yhbox/pkg/runctl"
)

type Stats struct {
	StartedAt   time.Time
	CastCount   int
	Success     int
	Fail        int
	CommonCount int // 普通鱼  (real 5-6s)
	PurpleCount int // 紫色鱼  (real 7-9s)
	GoldenCount int // 金色鱼  (real 10s+)
}

func (s Stats) Format() string {
	if s.StartedAt.IsZero() {
		return "未启动"
	}
	dur := time.Since(s.StartedAt).Round(time.Second)
	total := s.Success + s.Fail
	avg := time.Duration(0)
	if total > 0 {
		avg = dur / time.Duration(total)
	}
	return fmt.Sprintf("运行 %s  抛竿 %d  ✓ %d  ✗ %d  平均 %s/回",
		dur, s.CastCount, s.Success, s.Fail, avg)
}

// StatsHook 在 stats 变化时被调用（抛竿计数 / 收鱼成功失败 / 分类）。
// 实现方需要自己处理并发（hook 在 bot goroutine 调用）。nil 安全。
type StatsHook func(Stats)

type machine struct {
	hwnd      win.HWND
	cfg       *Config
	det       *Detector
	log       zerolog.Logger
	ctrl      runctl.Control
	state     State
	stats     Stats
	statsHook StatsHook
	clientW   int
	clientH   int

	idleLogNext time.Time

	waitingStart    time.Time
	waitingLogNext  time.Time
	hookStreak      int
	hookStreakStart time.Time

	fishingStart           time.Time
	fishingLogNext         time.Time
	fishingFullFrameAt     time.Time
	fishingBarMissingStart time.Time

	recoveryEscDone bool

	resultEnteredAt time.Time

	// lastStateChangeAt 记最近一次 setState 改变 state 的时间，给看门狗用。
	lastStateChangeAt time.Time

	// emptyCastStreak 连续 WAITING 超时回 IDLE 的次数。
	// state-watchdog 抓不到此 case（state 一直在 IDLE↔WAITING 跳，时间戳不停刷新）。
	// 超过 maxEmptyCastStreak 触发自动暂停，提示用户检查（鱼仓满/鱼饵不足等关键提示模板缺失时会这样卡）。
	emptyCastStreak int

	// 货币不足分支控制（abortIfNoCurrency 置位 → handleBuyBait/handleShopSell 消费）：
	//   buyBaitAfterSell: AutoSell=true，BUYBAIT → SHOPSELL → BUYBAIT 自动串联
	//   buyBaitAbortIdle: AutoSell=false，pause 等用户手动处理，恢复后改派 IDLE 重扫描
	// 两标记互斥；handleBuyBait 用来覆盖 buyBaitFlow.OnDone=CHANGEBAIT（鱼饵其实没买到，
	// 跑 changeBaitFlow 会白按 E）。
	buyBaitAfterSell bool
	buyBaitAbortIdle bool

	fc *fishingControl
}

func newMachine(hwnd win.HWND, cfg *Config, det *Detector, logger zerolog.Logger, ctrl runctl.Control, statsHook StatsHook) *machine {
	cw, ch, _ := capture.ClientSize(hwnd)
	if cw <= 0 || ch <= 0 {
		frame, err := capture.Frame(hwnd)
		if err == nil {
			cw = frame.Bounds().Dx()
			ch = frame.Bounds().Dy()
		}
	}
	return &machine{
		hwnd:              hwnd,
		cfg:               cfg,
		det:               det,
		log:               logger,
		ctrl:              ctrl,
		statsHook:         statsHook,
		state:             IDLE,
		fc:                newFishingControl(),
		clientW:           cw,
		clientH:           ch,
		lastStateChangeAt: time.Now(),
	}
}

// publishStats 把当前 stats 推给 hook（如果有）。stats 是值类型，copy 安全。
func (m *machine) publishStats() {
	if m.statsHook != nil {
		m.statsHook(m.stats)
	}
}

func (m *machine) run(ctx context.Context) Stats {
	m.stats.StartedAt = time.Now()
	m.publishStats()
	m.log.Info().Str("tag", "SYSTEM").Msgf("状态机启动 hwnd=0x%X", uint32(m.hwnd))

	defer input.ReleaseAll(m.hwnd)

	lastKeepActive := time.Time{}

	for {
		if ctx.Err() != nil {
			m.log.Info().Str("tag", "SYSTEM").Msg("收到取消信号，退出")
			return m.stats
		}

		if m.ctrl != nil {
			if m.ctrl.IsBack() {
				m.log.Info().Str("tag", "SYSTEM").Msg("已停止")
				return m.stats
			}
			if m.ctrl.IsPaused() {
				input.ReleaseAll(m.hwnd)
				m.log.Info().Str("tag", "SYSTEM").Msg("已暂停（GUI 点[继续]恢复 / [停止]退出）")
				if !m.ctrl.WaitUnpause() {
					m.log.Info().Str("tag", "SYSTEM").Msg("已停止")
					return m.stats
				}
				m.log.Info().Str("tag", "SYSTEM").Msg("已恢复")
				m.lastStateChangeAt = time.Now() // 防 pause 期间累计触发看门狗
				continue
			}
		}

		if time.Since(lastKeepActive) >= keepAliveEvery {
			input.FakeActivate(m.hwnd)
			lastKeepActive = time.Now()
		}

		// 看门狗：state 卡 watchdogStuckThreshold 无转换 → 进 RECOVERING 走 ESC + inspectPhase
		if m.shouldWatchdog() && time.Since(m.lastStateChangeAt) > watchdogStuckThreshold {
			stuck := int(time.Since(m.lastStateChangeAt).Seconds())
			m.enterRecover(fmt.Sprintf("看门狗：%s 卡 %ds", m.state, stuck), true)
			continue
		}

		// WAITING / FISHING / SHOPSELL 状态自己抓帧，不需要主循环预取
		if m.state == WAITING {
			m.handleWaiting(ctx)
			if !m.sleep(ctx, 250*time.Millisecond) {
				return m.stats
			}
			continue
		}
		if m.state == FISHING {
			m.handleFishing(ctx)
			if !m.sleep(ctx, loopInterval) {
				return m.stats
			}
			continue
		}
		if m.state == SHOPSELL {
			m.handleShopSell(ctx)
			if !m.sleep(ctx, 300*time.Millisecond) {
				return m.stats
			}
			continue
		}
		if m.state == BUYBAIT {
			m.handleBuyBait(ctx)
			if !m.sleep(ctx, 300*time.Millisecond) {
				return m.stats
			}
			continue
		}
		if m.state == CHANGEBAIT {
			m.handleChangeBait(ctx)
			if !m.sleep(ctx, 300*time.Millisecond) {
				return m.stats
			}
			continue
		}

		frame, err := capture.Frame(m.hwnd)
		if err != nil {
			m.log.Warn().Err(err).Str("tag", "SYSTEM").Msg("抓帧失败，2s 后重试")
			if !m.sleep(ctx, 2*time.Second) {
				return m.stats
			}
			continue
		}

		switch m.state {
		case IDLE:
			m.handleIdle(ctx, frame)
		case SETUP:
			m.handleSetup(ctx, frame)
		case RESULT:
			m.handleResult(ctx, frame)
		case RECOVERING:
			m.handleRecovering(ctx, frame)
		}

		if !m.sleep(ctx, 150*time.Millisecond) {
			return m.stats
		}
	}
}

func (m *machine) sleep(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	select {
	case <-time.After(d):
		return true
	case <-ctx.Done():
		return false
	}
}

// shouldExit 检查 ctx 取消 / GUI 触发 stop / GUI 触发 pause。
// 任一条件满足返回 true，调用方应立刻 return（让外层主循环统一处理）。
// 用于长时间内层循环（如 tryHookF）防止 60s 内 GUI 操作无响应。
func (m *machine) shouldExit() bool {
	if m.ctrl == nil {
		return false
	}
	return m.ctrl.IsBack() || m.ctrl.IsPaused()
}

func (m *machine) debugEnabled() bool {
	return false
}

func (m *machine) logState(format string, args ...any) {
	// 用 tag 字段而不是 state，让前端 LogPanel 的 tagClass(tag) 走色板（IDLE/SETUP/CAST/FIGHT...）
	m.log.Info().Str("tag", m.state.String()).Msgf(format, args...)
}

// watchdogStuckThreshold 看门狗触发阈值：state 超过此时长无转换则进 RECOVERING。
// 2 min：实战观察非 FISHING 状态都在秒级转换；2 min 不动必为异常（点击丢、UI 残留等）。
const watchdogStuckThreshold = 2 * time.Minute

// maxEmptyCastStreak 连续 N 次 WAITING 空抛回 IDLE 触发自动暂停。
// 5 次 × ~75s/cast ≈ 6 min 用户无感等待，已能确认确实卡死（不是单次偶然鱼跑了）。
const maxEmptyCastStreak = 5

// setState 改 state 并刷新 lastStateChangeAt。所有 state 转换都该走它，
// 直接赋值 m.state = X 会让看门狗"看不见"那次转换。
func (m *machine) setState(s State) {
	if m.state != s {
		m.state = s
		m.lastStateChangeAt = time.Now()
	}
}

// shouldWatchdog 当前 state 是否参与看门狗。
// 白名单：FISHING（溜鱼挣扎可达 1-2 min）、RECOVERING（已在恢复中，避免套娃）。
func (m *machine) shouldWatchdog() bool {
	return m.state != FISHING && m.state != RECOVERING
}

// ---------------- SHOPSELL / BUYBAIT / CHANGEBAIT ----------------
//
// 三套商店流程的 handler 已搬到 states_shop.go 中以声明式 flow{} 表达。
// 见 shopSellFlow / buyBaitFlow / changeBaitFlow。
//
// 钓鱼相关 handler (IDLE/SETUP/WAITING/FISHING/RESULT/RECOVERING) 已搬到 states_fishing.go。
