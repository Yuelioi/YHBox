package recording

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yottaapp/yotta/internal/apperr"
	"github.com/yottaapp/yotta/internal/automation/target"
	"github.com/yottaapp/yotta/internal/services/container"
	"github.com/yottaapp/yotta/internal/services/inputclip"
)

// HotkeySettingsProvider 给 Service 拿停录热键 VK + mouseMode.
// nil = 默认 F12 + relative.
type HotkeySettingsProvider interface {
	GetStopHotkeyVK() uint32  // 0x7B = F12 默认
	GetPauseHotkeyVK() uint32 // 0x7A = F11 默认 (暂停/继续切换); 0 = 不启用
	GetMouseMode() string     // 'relative' / 'absolute'
}

type clipStore interface {
	Save(clip *inputclip.InputClip) error
	List() []inputclip.ClipSummary
	Delete(id string) error
}

// ContainerGetter 窄接口 — recording 拿 container 解析 Win32WindowTarget hwnd 用.
// container.Store 已实现 Get(id) (Container, bool).
type ContainerGetter interface {
	Get(id string) (container.Container, bool)
}

// Service wails3 RPC 入口.
//
// 前端流程:
//  1. Start({containerID}) 启动 hook.
//  2. Stop() 只生成 pending token，不创建库资产.
//  3. Finalize(metadata) 创建 InputClip; Discard 释放 pending 数据.
//
// 停录热键: LL hook 直接检测 → return 1 拦截不透传游戏 → 异步调 StopAsync.
// StopAsync 完成时 emit 'recording:completed' pending payload 或 {error}.
type Service struct {
	rec          recorderLifecycle
	hkProv       HotkeySettingsProvider
	clipSvc      clipStore
	containerGet ContainerGetter
	emit         func(name string, data any)

	mu           sync.Mutex // 串行化 Start/Stop 命令 (防 F12 callback 跟 UI Stop 重入)
	pending      map[string]pendingRecording
	stateMu      sync.RWMutex // 保护 state 快照 — 跟 mu 分离, GetState 读路径不被慢的 rec.Stop 阻塞
	state        RecordingState
	closed       atomic.Bool
	shutdownOnce sync.Once
	shutdownDone chan struct{}
}

func NewService(rec recorderLifecycle, hkProv HotkeySettingsProvider, clipSvc clipStore) *Service {
	return &Service{
		rec: rec, hkProv: hkProv, clipSvc: clipSvc,
		state: RecordingState{Phase: PhaseIdle}, pending: map[string]pendingRecording{}, shutdownDone: make(chan struct{}),
	}
}

// Phase 常量 — 录制生命周期的三个权威阶段.
const (
	PhaseIdle       = "idle"
	PhaseRecording  = "recording"
	PhasePaused     = "paused"
	PhaseFinalizing = "finalizing"
)

// RecordingState 录制子系统的权威状态. 后端是唯一真相源; 前端/HUD 只镜像
// (recording:state 事件 + GetState 对账), 不自己存可 desync 的 flag.
type RecordingState struct {
	Phase       string `json:"phase"`       // idle | recording | paused | finalizing
	ContainerID string `json:"containerID"` // 录制目标容器
	TempID      string `json:"tempID"`
	StartedAtMs int64  `json:"startedAtMs"`
	PausedMs    int64  `json:"pausedMs"`   // 累计已暂停毫秒, HUD 算录制时长 = now-startedAt-pausedMs
	PausedAtMs  int64  `json:"pausedAtMs"` // 本次暂停起点 wall time (>0 即处于暂停, HUD 冻结计时); recording 态为 0
}

// GetState 返回当前权威状态快照. RPC — 前端任何时候可查 (reconcile 对账用).
// 走 stateMu 读路径, 不被慢的 rec.Stop() 阻塞.
func (s *Service) GetState() RecordingState {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.state
}

