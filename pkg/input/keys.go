package input

import (
	"fmt"
	"strings"
)

var vkMap = map[string]uint32{
	"esc": 0x1B, "escape": 0x1B,
	"space": 0x20,
	"enter": 0x0D, "return": 0x0D,
	"shift": 0x10, "ctrl": 0x11, "control": 0x11, "alt": 0x12,
	"tab":       0x09,
	"backspace": 0x08, "back": 0x08,
	"delete": 0x2E, "del": 0x2E,
	"insert": 0x2D, "ins": 0x2D,
	"home": 0x24, "end": 0x23,
	"pgup": 0x21, "pageup": 0x21,
	"pgdn": 0x22, "pagedown": 0x22,
	"up": 0x26, "down": 0x28, "left": 0x25, "right": 0x27,
	",": 0xBC, ".": 0xBE,
	"caps": 0x14, "capslock": 0x14,
}

// VK parses the key names shared by graph validation and native input adapters.
func VK(name string) uint32 {
	n := strings.ToLower(strings.TrimSpace(name))
	if v, ok := vkMap[n]; ok {
		return v
	}
	if len(n) == 1 {
		c := n[0]
		if c >= 'a' && c <= 'z' {
			return uint32(c - 'a' + 'A')
		}
		if c >= '0' && c <= '9' {
			return uint32(c)
		}
	}
	if len(n) >= 2 && n[0] == 'f' {
		var num int
		if _, err := fmt.Sscanf(n[1:], "%d", &num); err == nil && num >= 1 && num <= 12 {
			return 0x70 + uint32(num) - 1
		}
	}
	return 0
}
