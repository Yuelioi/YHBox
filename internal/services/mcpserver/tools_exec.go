package mcpserver

import (
	"context"

	"yotta/internal/node"
	"yotta/internal/services/container"
	"yotta/internal/services/container/runtime"
	"yotta/pkg/winutil"
)

func errResult(code, msg string) RunNodeResult {
	return RunNodeResult{Ok: false, Error: &RunNodeError{Code: code, Message: msg}}
}

// runNode 编排: arm/busy 闸 → 闸节点 → 句柄重校验 → 合成微容器 → validate → 跑 → 收割.
func (s *Server) runNode(ctx context.Context, kind string, params map[string]any, hwnd uintptr) (RunNodeResult, *node.Image) {
	if s.deps.Armed == nil || !s.deps.Armed() {
		return errResult("NOT_ARMED", "MCP 未武装, 去设置页打开 arm 开关"), nil
	}
	if s.deps.Busy != nil && s.deps.Busy() {
		return errResult("BUSY", "GUI 正在跑容器, 稍后重试"), nil
	}
	c, nodeID, err := buildMicroContainer(kind, params)
	if err != nil {
		return errResult("UNRUNNABLE_KIND", err.Error()), nil
	}
	// 句柄重校验 (HWND 可能被 OS 复用 / 窗口已关).
	wh, err := winutil.WindowMetadata(hwnd)
	if err != nil {
		return errResult("WINDOW_GONE", err.Error()), nil
	}
	// 参数校验 (缺必填 / 类型非法).
	for _, e := range container.ValidateContainer(c, nil) {
		if e.Severity == container.SeverityError {
			return errResult("INVALID_PARAMS", "params 校验未过 (详见节点 spec)"), nil
		}
	}
	// 串行化 run_node, 防 AI 并行交错输入.
	s.runMu.Lock()
	defer s.runMu.Unlock()

	rt := runtime.NewRuntimeContext(
		c, s.deps.InputBus, s.deps.Matcher, s.deps.Game,
		func(string, any) {}, // no-op emit: 收割走 execOutputs, 不靠事件
		s.deps.Clip, s.deps.MouseCounts(),
	)
	rt.SetActiveWindow(wh)
	return runMicroContainer(ctx, rt, c, nodeID)
}
