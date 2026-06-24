// internal/node/click_mods.go
// ClickWithMods / ParseMods / InterClickGapMs — 组合键+连点公共 helper.
// detect 包 (ClickTemplate) 和 input 包 (ClickAt) 共用, 所以提升到 internal/node.
package node

import (
	"strings"
	"time"
)

// InterClickGapMs 连点间隔 (ms), < 系统双击时限 (500ms).
const InterClickGapMs = 60

var validMods = map[string]bool{"ctrl": true, "shift": true, "alt": true, "win": true}

// ParseMods 解析 "ctrl+shift" → ["ctrl","shift"]; 全合法返 true. 空串 → (nil,true).
func ParseMods(keys string) ([]string, bool) {
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

// sleepOrCancel 等 d; 期间 ctx 取消则立即返 ctx.Err(). d<=0 立即返 nil.
func sleepOrCancel(ctx Ctx, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	select {
	case <-ctx.Context().Done():
		return ctx.Context().Err()
	case <-time.After(d):
		return nil
	}
}

// ClickWithMods 按住修饰键 → 连点 count 次 (每次 hold durationMs) → 逆序松开.
// count<=1 单击. 无论成败都逆序松开修饰键 (避免 ctrl 等卡住).
func ClickWithMods(ctx Ctx, pt Point, btn string, keys string, count int, durationMs int) error {
	mods, _ := ParseMods(keys) // 合法性由节点 Validate 保证
	for _, m := range mods {
		if err := ctx.Input().KeyDown(m); err != nil {
			// KeyDown 失败: 松开已按下的键后返错
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
		if clickErr = ctx.Input().Click(pt.X, pt.Y, btn, durationMs); clickErr != nil {
			break
		}
		if i < count-1 {
			if err := sleepOrCancel(ctx, InterClickGapMs*time.Millisecond); err != nil {
				clickErr = err
				break
			}
		}
	}
	// 无论点击成败, 逆序松开修饰键
	for i := len(mods) - 1; i >= 0; i-- {
		_ = ctx.Input().KeyUp(mods[i])
	}
	return clickErr
}
