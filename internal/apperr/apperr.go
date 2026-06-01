// Package apperr 是「用户可见业务错误」的结构化载体: 后端只发 code + params,
// FE 走 vue-i18n 翻译 (error.<CODE>)。后端永远不出现用户文案 (保持 locale-free)。
//
// 错误模型只许两套: ValidationError (图校验) + apperr.Error (业务错误)。
// 不得再造第三套。
package apperr

// Error 携带一个 i18n code 与可选插值参数。
type Error struct {
	Code   string         `json:"code"`
	Params map[string]any `json:"params,omitempty"`
}

// Error 只返 Code (供后端 log/debug)。要 params 走结构化日志字段, 不进 Error()。
func (e *Error) Error() string { return e.Code }

// New 构造一个 *Error。params 可为 nil。
func New(code string, params map[string]any) *Error {
	return &Error{Code: code, Params: params}
}

// 首批 code 常量 (与 FE error.* i18n 一一对应)。
const (
	CodeWailsNotReady           = "WAILS_NOT_READY"
	CodeContainerIDRequired     = "CONTAINER_ID_REQUIRED"
	CodeRecordingNoWindowTarget = "RECORDING_NO_WINDOW_TARGET"
)
