//go:build windows

package input

import (
	"errors"
	"fmt"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"github.com/lxn/win"
)

// siKeyUnicode KEYEVENTF_UNICODE — wScan 持 UTF-16 code unit, SendInput 全局注入 unicode 字符.
// 与 siKeyScanCode (0x0008) 互斥; 这里单独定义避免命名混淆.
const siKeyUnicode uint32 = 0x0004

// TypeText 通过 SendInput KEYEVENTF_UNICODE 向当前前台焦点窗口注入文本 (全局注入, 不挑 hwnd).
// 每个 rune 拆成 UTF-16 code unit(s), 每个 code unit 发一对 keydown+keyup.
// BMP 外字符 (>U+FFFF) 自动拆 surrogate pair (两个 code unit = 4 次 SendInput 调用).
//
// 全局路径 = sendinput backend 用 (它本就要前台). postmessage backend (可选, 后台不抢前台)
// 走 PostText (WM_CHAR targeted) 而非这里 —— 全局 SendInput 会把字符注入到真正持有键盘焦点
// 的窗口, 后台目标窗口收不到. hwnd 在此忽略.
func TypeText(_ win.HWND, s string) error {
	for _, r := range s {
		units := utf16.Encode([]rune{r})
		for _, u := range units {
			inputs := [2]sendInputKeyBlock{{
				Type: siInputKeyboard,
				Ki:   keybdInput{WVk: 0, WScan: u, DwFlags: siKeyUnicode},
			}, {
				Type: siInputKeyboard,
				Ki:   keybdInput{WVk: 0, WScan: u, DwFlags: siKeyUnicode | siKeyKeyUp},
			}}
			sent, _, errno := procSendInput.Call(2, uintptr(unsafe.Pointer(&inputs[0])), unsafe.Sizeof(inputs[0]))
			if sent == 2 {
				continue
			}
			var sendErr error
			if errno != syscall.Errno(0) {
				sendErr = fmt.Errorf("SendInput unicode failed after %d events: %w", sent, errno)
			} else {
				sendErr = fmt.Errorf("SendInput unicode sent %d events, want 2", sent)
			}
			if sent == 1 {
				released, _, releaseErrno := procSendInput.Call(1, uintptr(unsafe.Pointer(&inputs[1])), unsafe.Sizeof(inputs[1]))
				if released != 1 {
					if releaseErrno != syscall.Errno(0) {
						sendErr = errors.Join(sendErr, fmt.Errorf("release partial unicode input: %w", releaseErrno))
					} else {
						sendErr = errors.Join(sendErr, errors.New("release partial unicode input sent no event"))
					}
				}
			}
			return sendErr
		}
	}
	return nil
}

// PostText 通过 PostMessage WM_CHAR 向目标 hwnd 投递文本 (targeted, 后台可用, 不抢前台).
// 每个 rune 拆成 UTF-16 code unit(s), 每个 code unit 发一条 WM_CHAR (wParam=code unit, lParam=0).
// BMP 外字符 (>U+FFFF) 自动拆 surrogate pair (两条 WM_CHAR).
//
// 与 postmessage backend 的 KeyDown/Click 同语义 —— 字符直达目标窗口消息队列, 不依赖键盘焦点;
// 对照 TypeText 走全局 SendInput 需目标窗口前台持焦. 标准 Edit / RichEdit / Slate / Chromium
// 等处理 WM_CHAR 的控件均可收到; 极少数只认完整 keydown→char→keyup 的自绘输入框需切 sendinput 后端.
func PostText(hwnd win.HWND, s string) error {
	for _, r := range s {
		for _, u := range utf16.Encode([]rune{r}) {
			if err := postMessageChecked(hwnd, WM_CHAR, uintptr(u), 0); err != nil {
				return err
			}
		}
	}
	return nil
}
