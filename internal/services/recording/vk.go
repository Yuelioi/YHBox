// vk.go：Win32 VK_* 常量 + modifier flag + VK → action.Step.Vk 字符串映射。
//
// 字符串名跟 actions/hotkey.go parseHotkeyString 接受的格式对齐。
// 不支持的 VK 返空串，drain loop 直接跳过。
package recording

import "fmt"

// Win32 modifier flags（RegisterHotKey API 用，跟前端 hotkey 字符串 mask 对齐）
const (
	MOD_ALT     uint32 = 0x0001
	MOD_CONTROL uint32 = 0x0002
	MOD_SHIFT   uint32 = 0x0004
)

// 常用 VK 常量（Win32 SDK winuser.h）。只列录制需要的。
const (
	VK_LBUTTON  uint32 = 0x01
	VK_RBUTTON  uint32 = 0x02
	VK_MBUTTON  uint32 = 0x04
	VK_BACK     uint32 = 0x08
	VK_TAB      uint32 = 0x09
	VK_RETURN   uint32 = 0x0D
	VK_SHIFT    uint32 = 0x10
	VK_CONTROL  uint32 = 0x11
	VK_MENU     uint32 = 0x12 // Alt
	VK_PAUSE    uint32 = 0x13
	VK_CAPITAL  uint32 = 0x14
	VK_ESCAPE   uint32 = 0x1B
	VK_SPACE    uint32 = 0x20
	VK_PRIOR    uint32 = 0x21 // PgUp
	VK_NEXT     uint32 = 0x22 // PgDn
	VK_END      uint32 = 0x23
	VK_HOME     uint32 = 0x24
	VK_LEFT     uint32 = 0x25
	VK_UP       uint32 = 0x26
	VK_RIGHT    uint32 = 0x27
	VK_DOWN     uint32 = 0x28
	VK_INSERT   uint32 = 0x2D
	VK_DELETE   uint32 = 0x2E
	VK_LSHIFT   uint32 = 0xA0
	VK_RSHIFT   uint32 = 0xA1
	VK_LCONTROL uint32 = 0xA2
	VK_RCONTROL uint32 = 0xA3
	VK_LMENU    uint32 = 0xA4
	VK_RMENU    uint32 = 0xA5
)

// vkName Win32 VK 转 action.Step.Vk 字符串。
// 返回空串 = 不支持的键（小键盘 / 媒体键 / OEM 等），caller 应丢弃此事件。
func vkName(vk uint32) string {
	// A-Z（VK_A..VK_Z 的 code 跟 ASCII 一致）
	if vk >= 'A' && vk <= 'Z' {
		return string(rune(vk))
	}
	// 0-9（同样跟 ASCII 一致）
	if vk >= '0' && vk <= '9' {
		return string(rune(vk))
	}
	// F1-F12 = 0x70..0x7B
	if vk >= 0x70 && vk <= 0x7B {
		return fmt.Sprintf("F%d", vk-0x70+1)
	}
	switch vk {
	case VK_SPACE:
		return "Space"
	case VK_ESCAPE:
		return "Esc"
	case VK_RETURN:
		return "Enter"
	case VK_TAB:
		return "Tab"
	case VK_BACK:
		return "Backspace"
	case VK_DELETE:
		return "Delete"
	case VK_INSERT:
		return "Insert"
	case VK_HOME:
		return "Home"
	case VK_END:
		return "End"
	case VK_PRIOR:
		return "PgUp"
	case VK_NEXT:
		return "PgDn"
	case VK_UP:
		return "Up"
	case VK_DOWN:
		return "Down"
	case VK_LEFT:
		return "Left"
	case VK_RIGHT:
		return "Right"
	case VK_CONTROL, VK_LCONTROL, VK_RCONTROL:
		return "Ctrl"
	case VK_SHIFT, VK_LSHIFT, VK_RSHIFT:
		return "Shift"
	case VK_MENU, VK_LMENU, VK_RMENU:
		return "Alt"
	}
	return ""
}
