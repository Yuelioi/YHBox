// recorder.go：整合 LL hook + actiontransform pipeline 的录制控制器。
//
// 生命周期：Start → 用户操作 → Stop（或 Cancel）。同一 Recorder 可复用。
//
// 关键 Win32 约束：
//   - SetWindowsHookEx + GetMessage 必须同一个 OS 线程
//   - 所以起 worker goroutine + runtime.LockOSThread
//   - Stop 必须阻塞等 worker 真退出（done channel），否则 Start→Stop→Start
//     会触发 duplicate callback / hook leak
//
// 单测：真录制需要真 hwnd + 真键鼠输入，单测只覆盖 vkName / 构造 / Active=false。
package recording

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lxn/win"

	"yhbox/internal/services/actions"
	"yhbox/pkg/actiontransform"
	"yhbox/pkg/capture"
)

// Recorder 录制游戏窗口内的键鼠操作。线程安全：Start/Stop/Cancel/Active 都加锁。
type Recorder struct {
	mu sync.Mutex

	active  bool
	tempID  string // 临时 action ID（前端订阅 EventRecorderEvent 过滤用）
	hwnd    win.HWND
	clientW int
	clientH int

	// hook worker 线程相关
	threadID uint32        // 给 PostQuitToThread 用
	started  chan error    // worker 安装完 hook（或失败）push 一次
	done     chan struct{} // worker 退出时 close

	// 原始事件流
	rawEvents chan HookEvent // hook callback push，drain goroutine 消费
	drainDone chan struct{}  // drain goroutine 退出信号

	// 过滤选项 —— 录制 toggle 热键自身的 mods+vk，hook 收到时跳过，
	// 否则录制结果末尾会带一堆 Ctrl/Shift/R 抖动。
	ignoreMods uint32
	ignoreVK   uint32

	// 累积 events（drain goroutine 写，Stop 读）
	events  []actiontransform.Event
	eventMu sync.Mutex

	// 实时事件回调（可选，nil = 不推前端预览）
	onEvent func(actiontransform.Event)

	// mouseCounts360Getter Stop() 时调用快照当前 settings 的 360° HID counts。
	// nil 或返 0 = 录制 metadata 里不写（回放也就不缩放）。
	mouseCounts360Getter func() int
}

