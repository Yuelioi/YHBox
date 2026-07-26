package recording

import (
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/lxn/win"

	"github.com/yottaapp/yotta/internal/services/inputclip"
)

// Recorder 录制游戏窗口内的键鼠操作。线程安全：Start/Stop/Cancel/Active 都加锁。
type Recorder struct {
	mu sync.Mutex

	active bool
	tempID string // 临时 recording ID（前端订阅事件流过滤用）
	hwnd   win.HWND

	// 录制 metadata (Start 时传入)
	meta inputclip.ClipMeta

	// hook worker 线程相关
	threadID uint32        // 给 PostQuitToThread 用
	started  chan error    // worker 安装完 hook（或失败）push 一次
	done     chan struct{} // worker 退出时 close

	// 原始事件流
	rawEvents chan HookEvent // hook callback push，drain goroutine 消费
	drainDone chan struct{}  // drain goroutine 退出信号

	// 累积 inputclip events (drain goroutine 写，Stop 读)
	clipEvents []inputclip.Event
	eventMu    sync.Mutex

	// 时间基准 (Start 时设, drain 用来计算每个 Event.TUs 相对 0 的微秒偏移)
	tStartUs      uint64
	eventStartMs  uint32
	eventClockSet bool

	// 暂停 (切除间隔语义): 暂停期 drainLoop 丢事件, 时间戳扣除累计暂停时长 → 回放无空档.
	//   - paused: drainLoop 无锁高频读 → atomic.
	//   - pausedAccumUs: 累计已暂停微秒, drainLoop 无锁读 + Resume 写 → atomic.
	//   - pauseStartUs: 本次暂停起点, 仅 Pause/Resume 持 mu 读写, 不需 atomic.
	paused        atomic.Bool
	pausedAccumUs atomic.Uint64
	pauseStartUs  uint64

	// seq 单调递增, 同 TUs 内 tie-break
	seqCounter uint32
}

// NewRecorder 构造 Recorder。
func NewRecorder() *Recorder {
	return &Recorder{}
}

// Active 当前是否在录。
func (r *Recorder) Active() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.active
}

// Pause 暂停录制 — 标记 paused, 记录本次暂停起点. drainLoop 据此丢弃暂停期事件.
// 非 active 或已 paused 时 no-op (幂等). Service 层守状态机, 这里只做无害的本地标记.
func (r *Recorder) Pause() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.active || r.paused.Load() {
		return
	}
	r.pauseStartUs = nowMicros()
	r.paused.Store(true)
}

// Resume 继续录制 — 把本次暂停时长累加进 pausedAccumUs (后续事件时间戳扣除它 → 切除间隔).
// 非 active 或未 paused 时 no-op (幂等).
func (r *Recorder) Resume() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.active || !r.paused.Load() {
		return
	}
	r.pausedAccumUs.Add(nowMicros() - r.pauseStartUs)
	r.paused.Store(false)
}

// Start 启动录制。返回临时 recording ID（前端订阅事件流过滤用）。
//   - gameHwnd：游戏窗口；mouseMode=absolute 时不在它内部的鼠标事件丢弃
//   - meta：录制环境快照 (mouseMode / stopHotkeyVK 等)
//
// 失败时 active 仍为 false，可重试。
func (r *Recorder) Start(hwnd uintptr, meta inputclip.ClipMeta) (string, error) {
	r.mu.Lock()
	if r.active {
		r.mu.Unlock()
		return "", errors.New("recorder already active")
	}
	gameHwnd := win.HWND(hwnd)
	r.active = true
	r.tempID = uuid.NewString()
	r.hwnd = gameHwnd
	r.meta = meta
	r.started = make(chan error, 1)
	r.done = make(chan struct{})
	r.rawEvents = make(chan HookEvent, 256)
	r.drainDone = make(chan struct{})
	r.clipEvents = nil
	r.seqCounter = 0
	r.tStartUs = nowMicros()
	r.eventStartMs = 0
	r.eventClockSet = false
	r.paused.Store(false)
	r.pausedAccumUs.Store(0)
	r.pauseStartUs = 0
	tempID := r.tempID
	r.mu.Unlock()

	// worker：LockOSThread → InstallHooks → 发 started → RunMessageLoop → Uninstall
	go r.workerThread()

	// drain：从 rawEvents 转换成 inputclip.Event 并累积
	go r.drainLoop()

	// 等 hook 安装完毕（最多 1s）
	select {
	case err := <-r.started:
		if err != nil {
			// InstallHooks 失败：worker 已 close(done) 退出，只需让 drain 也退
			r.cleanupAfterWorkerExit()
			return "", err
		}
	case <-time.After(time.Second):
		// 超时路径上 worker 可能：1) 真挂在 InstallHooks 早期 OR 2) 刚好 InstallHooks
		// 成功还没来得及发 started。两种情况都先非阻塞 drain started，再走正常 Stop
		// 路径（PostQuit + 等 done）才能安全关 channel。
		r.cleanupOnTimeout()
		return "", errors.New("hook 安装超时")
	}
	return tempID, nil
}

