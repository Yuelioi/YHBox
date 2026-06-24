package input

import (
	"unicode/utf16"
	"unsafe"

	"github.com/lxn/win"
)

// siKeyUnicode KEYEVENTF_UNICODE — wScan 持 UTF-16 code unit, SendInput 全局注入 unicode 字符.
// 与 siKeyScanCode (0x0008) 互斥; 这里单独定义避免命名混淆.
const siKeyUnicode uint32 = 0x0004

// TypeText 通过 SendInput KEYEVENTF_UNICODE 向当前前台窗口注入文本.
// 每个 rune 拆成 UTF-16 code unit(s), 每个 code unit 发一对 keydown+keyup.
// BMP 外字符 (>U+FFFF) 自动拆 surrogate pair (两个 code unit = 4 次 SendInput 调用).
// hwnd 当前未使用 (SendInput 全局注入, 不挑 hwnd); 预留给将来 FakeActivate 需求.
func TypeText(_ win.HWND, s string) error {
	for _, r := range s {
		units := utf16.Encode([]rune{r})
		for _, u := range units {
			down := sendInputKeyBlock{
				Type: siInputKeyboard,
				Ki:   keybdInput{WVk: 0, WScan: u, DwFlags: siKeyUnicode},
			}
			procSendInput.Call(1, uintptr(unsafe.Pointer(&down)), unsafe.Sizeof(down))
			up := sendInputKeyBlock{
				Type: siInputKeyboard,
				Ki:   keybdInput{WVk: 0, WScan: u, DwFlags: siKeyUnicode | siKeyKeyUp},
			}
			procSendInput.Call(1, uintptr(unsafe.Pointer(&up)), unsafe.Sizeof(up))
		}
	}
	return nil
}
