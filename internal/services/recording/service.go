package recording

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/lxn/win"

	"yotta/internal/apperr"
	"yotta/internal/services/container"
	"yotta/internal/services/inputclip"
	"yotta/pkg/winutil"
)

// HotkeySettingsProvider 给 Service 拿停录热键 VK + mouseMode.
// nil = 默认 F12 + relative.
type HotkeySettingsProvider interface {
	GetStopHotkeyVK() uint32  // 0x7B = F12 默认
	GetPauseHotkeyVK() uint32 // 0x7A = F11 默认 (暂停/继续切换); 0 = 不启用
	GetMouseMode() string     // 'relative' / 'absolute'
}

// ContainerSubgraphSaver 窄接口 — 录制完落 Subgraph 到 container.
// 用接口注入避免循环 import (container 不直接进 recording 包依赖图).
// container.Store 已实现这个签名.
type ContainerSubgraphSaver interface {
	SaveSubgraph(containerID string, sg *container.Subgraph) error
}

// ContainerGetter 窄接口 — recording 拿 container 解析 WindowTarget hwnd 用.
// container.Store 已实现 Get(id) (Container, bool).
type ContainerGetter interface {
	Get(id string) (container.Container, bool)
}

// Service wails3 RPC 入口.
//
// 前端流程 (v2 Subgraph-only):
//  1. Start({filterMode, containerID}) → 立即返 tempID; hook 已挂上但用户还没停
//  2. (用户在 UI / F12 全局热键触发停止)
//  3. Stop() → 拿 StopResult → 走 transform → 落 *container.Subgraph 到
//     container.subgraphs/. 返回 Subgraph (前端在 activeGraph 加 Subgraph 引用节点).
//
// 停录热键: LL hook 直接检测 → return 1 拦截不透传游戏 → 异步调 StopAsync.
// StopAsync 完成时 emit 'recording:completed' {subgraphID, containerID, label} | {error}.
type Service struct {
	rec          *Recorder
	hkProv       HotkeySettingsProvider
	clipSvc      *inputclip.Service
	containers   ContainerSubgraphSaver
	containerGet ContainerGetter
	emit         func(name string, data any)

	mu      sync.Mutex   // 串行化 Start/Stop 命令 (防 F12 callback 跟 UI Stop 重入)
	stateMu sync.RWMutex // 保护 state 快照 — 跟 mu 分离, GetState 读路径不被慢的 rec.Stop 阻塞
	state   RecordingState
}