// cleanupAfterWorkerExit worker 已 close(done)（如 InstallHooks 失败的快速失败路径）。
// 这里只需让 drain 退 + 复位 active。
func (r *Recorder) cleanupAfterWorkerExit() {
	close(r.rawEvents)
	<-r.drainDone
	r.mu.Lock()
	r.active = false
	r.mu.Unlock()
}

// cleanupOnTimeout 超时分支：worker 状态不明（可能在 RunMessageLoop 跑着，可能
// 还在 InstallHooks 内部挂着）。如果 InstallHooks 已经成功，hook callback 仍可能
// 往 rawEvents push 事件——这种情况下直接 close rawEvents 会让 callback panic
// (send on closed channel)。
//
// 策略：1) 非阻塞 drain started，看 InstallHooks 是否已成功
//
//	2a) 已成功 → PostQuit + 等 done（正常 Stop 路径），再 close rawEvents
//	2b) 没成功 → worker 还卡在装 hook 阶段，等它最终失败 close(done)，
//	    为安全起见也等到 done 才 close rawEvents
//	两条路都等 done 后再 close，避免竞态。
func (r *Recorder) cleanupOnTimeout() {
	// 非阻塞 drain started
	select {
	case <-r.started:
	default:
	}

	// 让 worker 退（无论它现在到哪一步：成功安装的话会响应 WM_QUIT，
	// 失败/卡住的话最终走 defer close(done)）
	r.mu.Lock()
	tid := r.threadID
	r.mu.Unlock()
	if tid != 0 {
		PostQuitToThread(tid)
	}
	<-r.done

	// worker 已退 → hook 已卸 → callback 不会再 push → 安全关 rawEvents
	close(r.rawEvents)
	<-r.drainDone

	r.mu.Lock()
	r.active = false
	r.mu.Unlock()
}

func (r *Recorder) workerThread() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(r.done)

	tid := GetCurrentThreadID()
	r.mu.Lock()
	r.threadID = tid
	rawCh := r.rawEvents
	r.mu.Unlock()

	handle, err := InstallHooks(rawCh)
	if err != nil {
		r.started <- err
		return
	}

	// Raw Input 通路：装消息-only window + 注册 raw mouse。
	// 失败不阻塞录制（普通 click/key 仍能录），只是 camera 转向丢。
	rawWnd, rawErr := createRawInputWindow()
	if rawErr == nil {
		if err := registerRawMouse(rawWnd); err != nil {
			destroyRawInputWindow(rawWnd)
			rawWnd = 0
		}
	}

	r.started <- nil

	RunMessageLoop() // 阻塞直到 WM_QUIT

	if rawWnd != 0 {
		unregisterRawMouse()
		destroyRawInputWindow(rawWnd)
	}
	handle.Uninstall()
}

