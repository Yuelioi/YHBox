package hotkey

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sort"
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
//   - 内部维护 bindings map[int]*binding（id → spec/owner/onFire）
//   - Start: 批量加 OwnerBattle bindings；Stop: 批量删 OwnerBattle bindings
//   - Register/Unregister/Replace: 单条增删改（任意 owner）
//   - 任何写操作 → rebuildLocked：停现有线程 → 用 bindings 重起一遍
//   - bindings 空 → 不起线程；后续再 Register 自动起
//   - 失败用快照回滚（rebuild 内部）：保证 bindings/线程状态一致
//
// 不走 "PostThreadMessage 命令" 的复杂路线 —— 简单 Stop+Start 重建 < 10ms 完全可接受。
//
// handler 在热键线程被同步调用，按惯例应 dispatch goroutine 干活，别阻塞循环。

// Win32 RegisterHotKey 修饰键标志 + 数字 VK。导出版本给跨包消费者用
// （pkg/actiontransform / battle service 等），原 winXxx 名以 const alias 留在
// 包内简化引用。
const (
	MOD_ALT      uint32 = 0x0001
	MOD_CONTROL  uint32 = 0x0002
	MOD_SHIFT    uint32 = 0x0004
	MOD_NOREPEAT uint32 = 0x4000

	VK_1 uint32 = 0x31
	VK_2 uint32 = 0x32
	VK_3 uint32 = 0x33
	VK_4 uint32 = 0x34
	VK_5 uint32 = 0x35
	VK_6 uint32 = 0x36
)

const (
	winWM_HOTKEY = 0x0312
	winPM_REMOVE = 0x0001

	winMOD_ALT      = MOD_ALT
	winMOD_CONTROL  = MOD_CONTROL
	winMOD_SHIFT    = MOD_SHIFT
	winMOD_NOREPEAT = MOD_NOREPEAT

	winVK_1 = VK_1
	winVK_2 = VK_2
	winVK_3 = VK_3
	winVK_4 = VK_4
	winVK_5 = VK_5
	winVK_6 = VK_6

	// ERROR_HOTKEY_ALREADY_REGISTERED — 别的应用占了同样的组合
	errHotkeyAlreadyRegistered = 1409
)

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	procRegisterHotKey   = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey = user32.NewProc("UnregisterHotKey")
	procPeekMessageW     = user32.NewProc("PeekMessageW")
)

// HotkeySpec 一条要注册的热键。
//
// 注意：动态 API（Register/Replace）会**忽略** ID 字段并由 manager 内部分配；
// 老的 Start API 仍按 spec.ID 注册（保留 battle 现有语义）。
type HotkeySpec struct {
	ID   int    // 1..n 即可；动态 API 下被 manager 覆盖
	Mods uint32 // winMOD_CONTROL | winMOD_SHIFT ...
	VK   uint32 // winVK_1 ...
	Name string // 给日志/错误信息用，例 "Ctrl+Shift+1"
}

// BindingOwner 标记 binding 归属，便于 Stop 批量清 + debug 面板分组。
type BindingOwner string

const (
	OwnerBattle   BindingOwner = "battle"
	OwnerAction   BindingOwner = "action"
	OwnerRecorder BindingOwner = "recorder"
)

// BindingInfo 对外暴露的只读视图。
type BindingInfo struct {
	ID    int
	Owner BindingOwner
	Spec  HotkeySpec
}

// binding 内部状态：spec + 归属 + 触发回调。
type binding struct {
	spec   HotkeySpec
	owner  BindingOwner
	onFire func()
}

// HotkeyManager 一组热键的生命周期。线程安全。
// done channel：每次启动 loop 时新建，Stop/重建必须等线程跑完 UnregisterHotKey 才返回，
// 否则旧线程的反注册和新线程的注册会竞态，新热键可能被旧线程的延迟 UnregisterHotKey 反掉
// （hWnd=NULL 时 hotkey ID 跟线程绑定）。
type HotkeyManager struct {
	mu      sync.Mutex
	cancel  context.CancelFunc
	done    chan struct{}
	running bool

	// 动态 binding 状态
	bindings map[int]*binding
	nextID   int
}

func NewHotkeyManager() *HotkeyManager {
	return &HotkeyManager{
		bindings: make(map[int]*binding),
		nextID:   1000, // 给老 Start 路径（用 spec.ID=1..6）留空间，避免撞 id
	}
}

