package hotkey

import (
	"fmt"
	"strings"
)

// NormalizeHotkey 把任意大小写/顺序/别名的热键字符串归一成稳定 canonical form。
// 用于冲突检测、reserved 查表、entry 持久化。
//
// 规则：
//   - trim 空格
//   - 全小写
//   - 别名合并：control → ctrl, escape → esc, return → enter, pageup → pgup,
//     pagedown → pgdn, delete → del, insert → ins, period/dot → ., comma → ,
//   - modifier 顺序固定：ctrl < shift < alt（按物理键盘从左到右习惯）
//   - 主键放最后
//
// 例：
//
//	"Shift+Ctrl+R"   → "ctrl+shift+r"
//	"CTRL+SHIFT+R"   → "ctrl+shift+r"
//	"Control+Shift+R"→ "ctrl+shift+r"
//	"Escape"         → "esc"
//
// 解析失败 → 返 "", error。
func NormalizeHotkey(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("空热键")
	}
	parts := strings.Split(s, "+")
	var hasCtrl, hasShift, hasAlt bool
	var mainKey string
	for i, p := range parts {
		p = strings.TrimSpace(strings.ToLower(p))
		if p == "" {
			return "", fmt.Errorf("空 token in %q", s)
		}
		// modifier 合并
		switch p {
		case "ctrl", "control":
			if i == len(parts)-1 {
				return "", fmt.Errorf("modifier 不能放末尾: %q", s)
			}
			hasCtrl = true
			continue
		case "shift":
			if i == len(parts)-1 {
				return "", fmt.Errorf("modifier 不能放末尾: %q", s)
			}
			hasShift = true
			continue
		case "alt":
			if i == len(parts)-1 {
				return "", fmt.Errorf("modifier 不能放末尾: %q", s)
			}
			hasAlt = true
			continue
		}
		// 非 modifier — 必须是末尾的主键
		if i != len(parts)-1 {
			return "", fmt.Errorf("按键 %q 必须放在末尾", p)
		}
		// 借 vkOfName 校验是否是已知键名 — 不识别就拒绝
		if _, err := vkOfName(p); err != nil {
			return "", err
		}
		// 主键别名合并
		switch p {
		case "escape":
			mainKey = "esc"
		case "return":
			mainKey = "enter"
		case "pageup":
			mainKey = "pgup"
		case "pagedown":
			mainKey = "pgdn"
		case "delete":
			mainKey = "del"
		case "insert":
			mainKey = "ins"
		case "period", "dot":
			mainKey = "."
		case "comma":
			mainKey = ","
		default:
			mainKey = p
		}
	}
	if mainKey == "" {
		return "", fmt.Errorf("缺少按键: %q", s)
	}
	// 拼按固定顺序：ctrl + shift + alt + key
	var out []string
	if hasCtrl {
		out = append(out, "ctrl")
	}
	if hasShift {
		out = append(out, "shift")
	}
	if hasAlt {
		out = append(out, "alt")
	}
	out = append(out, mainKey)
	return strings.Join(out, "+"), nil
}