// drainLoop dumb 转换 — 每个 HookEvent 直接 append 进 clipEvents, 不做任何合并 /
// 配对 / dedupe. mouseMode='relative' (FPS) 时不做窗口检测.
func (r *Recorder) drainLoop() {
	defer close(r.drainDone)
	for ev := range r.rawEvents {
		// 暂停期: 丢弃事件 (在 drain 点判定; 暂停是用户级粗动作, drain 延迟内的边界事件丢弃可接受).
		if r.paused.Load() {
			continue
		}
		var clipEv inputclip.Event
		// Use the native event clock when available so hook-to-drain scheduling
		// jitter cannot change playback timing. Canonicalization later rebases the
		// first retained event to zero.
		elapsedUs := nowMicros() - r.tStartUs
		if ev.TimestampMs != 0 {
			if !r.eventClockSet {
				r.eventStartMs = ev.TimestampMs
				r.eventClockSet = true
			}
			elapsedUs = uint64(ev.TimestampMs-r.eventStartMs) * 1000
		}
		pausedUs := r.pausedAccumUs.Load()
		if elapsedUs > pausedUs {
			clipEv.TUs = elapsedUs - pausedUs
		}
		clipEv.Seq = r.seqCounter
		r.seqCounter++

		switch {
		case ev.IsKeyboard:
			if ev.IsKeyDown {
				clipEv.Type = inputclip.EventTypeKeyDown
			} else {
				clipEv.Type = inputclip.EventTypeKeyUp
			}
			clipEv.A = int32(ev.Vk)

		case ev.IsRawDelta:
			if r.meta.RecordingMode == inputclip.RecordingModeSimple || r.meta.MouseMode == "absolute" {
				continue
			}
			clipEv.Type = inputclip.EventTypeRawDelta
			clipEv.B = int32(ev.RawDx)
			clipEv.C = int32(ev.RawDy)

		case ev.IsScroll:
			// mouseMode='absolute' 时窗口外丢; mouseMode='relative' 不过滤
			if r.meta.MouseMode == "absolute" && !IsPointInsideGameWindow(r.hwnd, ev.ScreenX, ev.ScreenY) {
				continue
			}
			cx, cy, ok := screenToClient(r.hwnd, ev.ScreenX, ev.ScreenY)
			if !ok {
				continue
			}
			clipEv.Type = inputclip.EventTypeScroll
			clipEv.A = int32(ev.WheelNotches)
			clipEv.B = cx
			clipEv.C = cy

		case ev.IsMouseMove:
			if r.meta.RecordingMode == inputclip.RecordingModeSimple || r.meta.MouseMode == "relative" {
				continue
			}
			if r.meta.MouseMode == "absolute" && !IsPointInsideGameWindow(r.hwnd, ev.ScreenX, ev.ScreenY) {
				continue
			}
			cx, cy, ok := screenToClient(r.hwnd, ev.ScreenX, ev.ScreenY)
			if !ok {
				continue
			}
			clipEv.Type = inputclip.EventTypeMouseMove
			clipEv.B = cx
			clipEv.C = cy

		default:
			// 鼠标按键 down/up
			// 关键: mouseMode='relative' (FPS) 时不做窗口检测
			if r.meta.MouseMode == "absolute" && !IsPointInsideGameWindow(r.hwnd, ev.ScreenX, ev.ScreenY) {
				continue
			}
			cx, cy, ok := screenToClient(r.hwnd, ev.ScreenX, ev.ScreenY)
			if !ok {
				continue
			}
			if ev.IsMouseDown {
				clipEv.Type = inputclip.EventTypeMouseBtnDown
			} else {
				clipEv.Type = inputclip.EventTypeMouseBtnUp
			}
			clipEv.A = int32(ev.MouseBtn)
			clipEv.B = cx
			clipEv.C = cy
		}

		r.eventMu.Lock()
		r.clipEvents = append(r.clipEvents, clipEv)
		r.eventMu.Unlock()
	}
}

// nowMicros 当前时间微秒 (用于计算 Event.TUs 相对 r.tStartUs 的偏移).
func nowMicros() uint64 {
	return uint64(time.Now().UnixMicro())
}

// Stop 停止录制并返回 raw StopResult，Service 将其持久化为 InputClip.
// 阻塞等 worker + drain 完全退出, 防 hook leak.
func (r *Recorder) Stop() (*StopResult, error) {
	r.mu.Lock()
	if !r.active {
		r.mu.Unlock()
		return nil, ErrRecorderNotActive
	}
	tid := r.threadID
	rawCh := r.rawEvents
	tempID := r.tempID
	meta := r.meta
	r.mu.Unlock()

	// 1) 让 worker 退出 (PostQuit → GetMessage 返回 → Uninstall → close(done))
	PostQuitToThread(tid)
	<-r.done

	// 2) drain 还在跑 (channel 还开着), close 让它退
	close(rawCh)
	<-r.drainDone

	// 3) 取出 events
	r.eventMu.Lock()
	events := r.clipEvents
	r.clipEvents = nil
	r.eventMu.Unlock()

	r.mu.Lock()
	r.active = false
	r.mu.Unlock()

	return &StopResult{
		Events: events,
		Meta:   meta,
		TempID: tempID,
	}, nil
}

// Cancel 不保留 events 直接停。同样阻塞等 worker + drain 退干净。
func (r *Recorder) Cancel() {
	r.mu.Lock()
	if !r.active {
		r.mu.Unlock()
		return
	}
	tid := r.threadID
	rawCh := r.rawEvents
	r.mu.Unlock()

	PostQuitToThread(tid)
	<-r.done
	close(rawCh)
	<-r.drainDone

	r.mu.Lock()
	r.active = false
	r.clipEvents = nil
	r.mu.Unlock()
}
