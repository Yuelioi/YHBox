// internal/nodes/detect/click_common.go
// clickWithMods — 组合键 + 连点公共 helper；ClickTemplate (Task 2.4) 和 Phase 3 ClickAt 共用。
package detect

import (
	"strings"
	"time"

	"yotta/internal/node"
)

const interClickGapMs = 60 // 连点间隔，< 系统双击时限 (500ms)

var validMods = map[string]bool{"ctrl": true, "shift": true, "alt": true, "win": true}

// parseMods 解析 "ctrl+shift" → ["ctrl","shift"]; 全合法返 true。空串 → (nil,true)。
func parseMods(keys string) ([]string, bool) {
	keys = strings.TrimSpace(keys)
	if keys == "" {
		return nil, true
	}
	var mods []string
	for _, p := range strings.Split(keys, "+") {
		m := strings.ToLower(strings.TrimSpace(p))
		if !validMods[m] {
			return nil, false
		}
		mods = append(mods, m)
	}
	return mods, true
}

// clickWithMods 按住修饰键 → 连点 count 次 → 逆序松开。count<=1 单击。
// 无论点击成败都逆序松开修饰键（避免卡住 ctrl 等）。
func clickWithMods(ctx node.Ctx, pt node.Point, btn string, keys string, count int) error {
	mods, _ := parseMods(keys) // 合法性由节点 Validate 保证
	for _, m := range mods {
		if err := ctx.Input().KeyDown(m); err != nil {
			// KeyDown 失败：松开已按下的键后返错
			for i := len(mods) - 1; i >= 0; i-- {
				_ = ctx.Input().KeyUp(mods[i])
			}
			return err
		}
	}
	if count < 1 {
		count = 1
	}
	var clickErr error
	for i := 0; i < count; i++ {
		if clickErr = ctx.Input().Click(pt.X, pt.Y, btn, 50); clickErr != nil {
			break
		}
		if i < count-1 {
			if err := waitOrCancel(ctx, interClickGapMs*time.Millisecond); err != nil {
				clickErr = err
				break
			}
		}
	}
	// 无论点击成败，逆序松开修饰键
	for i := len(mods) - 1; i >= 0; i-- {
		_ = ctx.Input().KeyUp(mods[i])
	}
	return clickErr
}
