// recorder.go：整合 LL hook pipeline 的录制控制器 (v2 dumb 版).
//
// 生命周期：Start → 用户操作 → Stop（或 Cancel）。同一 Recorder 可复用。
//
// v2 设计原则: Recorder dumb 化 — drain loop 不做任何合并 / 配对 / dedupe,
// 每个 HookEvent 直接转换成 inputclip.Event append 进 slice. 智能化全部下放到
// inputclip transform 层 (Phase 3+).
//
// 关键 Win32 约束：
//   - SetWindowsHookEx + GetMessage 必须同一个 OS 线程
//   - 所以起 worker goroutine + runtime.LockOSThread
//   - Stop 必须阻塞等 worker 真退出（done channel），否则 Start→Stop→Start
//     会触发 duplicate callback / hook leak
package recording

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lxn/win"

	"yhbox/internal/services/inputclip"
	"yhbox/pkg/capture"
)

// StopResult Recorder.Stop 输出. Service 层据此分流:
//   - precise: 用 Events + Meta 建 InputClip 落到 clipSvc, 再 BuildPreciseSubgraph
//   - simple : 用 Events + ClientW/H 直接 BuildSimpleSubgraph
//
// TempID 用于下游建持久 ID (clip-<tempID> / sg-<tempID>) 让事件流期间前端能预订阅.
type StopResult struct {
	Events  []inputclip.Event
	Meta    inputclip.ClipMeta
	ClientW int
	ClientH int
	TempID  string
}

// Recorder 录制游戏窗口内的键鼠操作。线程安全：Start/Stop/Cancel/Active 都加锁。
type Recorder struct {
	mu sync.Mutex

	active  bool
	tempID  string // 临时 recording ID（前端订阅事件流过滤用）
	hwnd    win.HWND
	clientW int
	clientH int

	// 录制 metadata (Start 时传入, drainLoop 据此决定 filter 行为)
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
	tStartUs uint64

	// seq 单调递增, 同 TUs 内 tie-break
	seqCounter uint32

	// mouseCounts360Getter Stop() 时调用快照当前 settings 的 360° HID counts。
	// nil 或返 0 = 录制 metadata 里不写（回放也就不缩放）。
	mouseCounts360Getter func() int
}

// NewRecorder 构造 Recorder。
func NewRecorder() *Recorder {
	return &Recorder{}
}

// SetMouseCounts360Getter 注入 settings 取值函数。main.go 启动时调一次。
func (r *Recorder) SetMouseCounts360Getter(f func() int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.mouseCounts360Getter = f
}

// Active 当前是否在录。
func (r *Recorder) Active() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.active
}

// Start 启动录制。返回临时 recording ID（前端订阅事件流过滤用）。
//   - gameHwnd：游戏窗口；mouseMode=absolute 时不在它内部的鼠标事件丢弃
//   - meta：录制环境快照 (mouseMode / filterMode / stopHotkeyVK 等)
//
// 失败时 active 仍为 false，可重试。
func (r *Recorder) Start(gameHwnd win.HWND, meta inputclip.ClipMeta) (string, error) {
	r.mu.Lock()
	if r.active {
		r.mu.Unlock()
		return "", errors.New("recorder already active")
	}
	w, h, err := capture.ClientSize(gameHwnd)
	if err != nil {
		r.mu.Unlock()
		return "", fmt.Errorf("ClientSize: %w", err)
	}
	r.active = true
	r.tempID = uuid.NewString()
	r.hwnd = gameHwnd
	r.clientW = w
	r.clientH = h
	r.meta = meta
	r.started = make(chan error, 1)
	r.done = make(chan struct{})
	r.rawEvents = make(chan HookEvent, 256)
	r.drainDone = make(chan struct{})
	r.clipEvents = nil
	r.seqCounter = 0
	r.tStartUs = nowMicros()
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
//   2a) 已成功 → PostQuit + 等 done（正常 Stop 路径），再 close rawEvents
//   2b) 没成功 → worker 还卡在装 hook 阶段，等它最终失败 close(done)，
//       为安全起见也等到 done 才 close rawEvents
//   两条路都等 done 后再 close，避免竞态。
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
// 配对 / dedupe. mouseMode='relative' (FPS) 时不做窗口检测 — 防 Gemini 隐藏坑.
func (r *Recorder) drainLoop() {
	defer close(r.drainDone)
	for ev := range r.rawEvents {
		var clipEv inputclip.Event
		clipEv.TUs = nowMicros() - r.tStartUs
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
			// simple 模式不录相机转向 — 用户定义"简易 = 单击/按键 这种一次性动作".
			// 不录 RawDelta 让简易模式无需 MouseCalibration, 也不会产生需要校准的回放路径.
			if r.meta.FilterMode == "simple" {
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
			// filterMode='simple' 时丢 mouseMove
			if r.meta.FilterMode == "simple" {
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
			// 关键: mouseMode='relative' (FPS) 时不做窗口检测 — 防 Gemini 隐藏坑
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

// Stop 停止录制, 返回 raw StopResult. Service 层根据 meta.FilterMode 决定构造路径
// (precise → 建 InputClip + BuildPreciseSubgraph, simple → BuildSimpleSubgraph).
// 阻塞等 worker + drain 完全退出, 防 hook leak.
func (r *Recorder) Stop() (*StopResult, error) {
	r.mu.Lock()
	if !r.active {
		r.mu.Unlock()
		return nil, errors.New("recorder not active")
	}
	tid := r.threadID
	rawCh := r.rawEvents
	tempID := r.tempID
	meta := r.meta
	clientW := r.clientW
	clientH := r.clientH
	getter := r.mouseCounts360Getter
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

	if meta.MouseCounts360 == 0 && getter != nil {
		meta.MouseCounts360 = getter()
	}

	return &StopResult{
		Events:  events,
		Meta:    meta,
		ClientW: clientW,
		ClientH: clientH,
		TempID:  tempID,
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
