package mcpserver

import (
	"context"
	"testing"

	"yotta/internal/services/container"

	_ "yotta/internal/nodes/all"
)

func TestRunNode_NotArmed_Rejects(t *testing.T) {
	s := NewServer(Deps{Armed: func() bool { return false }, Busy: func() bool { return false }})
	res, _ := s.runNode(context.Background(), "ClickAt", map[string]any{"X": 1, "Y": 1}, 0)
	if res.Ok || res.Error == nil || res.Error.Code != "NOT_ARMED" {
		t.Fatalf("未武装应返 NOT_ARMED, got %+v", res)
	}
}

func TestRunNode_Busy_Rejects(t *testing.T) {
	s := NewServer(Deps{Armed: func() bool { return true }, Busy: func() bool { return true }})
	res, _ := s.runNode(context.Background(), "ClickAt", map[string]any{"X": 1, "Y": 1}, 0)
	if res.Error == nil || res.Error.Code != "BUSY" {
		t.Fatalf("GUI 在跑应返 BUSY, got %+v", res)
	}
}

func TestRunNode_UnrunnableKind_Rejects(t *testing.T) {
	s := NewServer(Deps{Armed: func() bool { return true }, Busy: func() bool { return false }})
	res, _ := s.runNode(context.Background(), "Loop", nil, 0)
	if res.Error == nil || res.Error.Code != "UNRUNNABLE_KIND" {
		t.Fatalf("Loop 应返 UNRUNNABLE_KIND, got %+v", res)
	}
}

// TestHasBlockingValidationError_ClickAt 验证 MISSING_WIN32_WINDOW_TARGET 豁免逻辑:
// ClickAt 微容器校验会产生 MISSING_WIN32_WINDOW_TARGET (因没有 Win32WindowTarget 节点),
// 但 hasBlockingValidationError 应返回 false — 这是此次 critical bug 的回归防护。
func TestHasBlockingValidationError_ClickAt(t *testing.T) {
	c, _, err := buildMicroContainer("ClickAt", map[string]any{"X": 1, "Y": 1})
	if err != nil {
		t.Fatalf("buildMicroContainer failed: %v", err)
	}

	// 前提验证: validator 确实会对此微容器报 MISSING_WIN32_WINDOW_TARGET (error severity).
	// 这证明了 bug 的存在前提 — 没有豁免时 runNode 会被误拦。
	errs := container.ValidateContainer(c, nil)
	foundMWT := false
	for _, e := range errs {
		if e.Code == container.CodeMissingWin32WindowTarget && e.Severity == container.SeverityError {
			foundMWT = true
			break
		}
	}
	if !foundMWT {
		t.Fatal("前提失效: ClickAt 微容器未触发 MISSING_WIN32_WINDOW_TARGET error — 测试需更新")
	}

	// 核心断言: hasBlockingValidationError 必须豁免 MISSING_WIN32_WINDOW_TARGET, 返回 false.
	if hasBlockingValidationError(c) {
		t.Fatal("hasBlockingValidationError 应豁免 MISSING_WIN32_WINDOW_TARGET 返 false, 但返了 true (critical bug 未修)")
	}
}
