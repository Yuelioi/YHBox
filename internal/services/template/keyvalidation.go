package template

import (
	"fmt"
	"regexp"
)

var keyPattern = regexp.MustCompile(`^[a-z0-9_]+(\.[a-z0-9_]+)+$`)

// ValidateKey 校验 namespace key 格式. 至少 1 个 dot, 多级 (e.g. fishing.hook_icon / fishing.ui.close).
// 字母数字下划线 only, 总长 <= 64.
func ValidateKey(key string) error {
	if len(key) == 0 {
		return fmt.Errorf("empty key")
	}
	if len(key) > 64 {
		return fmt.Errorf("key %q exceeds 64 chars", key)
	}
	if !keyPattern.MatchString(key) {
		return fmt.Errorf("key %q must match ^[a-z0-9_]+(\\.[a-z0-9_]+)+$ (e.g. fishing.hook_icon)", key)
	}
	return nil
}
