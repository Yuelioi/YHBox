package hotkey

import "strings"

// reservedHotkeys 不允许用户绑定的热键（小写归一化字符串）。
// 包括系统级保留 + Windows 通用快捷键。
var reservedHotkeys = map[string]string{
	// 系统级
	"ctrl+shift+r": "录制 toggle 全局热键",
	"esc":          "Esc（容易跟游戏菜单冲突）",
	"escape":       "Esc（容易跟游戏菜单冲突）",

	// Windows 系统通用快捷键
	"ctrl+a": "系统快捷键：全选",
	"ctrl+c": "系统快捷键：复制",
	"ctrl+v": "系统快捷键：粘贴",
	"ctrl+x": "系统快捷键：剪切",
	"ctrl+z": "系统快捷键：撤销",
	"ctrl+y": "系统快捷键：重做",
	"ctrl+s": "系统快捷键：保存",
	"ctrl+f": "系统快捷键：查找",
	"ctrl+n": "系统快捷键：新建",
	"ctrl+o": "系统快捷键：打开",
	"ctrl+p": "系统快捷键：打印",
	"ctrl+t": "系统快捷键：新标签",
	"ctrl+w": "系统快捷键：关闭标签",
	"alt+f4": "系统快捷键：关闭窗口",
}

// IsReservedHotkey 检查 hotkey 字符串是否在保留列表。
// 返回 (hit, reason)。大小写不敏感，trim 后比较。
// 额外规则：无 modifier 的纯单字母/单数字键拒绝（聊天打字易误触）。
// F1-F12 / Tab / 方向键等"特殊键"允许。
func IsReservedHotkey(s string) (bool, string) {
	norm := strings.ToLower(strings.TrimSpace(s))
	if norm == "" {
		return false, ""
	}
	if reason, ok := reservedHotkeys[norm]; ok {
		return true, reason
	}
	if !strings.Contains(norm, "+") {
		if len(norm) == 1 {
			c := norm[0]
			if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
				return true, "纯字母/数字键无修饰键易误触日常输入"
			}
		}
	}
	return false, ""
}