// phase 读当前 phase (命令幂等判定用).
func (s *Service) phase() string {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.state.Phase
}

// setState 写状态快照 + 广播 recording:state (全量). 不持 stateMu emit, 防 re-entry / 慢 emit 卡转换.
func (s *Service) setState(st RecordingState) {
	s.stateMu.Lock()
	s.state = st
	s.stateMu.Unlock()
	if s.emit != nil {
		s.emit("recording:state", st)
	}
}

// ConfigureEmitter injects the presentation event transport during startup.
// It is deliberately a package function so it cannot become a Wails RPC method.
func ConfigureEmitter(s *Service, emit func(name string, data any)) { s.emit = emit }

// ConfigureContainerGetter injects target lookup used when a recording starts.
func ConfigureContainerGetter(s *Service, getter ContainerGetter) { s.containerGet = getter }

// Shutdown cancels an in-progress recording without persisting a partial asset.
// It is a package function so lifecycle wiring does not become a Wails RPC.
func Shutdown(ctx context.Context, s *Service) error {
	s.shutdownOnce.Do(func() {
		s.closed.Store(true)
		go s.shutdown()
	})
	select {
	case <-s.shutdownDone:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Service) shutdown() {
	s.mu.Lock()
	wasActive := s.phase() != PhaseIdle
	if wasActive {
		s.rec.Cancel()
		setActiveStopHotkey(0, nil)
		setActivePauseHotkey(0, nil)
	}
	s.mu.Unlock()
	if wasActive {
		s.setState(RecordingState{Phase: PhaseIdle})
	}
	close(s.shutdownDone)
}

// ValidateTarget 录制前预检 — 解析容器 Win32WindowTarget 找窗口 (找不到返 error), 并把窗口拉到前台.
// 前端在 3s 倒计时**之前**调: 没设/找不到窗口立刻报错 (不用等录完), 成功则游戏已置前台省去用户 Alt-Tab.
// 纯预检 — 不装 hook 不起 recorder. Start 内仍保留同样校验作 race 兜底 (倒计时期间窗口可能消失).
func (s *Service) ValidateTarget(containerID string) error {
	if containerID == "" {
		return apperr.New(apperr.CodeContainerIDRequired, nil)
	}
	if s.containerGet == nil {
		return errors.New("ContainerGetter 未注入")
	}
	cont, ok := s.containerGet.Get(containerID)
	if !ok {
		return fmt.Errorf("container %q not found", containerID)
	}
	wtNode := findWin32WindowTargetNode(&cont)
	if wtNode == nil {
		return apperr.New(apperr.CodeRecordingNoWin32WindowTarget, nil)
	}
	spec := readMatchSpecFromConfig(wtNode)
	window, err := resolveRecordingWindow(spec)
	if err != nil {
		return fmt.Errorf("窗口未找到: %w", err)
	}
	_ = bringRecordingWindowToFront(window.HWND)
	return nil
}

// StartArgs 前端传入的录制开关.
type StartArgs struct {
	ContainerID string `json:"containerID"` // 必传；用于解析录制目标窗口
}

// Start 启动录制 (非阻塞). 返回临时录制 ID (前端订阅事件流过滤用).
//
// Start 后会 atomic 设 LL hook 的 stopHotkeyVK + callback — 用户在游戏前台按 F12 时,
// hook 直接拦截 (不透传游戏) 并异步触发 StopAsync.
func (s *Service) Start(args StartArgs) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed.Load() {
		return "", errors.New("recording service is closed")
	}

	// 幂等: 已经在录 (或正在收尾) → 不重复启动, 返当前 tempID. 前端误触/重入无害.
	if s.phase() != PhaseIdle {
		return s.GetState().TempID, nil
	}

	if args.ContainerID == "" {
		return "", apperr.New(apperr.CodeContainerIDRequired, nil)
	}
	if s.containerGet == nil {
		return "", errors.New("ContainerGetter 未注入 (main.go 启动期 ConfigureContainerGetter?)")
	}
	cont, ok := s.containerGet.Get(args.ContainerID)
	if !ok {
		return "", fmt.Errorf("container %q not found", args.ContainerID)
	}
	wtNode := findWin32WindowTargetNode(&cont)
	if wtNode == nil {
		return "", apperr.New(apperr.CodeRecordingNoWin32WindowTarget, nil)
	}
	spec := readMatchSpecFromConfig(wtNode)
	wh, err := resolveRecordingWindow(spec)
	if err != nil {
		return "", fmt.Errorf("窗口未找到: %w", err)
	}
	mouseMode := "relative"
	stopVK := uint32(0x7B)
	if s.hkProv != nil {
		mouseMode = s.hkProv.GetMouseMode()
		stopVK = s.hkProv.GetStopHotkeyVK()
	}
	// 录制基准分辨率取目标窗口客户区实际尺寸 (回放跨分辨率缩放用). 取不到 (≤0) 直接
	// 返 error 让用户重试 —— 兜底 1080p 反而让回放按错基准缩放绝对坐标, 比不缩放更糟.
	baseW, baseH := wh.ClientW, wh.ClientH
	if baseW <= 0 || baseH <= 0 {
		return "", fmt.Errorf("无法读取目标窗口客户区尺寸 (得 %dx%d), 请确认窗口已正常显示后重试", baseW, baseH)
	}
	meta := inputclip.ClipMeta{
		MouseMode:      mouseMode,
		StopHotkeyVK:   stopVK,
		BaseResolution: [2]int{baseW, baseH},
	}
	id, recErr := s.rec.Start(wh.HWND, meta)
	if recErr != nil {
		return "", fmt.Errorf("recorder.Start: %w", recErr)
	}
	s.setState(RecordingState{
		Phase:       PhaseRecording,
		ContainerID: args.ContainerID,
		TempID:      id,
		StartedAtMs: time.Now().UnixMilli(),
	})
	setActiveStopHotkey(stopVK, func() {
		s.StopAsync()
	})
	// 暂停/继续切换热键 (可选). 录制中按 → 暂停; 暂停中按 → emit resume-hotkey 让 HUD 走 3s 倒计时再继续.
	var pauseVK uint32
	if s.hkProv != nil {
		pauseVK = s.hkProv.GetPauseHotkeyVK()
	}
	if pauseVK != 0 {
		setActivePauseHotkey(pauseVK, func() {
			switch s.phase() {
			case PhaseRecording:
				_ = s.Pause()
			case PhasePaused:
				if s.emit != nil {
					s.emit("recording:resume-hotkey", nil)
				}
			}
		})
	}
	return id, nil
}

