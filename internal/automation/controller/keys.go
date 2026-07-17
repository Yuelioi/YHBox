package controller

import (
	"strconv"
	"strings"
)

// ValidKeyName validates the portable key vocabulary accepted at the
// automation seam. Native key-code conversion remains an adapter detail.
func ValidKeyName(value string) bool {
	if value == "" || strings.TrimSpace(value) != value {
		return false
	}
	normalized := strings.ToLower(value)
	switch normalized {
	case "esc", "escape", "space", "enter", "return", "shift", "ctrl", "control", "alt", "tab", "backspace", "back", "delete", "del", "insert", "ins", "home", "end", "pgup", "pageup", "pgdn", "pagedown", "up", "down", "left", "right", ",", ".", "caps", "capslock":
		return true
	}
	if len(normalized) == 1 {
		return normalized[0] >= 'a' && normalized[0] <= 'z' || normalized[0] >= '0' && normalized[0] <= '9'
	}
	if normalized[0] != 'f' {
		return false
	}
	number, err := strconv.Atoi(normalized[1:])
	return err == nil && number >= 1 && number <= 12
}
