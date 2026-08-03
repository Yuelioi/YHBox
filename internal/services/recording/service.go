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
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/services/asset"
	"github.com/yottaapp/yotta/internal/services/inputclip"
	"github.com/yottaapp/yotta/internal/services/macro"
	"github.com/yottaapp/yotta/internal/services/resourceauthoring"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

// HotkeySettingsProvider 给 Service 拿录制热键 VK + mouseMode.
// nil = 默认 F10/F11/F12 + relative.
type HotkeySettingsProvider interface {
	GetStartHotkeyVK() uint32 // 0x79 = F10 默认
	GetStopHotkeyVK() uint32  // 0x7B = F12 默认
	GetPauseHotkeyVK() uint32 // 0x7A = F11 默认 (暂停/继续切换); 0 = 不启用
	GetMouseMode() string     // 'relative' / 'absolute'
}

type startHotkeyWatch interface {
	Start() error
	Stop()
}

type startHotkeyFactory func(vk uint32, callback func()) startHotkeyWatch

type clipStore interface {
	Save(clip *inputclip.InputClip) error
}

type macroStore interface {
	Save(value *macro.Macro) (*macro.Macro, error)
}

// TargetResolver provides trusted local recording access to installed targets.
type TargetResolver interface {
	AcquireRecordingTarget(context.Context, string) (target.WindowHandle, int, func(), error)
}

// Service wails3 RPC 入口.
//
// 前端流程:
//  1. Start({targetSlot}) 锁定目标并进入 armed，不采集输入.
//  2. F10 / BeginCountdown() 触发 3 秒倒计时，结束后才启动 recorder.
//  3. Stop() 只生成 pending token，不创建库资产.
//  4. Finalize(metadata) 创建 InputClip; Discard 释放 pending 数据.
//
// 停录热键: LL hook 直接检测 → return 1 拦截不透传游戏 → 异步调 StopAsync.
// StopAsync 完成时 emit 'recording:completed' pending payload 或 {error}.
type Service struct {
	rec       recorderLifecycle
	hkProv    HotkeySettingsProvider
	clipSvc   clipStore
	macroSvc  macroStore
	resources *resourceauthoring.Creator
	targets   TargetResolver
	emit      func(name string, data any)

	mu                  sync.Mutex // 串行化 Start/Stop 命令 (防 F12 callback 跟 UI Stop 重入)
	pending             *pendingRecording
	activeRelease       func()
	armed               *armedRecording
	startHotkey         startHotkeyWatch
	startHotkeyFactory  startHotkeyFactory
	countdownGeneration atomic.Uint64
	stateMu             sync.RWMutex // 保护 state 快照 — 跟 mu 分离, GetState 读路径不被慢的 rec.Stop 阻塞
	state               RecordingState
	closed              atomic.Bool
	shutdownOnce        sync.Once
	shutdownDone        chan struct{}
}

func NewService(rec recorderLifecycle, hkProv HotkeySettingsProvider, clipSvc clipStore, macroSvc macroStore, resources *resourceauthoring.Creator, targets TargetResolver, emit ...func(name string, data any)) *Service {
	service := &Service{
		rec: rec, hkProv: hkProv, clipSvc: clipSvc, macroSvc: macroSvc, resources: resources, targets: targets,
		state: RecordingState{Phase: PhaseIdle}, shutdownDone: make(chan struct{}),
		startHotkeyFactory: newStartHotkeyWatch,
	}
	if len(emit) != 0 {
		service.emit = emit[0]
	}
	return service
}

// Phase 常量 — 录制生命周期的权威阶段.
const (
	PhaseIdle       = "idle"
	PhaseArmed      = "armed"
	PhaseCountdown  = "countdown"
	PhaseRecording  = "recording"
	PhasePaused     = "paused"
	PhaseFinalizing = "finalizing"
	PhasePending    = "pending"
)