// Pause 暂停录制 (HUD 按钮触发). 切除间隔语义: 暂停期不录, 时间戳扣除该段 → 回放无空档.
// 仅 recording → paused; 其它 phase no-op 返 nil (幂等). 持 s.mu 串行化跟 Stop/Resume 互斥.
func (s *Service) Pause() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed.Load() {
		return nil
	}
	if s.phase() != PhaseRecording {
		return nil
	}
	s.rec.Pause()
	cur := s.GetState()
	cur.Phase = PhasePaused
	cur.PausedAtMs = time.Now().UnixMilli()
	s.setState(cur)
	return nil
}

// Resume 继续录制 (HUD 按钮触发). 把本次暂停时长累加进 PausedMs, 清 PausedAtMs.
// 仅 paused → recording; 其它 phase no-op 返 nil (幂等).
func (s *Service) Resume() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed.Load() {
		return nil
	}
	if s.phase() != PhasePaused {
		return nil
	}
	s.rec.Resume()
	cur := s.GetState()
	if cur.PausedAtMs > 0 {
		cur.PausedMs += time.Now().UnixMilli() - cur.PausedAtMs
	}
	cur.PausedAtMs = 0
	cur.Phase = PhaseRecording
	s.setState(cur)
	return nil
}

// StopResultPayload 描述尚未入库的录制结果，供前端打开命名表单.
type StopResultPayload struct {
	PendingID   string `json:"pendingID"`
	ContainerID string `json:"containerID"`
	DurationUs  uint64 `json:"durationUs"`
	EventCount  int    `json:"eventCount"`
}

