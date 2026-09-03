package hotkey

import (
	"errors"
	"fmt"

	"github.com/yottaapp/yotta/internal/apperr"
)

// ErrDuplicateKey Register 同 key 二次调用返这个。
var ErrDuplicateKey = errors.New("hotkey registry: duplicate key")

// HotkeyConflictError 新热键撞到另一个已注册 entry。
// 前端通过 error message 前缀 [conflict] 识别（wails3 alpha 不传 typed error，只传 string）。
type HotkeyConflictError struct {
	Key         string // 当前要改的 entry key
	ConflictKey string // 被撞的 entry key
	Label       string // 被撞 entry 的中文 label
}

func (e *HotkeyConflictError) Error() string {
	return fmt.Sprintf("[conflict] 热键已被 %q 占用", e.Label)
}

func (e *HotkeyConflictError) RPCErrorEnvelope() apperr.Envelope {
	return apperr.Envelope{ID: "hotkey.conflict", Category: apperr.CategoryValidation, Params: map[string]any{
		"key": e.Key, "conflictKey": e.ConflictKey, "conflictLabel": e.Label,
	}}
}

// HotkeyReservedError 新热键命中 reserved 黑名单。
type HotkeyReservedError struct {
	Hotkey string
	Reason string
}

func (e *HotkeyReservedError) Error() string {
	return fmt.Sprintf("[reserved] %s", e.Reason)
}

func (e *HotkeyReservedError) RPCErrorEnvelope() apperr.Envelope {
	return apperr.Envelope{ID: "hotkey.reserved", Category: apperr.CategoryValidation, Params: map[string]any{"hotkey": e.Hotkey}}
}

// HotkeyInvalidError 热键字符串语法错误（解析不出 mods+vk）。
type HotkeyInvalidError struct {
	Hotkey string
	Reason string
}

func (e *HotkeyInvalidError) Error() string {
	return fmt.Sprintf("[invalid] %s", e.Reason)
}

func (e *HotkeyInvalidError) RPCErrorEnvelope() apperr.Envelope {
	return apperr.Envelope{ID: "hotkey.invalid", Category: apperr.CategoryValidation, Params: map[string]any{"hotkey": e.Hotkey}}
}
