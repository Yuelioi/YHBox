// Package locale 定义全项目唯一的 locale 常量。
// 所有 "zh" / "en" 字面量都应该走这里的常量。
package locale

// Locale 是强类型 locale 值。
type Locale string

const (
	Zh Locale = "zh"
	En Locale = "en"
)

// All 返回所有支持的 locale。给 Validate / settings UI 用。
func All() []Locale {
	return []Locale{Zh, En}
}

// Valid 检查 s 是不是已注册 locale。
func Valid(s string) bool {
	for _, l := range All() {
		if string(l) == s {
			return true
		}
	}
	return false
}

// String 让 Locale 可以直接用作 string 上下文（filepath 拼接等）。
func (l Locale) String() string { return string(l) }
