package system

import (
	"context"
	"errors"
	"testing"

	"yotta/internal/node"
)

func TestWindowTarget_SpecHasExecInNoBackendPins(t *testing.T) {
	s := WindowTarget{}.Spec()
	in := map[string]string{}
	for _, p := range s.Inputs {
		in[p.Name] = p.Type
	}
	if in["In"] != "Exec" {
		t.Fatal("WindowTarget 应有 exec-in 'In'")
	}
	for _, gone := range []string{"InputBackend", "CaptureBackend", "ScaleTolerance"} {
		if _, ok := in[gone]; ok {
			t.Fatalf("pin %q 应已移除", gone)
		}
	}
	for _, keep := range []string{"Title", "Class", "ProcessName", "TitleMatch"} {
		if _, ok := in[keep]; !ok {
			t.Fatalf("pin %q 应保留", keep)
		}
	}
}

// stubWindowService 记录 SetActive 调用参数, 可注入 error.
type stubWindowService struct {
	err         error
	calledTitle string
	called      bool
}

func (s *stubWindowService) BringForeground() error { return nil }
func (s *stubWindowService) HWND() uintptr          { return 1 }
func (s *stubWindowService) ClientSize() (int, int, error) {
	return 1920, 1080, nil
}
func (s *stubWindowService) SetActive(_ context.Context, title, _, _, _ string) error {
	s.called = true
	s.calledTitle = title
	return s.err
}
func (s *stubWindowService) Maximize() error              { return nil }
func (s *stubWindowService) Minimize() error              { return nil }
func (s *stubWindowService) Restore() error               { return nil }
func (s *stubWindowService) BorderlessFullscreen() error  { return nil }
func (s *stubWindowService) RestoreBorders() error        { return nil }
func (s *stubWindowService) MoveResize(_, _, _, _ int) error { return nil }
func (s *stubWindowService) Close() error                 { return nil }
func (s *stubWindowService) Snapshot() (node.Window, error) { return node.Window{}, nil }

func TestWindowTarget_Run_CallsSetActive(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&WindowTarget{})
	rn, _ := node.Get("WindowTarget")

	stub := &stubWindowService{}
	svc := node.StubServices()
	svc.Window = stub

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			wtInTitle:      "GameWindow",
			wtInTitleMatch: "exact",
		},
		nil, svc, false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != wtOutDone {
		t.Errorf("exit = %q, want %q", r.ExitName, wtOutDone)
	}
	if !stub.called {
		t.Fatal("SetActive 应被调用")
	}
	if stub.calledTitle != "GameWindow" {
		t.Errorf("SetActive title = %q, want GameWindow", stub.calledTitle)
	}
}

func TestWindowTarget_Run_PropagatesSetActiveError(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&WindowTarget{})
	rn, _ := node.Get("WindowTarget")

	stub := &stubWindowService{err: errors.New("窗口未找到")}
	svc := node.StubServices()
	svc.Window = stub

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{wtInTitle: "Nope", wtInTitleMatch: "exact"},
		nil, svc, false)
	if r.Error == nil {
		t.Fatal("SetActive 报错时 Run 应返回 error")
	}
}