// RecordingState 录制子系统的权威状态. 后端是唯一真相源; 前端/HUD 只镜像
// (recording:state 事件 + GetState 对账), 不自己存可 desync 的 flag.
type RecordingState struct {
	Revision          uint64                  `json:"revision"`
	Phase             string                  `json:"phase"` // idle | armed | countdown | recording | paused | finalizing | pending
	Mode              inputclip.RecordingMode `json:"mode"`
	TargetSlot        string                  `json:"targetSlot"`
	TempID            string                  `json:"tempID"`
	StartedAtMs       int64                   `json:"startedAtMs"`
	PausedMs          int64                   `json:"pausedMs"`   // 累计已暂停毫秒, HUD 算录制时长 = now-startedAt-pausedMs
	PausedAtMs        int64                   `json:"pausedAtMs"` // 本次暂停起点 wall time (>0 即处于暂停, HUD 冻结计时); recording 态为 0
	CountdownEndsAtMs int64                   `json:"countdownEndsAtMs"`
	Pending           *StopResultPayload      `json:"pending,omitempty"`
}

// GetState 返回当前权威状态快照. RPC — 前端任何时候可查 (reconcile 对账用).
// 走 stateMu 读路径, 不被慢的 rec.Stop() 阻塞.
func (s *Service) GetState() RecordingState {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return cloneRecordingState(s.state)
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
	st.Revision = s.state.Revision + 1
	s.state = cloneRecordingState(st)
	snapshot := cloneRecordingState(s.state)
	s.stateMu.Unlock()
	if s.emit != nil {
		s.emit("recording:state", snapshot)
	}
}

func cloneRecordingState(state RecordingState) RecordingState {
	if state.Pending == nil {
		return state
	}
	pending := *state.Pending
	// These collections are part of the RPC contract. Cloning an empty slice
	// into a nil destination turns [] into JSON null and makes otherwise valid
	// pending recordings disappear at strict clients.
	pending.Preview.Steps = append([]RecordingPreviewStep{}, pending.Preview.Steps...)
	pending.Preview.Tracks = append([]RecordingTrack{}, pending.Preview.Tracks...)
	for index := range pending.Preview.Steps {
		if point := pending.Preview.Steps[index].Point; point != nil {
			copyOfPoint := *point
			pending.Preview.Steps[index].Point = &copyOfPoint
		}
	}
	if pending.Document != nil {
		document := macro.CloneDocument(*pending.Document)
		pending.Document = &document
	}
	state.Pending = &pending
	return state
}

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
	phase := s.phase()
	wasOpen := phase != PhaseIdle
	if (phase == PhaseRecording || phase == PhasePaused || phase == PhaseFinalizing) && s.rec.Active() {
		s.rec.Cancel()
		setActiveStopHotkey(0, nil)
		setActivePauseHotkey(0, nil)
	}
	s.countdownGeneration.Add(1)
	s.stopStartHotkeyLocked()
	s.releaseActiveLocked()
	s.armed = nil
	s.pending = nil
	s.mu.Unlock()
	if wasOpen {
		s.setState(RecordingState{Phase: PhaseIdle})
	}
	close(s.shutdownDone)
}

// ValidateTarget resolves and activates one installed target before countdown.
// 前端在 3s 倒计时**之前**调: 没设/找不到窗口立刻报错 (不用等录完), 成功则游戏已置前台省去用户 Alt-Tab.
// 纯预检 — 不装 hook 不起 recorder. Start 内仍保留同样校验作 race 兜底 (倒计时期间窗口可能消失).
func (s *Service) ValidateTarget(targetSlot string) error {
	if targetSlot == "" {
		return apperr.New(apperr.CodeAutomationTargetSlotRequired, nil)
	}
	if s.targets == nil {
		return errors.New("installed target resolver is unavailable")
	}
	_, _, release, err := s.targets.AcquireRecordingTarget(context.Background(), targetSlot)
	if err != nil {
		return apperr.New(apperr.CodeRecordingTargetUnavailable, map[string]any{"targetSlot": targetSlot, "cause": err.Error()})
	}
	if release == nil {
		return errors.New("installed target resolver returned no recording lease")
	}
	release()
	return nil
}

