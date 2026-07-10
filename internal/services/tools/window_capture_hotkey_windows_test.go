package tools

import (
	"testing"
)

// TestCancelWin32WindowTargetCapture_NoActive 没 active session 时 cancel 返 nil.
// (避免前端 stale captureID 调 cancel 报错)
func TestCancelWin32WindowTargetCapture_NoActive(t *testing.T) {
	captureMu.Lock()
	activeCapture = nil
	captureMu.Unlock()

	if err := cancelWin32WindowTargetCapture("any-id"); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

// TestCancelWin32WindowTargetCapture_WrongID id 不匹配 idempotent 返 nil.
// (前端 captureID 跟 backend 不一致时 cancel 不要 fail)
func TestCancelWin32WindowTargetCapture_WrongID(t *testing.T) {
	captureMu.Lock()
	activeCapture = &captureSession{
		id:     "real-id",
		done:   make(chan struct{}),
		cancel: make(chan struct{}),
	}
	captureMu.Unlock()
	defer func() {
		captureMu.Lock()
		activeCapture = nil
		captureMu.Unlock()
	}()

	if err := cancelWin32WindowTargetCapture("wrong-id"); err != nil {
		t.Fatalf("expected nil error for wrong id, got %v", err)
	}
	// active session 应保留 (cancel 无效)
	captureMu.Lock()
	stillActive := activeCapture != nil
	captureMu.Unlock()
	if !stillActive {
		t.Fatalf("active session was cleared by wrong-id cancel")
	}
}

// TestRandID_Unique 连续生成 ID 不撞 (基本正确性, 不是 crypto 测试)
func TestRandID_Unique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		id := randID()
		if seen[id] {
			t.Fatalf("randID collision after %d iterations: %q", i, id)
		}
		seen[id] = true
	}
}
