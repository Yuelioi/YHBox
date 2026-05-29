package runtime

import (
	"time"

	"github.com/google/uuid"

	"yhbox/internal/services/container"
)

// ExecFrame 单层 graph 调用栈帧.
// LocalVars: frame-local 变量, 子图调用之间互不干扰.
// LocalParams: 子图入参 (Subgraph.InputParams), 调用入栈时从父图 pull 进来,
//    子图内 GetParam 节点读取. 跟 LocalVars 同生命周期 (pop frame 即销毁).
type ExecFrame struct {
	ContainerID string
	Graph       container.GraphRef
	NodeID      string
	Parent      *ExecFrame
	LocalVars   map[string]any
	LocalParams map[string]any
	SubgraphRef *container.Subgraph // Graph.Kind == "subgraph" 时非 nil；指向当前 frame 所属 subgraph 实例
	CallNodeID  string              // 父图里 push 本 frame 的调用方节点 ID; "" 表示 main frame.
}

// ExecState 整个 run 的运行时状态。每次 RunOnce 一次新 state。
type ExecState struct {
	RunID        string         // UUID per run
	CurrentFrame *ExecFrame
	GlobalVars   map[string]any // 容器级变量（"全局"，scope == global 的 SetVar/IncVar 用）
	CalibCounts  int            // 主图 MouseCalibration 节点 config.counts360 启动 snapshot；运行中改节点无效
	StartedAt    time.Time
}

// NewExecState 起一个新 run。containerID + 启动时校准值。
// 自动 push 一个 main frame，CurrentFrame 就是它。
func NewExecState(containerID string, calibCounts int) *ExecState {
	main := &ExecFrame{
		ContainerID: containerID,
		Graph:       container.MainGraphRef(),
		LocalVars:   map[string]any{},
		LocalParams: map[string]any{},
		SubgraphRef: nil,
	}
	return &ExecState{
		RunID:        uuid.NewString(),
		CurrentFrame: main,
		GlobalVars:   map[string]any{},
		CalibCounts:  calibCounts,
		StartedAt:    time.Now().UTC(),
	}
}

// PushFrame 进入一个子图调用：push 新帧。callNodeID 是父图里发起调用的节点 ID。
func (s *ExecState) PushFrame(graph container.GraphRef, sg *container.Subgraph, callNodeID string) {
	f := &ExecFrame{
		ContainerID: s.CurrentFrame.ContainerID,
		Graph:       graph,
		Parent:      s.CurrentFrame,
		LocalVars:   map[string]any{},
		LocalParams: map[string]any{}, // input param storage (populated by execSubgraph)
		SubgraphRef: sg,
		CallNodeID:  callNodeID,
	}
	s.CurrentFrame = f
}

// PopFrame 子图执行到 SubgraphOutput 后退栈。
// 安全：到 main frame 已经在栈底，再 pop 是 no-op。
func (s *ExecState) PopFrame() {
	if s.CurrentFrame == nil || s.CurrentFrame.Parent == nil {
		return
	}
	s.CurrentFrame = s.CurrentFrame.Parent
}

// SetLocalVar 写当前 frame 的 LocalVar。
func (s *ExecState) SetLocalVar(name string, value any) {
	if s.CurrentFrame == nil {
		return
	}
	if s.CurrentFrame.LocalVars == nil {
		s.CurrentFrame.LocalVars = map[string]any{}
	}
	s.CurrentFrame.LocalVars[name] = value
}

// GetLocalVarHere 只查当前 frame 的 LocalVars，不沿链上跑。
func (s *ExecState) GetLocalVarHere(name string) (any, bool) {
	if s.CurrentFrame == nil {
		return nil, false
	}
	v, ok := s.CurrentFrame.LocalVars[name]
	return v, ok
}

// GetLocalVarChain 沿 frame chain 从当前向 root 走, 第一个命中即返回.
// GetVar scope=auto 用. 不 fallback 到 rt.vars (调用方自己组合).
func (s *ExecState) GetLocalVarChain(name string) (any, bool) {
	for f := s.CurrentFrame; f != nil; f = f.Parent {
		if v, ok := f.LocalVars[name]; ok {
			return v, true
		}
	}
	return nil, false
}

// ResolveVar 按 local → parent chain → global 顺序查找.
func (s *ExecState) ResolveVar(name string) (any, bool) {
	for f := s.CurrentFrame; f != nil; f = f.Parent {
		if v, ok := f.LocalVars[name]; ok {
			return v, true
		}
	}
	if v, ok := s.GlobalVars[name]; ok {
		return v, true
	}
	return nil, false
}

// SetGlobalVar 显式写 GlobalVars（SetVar/IncVar scope=global 时用）。
func (s *ExecState) SetGlobalVar(name string, value any) {
	if s.GlobalVars == nil {
		s.GlobalVars = map[string]any{}
	}
	s.GlobalVars[name] = value
}

// ResolveSourceCalibCounts 沿 frame.Parent 链向上找最近一个非 nil 的
// SubgraphRef.RecordingContext.MouseCounts360 当作 MouseMoveRel scale 的源值。
// 主图层级 (CurrentFrame.SubgraphRef == nil) 或者所有祖先都没有 RecordingContext → 返 0
// (caller 应解释为 "不缩放, raw 值原样输出").
func (s *ExecState) ResolveSourceCalibCounts() int {
	for f := s.CurrentFrame; f != nil; f = f.Parent {
		if f.SubgraphRef == nil {
			continue
		}
		if f.SubgraphRef.RecordingContext == nil {
			continue
		}
		if c := f.SubgraphRef.RecordingContext.MouseCounts360; c > 0 {
			return c
		}
	}
	return 0
}
