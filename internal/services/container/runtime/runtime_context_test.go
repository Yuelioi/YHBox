package runtime

import (
	"image"
	"testing"

	"yotta/internal/automation/target"
	winutil "yotta/pkg/winutil"
)

func TestRuntimeContext_SetActiveWindow_StickyAndGuard(t *testing.T) {
	rt := &RuntimeContext{}
	if _, err := rt.ActiveHWND(); err == nil {
		t.Fatal("want ErrNoActiveWindow when unset, got nil")
	}
	rt.SetActiveWindow(winutil.WindowHandle{HWND: 42, ClientW: 800, ClientH: 600})
	wh := rt.WindowHandle()
	if wh.HWND != 42 || wh.ClientW != 800 {
		t.Fatalf("WindowHandle not sticky: %+v", wh)
	}
	h, err := rt.ActiveHWND()
	if err != nil || h != 42 {
		t.Fatalf("ActiveHWND = %v, %v; want 42, nil", h, err)
	}
	tg, ok := rt.ActiveTarget()
	if !ok {
		t.Fatal("ActiveTarget missing after SetActiveWindow")
	}
	if tg.ID != "win32:42" || tg.Kind != target.KindWin32Window || tg.Ref.HWND != 42 || tg.Resolution.W != 800 || tg.Resolution.H != 600 {
		t.Fatalf("ActiveTarget after SetActiveWindow = %#v", tg)
	}
}

func TestRuntimeContext_SetActiveTarget_Sticky(t *testing.T) {
	rt := &RuntimeContext{}
	tg := target.Target{
		ID:         "android:emulator-5554",
		Kind:       target.KindAndroidADB,
		Ref:        target.TargetRef{ADBSerial: "emulator-5554"},
		Resolution: target.Size{W: 1080, H: 1920},
	}
	rt.SetActiveTarget(tg)
	got, ok := rt.ActiveTarget()
	if !ok {
		t.Fatal("ActiveTarget missing after SetActiveTarget")
	}
	if got.ID != tg.ID || got.Kind != tg.Kind || got.Ref.ADBSerial != tg.Ref.ADBSerial {
		t.Fatalf("ActiveTarget = %#v, want %#v", got, tg)
	}
}

func TestRuntimeContext_WindowOverrideStack(t *testing.T) {
	rt := &RuntimeContext{}
	rt.SetActiveWindow(winutil.WindowHandle{HWND: 1})
	rt.PushWindowOverride(winutil.WindowHandle{HWND: 2})
	if h, _ := rt.ActiveHWND(); h != 2 {
		t.Fatal("栈顶应为 2")
	}
	rt.PushWindowOverride(winutil.WindowHandle{HWND: 3})
	if h, _ := rt.ActiveHWND(); h != 3 {
		t.Fatal("嵌套栈顶应为 3")
	}
	rt.PopWindowOverride()
	if h, _ := rt.ActiveHWND(); h != 2 {
		t.Fatal("pop 后应回 2")
	}
	rt.PopWindowOverride()
	if h, _ := rt.ActiveHWND(); h != 1 {
		t.Fatal("清空后回粘性 1")
	}
}

func TestFrameCache_PerHWND(t *testing.T) {
	rt := &RuntimeContext{}
	rt.initFrameCache()
	imgA := image.NewRGBA(image.Rect(0, 0, 2, 2))
	rt.putFrameCache(1, imgA)
	if got := rt.peekFrameCache(1); got != imgA {
		t.Fatal("hwnd=1 应命中")
	}
	if got := rt.peekFrameCache(2); got != nil {
		t.Fatal("hwnd=2 不应命中")
	}
	rt.invalidateFrameCacheFor(1)
	if got := rt.peekFrameCache(1); got != nil {
		t.Fatal("invalidate 后应 miss")
	}
}
