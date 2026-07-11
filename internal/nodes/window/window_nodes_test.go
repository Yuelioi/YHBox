package window

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/node"
	"github.com/yottaapp/yotta/pkg/winutil"
)

func jsonNum(s string) json.Number { return json.Number(s) }

func TestGetWindow_ResolvesToDoneWindow(t *testing.T) {
	orig := resolveWindowFn
	defer func() { resolveWindowFn = orig }()
	resolveWindowFn = func(_ context.Context, _ winutil.MatchSpec, _, _ time.Duration) (winutil.WindowHandle, error) {
		return winutil.WindowHandle{HWND: 42, Title: "记事本", ClientW: 800, ClientH: 600}, nil
	}

	registry := node.NewRegistry()
	registry.Register(&GetWindow{})
	rn, _ := registry.Get("GetWindow")
	svc := node.StubServices()

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{selInTitle: "记事本"}, nil, svc, false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != gwDone {
		t.Fatalf("应走 Done, got %q", r.ExitName)
	}
	w, ok := r.OutputData["Window"].(node.Window)
	if !ok || w.HWND != 42 || w.Title != "记事本" {
		t.Fatalf("Done.Window 错: %+v (ok=%v)", w, ok)
	}
}

// recordingWindowService 记录调了哪个控制方法 + Snapshot 返预设窗口。
type recordingWindowService struct {
	snap             node.Window
	maximized        bool
	minimized        bool
	restored         bool
	borderless       bool
	restoredBorders  bool
	closed           bool
	moveResize       [4]int
	moveResizeCalled bool
}

func (r *recordingWindowService) BringForeground() error { return nil }
func (r *recordingWindowService) HWND() uintptr          { return r.snap.HWND }
func (r *recordingWindowService) ClientSize() (int, int, error) {
	return r.snap.ClientW, r.snap.ClientH, nil
}
func (r *recordingWindowService) SetActive(_ context.Context, _, _, _, _ string) error { return nil }
func (r *recordingWindowService) Maximize() error {
	r.maximized = true
	return nil
}
func (r *recordingWindowService) Minimize() error {
	r.minimized = true
	return nil
}
func (r *recordingWindowService) Restore() error {
	r.restored = true
	return nil
}
func (r *recordingWindowService) BorderlessFullscreen() error {
	r.borderless = true
	return nil
}
func (r *recordingWindowService) RestoreBorders() error {
	r.restoredBorders = true
	return nil
}
func (r *recordingWindowService) MoveResize(x, y, w, h int) error {
	r.moveResize = [4]int{x, y, w, h}
	r.moveResizeCalled = true
	return nil
}
func (r *recordingWindowService) Close() error { r.closed = true; return nil }
func (r *recordingWindowService) Snapshot() (node.Window, error) {
	return r.snap, nil
}

func TestWindowState_AllStates(t *testing.T) {
	cases := []struct {
		state string
		check func(*recordingWindowService) bool
		label string
	}{
		{"maximize", func(r *recordingWindowService) bool { return r.maximized }, "r.maximized"},
		{"minimize", func(r *recordingWindowService) bool { return r.minimized }, "r.minimized"},
		{"restore", func(r *recordingWindowService) bool { return r.restored }, "r.restored"},
		{"borderlessFullscreen", func(r *recordingWindowService) bool { return r.borderless }, "r.borderless"},
		{"restoreBorders", func(r *recordingWindowService) bool { return r.restoredBorders }, "r.restoredBorders"},
	}
	for _, tc := range cases {
		t.Run(tc.state, func(t *testing.T) {
			rec := &recordingWindowService{snap: node.Window{HWND: 9, ClientW: 1920, ClientH: 1080}}
			registry := node.NewRegistry()
			registry.Register(&WindowState{})
			rn, _ := registry.Get("WindowState")
			svc := node.StubServices()
			svc.Window = rec

			r := node.RunNode(context.Background(), rn, nil,
				map[string]any{wsInState: tc.state}, nil, svc, false)
			if r.Error != nil {
				t.Fatalf("state=%q: r.Error=%v", tc.state, r.Error)
			}
			if !tc.check(rec) {
				t.Fatalf("state=%q: %s 未被调用", tc.state, tc.label)
			}
			if r.ExitName != wsDone {
				t.Fatalf("state=%q: ExitName=%q, want %q", tc.state, r.ExitName, wsDone)
			}
			w, ok := r.OutputData["Window"].(node.Window)
			if !ok || w.HWND != 9 {
				t.Fatalf("state=%q: Done.Window 错: %+v", tc.state, r.OutputData["Window"])
			}
		})
	}

	// unknown state → r.Error 携带 CodeError
	t.Run("unknown", func(t *testing.T) {
		rec := &recordingWindowService{snap: node.Window{HWND: 9}}
		registry := node.NewRegistry()
		registry.Register(&WindowState{})
		rn, _ := registry.Get("WindowState")
		svc := node.StubServices()
		svc.Window = rec

		r := node.RunNode(context.Background(), rn, nil,
			map[string]any{wsInState: "bogus"}, nil, svc, false)
		if r.Error == nil {
			t.Fatal("unknown State 应返 error")
		}
		var coded node.Coded
		if !errors.As(r.Error, &coded) || coded.ErrCode() != node.CodeError {
			t.Fatalf("error 应为 CodeError, got %v", r.Error)
		}
	})
}

func TestMoveResizeWindow_FiresDoneWithFreshWindow(t *testing.T) {
	rec := &recordingWindowService{snap: node.Window{HWND: 7, ClientW: 800, ClientH: 600}}
	registry := node.NewRegistry()
	registry.Register(&MoveResizeWindow{})
	rn, _ := registry.Get("MoveResizeWindow")
	svc := node.StubServices()
	svc.Window = rec

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{mrInX: jsonNum("100"), mrInY: jsonNum("200"), mrInW: jsonNum("800"), mrInH: jsonNum("600")},
		nil, svc, false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if !rec.moveResizeCalled {
		t.Fatal("应调 MoveResize")
	}
	if rec.moveResize != [4]int{100, 200, 800, 600} {
		t.Fatalf("MoveResize 参数错: %v", rec.moveResize)
	}
	w, ok := r.OutputData["Window"].(node.Window)
	if !ok || w.ClientW != 800 {
		t.Fatalf("Done.Window 应是操作后重读: %+v", r.OutputData["Window"])
	}
}

func TestCloseWindow_FiresDoneNoWindow(t *testing.T) {
	rec := &recordingWindowService{snap: node.Window{HWND: 5}}
	registry := node.NewRegistry()
	registry.Register(&CloseWindow{})
	rn, _ := registry.Get("CloseWindow")
	svc := node.StubServices()
	svc.Window = rec

	r := node.RunNode(context.Background(), rn, nil, nil, nil, svc, false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if !rec.closed {
		t.Fatal("应调 Close")
	}
	if r.ExitName != cwDone {
		t.Fatalf("应走 Done, got %q", r.ExitName)
	}
	if r.OutputData["Window"] != nil {
		t.Fatalf("CloseWindow 不透传窗口, got %+v", r.OutputData["Window"])
	}
}
