// internal/services/container/window.go
// 共享 Win32WindowTarget 节点查找 + matchspec 读取 + resolve 入口。
// 纯依赖 container 包自身类型 + winutil，无 import cycle 风险。
package container

import (
	"context"
	"errors"
	"time"

	"github.com/yottaapp/yotta/pkg/winutil"
)

// ErrNoWin32WindowTarget 容器主图里找不到 Win32WindowTarget 节点时返回。
var ErrNoWin32WindowTarget = errors.New("MISSING_WIN32_WINDOW_TARGET — 容器缺 Win32WindowTarget 节点，先放一个并捕获 Windows 窗口")

// FindMainGraphNode 在容器主图里找指定 kind 的第一个节点。
// 返回主图中第一个匹配 kind 的节点；主图可包含多个同 kind 节点（如多 Win32WindowTarget），找到即停。
func FindMainGraphNode(c *Container, kind string) *GraphNode {
	for i := range c.Graph.Nodes {
		if c.Graph.Nodes[i].Kind == kind {
			return &c.Graph.Nodes[i]
		}
	}
	return nil
}

// ReadWin32WindowTargetMatchSpec 从 Win32WindowTarget 节点的 Config 读取四个匹配字段。
// n 或 n.Config 为 nil 返空 MatchSpec。
func ReadWin32WindowTargetMatchSpec(n *GraphNode) winutil.MatchSpec {
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

// ReadWin32WindowTargetCaptureBackend 读容器级 CaptureBackend 配置.
// 无配置 → "auto". 制作工具(截模板/取色)按此现建一次性 IBackend, 与运行时同后端.
func ReadWin32WindowTargetCaptureBackend(c *Container) string {
	if c.CaptureBackend != "" {
		return c.CaptureBackend
	}
	return "auto"
}

// DefaultInputBackend is the container-level default for Windows input.
// SendInput uses real foreground OS input and covers applications that ignore window messages.
const DefaultInputBackend = "sendinput"

// ReadWin32WindowTargetInputBackend reads the container input mode and applies the default for
// legacy or incomplete records. Explicit postmessage remains supported for background automation.
func ReadWin32WindowTargetInputBackend(c *Container) string {
	if c != nil && c.InputBackend != "" {
		return c.InputBackend
	}
	return DefaultInputBackend
}

// DefaultScaleTolerance 模板跨分辨率缩放容差默认值. k=2.0 → 允许缩放比 ∈ [0.5, 2.0].
const DefaultScaleTolerance = 2.0

// ReadWin32WindowTargetScaleTolerance 读容器级 ScaleTolerance 配置.
// 未填/非法 (<1.0) → DefaultScaleTolerance. 模板匹配按此做跨分辨率缩放兜底.
func ReadWin32WindowTargetScaleTolerance(c *Container) float64 {
	if c.ScaleTolerance >= 1.0 {
		return c.ScaleTolerance
	}
	return DefaultScaleTolerance
}

// ResolveWin32WindowTarget 找主图 Win32WindowTarget 节点并 resolve 成 WindowHandle。
// 无节点 → ErrNoWin32WindowTarget；resolve 失败 → winutil 原始 error。
func ResolveWin32WindowTarget(ctx context.Context, c *Container, timeout, interval time.Duration) (winutil.WindowHandle, error) {
	wt := FindMainGraphNode(c, "Win32WindowTarget")
	if wt == nil {
		return winutil.WindowHandle{}, ErrNoWin32WindowTarget
	}
	return winutil.ResolveWindow(ctx, ReadWin32WindowTargetMatchSpec(wt), timeout, interval)
}
