// internal/services/container/window.go
// 共享 WindowTarget 节点查找 + matchspec 读取 + resolve 入口。
// 纯依赖 container 包自身类型 + winutil，无 import cycle 风险。
package container

import (
	"context"
	"errors"
	"time"

	"yotta/pkg/winutil"
)

// ErrNoWindowTarget 容器主图里找不到 WindowTarget 节点时返回。
var ErrNoWindowTarget = errors.New("MISSING_WINDOW_TARGET — 容器缺 WindowTarget 节点，先放一个并捕获目标窗口")

// FindMainGraphNode 在容器主图里找指定 kind 的第一个节点。
// WindowTarget / MouseCalibration 等声明式节点均 single-instance per container — 找到即停。
func FindMainGraphNode(c *Container, kind string) *GraphNode {
	for i := range c.Graph.Nodes {
		if c.Graph.Nodes[i].Kind == kind {
			return &c.Graph.Nodes[i]
		}
	}
	return nil
}

// ReadWindowTargetMatchSpec 从 WindowTarget 节点的 Config 读取四个匹配字段。
// n 或 n.Config 为 nil 返空 MatchSpec。
func ReadWindowTargetMatchSpec(n *GraphNode) winutil.MatchSpec {
	if n == nil || n.Config == nil {
		return winutil.MatchSpec{}
	}
	return winutil.MatchSpec{
		Title:       PinString(n, "Title"),
		Class:       PinString(n, "Class"),
		ProcessName: PinString(n, "ProcessName"),
		TitleMatch:  PinString(n, "TitleMatch"),
	}
}

// ReadWindowTargetCaptureBackend 读容器主图 WindowTarget 节点的 CaptureBackend 配置.
// 无节点/无配置 → "auto". 制作工具(截模板/取色)按此现建一次性 IBackend, 与运行时同后端.
func ReadWindowTargetCaptureBackend(c *Container) string {
	wt := FindMainGraphNode(c, "WindowTarget")
	if wt == nil {
		return "auto"
	}
	if s := PinString(wt, "CaptureBackend"); s != "" {
		return s
	}
	return "auto"
}

// DefaultScaleTolerance 模板跨分辨率缩放容差默认值. k=2.0 → 允许缩放比 ∈ [0.5, 2.0].
const DefaultScaleTolerance = 2.0

// ReadWindowTargetScaleTolerance 读容器主图 WindowTarget 节点的 ScaleTolerance 配置.
// 无节点/无配置/非法 (<1.0) → DefaultScaleTolerance. 模板匹配按此做跨分辨率缩放兜底.
func ReadWindowTargetScaleTolerance(c *Container) float64 {
	wt := FindMainGraphNode(c, "WindowTarget")
	if wt == nil {
		return DefaultScaleTolerance
	}
	if v, ok := PinFloat(wt, "ScaleTolerance"); ok && v >= 1.0 {
		return v
	}
	return DefaultScaleTolerance
}

// ResolveWindowTarget 找主图 WindowTarget 节点并 resolve 成 WindowHandle。
// 无节点 → ErrNoWindowTarget；resolve 失败 → winutil 原始 error。
func ResolveWindowTarget(ctx context.Context, c *Container, timeout, interval time.Duration) (winutil.WindowHandle, error) {
	wt := FindMainGraphNode(c, "WindowTarget")
	if wt == nil {
		return winutil.WindowHandle{}, ErrNoWindowTarget
	}
	return winutil.ResolveWindow(ctx, ReadWindowTargetMatchSpec(wt), timeout, interval)
}