func NewService(rec *Recorder, hkProv HotkeySettingsProvider, clipSvc *inputclip.Service) *Service {
	return &Service{
		rec: rec, hkProv: hkProv, clipSvc: clipSvc,
		state: RecordingState{Phase: PhaseIdle},
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
	ContainerID string `json:"containerID"` // 录制目标容器 (录完子图落这)
	FilterMode  string `json:"filterMode"`  // precise | simple
	TempID      string `json:"tempID"`
	StartedAtMs int64  `json:"startedAtMs"`
	PausedMs    int64  `json:"pausedMs"`    // 累计已暂停毫秒, HUD 算录制时长 = now-startedAt-pausedMs
	PausedAtMs  int64  `json:"pausedAtMs"`  // 本次暂停起点 wall time (>0 即处于暂停, HUD 冻结计时); recording 态为 0
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

// SetEmit main.go 启动期注入. wails3 application.Event.Emit 包一层.
func (s *Service) SetEmit(emit func(name string, data any)) { s.emit = emit }

// SetContainerSaver main.go 启动期注入. nil = Stop 时报错 (录制没出口).
func (s *Service) SetContainerSaver(c ContainerSubgraphSaver) { s.containers = c }

// SetContainerGetter main.go 启动期注入. nil = Start 时报错 (没法解 WindowTarget hwnd).
func (s *Service) SetContainerGetter(c ContainerGetter) { s.containerGet = c }

// ValidateTarget 录制前预检 — 解析容器 WindowTarget 找窗口 (找不到返 error), 并把窗口拉到前台.
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
	wtNode := findWindowTargetNode(&cont)
	if wtNode == nil {
		return apperr.New(apperr.CodeRecordingNoWindowTarget, nil)
	}
	spec := readMatchSpecFromConfig(wtNode)
	wh, err := winutil.ResolveWindow(spec, 3*time.Second, 500*time.Millisecond)
	if err != nil {
		return fmt.Errorf("窗口未找到: %w", err)
	}
	// 找到 → 拉到前台 (best-effort; 独占全屏 OS 可能拒绝, 不算失败).
	winutil.BringToFront(win.HWND(wh.HWND))
	return nil
}

// StartArgs 前端传入的录制开关.
type StartArgs struct {
	FilterMode  string `json:"filterMode"`  // 'precise' | 'simple'
	ContainerID string `json:"containerID"` // 必传; 录完 Subgraph 落到这个 container 的 subgraphs/
}

// Start 启动录制 (非阻塞). 返回临时录制 ID (前端订阅事件流过滤用).
//
// Start 后会 atomic 设 LL hook 的 stopHotkeyVK + callback — 用户在游戏前台按 F12 时,
// hook 直接拦截 (不透传游戏) 并异步触发 StopAsync.
func (s *Service) Start(args StartArgs) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 幂等: 已经在录 (或正在收尾) → 不重复启动, 返当前 tempID. 前端误触/重入无害.
	if s.phase() != PhaseIdle {
		return s.GetState().TempID, nil
	}

	if args.ContainerID == "" {
		return "", apperr.New(apperr.CodeContainerIDRequired, nil)
	}
	if s.containerGet == nil {
		return "", errors.New("ContainerGetter 未注入 (main.go 启动期 SetContainerGetter?)")
	}
	cont, ok := s.containerGet.Get(args.ContainerID)
	if !ok {
		return "", fmt.Errorf("container %q not found", args.ContainerID)
	}
	wtNode := findWindowTargetNode(&cont)
	if wtNode == nil {
		return "", apperr.New(apperr.CodeRecordingNoWindowTarget, nil)
	}
	spec := readMatchSpecFromConfig(wtNode)
	wh, err := winutil.ResolveWindow(spec, 3*time.Second, 500*time.Millisecond)
	if err != nil {
		return "", fmt.Errorf("窗口未找到: %w", err)
	}
	hwnd := uintptr(wh.HWND)
	mouseMode := "relative"
	stopVK := uint32(0x7B)
	if s.hkProv != nil {
		mouseMode = s.hkProv.GetMouseMode()
		stopVK = s.hkProv.GetStopHotkeyVK()
	}
	filterMode := args.FilterMode
	if filterMode == "" {
		filterMode = "precise"
	}
	// 录制基准分辨率取目标窗口客户区实际尺寸 (回放跨分辨率缩放用). 取不到 (≤0) 直接
	// 返 error 让用户重试 —— 兜底 1080p 反而让回放按错基准缩放绝对坐标, 比不缩放更糟.
	baseW, baseH := wh.ClientW, wh.ClientH
	if baseW <= 0 || baseH <= 0 {
		return "", fmt.Errorf("无法读取目标窗口客户区尺寸 (得 %dx%d), 请确认窗口已正常显示后重试", baseW, baseH)
	}
	meta := inputclip.ClipMeta{
		MouseMode:      mouseMode,
		FilterMode:     filterMode,
		StopHotkeyVK:   stopVK,
		BaseResolution: [2]int{baseW, baseH},
	}
	id, recErr := s.rec.Start(win.HWND(hwnd), meta)
	if recErr != nil {
		return "", fmt.Errorf("recorder.Start: %w", recErr)
	}
	s.setState(RecordingState{
		Phase:       PhaseRecording,
		ContainerID: args.ContainerID,
		FilterMode:  filterMode,
		TempID:      id,
		StartedAtMs: time.Now().UnixMilli(),
	})
	SetActiveStopHotkey(stopVK, func() {
		s.StopAsync()
	})
	// 暂停/继续切换热键 (可选). 录制中按 → 暂停; 暂停中按 → emit resume-hotkey 让 HUD 走 3s 倒计时再继续.
	var pauseVK uint32
	if s.hkProv != nil {
		pauseVK = s.hkProv.GetPauseHotkeyVK()
	}
	if pauseVK != 0 {
		SetActivePauseHotkey(pauseVK, func() {
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

// StopResultPayload 给前端 RPC / event payload 用. Subgraph 完整结构对前端是不透明的
// (会通过 container.Get 重新拉一次), 这里只回最小定位信息.
type StopResultPayload struct {
	SubgraphID  string `json:"subgraphID"`
	ContainerID string `json:"containerID"`
	Label       string `json:"label"`
	FilterMode  string `json:"filterMode"`
}

// Stop 同步停止录制 — 走 transform 输出 Subgraph 落到 container.
// precise 模式额外把 InputClip 落到 clipSvc.Store (PlayClip 节点要 clipID 解析).
//
// 注意: 用 internal mutex 防 F12 stop callback 跟 UI Stop 重入 (Recorder.Stop 不可重入).
func (s *Service) Stop() (*StopResultPayload, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

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

	if s.containers == nil {
		return nil, errors.New("ContainerSubgraphSaver 未注入 (main.go 启动期 SetContainerSaver?)")
	}

	res, err := s.rec.Stop()
	// 不管成败都清停录 + 暂停热键, 避免悬挂 callback 引发误触
	SetActiveStopHotkey(0, nil)
	SetActivePauseHotkey(0, nil)
	if err != nil {
		// recorder 自己已不活跃 (理论上 phase 守卫挡住, 防御性): 当 no-op, 不抛伪错误.
		if errors.Is(err, ErrRecorderNotActive) {
			return nil, nil
		}
		return nil, err
	}

	label := "录制 " + time.Now().Format("15:04")

	var sg container.Subgraph
	switch res.Meta.FilterMode {
	case "precise":
		// 1) 建 InputClip 落到 clipSvc — PlayClip 节点要靠 clipID 解析回放
		clip := &inputclip.InputClip{
			ID:        "clip-" + res.TempID,
			Label:     label,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			Meta:      res.Meta,
			Events:    res.Events,
		}
		clip.UpdateDuration()
		if s.clipSvc != nil {
			if err := s.clipSvc.Save(clip); err != nil {
				return nil, fmt.Errorf("save clip: %w", err)
			}
		}
		// 2) 包成单 PlayClip 节点的 Subgraph
		sg = BuildPreciseSubgraph(clip.ID, res.Meta, label)
	case "simple":
		sg = BuildSimpleSubgraph(res.Events, res.Meta, res.ClientW, res.ClientH, label)
	default:
		return nil, fmt.Errorf("unknown filterMode %q (前端 StartArgs.FilterMode 漏传?)", res.Meta.FilterMode)
	}

	if err := s.containers.SaveSubgraph(containerID, &sg); err != nil {
		return nil, fmt.Errorf("save subgraph to container %q: %w", containerID, err)
	}

	return &StopResultPayload{
		SubgraphID:  sg.ID,
		ContainerID: containerID,
		Label:       sg.Label,
		FilterMode:  res.Meta.FilterMode,
	}, nil
}

// StopAsync 异步停录 — 跑后 emit 'recording:completed' payload 或 {error}.
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
			"subgraphID":  payload.SubgraphID,
			"containerID": payload.ContainerID,
			"label":       payload.Label,
			"filterMode":  payload.FilterMode,
		})
	}()
}

// findWindowTargetNode 在 container.Graph.Nodes 里找第一个 Kind=="WindowTarget" 的节点.
// container 强制有且只有一个 WindowTarget 节点, 这里假设也是 1.
func findWindowTargetNode(c *container.Container) *container.GraphNode {
	for i := range c.Graph.Nodes {
		if c.Graph.Nodes[i].Kind == "WindowTarget" {
			return &c.Graph.Nodes[i]
		}
	}
	return nil
}

// readMatchSpecFromConfig 从 WindowTarget 节点 config 顶级字段解出 winutil.MatchSpec.
// 字段是扁平的, 跟 Spec.Inputs 对齐.
// config 缺字段或类型不对 → 留空字段 (winutil.ResolveWindow 会报 IsEmptyMatch).
func readMatchSpecFromConfig(n *container.GraphNode) winutil.MatchSpec {
	if n.Config == nil {
		return winutil.MatchSpec{}
	}
	return winutil.MatchSpec{
		Title:       container.PinString(n, "Title"),
		Class:       container.PinString(n, "Class"),
		ProcessName: container.PinString(n, "ProcessName"),
		TitleMatch:  container.PinString(n, "TitleMatch"),
	}
}