type pendingRecording struct {
	result      *StopResult
	containerID string
}

// FinalizeArgs supplies user-owned metadata for a pending recording.
type FinalizeArgs struct {
	PendingID   string   `json:"pendingID"`
	Label       string   `json:"label"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
}

// FinalizeResult identifies the durable asset created from a pending recording.
type FinalizeResult struct {
	ClipID      string `json:"clipID"`
	ContainerID string `json:"containerID"`
	Label       string `json:"label"`
}

// Stop 同步停止录制并保留为内存 pending，等待用户命名后 Finalize.
//
// 注意: 用 internal mutex 防 F12 stop callback 跟 UI Stop 重入 (Recorder.Stop 不可重入).
func (s *Service) Stop() (*StopResultPayload, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed.Load() {
		return nil, nil
	}

	// 幂等: 仅 recording / paused 可停 (paused 直接停, 不必先 resume); idle / 已 finalizing → no-op.
	// 杀掉 ErrRecorderNotActive 这个伪错误 — 陈旧/重复 stop 点击对前端无害.
	if p := s.phase(); p != PhaseRecording && p != PhasePaused {
		return nil, nil
	}
	cur := s.GetState()
	containerID := cur.ContainerID

	// 进 finalizing — 收尾期 GetState 反映真实阶段; 同时挡住并发 Stop (phase != recording/paused 直接 no-op).
	finalizing := cur
	finalizing.Phase = PhaseFinalizing
	finalizing.PausedAtMs = 0
	s.setState(finalizing)
	// 无论成败回 idle (前端镜像始终收敛).
	defer s.setState(RecordingState{Phase: PhaseIdle})

	res, err := s.rec.Stop()
	// 不管成败都清停录 + 暂停热键, 避免悬挂 callback 引发误触
	setActiveStopHotkey(0, nil)
	setActivePauseHotkey(0, nil)
	if err != nil {
		// recorder 自己已不活跃 (理论上 phase 守卫挡住, 防御性): 当 no-op, 不抛伪错误.
		if errors.Is(err, ErrRecorderNotActive) {
			return nil, nil
		}
		return nil, err
	}
	// Shutdown owns cancellation, not persistence. If it arrived while native
	// Stop was draining, discard the result before creating any durable asset.
	if s.closed.Load() {
		return nil, nil
	}

	if len(res.Events) == 0 {
		return nil, nil
	}
	pendingID := "pending-" + res.TempID
	s.pending[pendingID] = pendingRecording{result: res, containerID: containerID}
	durationUs := res.Events[len(res.Events)-1].TUs
	return &StopResultPayload{
		PendingID: pendingID, ContainerID: containerID,
		DurationUs: durationUs, EventCount: len(res.Events),
	}, nil
}

// Cancel stops the active recording and discards all captured events.
func (s *Service) Cancel() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p := s.phase(); p != PhaseRecording && p != PhasePaused {
		return nil
	}
	s.rec.Cancel()
	setActiveStopHotkey(0, nil)
	setActivePauseHotkey(0, nil)
	s.setState(RecordingState{Phase: PhaseIdle})
	if s.emit != nil {
		s.emit("recording:cancelled", map[string]any{})
	}
	return nil
}

// Finalize persists a pending recording after the user supplies library metadata.
func (s *Service) Finalize(args FinalizeArgs) (*FinalizeResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	label := strings.TrimSpace(args.Label)
	if label == "" {
		return nil, errors.New("录制名称不能为空")
	}
	if len([]rune(label)) > 80 {
		return nil, errors.New("录制名称不能超过 80 个字符")
	}
	pending, ok := s.pending[args.PendingID]
	if !ok {
		return nil, fmt.Errorf("pending recording %q not found", args.PendingID)
	}
	tags := normalizeTags(args.Tags)
	description := strings.TrimSpace(args.Description)
	category := strings.TrimSpace(args.Category)
	res := pending.result
	if s.clipSvc == nil {
		return nil, errors.New("clip store 未注入")
	}
	clip := &inputclip.InputClip{
		ID: "clip-" + res.TempID, Label: label, Description: description, Category: category,
		Tags: tags, CreatedAt: time.Now().UTC().Format(time.RFC3339), Meta: res.Meta, Events: res.Events,
	}
	clip.UpdateDuration()
	if err := s.clipSvc.Save(clip); err != nil {
		return nil, fmt.Errorf("save clip: %w", err)
	}
	result := &FinalizeResult{ClipID: clip.ID, ContainerID: pending.containerID, Label: label}
	delete(s.pending, args.PendingID)
	return result, nil
}

// Discard releases a pending recording without creating an asset.
func (s *Service) Discard(pendingID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.pending, pendingID)
	return nil
}

func normalizeTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	seen := map[string]struct{}{}
	for _, raw := range tags {
		tag := strings.TrimSpace(raw)
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
	}
	return out
}

// StopAsync 异步停录，完成后广播 pending payload 或 {error}.
// F12 hook callback 走这条 (callback 不能阻塞 hook 线程 50ms+).
// 前端 RecordingHUD 子窗口主动调它 (拿不到 RPC 返回值, 靠订阅事件).
//
// ErrRecorderNotActive 静默吞: F12 + toolbar/HUD 同时触发 stop 时, 第二条会撞这个错;
// 此时第一条已经 emit 过正常 payload, 再 emit error 会覆盖前端 toast.
func (s *Service) StopAsync() {
	go func() {
		payload, err := s.Stop()
		if s.emit == nil {
			return
		}
		if err != nil {
			if errors.Is(err, ErrRecorderNotActive) {
				return
			}
			s.emit("recording:completed", map[string]any{"error": err.Error()})
			return
		}
		// 幂等 no-op (payload nil): 不在录时被触发, 状态已走 recording:state, 不发结果事件.
		if payload == nil {
			return
		}
		s.emit("recording:completed", map[string]any{
			"pendingID":   payload.PendingID,
			"containerID": payload.ContainerID,
			"durationUs":  payload.DurationUs,
			"eventCount":  payload.EventCount,
		})
	}()
}

// findWin32WindowTargetNode 在 container.Graph.Nodes 里找第一个 Kind=="Win32WindowTarget" 的节点.
// container 强制有且只有一个 Win32WindowTarget 节点, 这里假设也是 1.
func findWin32WindowTargetNode(c *container.Container) *container.GraphNode {
	for i := range c.Graph.Nodes {
		if c.Graph.Nodes[i].Kind == "Win32WindowTarget" {
			return &c.Graph.Nodes[i]
		}
	}
	return nil
}

// readMatchSpecFromConfig 从 Win32WindowTarget 节点 config 顶级字段解出 target.WindowMatchSpec.
// 字段是扁平的, 跟 Spec.Inputs 对齐.
// config 缺字段或类型不对 → 留空字段，平台 resolver 负责校验。
func readMatchSpecFromConfig(n *container.GraphNode) target.WindowMatchSpec {
	if n.Config == nil {
		return target.WindowMatchSpec{}
	}
	return target.WindowMatchSpec{
		Title:       container.PinString(n, "Title"),
		Class:       container.PinString(n, "Class"),
		ProcessName: container.PinString(n, "ProcessName"),
		TitleMatch:  container.PinString(n, "TitleMatch"),
	}
}
