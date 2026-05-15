package bots

import (
	"context"
	"os"
	"runtime/debug"
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog"
	"yhbox/internal/services"
)

// botState 用 atomic.Int32 存。值跟 events.go BotStateXxx 字符串映射。
const (
	bsIdle int32 = iota
	bsRunning
	bsPaused
)

func bsToString(s int32) string {
	switch s {
	case bsRunning:
		return services.BotStateRunning
	case bsPaused:
		return services.BotStatePaused
	default:
		return services.BotStateIdle
	}
}

// botRun 是所有 bot service 共享的生命周期编排骨架。
//
// 不变式：
//  1. done chan 的所有权严格属于 worker goroutine —— 只能在 worker defer 里 close(done)，
//     其它任何位置（Stop/cancel/panic 处理）都不许 close。
//  2. afterExit 必须用 exitOnce.Do(...) 包，多次进入（panic / Stop race / ctx cancel）安全。
//  3. 每次 Start 必须用 make(chan struct{}) 新建 done，不复用旧实例。
type botRun struct {
	name     string
	opMu     sync.Mutex // 串行 Start/Pause/Resume/Stop（wails3 默认每 RPC 一 goroutine）
	state    atomic.Int32
	ctrl     *botControl
	ctx      context.Context
	cancel   context.CancelFunc
	done     chan struct{}
	logFile  *os.File
	lease    *services.BotLease
	exitOnce sync.Once

	app *services.App
}

func newBotRun(name string, app *services.App) *botRun {
	return &botRun{name: name, app: app}
}

func (r *botRun) State() string { return bsToString(r.state.Load()) }

// markRunning 设置 state 并 emit。
func (r *botRun) markRunning() {
	r.state.Store(bsRunning)
	r.app.Emit(eventNameFor(r.name), services.BotStateEvent{State: services.BotStateRunning})
}

func (r *botRun) markPaused() {
	r.state.Store(bsPaused)
	r.app.Emit(eventNameFor(r.name), services.BotStateEvent{State: services.BotStatePaused})
}

func (r *botRun) pause() {
	if r.ctrl != nil {
		r.ctrl.Pause()
	}
	r.markPaused()
}

func (r *botRun) resume() {
	if r.ctrl != nil {
		r.ctrl.Resume()
	}
	r.markRunning()
}

// stop 通知 worker 停（ctrl.Stop + cancel）。不等 done —— 调用方等。
func (r *botRun) stop() {
	if r.ctrl != nil {
		r.ctrl.Stop()
	}
	if r.cancel != nil {
		r.cancel()
	}
}

// stopSync 通知 worker 停并阻塞直到 worker 真正退出（done 关闭）。
// Shutdown 时用 —— 必须等 worker 释放游戏窗口输入再继续退出流程。
func (r *botRun) stopSync() {
	r.stop()
	if r.done != nil {
		<-r.done
	}
}

// afterExit 必须在 worker goroutine 的 defer 里调，exitOnce 包住。
// 顺序：close logFile → lease.Release → emit Idle。
func (r *botRun) afterExit() {
	r.exitOnce.Do(func() {
		if r.logFile != nil {
			r.logFile.Close()
			r.logFile = nil
		}
		if r.lease != nil {
			r.lease.Release()
			r.lease = nil
		}
		r.state.Store(bsIdle)
		r.app.Emit(eventNameFor(r.name), services.BotStateEvent{State: services.BotStateIdle})
	})
}

// runWorker worker goroutine 模板。run 函数是 bot 实际工作（fish.Run / cook.Run / piano.Play）。
// panic recovery 是强制项：没有这个 recover，tools/* 里任何 panic 都会让 lease 永久泄漏，
// bot 槽位锁死，用户必须重启进程。
func (r *botRun) runWorker(logger zerolog.Logger, run func()) {
	defer close(r.done) // ★ 唯一 close 点
	defer func() {
		if rec := recover(); rec != nil {
			logger.Error().
				Interface("panic", rec).
				Bytes("stack", debug.Stack()).
				Msg("bot panicked")
		}
		r.afterExit()
	}()
	run()
}

// eventNameFor 映射 bot name → state 事件名。
// 等价于 services.EventStateName(services.BotKind(name))；保留 string 入参跟 newBotRun(name) 一致。
func eventNameFor(name string) string {
	return services.EventStateName(services.BotKind(name))
}