// StartArgs 前端传入的录制开关.
type StartArgs struct {
	TargetSlot string                  `json:"targetSlot"`
	Mode       inputclip.RecordingMode `json:"mode"`
}

type armedRecording struct {
	window  target.WindowHandle
	meta    inputclip.ClipMeta
	pauseVK uint32
}

// Start 准备录制 (非阻塞). 只校验并锁定目标、监听开始热键，不采集任何输入。
//
// 倒计时结束、recorder 真正启动后才设置 F11/F12 hook；准备阶段仅监听 F10.
func (s *Service) Start(args StartArgs) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed.Load() {
		return "", errors.New("recording service is closed")
	}

	if current := s.GetState(); current.Phase != PhaseIdle {
		if (current.Phase == PhaseArmed || current.Phase == PhaseCountdown || current.Phase == PhaseRecording || current.Phase == PhasePaused) && current.TargetSlot == args.TargetSlot && current.Mode == args.Mode {
			return current.TempID, nil
		}
		return "", apperr.New(apperr.CodeRecordingSessionBusy, map[string]any{"phase": current.Phase})
	}

	if args.TargetSlot == "" {
		return "", apperr.New(apperr.CodeAutomationTargetSlotRequired, nil)
	}
	if !args.Mode.Valid() {
		return "", apperr.New(apperr.CodeRecordingModeRequired, nil)
	}
	if s.targets == nil {
		return "", errors.New("installed target resolver is unavailable")
	}
	wh, targetCounts360, release, err := s.targets.AcquireRecordingTarget(context.Background(), args.TargetSlot)
	if err != nil {
		return "", apperr.New(apperr.CodeRecordingTargetUnavailable, map[string]any{"targetSlot": args.TargetSlot, "cause": err.Error()})
	}
	if release == nil {
		return "", errors.New("installed target resolver returned no recording lease")
	}
	releaseOnFailure := true
	defer func() {
		if releaseOnFailure {
			release()
		}
	}()
	mouseMode := "relative"
	startVK := uint32(0x79)
	stopVK := uint32(0x7B)
	pauseVK := uint32(0x7A)
	if s.hkProv != nil {
		mouseMode = s.hkProv.GetMouseMode()
		startVK = s.hkProv.GetStartHotkeyVK()
		stopVK = s.hkProv.GetStopHotkeyVK()
		pauseVK = s.hkProv.GetPauseHotkeyVK()
	}
	// 录制基准分辨率取目标窗口客户区实际尺寸 (回放跨分辨率缩放用). 取不到 (≤0) 直接
	// 返 error 让用户重试 —— 兜底 1080p 反而让回放按错基准缩放绝对坐标, 比不缩放更糟.
	baseW, baseH := wh.ClientW, wh.ClientH
	if baseW <= 0 || baseH <= 0 {
		return "", fmt.Errorf("无法读取目标窗口客户区尺寸 (得 %dx%d), 请确认窗口已正常显示后重试", baseW, baseH)
	}
	if args.Mode == inputclip.RecordingModePrecise && mouseMode == "relative" && targetCounts360 <= 0 {
		return "", apperr.New(apperr.CodeRecordingCalibrationRequired, map[string]any{"targetSlot": args.TargetSlot})
	}
	meta := inputclip.ClipMeta{
		RecordingMode:  args.Mode,
		MouseMode:      mouseMode,
		MouseCounts360: targetCounts360,
		StopHotkeyVK:   stopVK,
		BaseResolution: [2]int{baseW, baseH},
	}
	if startVK == 0 || s.startHotkeyFactory == nil {
		return "", errors.New("recording start hotkey is unavailable")
	}
	watch := s.startHotkeyFactory(startVK, func() { _ = s.BeginCountdown() })
	if watch == nil {
		return "", errors.New("recording start hotkey watcher is unavailable")
	}
	if err := watch.Start(); err != nil {
		return "", fmt.Errorf("start recording hotkey watch: %w", err)
	}
	s.startHotkey = watch
	s.armed = &armedRecording{window: wh, meta: meta, pauseVK: pauseVK}
	s.activeRelease = release
	releaseOnFailure = false
	s.setState(RecordingState{
		Phase:      PhaseArmed,
		Mode:       args.Mode,
		TargetSlot: args.TargetSlot,
	})
	return "", nil
}

