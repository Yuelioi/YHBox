package gui

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/lxn/win"
)

// 全局热键封装。Win32 RegisterHotKey 注册到当前线程的消息队列（hWnd=NULL），
// 因此必须在固定 OS 线程上跑一个 PeekMessage 循环——RegisterHotKey/UnregisterHotKey/
// 收消息都得在同一个线程，不然消息直接被发到了别的线程的队列就丢了。
//
// 设计：
//   - Start 开 goroutine + runtime.LockOSThread
//   - 在该线程里 RegisterHotKey 全部 specs → PeekMessage 轮询 WM_HOTKEY
//   - 每个 WM_HOTKEY 派发到 handler（在 hotkey 线程里同步调用）
//   - Stop 关 ctx，循环退出前 UnregisterHotKey 全部
//
// handler 一般会异步启动 SwitchTeam（带自己的 mutex 防重入），不阻塞热键循环。

const (
	winWM_HOTKEY    = 0x0312
	winMOD_ALT      = 0x0001
	winMOD_CONTROL  = 0x0002
	winMOD_SHIFT    = 0x0004
	winMOD_NOREPEAT = 0x4000
	winPM_REMOVE    = 0x0001

	// ERROR_HOTKEY_ALREADY_REGISTERED — 别的应用占了同样的组合
	errHotkeyAlreadyRegistered = 1409
)

// VK codes for 1..9（数字键 ASCII）。
const (
	winVK_1 = 0x31
	winVK_2 = 0x32
	winVK_3 = 0x33
	winVK_4 = 0x34
	winVK_5 = 0x35
	winVK_6 = 0x36
)

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	procRegisterHotKey   = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey = user32.NewProc("UnregisterHotKey")
	procPeekMessageW     = user32.NewProc("PeekMessageW")
)

// HotkeySpec 一条要注册的热键。ID 在 manager 内唯一。
type HotkeySpec struct {
	ID   int    // 1..n 即可
	Mods uint32 // winMOD_CONTROL | winMOD_SHIFT ...
	VK   uint32 // winVK_1 ...
	Name string // 给日志/错误信息用，例 "Ctrl+Shift+1"
}

// HotkeyManager 一组热键的生命周期。线程安全。
// done channel：Stop 必须等 hotkey 线程跑完 UnregisterHotKey 退出再返回，否则
// "Stop → 立刻 Start" 路径下旧线程的反注册和新线程的注册会竞态，新热键可能被
// 旧线程的延迟 UnregisterHotKey 反掉（hWnd=NULL 时 hotkey ID 跟线程绑定）。
type HotkeyManager struct {
	mu      sync.Mutex
	cancel  context.CancelFunc
	done    chan struct{}
	running bool
}

func NewHotkeyManager() *HotkeyManager {
	return &HotkeyManager{}
}

// Start 注册所有 specs 并起一个 hotkey 线程轮询消息。
// 返回错误说明某个热键被别的应用占了（specs 索引 + 名字会带在 err 里）。
// 失败时已注册的会被全部反注册；调用方不需要 Stop。
//
// handler 在热键线程里被调用，应当尽快返回——通常做法是把工作 dispatch 到自己的 goroutine。
func (m *HotkeyManager) Start(specs []HotkeySpec, handler func(id int)) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.running {
		return fmt.Errorf("已在运行")
	}
	ready := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	m.cancel = cancel
	m.done = done
	go runHotkeyLoop(ctx, specs, handler, ready, done)
	if err := <-ready; err != nil {
		cancel()
		<-done // 等线程退出（注册失败路径也会 close(done)）
		m.cancel = nil
		m.done = nil
		return err
	}
	m.running = true
	return nil
}

// Stop 取消热键线程的 ctx 并等线程退出（含反注册全部热键）。幂等。
// 阻塞直到 done 收到 close —— 给 Start 用的「立刻重注册」路径保证旧热键已彻底走人。
func (m *HotkeyManager) Stop() {
	m.mu.Lock()
	cancel, done := m.cancel, m.done
	m.cancel = nil
	m.done = nil
	m.running = false
	m.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

func (m *HotkeyManager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

// runHotkeyLoop 跑在锁定的 OS 线程：RegisterHotKey → PeekMessage 轮询 → 反注册。
// ready 通道：成功一次（nil）→ caller 解锁；失败一次（err）→ caller 收错。
// done 通道：函数退出前必 close —— 给 Stop() 阻塞等线程退出用。
func runHotkeyLoop(ctx context.Context, specs []HotkeySpec, handler func(int), ready chan<- error, done chan<- struct{}) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(done)

	// 单一退出点：defer 反注册 registered，append 跟反注册逻辑用同一份。
	// 不在失败路径手动回滚 —— defer 反注册全 registered 即可。
	registered := make([]int, 0, len(specs))
	defer func() {
		for _, id := range registered {
			procUnregisterHotKey.Call(0, uintptr(id))
		}
	}()

	for _, s := range specs {
		r, _, callErr := procRegisterHotKey.Call(
			0, uintptr(s.ID), uintptr(s.Mods), uintptr(s.VK))
		if r == 0 {
			var errno syscall.Errno
			if errors.As(callErr, &errno) && errno == errHotkeyAlreadyRegistered {
				ready <- fmt.Errorf("热键 %s 已被其它应用占用", s.Name)
			} else {
				ready <- fmt.Errorf("注册热键 %s 失败: %v", s.Name, callErr)
			}
			return
		}
		registered = append(registered, s.ID)
	}
	ready <- nil

	// PeekMessage 轮询。20ms 间隔够热键响应，CPU 占用可忽略。
	// 不用 GetMessage：GetMessage 阻塞，要靠 PostThreadMessage 唤醒退出更绕；
	// PeekMessage 配 ctx 轮询更直观。
	var msg win.MSG
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
		// PeekMessage(&msg, NULL, 0, 0, PM_REMOVE)
		r, _, _ := procPeekMessageW.Call(
			uintptr(unsafe.Pointer(&msg)), 0, 0, 0, winPM_REMOVE)
		if r == 0 {
			continue
		}
		if msg.Message == winWM_HOTKEY && handler != nil {
			handler(int(msg.WParam))
		}
	}
}
