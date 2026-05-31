package input

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"yhbox/internal/node"
)

// recordingInput 实现 node.InputService, 把每次调用 append 到 calls.
// 故意写在 key_press_test.go (字母序最早), 同包其它 _test.go 复用 — 不抽 helpers.go.
type recordingInput struct {
	calls []string
	err   error // 注入到下一次调用返回
}

func (r *recordingInput) KeyDown(vk string) error {
	r.calls = append(r.calls, fmt.Sprintf("KeyDown:%s", vk))
	return r.err
}
func (r *recordingInput) KeyUp(vk string) error {
	r.calls = append(r.calls, fmt.Sprintf("KeyUp:%s", vk))
	return r.err
}
func (r *recordingInput) Click(xRatio, yRatio float64, button string, durationMs int) error {
	r.calls = append(r.calls, fmt.Sprintf("Click:%.3f:%.3f:%s:%d", xRatio, yRatio, button, durationMs))
	return r.err
}
func (r *recordingInput) MouseMoveRel(dx, dy, durationMs int) error {
	r.calls = append(r.calls, fmt.Sprintf("MouseMoveRel:%d:%d:%d", dx, dy, durationMs))
	return r.err
}
func (r *recordingInput) Scroll(xRatio, yRatio float64, notches int) error {
	r.calls = append(r.calls, fmt.Sprintf("Scroll:%.3f:%.3f:%d", xRatio, yRatio, notches))
	return r.err
}
func (r *recordingInput) MouseDown(xRatio, yRatio float64, button string) error {
	r.calls = append(r.calls, fmt.Sprintf("MouseDown:%.3f:%.3f:%s", xRatio, yRatio, button))
	return r.err
}
func (r *recordingInput) MouseUp(button string) error {
	r.calls = append(r.calls, fmt.Sprintf("MouseUp:%s", button))
	return r.err
}
func (r *recordingInput) MoveTo(xRatio, yRatio float64) error {
	r.calls = append(r.calls, fmt.Sprintf("MoveTo:%.3f:%.3f", xRatio, yRatio))
	return r.err
}
func (r *recordingInput) CursorRatio() (float64, float64, error) {
	// spy 起点固定原点; err 注入时一并返回 (复用 r.err).
	return 0, 0, r.err
}

// withInput 替 ServiceBundle.Input 字段, 其余 stub.
func withInput(in node.InputService) node.ServiceBundle {
	b := node.StubServices()
	b.Input = in
	return b
}

func TestKeyPress_HappyPath(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&KeyPress{})
	rn, _ := node.Get("KeyPress")

	rec := &recordingInput{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{kpInVK: "F", kpInDurationMs: 100},
		nil, withInput(rec))

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != kpOutDone {
		t.Errorf("exit = %q, want Done", r.ExitName)
	}
	if len(rec.calls) != 2 || rec.calls[0] != "KeyDown:F" || rec.calls[1] != "KeyUp:F" {
		t.Errorf("calls = %v, want [KeyDown:F KeyUp:F]", rec.calls)
	}
}

func TestKeyPress_CtxCancel_ReleasesAndReturns(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&KeyPress{})
	rn, _ := node.Get("KeyPress")

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(20 * time.Millisecond); cancel() }()

	rec := &recordingInput{}
	start := time.Now()
	r := node.RunNode(ctx, rn, nil,
		map[string]any{kpInVK: "F", kpInDurationMs: 10000},
		nil, withInput(rec))
	elapsed := time.Since(start)

	if elapsed > time.Second {
		t.Fatalf("elapsed %v — KeyPress 没响应 ctx 取消, 仍等满 10s", elapsed)
	}
	if r.Error == nil || !errors.Is(r.Error, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", r.Error)
	}
	// 取消也必须松键: KeyDown 后 KeyUp
	if len(rec.calls) != 2 || rec.calls[0] != "KeyDown:F" || rec.calls[1] != "KeyUp:F" {
		t.Errorf("calls = %v, want [KeyDown:F KeyUp:F]", rec.calls)
	}
}

func TestKeyPress_JitterPct_StillDownUp(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&KeyPress{})
	rn, _ := node.Get("KeyPress")

	rec := &recordingInput{}
	// JitterPct 仅扰动按住时长, 不改 down/up 行为
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{kpInVK: "F", kpInDurationMs: 10, kpInJitterPct: 20},
		nil, withInput(rec))
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if len(rec.calls) != 2 || rec.calls[0] != "KeyDown:F" || rec.calls[1] != "KeyUp:F" {
		t.Errorf("calls = %v, want [KeyDown:F KeyUp:F]", rec.calls)
	}
}

func TestKeyPress_BackendError_Propagates(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&KeyPress{})
	rn, _ := node.Get("KeyPress")

	rec := &recordingInput{err: errors.New("hwnd closed")}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{kpInVK: "A", kpInDurationMs: 50},
		nil, withInput(rec))

	if r.Error == nil {
		t.Fatal("expected backend error to propagate")
	}
}

func TestKeyPress_EmptyVK_ValidationError(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&KeyPress{})
	rn, _ := node.Get("KeyPress")

	// vk="" 触发 Validate
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{kpInVK: "", kpInDurationMs: 50},
		nil, withInput(&recordingInput{}))

	if len(r.Validation) != 1 || r.Validation[0].Code != "INVALID_VK" {
		t.Errorf("validation = %v, want INVALID_VK", r.Validation)
	}
}

func TestKeyPress_UnknownVK_ValidationError(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&KeyPress{})
	rn, _ := node.Get("KeyPress")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{kpInVK: "not-a-real-key", kpInDurationMs: 50},
		nil, withInput(&recordingInput{}))

	if len(r.Validation) != 1 || r.Validation[0].Code != "INVALID_VK" {
		t.Errorf("validation = %v, want INVALID_VK for unknown keyname", r.Validation)
	}
}
