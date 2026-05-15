package actions

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/google/uuid"
)

// ---- 抽象接口（解耦 actions ↔ services）----
//
// actions 包不能 import services（main.go 同时构造两边，反向 import 会 cycle）。
// 这些接口由 main.go 写适配器实现 *services.App / *runtime.Runner。

// BotLease 跟 services.BotLease 同语义。
type BotLease interface {
	Release()
}

// BotGate 由 services.App 适配。负责 bot mutex（避免 action 和长跑 bot 抢游戏输入）。
type BotGate interface {
	AcquireBot(name string) (BotLease, error)
}

// GameProvider 提供当前游戏窗口的 HWND。uintptr 避免 actions 包 import lxn/win。
type GameProvider interface {
	HWND() (hwnd uintptr, ok bool)
	BringToForeground(hwnd uintptr)
}

// Runner 抽象 runtime.Runner。SetHWND 接 uintptr —— 适配器内部转 win.HWND。
type Runner interface {
	Start(a *Action) error
	Stop() error
	SetHWND(hwnd uintptr)
}

// WindowOpener 用来开独立编辑器窗口（actions 不能直 import wails3）。
type WindowOpener interface {
	OpenEditor(actionID string) error
}

// RecorderHost 抽象 recording.Recorder。同上避免循环依赖：actions 不能 import recording
// （recording 已经 import actions）。
type RecorderHost interface {
	Start(gameHwnd uintptr, ignoreMods, ignoreVK uint32) (tempActionID string, err error)
	Stop() (Action, error)
	Cancel()
	Active() bool
}

// ---- Service ----

// Service Action 是纯录制 macro，不绑热键 / 不 require foreground
// （这两层在 Container/Schedule）。暴露 CRUD + RunOnce/StopRunning + 录制 RPC。
type Service struct {
	store    *Store
	runner   Runner
	botGate  BotGate
	game     GameProvider
	recorder RecorderHost // 可为 nil（Phase 1/2 测试场景）
	windows  WindowOpener // 可为 nil（单测/平台不支持独立窗口时）

	// 录制 toggle 热键的 Win32 mods+vk —— 传给 recorder 用做过滤
	recordToggleMods uint32
	recordToggleVK   uint32

	// emit wails Event.Emit 包装。
	emit func(name string, data any)

	mu sync.Mutex
	// 当前正在跑的 action → bot lease。
	leasesByActionID map[string]BotLease
}

// NewService 全部依赖必须显式注入。
func NewService(store *Store, runner Runner, botGate BotGate, game GameProvider) *Service {
	return &Service{
		store:            store,
		runner:           runner,
		botGate:          botGate,
		game:             game,
		leasesByActionID: map[string]BotLease{},
	}
}

// SetRecorder 注入 recorder + 当前录制 toggle 热键（mods+vk）。
func (s *Service) SetRecorder(rec RecorderHost, toggleMods, toggleVK uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recorder = rec
	s.recordToggleMods = toggleMods
	s.recordToggleVK = toggleVK
}

// SetWindowOpener 注入独立编辑器窗口的开窗器。
func (s *Service) SetWindowOpener(w WindowOpener) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.windows = w
}

// OpenEditorWindow 给前端 RPC 用：打开 actionID 对应的编辑器独立窗口。
func (s *Service) OpenEditorWindow(actionID string) error {
	if _, ok := s.store.Get(actionID); !ok {
		return fmt.Errorf("action %q not found", actionID)
	}
	s.mu.Lock()
	w := s.windows
	s.mu.Unlock()
	if w == nil {
		return errors.New("window opener 未注入")
	}
	return w.OpenEditor(actionID)
}

// SetEmit 注入 wails Event.Emit 包装。
func (s *Service) SetEmit(emit func(name string, data any)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.emit = emit
}

func (s *Service) List() []Action {
	return s.store.List()
}

// emitChanged 广播给所有窗口"action 列表变了"。
func (s *Service) emitChanged() {
	emit := s.emit
	if emit != nil {
		emit("action:changed", map[string]any{})
	}
}

// Create 以 name 起 action，其它字段默认。
func (s *Service) Create(name string) (Action, error) {
	a := Action{
		ID:   uuid.NewString(),
		Name: name,
	}
	NormalizeAction(&a)
	if err := a.Validate(); err != nil {
		return Action{}, err
	}
	if err := s.store.Create(&a); err != nil {
		return Action{}, err
	}
	s.emitChanged()
	return a, nil
}