// Start 注册一批 OwnerBattle 热键。保留旧契约：
//   - 已运行（任意 owner 已有 bindings 或线程在跑） → "已在运行" 错
//   - 内部把 specs 写进 bindings 后调 rebuildLocked
//   - 失败时回滚清除新加的 battle bindings
//
// handler 在热键线程里同步调用 —— 派发 goroutine 是 caller 的责任。
func (m *HotkeyManager) Start(specs []HotkeySpec, handler func(id int)) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// "已在运行" 的语义是 battle 已 Enable —— action binding 不算
	if m.hasOwnerLocked(OwnerBattle) {
		return fmt.Errorf("已在运行")
	}
	snapshot := m.snapshotBindingsLocked()
	// 把 specs 转 binding（spec.ID 保持，对外仍是 1..6，battle 的 onHotkeyFired 不变）
	for _, s := range specs {
		capS := s
		id := s.ID
		// 避免覆盖既有 binding（例如 action 已注册某 id）
		if _, exists := m.bindings[id]; exists {
			m.restoreBindingsLocked(snapshot)
			return fmt.Errorf("内部 binding id 冲突: %d", id)
		}
		m.bindings[id] = &binding{
			spec:   s,
			owner:  OwnerBattle,
			onFire: func() { handler(capS.ID) },
		}
		if id >= m.nextID {
			m.nextID = id + 1
		}
	}
	if err := m.rebuildLocked(); err != nil {
		m.restoreBindingsLocked(snapshot)
		return err
	}
	return nil
}

// Stop 反注册所有 OwnerBattle bindings，rebuild。
// 如果还有其它 owner 的 binding，loop 继续跑；全空才彻底停。幂等。
func (m *HotkeyManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.bindings) == 0 && !m.running {
		return
	}
	for id, b := range m.bindings {
		if b.owner == OwnerBattle {
			delete(m.bindings, id)
		}
	}
	// rebuild 失败这里没法返回 error —— 但 Stop 的语义是尽力，记不下也只能继续。
	// 实践上 rebuild 失败只可能是新注册冲突，而 Stop 不引入新注册，不会出错。
	_ = m.rebuildLocked()
}

// IsRunning 等价于"是否有 OwnerBattle 的 binding"——保留 battle 现有用法：
// battle 用 IsRunning 做 "我有没有 Enable 过" 的 proxy。
//
// 注意：这跟"hotkey 线程是否在跑"不完全相等（action 可能有 binding 让线程跑着，
// 但 battle 没 Enable）。battle 关心的就是 battle 自己的状态。
func (m *HotkeyManager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.hasOwnerLocked(OwnerBattle)
}

// HasOwner 检查指定 owner 是否有 binding。debug 面板/其它 service 用。
func (m *HotkeyManager) HasOwner(owner BindingOwner) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.hasOwnerLocked(owner)
}

func (m *HotkeyManager) hasOwnerLocked(owner BindingOwner) bool {
	for _, b := range m.bindings {
		if b.owner == owner {
			return true
		}
	}
	return false
}

// Register 动态注册单个热键，spec.ID 被忽略，manager 分配新 id 返回。
// 冲突（其它进程占用）→ 返 error 且 bindings 回滚到调用前 + 旧 loop 重启保持原热键工作。
func (m *HotkeyManager) Register(spec HotkeySpec, owner BindingOwner, onFire func()) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	snapshot := m.snapshotBindingsLocked()
	id := m.nextID
	m.nextID++
	spec.ID = id
	m.bindings[id] = &binding{spec: spec, owner: owner, onFire: onFire}
	if err := m.rebuildLocked(); err != nil {
		m.restoreBindingsLocked(snapshot)
		return 0, err
	}
	return id, nil
}

// Unregister 删一个 binding 并 rebuild。id 不存在 → 当 no-op 返 nil。
// rebuild 失败（理论上少 binding 不应失败）→ 回滚 + 重启 snapshot loop。
func (m *HotkeyManager) Unregister(id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.bindings[id]; !ok {
		return nil
	}
	snapshot := m.snapshotBindingsLocked()
	delete(m.bindings, id)
	if err := m.rebuildLocked(); err != nil {
		m.restoreBindingsLocked(snapshot)
		return err
	}
	return nil
}

