package input

import (
	"context"
	"errors"
	"testing"

	"yotta/internal/node"
)

// recordingWindow 接 node.WindowService, 计 BringForeground 调用次数 + 注入 err.
type recordingWindow struct {
	calls int
	err   error
}

func (r *recordingWindow) BringForeground() error {
	r.calls++
	return r.err
}
func (r *recordingWindow) HWND() uintptr                 { return 0 }
func (r *recordingWindow) ClientSize() (int, int, error) { return 0, 0, nil }
func (r *recordingWindow) SetActive(ctx context.Context, title, class, processName, titleMatch string) error {
	return nil
}

func withWindow(w node.WindowService) node.ServiceBundle {
	b := node.StubServices()
	b.Window = w
	return b
}

func TestBringWindowForeground_HappyPath(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&BringWindowForeground{})
	rn, _ := node.Get("BringWindowForeground")

	win := &recordingWindow{}
	r := node.RunNode(context.Background(), rn, nil, nil, nil, withWindow(win), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != bgfOutDone {
		t.Errorf("exit = %q, want Done", r.ExitName)
	}
	if win.calls != 1 {
		t.Errorf("BringForeground calls = %d, want 1", win.calls)
	}
}

// 失败仅 warn log, 不报 error, 仍走 Done.
func TestBringWindowForeground_BackendError_StillDone(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&BringWindowForeground{})
	rn, _ := node.Get("BringWindowForeground")

	win := &recordingWindow{err: errors.New("fullscreen exclusive")}
	r := node.RunNode(context.Background(), rn, nil, nil, nil, withWindow(win), false)

	if r.Error != nil {
		t.Fatalf("backend error should NOT propagate (warn log only), got %v", r.Error)
	}
	if r.ExitName != bgfOutDone {
		t.Errorf("exit = %q, want Done even on backend error", r.ExitName)
	}
	if win.calls != 1 {
		t.Errorf("BringForeground calls = %d, want 1", win.calls)
	}
}