// Update 接 RFC7386 merge patch。
func (s *Service) Update(id, patchJSON string) error {
	err := s.store.Update(id, func(a *Action) error {
		cur, err := json.Marshal(a)
		if err != nil {
			return err
		}
		merged, err := jsonpatch.MergePatch(cur, []byte(patchJSON))
		if err != nil {
			return fmt.Errorf("apply patch: %w", err)
		}
		if err := json.Unmarshal(merged, a); err != nil {
			return err
		}
		NormalizeAction(a)
		return a.Validate()
	})
	if err == nil {
		s.emitChanged()
	}
	return err
}

// Delete 删 action。
func (s *Service) Delete(id string) error {
	if err := s.store.Delete(id); err != nil {
		return err
	}
	s.emitChanged()
	return nil
}

// RunOnce 跑一次 action：拿 bot lease + 绑 hwnd + Runner.Start。
// 不在这里管前台（含 click 的 action 由 Container/Schedule 调用前确保前台）。
func (s *Service) RunOnce(id string) error {
	a, ok := s.store.Get(id)
	if !ok {
		return fmt.Errorf("action %q not found", id)
	}

	hwnd, hwndOK := s.game.HWND()
	if !hwndOK {
		return errors.New("未检测到游戏窗口")
	}
	s.runner.SetHWND(hwnd)

	lease, err := s.botGate.AcquireBot("action:" + a.Name)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.leasesByActionID[a.ID] = lease
	s.mu.Unlock()

	if err := s.runner.Start(&a); err != nil {
		s.mu.Lock()
		delete(s.leasesByActionID, a.ID)
		s.mu.Unlock()
		lease.Release()
		return err
	}
	return nil
}

// OnRunnerEvent 由 main.go 在 Runner emit 回调里调，释放 lease。
func (s *Service) OnRunnerEvent(actionID, status string) {
	if status == "running" {
		return
	}
	s.mu.Lock()
	lease := s.leasesByActionID[actionID]
	delete(s.leasesByActionID, actionID)
	s.mu.Unlock()
	if lease != nil {
		lease.Release()
	}
}

// StopRunning 打断当前正在跑的 action。
func (s *Service) StopRunning() error {
	return s.runner.Stop()
}

// ---- 录制 ----

// StartRecording 开始录制。
func (s *Service) StartRecording() (string, error) {
	s.mu.Lock()
	rec := s.recorder
	mods := s.recordToggleMods
	vk := s.recordToggleVK
	s.mu.Unlock()
	if rec == nil {
		return "", errors.New("recorder 未初始化")
	}
	if rec.Active() {
		return "", nil
	}
	hwnd, ok := s.game.HWND()
	if !ok {
		return "", errors.New("未检测到游戏窗口")
	}
	tempID, err := rec.Start(hwnd, mods, vk)
	if err != nil {
		return "", err
	}
	if s.emit != nil {
		s.emit("action:recorder-state", map[string]any{
			"status":       "started",
			"tempActionId": tempID,
		})
	}
	return tempID, nil
}

// StopRecording 停录、转 step、写入 store。
func (s *Service) StopRecording() (Action, error) {
	s.mu.Lock()
	rec := s.recorder
	s.mu.Unlock()
	if rec == nil {
		return Action{}, errors.New("recorder 未初始化")
	}
	if !rec.Active() {
		return Action{}, nil
	}
	a, err := rec.Stop()
	if err != nil {
		return Action{}, err
	}
	if err := a.Validate(); err != nil {
		return a, fmt.Errorf("录制结果 invalid: %w", err)
	}
	if err := s.store.Create(&a); err != nil {
		return a, fmt.Errorf("保存录制结果失败: %w", err)
	}
	if s.emit != nil {
		s.emit("action:recorder-state", map[string]any{
			"status": "stopped",
			"action": a,
		})
	}
	s.emitChanged()
	return a, nil
}

// CancelRecording 丢弃当前录制，不写库。
func (s *Service) CancelRecording() {
	s.mu.Lock()
	rec := s.recorder
	s.mu.Unlock()
	if rec == nil || !rec.Active() {
		return
	}
	rec.Cancel()
}

// IsRecording 给 toggle 热键回调判方向用。
func (s *Service) IsRecording() bool {
	s.mu.Lock()
	rec := s.recorder
	s.mu.Unlock()
	if rec == nil {
		return false
	}
	return rec.Active()
}
