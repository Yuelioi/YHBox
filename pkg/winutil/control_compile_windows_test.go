package winutil

import "testing"

func TestControlPrimitives_ZeroHandleSafe(t *testing.T) {
	if IsWindow(0) {
		t.Fatal("0 句柄应非窗口")
	}
	if err := Maximize(0); err == nil {
		t.Fatal("0 句柄应报错")
	}
	if err := MoveResize(0, 0, 0, 0, 0); err == nil {
		t.Fatal("0 句柄应报错")
	}
	if err := CloseWindow(0); err == nil {
		t.Fatal("0 句柄应报错")
	}
	if _, err := InspectWindowState(0); err == nil {
		t.Fatal("0 句柄读取状态应报错")
	}
	if _, err := EnterBorderless(0); err == nil {
		t.Fatal("0 句柄应报错")
	}
}
