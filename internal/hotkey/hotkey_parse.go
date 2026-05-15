package hotkey

import (
	"fmt"
	"strings"
)

// hotkey 解析工具：把 "Ctrl+Shift+F" 字符串解析成 mods/vk。
// 从 actions 包迁过来 — 现在归 services 包，作 system-level infra。

// modifier Win32 标志（跟 winMOD_* 别名相同，保持局部命名简短）
const (
	hkModAlt   uint32 = 0x0001
	hkModCtrl  uint32 = 0x0002
	hkModShift uint32 = 0x0004
)

// parseHotkey 把 "Ctrl+Shift+F" 解析为 (mods, vk)。
// 规则：
//   - "+" 分割，每段 trim 空格
//   - modifier（Ctrl/Control/Shift/Alt，大小写不敏感）必须放在前面
//   - 最后一段必须是主键（A-Z / 0-9 / F1-F12 / 常见特殊键）
//   - 空串或语法错误 → error
func parseHotkey(s string) (mods uint32, vk uint32, err error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, 0, fmt.Errorf("空热键")
	}
	parts := strings.Split(s, "+")
	for i, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			return 0, 0, fmt.Errorf("空 token in %q", s)
		}
		isMod, mflag := modifierFlag(p)
		if isMod {
			if i == len(parts)-1 {
				return 0, 0, fmt.Errorf("modifier %q 不能放在末尾", p)
			}
			mods |= mflag
			continue
		}
		if i != len(parts)-1 {
			return 0, 0, fmt.Errorf("按键 %q 必须放在末尾", p)
		}
		v, e := vkOfName(p)
		if e != nil {
			return 0, 0, e
		}
		vk = v
	}
	if vk == 0 {
		return 0, 0, fmt.Errorf("缺少按键: %q", s)
	}
	return mods, vk, nil
}

// ParseHotkey 公开版，给跨包调用方（main.go / 未来其它）用。
func ParseHotkey(s string) (mods uint32, vk uint32, err error) {
	return parseHotkey(s)
}

func modifierFlag(token string) (bool, uint32) {
	switch strings.ToLower(token) {
	case "ctrl", "control":
		return true, hkModCtrl
	case "shift":
		return true, hkModShift
	case "alt":
		return true, hkModAlt
	}
	return false, 0
}

// vkOfName 简表覆盖常用键。大小写不敏感。
func vkOfName(name string) (uint32, error) {
	n := strings.ToUpper(strings.TrimSpace(name))
	if len(n) == 1 {
		c := n[0]
		if (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			return uint32(c), nil
		}
	}
	if strings.HasPrefix(n, "F") && len(n) > 1 {
		var num int
		if _, err := fmt.Sscanf(n[1:], "%d", &num); err == nil && num >= 1 && num <= 12 {
			return 0x70 + uint32(num) - 1, nil // VK_F1=0x70
		}
	}
	switch n {
	case "SPACE":
		return 0x20, nil
	case "ESC", "ESCAPE":
		return 0x1B, nil
	case "ENTER", "RETURN":
		return 0x0D, nil
	case "TAB":
		return 0x09, nil
	case "DELETE", "DEL":
		return 0x2E, nil
	case "INSERT", "INS":
		return 0x2D, nil
	case "HOME":
		return 0x24, nil
	case "END":
		return 0x23, nil
	case "PGUP", "PAGEUP":
		return 0x21, nil
	case "PGDN", "PAGEDOWN":
		return 0x22, nil
	case "UP":
		return 0x26, nil
	case "DOWN":
		return 0x28, nil
	case "LEFT":
		return 0x25, nil
	case "RIGHT":
		return 0x27, nil
	case ".", "PERIOD", "DOT":
		return 0xBE, nil
	case ",", "COMMA":
		return 0xBC, nil
	}
	return 0, fmt.Errorf("不支持的键名: %q", name)
}