// Replace 改一个 binding 的 spec / onFire。spec.ID 被覆盖为 id。
// 失败 → bindings 回滚到调用前（旧 binding 保留），调用方不会因为配错把热键丢了。
func (m *HotkeyManager) Replace(id int, spec HotkeySpec, onFire func()) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	old, ok := m.bindings[id]
	if !ok {
		return fmt.Errorf("binding id=%d 不存在", id)
	}
	snapshot := m.snapshotBindingsLocked()
	spec.ID = id
	m.bindings[id] = &binding{spec: spec, owner: old.owner, onFire: onFire}
	if err := m.rebuildLocked(); err != nil {
		m.restoreBindingsLocked(snapshot)
		return err
	}
	return nil
}

// Bindings 当前所有 binding 的快照，按 id 升序。debug 面板用。
func (m *HotkeyManager) Bindings() []BindingInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]BindingInfo, 0, len(m.bindings))
	for _, b := range m.bindings {
		out = append(out, BindingInfo{ID: b.spec.ID, Owner: b.owner, Spec: b.spec})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// rebuildLocked 停现有 hotkey 线程 → 用当前 bindings 启 loop。
// 假设 m.mu 已持有。**不**自己 rollback —— 失败时 caller 决定是否回滚 bindings
// 再重试。caller 应当在 modify bindings 之前自己 snapshot；rebuildLocked 入口 snapshot
// 已包含 caller 的新 binding，回滚会污染。
func (m *HotkeyManager) rebuildLocked() error {
	// 停老线程
	if m.cancel != nil {
		m.cancel()
		<-m.done
		m.cancel = nil
		m.done = nil
		m.running = false
	}
	return m.startLoopLocked()
}

// snapshotBindingsLocked 给 mutator 用：modify 前拷一份 m.bindings，失败时还原。
// 假设 m.mu 已持有。值是 *binding 浅拷贝（内容不 mutate，只整体替换）。
func (m *HotkeyManager) snapshotBindingsLocked() map[int]*binding {
	out := make(map[int]*binding, len(m.bindings))
	for k, v := range m.bindings {
		out[k] = v
	}
	return out
}

// restoreBindingsLocked rebuildLocked 失败后 caller 调用：还原 bindings + 重启 snapshot loop。
// snapshot 自己也启不起来（理论不应该）→ bindings 仍是 snapshot 但 loop 死，
// 此时系统不健康但不会再次报错。
func (m *HotkeyManager) restoreBindingsLocked(snapshot map[int]*binding) {
	m.bindings = snapshot
	_ = m.rebuildLocked()
}

// startLoopLocked 用当前 m.bindings 启 hotkey loop。假设 m.mu 已持有，且 m.cancel == nil。
// bindings 空 → no-op 成功返回。
// 失败：m.cancel/m.done/m.running 保持 nil/false；caller 决定回滚。
func (m *HotkeyManager) startLoopLocked() error {
	if len(m.bindings) == 0 {
		return nil
	}

	specs := make([]HotkeySpec, 0, len(m.bindings))
	for _, b := range m.bindings {
		specs = append(specs, b.spec)
	}
	sort.Slice(specs, func(i, j int) bool { return specs[i].ID < specs[j].ID })

	// dispatcher 拿当前 m.bindings 的引用做 id→onFire 查表。
	// 关键：捕获 m 自身；rebuild 后 m.bindings 是新版本，但本次 loop 启动时 specs
	// 也是新版本，所以 id 必然在 m.bindings 里。用 mu 短临界区保护读。
	dispatcher := func(id int) {
		m.mu.Lock()
		b, ok := m.bindings[id]
		m.mu.Unlock()
		if ok && b.onFire != nil {
			b.onFire()
		}
	}

	ready := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go runHotkeyLoop(ctx, specs, dispatcher, ready, done)
	if err := <-ready; err != nil {
		cancel()
		<-done
		return err
	}
	m.cancel = cancel
	m.done = done
	m.running = true
	return nil
}

// runHotkeyLoop 跑在锁定的 OS 线程：RegisterHotKey → PeekMessage 轮询 → 反注册。
// ready 通道：成功一次（nil）→ caller 解锁；失败一次（err）→ caller 收错。
// done 通道：函数退出前必 close —— 给 rebuild/Stop 阻塞等线程退出用。
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
