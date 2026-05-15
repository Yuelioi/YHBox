// recorder_test.go：单测只覆盖 vkName / Recorder 构造 / Active=false 这些
// 不需要真 hwnd + 真键鼠输入的路径。真录制集成阶段手测。
package recording

import "testing"

func TestVKName(t *testing.T) {
	cases := []struct {
		in   uint32
		want string
	}{
		{'A', "A"},
		{'Z', "Z"},
		{'0', "0"},
		{'9', "9"},
		{0x70, "F1"},
		{0x7B, "F12"},
		{VK_SPACE, "Space"},
		{VK_ESCAPE, "Esc"},
		{VK_RETURN, "Enter"},
		{VK_LCONTROL, "Ctrl"},
		{VK_RCONTROL, "Ctrl"},
		{VK_CONTROL, "Ctrl"},
		{VK_LSHIFT, "Shift"},
		{VK_LMENU, "Alt"},
		{VK_UP, "Up"},
		{VK_DOWN, "Down"},
		{VK_LEFT, "Left"},
		{VK_RIGHT, "Right"},
		{0xFF, ""},  // 不支持
		{0x00, ""},  // 不支持
		{0x6F, ""},  // 紧挨 F1 前一位
		{0x7C, ""},  // 紧挨 F12 后一位
	}
	for _, c := range cases {
		if got := vkName(c.in); got != c.want {
			t.Errorf("vkName(0x%X) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRecorder_NotActiveInitially(t *testing.T) {
	r := NewRecorder(nil)
	if r.Active() {
		t.Error("新建 Recorder 不应是 active")
	}
}

func TestRecorder_StopWhenNotActive(t *testing.T) {
	r := NewRecorder(nil)
	if _, err := r.Stop(); err == nil {
		t.Error("非 active 时 Stop 应返 error")
	}
}

func TestRecorder_CancelWhenNotActive(t *testing.T) {
	r := NewRecorder(nil)
	// Cancel 在非 active 状态应静默返回，不 panic
	defer func() {
		if rec := recover(); rec != nil {
			t.Errorf("Cancel 不应 panic, 但 panic 了: %v", rec)
		}
	}()
	r.Cancel()
	if r.Active() {
		t.Error("Cancel 后仍 Active")
	}
}
