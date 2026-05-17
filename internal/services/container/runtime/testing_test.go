package runtime

import (
	"github.com/lxn/win"

	pkginput "yhbox/pkg/input"
)

// fakeInputBackend 测试用 pkginput.Backend 实现. 计数 Click 调用次数, 其他方法返 nil.
// 共享给所有 _test.go (runner_test / playclip_test / safe_backend_test 等).
type fakeInputBackend struct {
	clicks int
}

func (f *fakeInputBackend) Name() string                        { return "fake" }
func (f *fakeInputBackend) Capabilities() pkginput.Capabilities { return pkginput.Capabilities{} }
func (f *fakeInputBackend) Click(_ win.HWND, _, _ float64, _ string, _ int) error {
	f.clicks++
	return nil
}
func (f *fakeInputBackend) KeyPress(win.HWND, string, int) error               { return nil }
func (f *fakeInputBackend) KeyDown(win.HWND, string) error                     { return nil }
func (f *fakeInputBackend) KeyUp(win.HWND, string) error                       { return nil }
func (f *fakeInputBackend) MouseDown(win.HWND, float64, float64, string) error { return nil }
func (f *fakeInputBackend) MouseUp(win.HWND, string) error                     { return nil }
func (f *fakeInputBackend) MouseMoveRel(win.HWND, int, int, int) error         { return nil }
func (f *fakeInputBackend) Scroll(win.HWND, float64, float64, int) error       { return nil }
func (f *fakeInputBackend) ReleaseAll() error                                  { return nil }
func (f *fakeInputBackend) Close() error                                       { return nil }