// NewRecorder 构造 Recorder。onEvent 可为 nil。
func NewRecorder(onEvent func(actiontransform.Event)) *Recorder {
	return &Recorder{onEvent: onEvent}
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

// Start 启动录制。返回临时 action ID（前端订阅事件流过滤用）。
//   - gameHwnd：游戏窗口，鼠标事件不在它内部的丢弃
//   - ignoreMods/ignoreVK：录制 toggle 热键自身的 mods+vk，hook 收到时跳过
//
// 失败时 active 仍为 false，可重试。
func (r *Recorder) Start(gameHwnd win.HWND, ignoreMods, ignoreVK uint32) (string, error) {
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
	r.started = make(chan error, 1)
	r.done = make(chan struct{})
	r.rawEvents = make(chan HookEvent, 256)
	r.drainDone = make(chan struct{})
	r.ignoreMods = ignoreMods
	r.ignoreVK = ignoreVK
	r.events = nil
	tempID := r.tempID
	r.mu.Unlock()

	// worker：LockOSThread → InstallHooks → 发 started → RunMessageLoop → Uninstall
	go r.workerThread()

	// drain：从 rawEvents 转换成 actiontransform.Event 并累积
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

func (r *Recorder) drainLoop() {
	defer close(r.drainDone)
	r.mu.Lock()
	rawCh := r.rawEvents
	gameHwnd := r.hwnd
	cw, ch := r.clientW, r.clientH
	onEvent := r.onEvent
	ignoreMods := r.ignoreMods
	ignoreVK := r.ignoreVK
	r.mu.Unlock()

	// 持续追踪 modifier 状态，用于识别 record-toggle 触发那一刻的组合键
	var ctrlDown, shiftDown, altDown bool
	updateMod := func(vk uint32, down bool) {
		switch vk {
		case VK_CONTROL, VK_LCONTROL, VK_RCONTROL:
			ctrlDown = down
		case VK_SHIFT, VK_LSHIFT, VK_RSHIFT:
			shiftDown = down
		case VK_MENU, VK_LMENU, VK_RMENU:
			altDown = down
		}
	}
	matchesToggle := func(vk uint32) bool {
		if vk != ignoreVK {
			return false
		}
		var cur uint32
		if ctrlDown {
			cur |= MOD_CONTROL
		}
		if shiftDown {
			cur |= MOD_SHIFT
		}
		if altDown {
			cur |= MOD_ALT
		}
		return cur&ignoreMods == ignoreMods
	}

	for ev := range rawCh {
		if ev.IsKeyboard {
			// 过滤 record-toggle 自身 —— modifier 状态先看再 update，因为 toggle
			// 触发那一刻 modifier 还按着。
			if matchesToggle(ev.Vk) {
				updateMod(ev.Vk, ev.IsKeyDown)
				continue
			}
			updateMod(ev.Vk, ev.IsKeyDown)

			vk := vkName(ev.Vk)
			if vk == "" {
				continue // 不识别的 VK 丢
			}
			atEv := actiontransform.Event{
				TimestampMs: ev.TimestampMs,
				Type:        actiontransform.EventKeyboard,
				Vk:          vk,
				IsDown:      ev.IsKeyDown,
			}
			r.appendEvent(atEv, onEvent)
		} else if ev.IsRawDelta {
			// Raw Input 相对位移 —— 不做窗口过滤（游戏全屏/失焦时也录，camera 转向常态）
			r.appendEvent(actiontransform.Event{
				TimestampMs: ev.TimestampMs,
				Type:        actiontransform.EventRawMouseDelta,
				Dx:          ev.RawDx,
				Dy:          ev.RawDy,
			}, onEvent)
		} else {
			// 鼠标事件：先按位置过滤（不在游戏窗口内的全丢）
			if !IsPointInsideGameWindow(gameHwnd, ev.ScreenX, ev.ScreenY) {
				continue
			}
			cx, cy, ok := screenToClient(gameHwnd, ev.ScreenX, ev.ScreenY)
			if !ok {
				continue
			}
			if cx < 0 || cy < 0 || cx >= int32(cw) || cy >= int32(ch) {
				continue
			}
			switch {
			case ev.IsScroll:
				r.appendEvent(actiontransform.Event{
					TimestampMs:  ev.TimestampMs,
					Type:         actiontransform.EventMouseScroll,
					X:            int(cx),
					Y:            int(cy),
					WheelNotches: ev.WheelNotches,
				}, onEvent)
			case ev.IsMouseMove:
				r.appendEvent(actiontransform.Event{
					TimestampMs: ev.TimestampMs,
					Type:        actiontransform.EventMouseMove,
					X:           int(cx),
					Y:           int(cy),
				}, onEvent)
			default:
				// 按键 down/up
				r.appendEvent(actiontransform.Event{
					TimestampMs: ev.TimestampMs,
					Type:        actiontransform.EventMouse,
					MouseBtn:    ev.MouseBtn,
					X:           int(cx),
					Y:           int(cy),
					IsDown:      ev.IsMouseDown,
				}, onEvent)
			}
		}
	}
}

func (r *Recorder) appendEvent(ev actiontransform.Event, onEvent func(actiontransform.Event)) {
	r.eventMu.Lock()
	r.events = append(r.events, ev)
	r.eventMu.Unlock()
	if onEvent != nil {
		onEvent(ev)
	}
}

// Stop 停止录制，把累积 events 转成 Action 并返回（不写库，caller 决定保存/丢弃）。
// 阻塞等 worker + drain 完全退出，防 hook leak。
func (r *Recorder) Stop() (actions.Action, error) {
	r.mu.Lock()
	if !r.active {
		r.mu.Unlock()
		return actions.Action{}, errors.New("recorder not active")
	}
	tid := r.threadID
	tempID := r.tempID
	cw, ch := r.clientW, r.clientH
	rawCh := r.rawEvents
	r.mu.Unlock()

	// 1) 让 worker 退出（PostQuit → GetMessage 返回 → Uninstall → close(done)）
	PostQuitToThread(tid)
	<-r.done

	// 2) drain 还在跑（channel 还开着），close 让它退
	close(rawCh)
	<-r.drainDone

	// 3) 转换
	r.eventMu.Lock()
	raw := r.events
	r.events = nil
	r.eventMu.Unlock()

	r.mu.Lock()
	r.active = false
	r.mu.Unlock()

	steps := actiontransform.EventsToSteps(raw, cw, ch)
	var counts int
	if r.mouseCounts360Getter != nil {
		counts = r.mouseCounts360Getter()
	}
	a := actions.Action{
		ID:    tempID,
		Name:  "录制 " + time.Now().Format("01-02 15:04"),
		Steps: steps,
		RecordingContext: actions.RecordingContext{
			Resolution:     [2]int{cw, ch},
			DPIScale:       1.0,
			MouseCounts360: counts,
		},
	}
	actions.NormalizeAction(&a)
	return a, nil
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
	r.events = nil
	r.mu.Unlock()
}