// BeginCountdown starts the authoritative three-second preparation window.
// Both the configured global start key and the visible HUD button call this method.
func (s *Service) BeginCountdown() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed.Load() || s.phase() != PhaseArmed {
		return nil
	}
	s.stopStartHotkeyLocked()
	generation := s.countdownGeneration.Add(1)
	cur := s.GetState()
	cur.Phase = PhaseCountdown
	cur.CountdownEndsAtMs = time.Now().Add(3 * time.Second).UnixMilli()
	s.setState(cur)
	go func() {
		timer := time.NewTimer(3 * time.Second)
		defer timer.Stop()
		<-timer.C
		s.finishCountdown(generation)
	}()
	return nil
}

func (s *Service) finishCountdown(generation uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed.Load() || generation != s.countdownGeneration.Load() || s.phase() != PhaseCountdown || s.armed == nil {
		return
	}
	prepared := s.armed
	id, err := s.rec.Start(prepared.window.HWND, prepared.meta)
	if err != nil {
		s.armed = nil
		s.releaseActiveLocked()
		s.setState(RecordingState{Phase: PhaseIdle})
		if s.emit != nil {
			s.emit("recording:completed", map[string]any{"error": fmt.Errorf("recorder.Start: %w", err).Error()})
		}
		return
	}
	s.armed = nil
	cur := s.GetState()
	cur.Phase = PhaseRecording
	cur.TempID = id
	cur.StartedAtMs = time.Now().UnixMilli()
	cur.CountdownEndsAtMs = 0
	s.setState(cur)
	setActiveStopHotkey(prepared.meta.StopHotkeyVK, func() {
		s.StopAsync()
	})
	if prepared.pauseVK != 0 {
		setActivePauseHotkey(prepared.pauseVK, func() {
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
	PendingID   string                  `json:"pendingID"`
	TargetSlot  string                  `json:"targetSlot"`
	Mode        inputclip.RecordingMode `json:"mode"`
	DurationUs  uint64                  `json:"durationUs"`
	EventCount  int                     `json:"eventCount"`
	Preview     RecordingPreview        `json:"preview"`
	Document    *macro.Document         `json:"document,omitempty"`
	Environment RecordingEnvironment    `json:"environment"`
}

type RecordingEnvironment struct {
	BaseResolution [2]int `json:"baseResolution"`
	MouseMode      string `json:"mouseMode"`
	MouseCounts360 int    `json:"mouseCounts360"`
}

type pendingRecording struct {
	result     *StopResult
	targetSlot string
	document   *macro.Document
}

// FinalizeArgs supplies user-owned metadata for a pending recording.
type FinalizeArgs struct {
	PendingID   string          `json:"pendingID"`
	Destination string          `json:"destination"`
	Label       string          `json:"label"`
	Description string          `json:"description"`
	Category    string          `json:"category"`
	Tags        []string        `json:"tags"`
	Document    *macro.Document `json:"document,omitempty"`
	TrimStartUs *uint64         `json:"trimStartUs,omitempty"`
	TrimEndUs   *uint64         `json:"trimEndUs,omitempty"`
}

const (
	DestinationGlobalAsset      = "global-asset"
	DestinationWorkflowResource = "workflow-resource"
)

type FinalizedAsset struct {
	GUID  string       `json:"guid"`
	Kind  string       `json:"kind"`
	Label string       `json:"label"`
	Blob  blob.BlobRef `json:"blob"`
}

// FinalizeResult is a destination-tagged recording creation result. Exactly
// one of Asset or Resource is present.
type FinalizeResult struct {
	Destination string                   `json:"destination"`
	TargetSlot  string                   `json:"targetSlot"`
	Asset       *FinalizedAsset          `json:"asset,omitempty"`
	Resource    *schema.WorkflowResource `json:"resource,omitempty"`
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
	if phase := s.phase(); phase == PhaseArmed || phase == PhaseCountdown {
		s.cancelPreparationLocked()
		s.setState(RecordingState{Phase: PhaseIdle})
		if s.emit != nil {
			s.emit("recording:cancelled", map[string]any{})
		}
		return nil, nil
	}

	// 幂等: 仅 recording / paused 可停 (paused 直接停, 不必先 resume); idle / 已 finalizing → no-op.
	// 杀掉 ErrRecorderNotActive 这个伪错误 — 陈旧/重复 stop 点击对前端无害.
	if p := s.phase(); p != PhaseRecording && p != PhasePaused {
		return nil, nil
	}
	cur := s.GetState()
	targetSlot := cur.TargetSlot

	// 进 finalizing — 收尾期 GetState 反映真实阶段; 同时挡住并发 Stop (phase != recording/paused 直接 no-op).
	finalizing := cur
	finalizing.Phase = PhaseFinalizing
	finalizing.PausedAtMs = 0
	s.setState(finalizing)

	res, err := s.rec.Stop()
	// 不管成败都清停录 + 暂停热键, 避免悬挂 callback 引发误触
	setActiveStopHotkey(0, nil)
	setActivePauseHotkey(0, nil)
	s.releaseActiveLocked()
	if err != nil {
		// recorder 自己已不活跃 (理论上 phase 守卫挡住, 防御性): 当 no-op, 不抛伪错误.
		if errors.Is(err, ErrRecorderNotActive) {
			s.setState(RecordingState{Phase: PhaseIdle})
			return nil, nil
		}
		s.setState(RecordingState{Phase: PhaseIdle})
		return nil, err
	}
	// Shutdown owns cancellation, not persistence. If it arrived while native
	// Stop was draining, discard the result before creating any durable asset.
	if s.closed.Load() {
		s.setState(RecordingState{Phase: PhaseIdle})
		return nil, nil
	}

	if err := canonicalizeStopResult(res); err != nil {
		s.setState(RecordingState{Phase: PhaseIdle})
		return nil, fmt.Errorf("canonicalize recording: %w", err)
	}
	if len(res.Events) == 0 {
		s.setState(RecordingState{Phase: PhaseIdle})
		return nil, nil
	}
	pendingID := "pending-" + res.TempID
	durationUs := res.Events[len(res.Events)-1].TUs
	payload := &StopResultPayload{
		PendingID: pendingID, TargetSlot: targetSlot,
		Mode: res.Meta.RecordingMode, DurationUs: durationUs, EventCount: len(res.Events), Preview: recordingPreview(res),
		Environment: RecordingEnvironment{BaseResolution: res.Meta.BaseResolution, MouseMode: res.Meta.MouseMode, MouseCounts360: res.Meta.MouseCounts360},
	}
	if res.Meta.RecordingMode == inputclip.RecordingModeSimple {
		document, documentErr := buildMacroDocument(res)
		if documentErr != nil {
			s.setState(RecordingState{Phase: PhaseIdle})
			return nil, fmt.Errorf("build macro recording: %w", documentErr)
		}
		payload.DurationUs = macro.Analyze(document).DurationUs
		payload.Preview.DurationUs = payload.DurationUs
		payload.Document = &document
		document = macro.CloneDocument(document)
		s.pending = &pendingRecording{result: res, targetSlot: targetSlot, document: &document}
	} else {
		s.pending = &pendingRecording{result: res, targetSlot: targetSlot}
	}
	s.setState(RecordingState{
		Phase: PhasePending, Mode: res.Meta.RecordingMode, TargetSlot: targetSlot,
		TempID: res.TempID, StartedAtMs: cur.StartedAtMs, PausedMs: cur.PausedMs, Pending: payload,
	})
	return cloneStopResultPayload(payload), nil
}

func cloneStopResultPayload(payload *StopResultPayload) *StopResultPayload {
	if payload == nil {
		return nil
	}
	return cloneRecordingState(RecordingState{Pending: payload}).Pending
}

// Cancel stops the active recording and discards all captured events.
func (s *Service) Cancel() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.phase()
	if p == PhaseArmed || p == PhaseCountdown {
		s.cancelPreparationLocked()
		s.setState(RecordingState{Phase: PhaseIdle})
		if s.emit != nil {
			s.emit("recording:cancelled", map[string]any{})
		}
		return nil
	}
	if p != PhaseRecording && p != PhasePaused {
		return nil
	}
	s.rec.Cancel()
	setActiveStopHotkey(0, nil)
	setActivePauseHotkey(0, nil)
	s.releaseActiveLocked()
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
	if args.Destination != DestinationGlobalAsset && args.Destination != DestinationWorkflowResource {
		return nil, errors.New("recording destination is invalid")
	}
	label := strings.TrimSpace(args.Label)
	if label == "" {
		return nil, errors.New("录制名称不能为空")
	}
	if len([]rune(label)) > 80 {
		return nil, errors.New("录制名称不能超过 80 个字符")
	}
	if s.pending == nil || "pending-"+s.pending.result.TempID != args.PendingID {
		return nil, fmt.Errorf("pending recording %q not found", args.PendingID)
	}
	pending := *s.pending
	tags := normalizeTags(args.Tags)
	description := strings.TrimSpace(args.Description)
	category := strings.TrimSpace(args.Category)
	pendingState := s.GetState()
	finalizing := pendingState
	finalizing.Phase = PhaseFinalizing
	s.setState(finalizing)
	result, err := s.finalizePending(pending, args.Destination, label, description, category, tags, args.Document, args.TrimStartUs, args.TrimEndUs)
	if err != nil {
		s.setState(pendingState)
		return nil, err
	}
	s.pending = nil
	s.setState(RecordingState{Phase: PhaseIdle})
	return result, nil
}

func (s *Service) finalizePending(pending pendingRecording, destination, label, description, category string, tags []string, editedDocument *macro.Document, trimStartUs, trimEndUs *uint64) (*FinalizeResult, error) {
	result := pending.result
	if result.Meta.RecordingMode == inputclip.RecordingModeSimple {
		if trimStartUs != nil || trimEndUs != nil {
			return nil, errors.New("macros do not accept precise trim boundaries")
		}
		if pending.document == nil {
			return nil, errors.New("pending macro document is missing")
		}
		document := macro.CloneDocument(*pending.document)
		if editedDocument != nil {
			document = macro.CloneDocument(*editedDocument)
		}
		if err := macro.Validate(document); err != nil {
			return nil, fmt.Errorf("validate macro: %w", err)
		}
		if destination == DestinationWorkflowResource {
			if s.resources == nil {
				return nil, errors.New("Workflow Resource creator is unavailable")
			}
			resource, err := s.resources.CreateMacro(context.Background(), resourceauthoring.MacroDraft{
				Metadata: resourceauthoring.Metadata{
					Name: label, Description: description, Category: category, Tags: tags,
				},
				Document: document,
			})
			if err != nil {
				return nil, err
			}
			return &FinalizeResult{
				Destination: destination, TargetSlot: pending.targetSlot, Resource: &resource,
			}, nil
		}
		if s.macroSvc == nil {
			return nil, errors.New("macro store is unavailable")
		}
		saved, err := s.macroSvc.Save(&macro.Macro{
			ID: "macro-" + result.TempID, Label: label, Description: description, Category: category,
			Tags: tags, CreatedAt: time.Now().UTC().Format(time.RFC3339), Document: document,
		})
		if err != nil {
			return nil, fmt.Errorf("save macro: %w", err)
		}
		return &FinalizeResult{
			Destination: destination, TargetSlot: pending.targetSlot,
			Asset: &FinalizedAsset{GUID: saved.ID, Kind: asset.KindMacro, Label: saved.Label, Blob: saved.Blob},
		}, nil
	}
	if editedDocument != nil {
		return nil, errors.New("precise recordings do not accept a macro document")
	}
	if trimStartUs != nil || trimEndUs != nil {
		start := uint64(0)
		end := result.Events[len(result.Events)-1].TUs
		if trimStartUs != nil {
			start = *trimStartUs
		}
		if trimEndUs != nil {
			end = *trimEndUs
		}
		trimmed, err := trimPreciseEvents(result.Events, start, end)
		if err != nil {
			return nil, fmt.Errorf("trim precise recording: %w", err)
		}
		result = &StopResult{TempID: result.TempID, Meta: result.Meta, Events: trimmed}
	}
	clip := &inputclip.InputClip{
		ID: "clip-" + result.TempID, Label: label, Description: description, Category: category,
		Tags: tags, CreatedAt: time.Now().UTC().Format(time.RFC3339), Meta: result.Meta, Events: result.Events,
	}
	clip.UpdateDuration()
	if destination == DestinationWorkflowResource {
		if s.resources == nil {
			return nil, errors.New("Workflow Resource creator is unavailable")
		}
		resource, err := s.resources.CreateInputClip(context.Background(), resourceauthoring.InputClipDraft{
			Metadata: resourceauthoring.Metadata{
				Name: label, Description: description, Category: category, Tags: tags,
			},
			Clip: *clip,
		})
		if err != nil {
			return nil, err
		}
		return &FinalizeResult{
			Destination: destination, TargetSlot: pending.targetSlot, Resource: &resource,
		}, nil
	}
	if s.clipSvc == nil {
		return nil, errors.New("clip store is unavailable")
	}
	if err := s.clipSvc.Save(clip); err != nil {
		return nil, fmt.Errorf("save clip: %w", err)
	}
	return &FinalizeResult{
		Destination: destination, TargetSlot: pending.targetSlot,
		Asset: &FinalizedAsset{GUID: clip.ID, Kind: asset.KindClip, Label: clip.Label, Blob: clip.Blob},
	}, nil
}

// Discard releases a pending recording without creating an asset.
func (s *Service) Discard(pendingID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pending == nil {
		return nil
	}
	if "pending-"+s.pending.result.TempID != pendingID {
		return fmt.Errorf("pending recording %q not found", pendingID)
	}
	s.pending = nil
	s.setState(RecordingState{Phase: PhaseIdle})
	return nil
}

func (s *Service) releaseActiveLocked() {
	if s.activeRelease == nil {
		return
	}
	s.activeRelease()
	s.activeRelease = nil
}

func (s *Service) stopStartHotkeyLocked() {
	if s.startHotkey == nil {
		return
	}
	s.startHotkey.Stop()
	s.startHotkey = nil
}

func (s *Service) cancelPreparationLocked() {
	s.countdownGeneration.Add(1)
	s.stopStartHotkeyLocked()
	s.armed = nil
	s.releaseActiveLocked()
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
			"pendingID": payload.PendingID, "targetSlot": payload.TargetSlot,
			"mode":       payload.Mode,
			"durationUs": payload.DurationUs, "eventCount": payload.EventCount,
			"preview": payload.Preview, "document": payload.Document, "environment": payload.Environment,
		})
	}()
}
