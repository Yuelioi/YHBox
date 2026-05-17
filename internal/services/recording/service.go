package recording

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/lxn/win"

	"yhbox/internal/services/container"
	"yhbox/internal/services/inputclip"
)

// GameHwndProvider 给录制器拿当前游戏窗口 hwnd. main.go 启动时注入.
type GameHwndProvider interface {
	HWND() (uintptr, bool)
}

// HotkeySettingsProvider 给 Service 拿停录热键 VK + mouseMode.
// nil = 默认 F12 + relative.
type HotkeySettingsProvider interface {
	GetStopHotkeyVK() uint32 // 0x7B = F12 默认
	GetMouseMode() string    // 'relative' / 'absolute'
}

// ContainerSubgraphSaver 窄接口 — 录制完落 Subgraph 到 container.
// 用接口注入避免循环 import (container 不直接进 recording 包依赖图).
// container.Store 已实现这个签名.
type ContainerSubgraphSaver interface {
	SaveSubgraph(containerID string, sg *container.Subgraph) error
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
	rec        *Recorder
	game       GameHwndProvider
	hkProv     HotkeySettingsProvider
	clipSvc    *inputclip.Service
	containers ContainerSubgraphSaver
	emit       func(name string, data any)

	mu                sync.Mutex // 防 F12 stop callback 跟 UI Stop 同时跑
	activeContainerID string     // Start 时记下, Stop 时落 subgraph 用
}

func NewService(rec *Recorder, game GameHwndProvider, hkProv HotkeySettingsProvider, clipSvc *inputclip.Service) *Service {
	return &Service{rec: rec, game: game, hkProv: hkProv, clipSvc: clipSvc}
}

// SetEmit main.go 启动期注入. wails3 application.Event.Emit 包一层.
func (s *Service) SetEmit(emit func(name string, data any)) { s.emit = emit }

// SetContainerSaver main.go 启动期注入. nil = Stop 时报错 (录制没出口).
func (s *Service) SetContainerSaver(c ContainerSubgraphSaver) { s.containers = c }

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
	if s.game == nil {
		return "", errors.New("game provider 未注入")
	}
	if args.ContainerID == "" {
		return "", errors.New("containerID 必填 — 录制 Subgraph 要落到某个 container")
	}
	hwnd, ok := s.game.HWND()
	if !ok || hwnd == 0 {
		return "", errors.New("游戏窗口未就绪")
	}
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
	meta := inputclip.ClipMeta{
		MouseMode:      mouseMode,
		FilterMode:     filterMode,
		StopHotkeyVK:   stopVK,
		BaseResolution: [2]int{1920, 1080}, // TODO: 真实读取游戏窗口尺寸
		WindowMode:     "fullscreen",
	}
	id, err := s.rec.Start(win.HWND(hwnd), meta)
	if err != nil {
		return "", fmt.Errorf("recorder.Start: %w", err)
	}
	s.mu.Lock()
	s.activeContainerID = args.ContainerID
	s.mu.Unlock()
	SetActiveStopHotkey(stopVK, func() {
		s.StopAsync()
	})
	return id, nil
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
	if !s.rec.Active() {
		return nil, errors.New("recorder not active")
	}
	if s.containers == nil {
		return nil, errors.New("ContainerSubgraphSaver 未注入 (main.go 启动期 SetContainerSaver?)")
	}
	containerID := s.activeContainerID
	s.activeContainerID = ""

	res, err := s.rec.Stop()
	// 不管成败都清 stopHotkey, 避免悬挂 callback 引发误触
	SetActiveStopHotkey(0, nil)
	if err != nil {
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
func (s *Service) StopAsync() {
	go func() {
		payload, err := s.Stop()
		if s.emit == nil {
			return
		}
		if err != nil {
			s.emit("recording:completed", map[string]any{"error": err.Error()})
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

// IsRecording 状态查询.
func (s *Service) IsRecording() bool {
	return s.rec.Active()
}
