package system

import (
	"errors"

	"github.com/yottaapp/yotta/internal/node"
)

var errThrow = errors.New("throw")

// ThrowError 由 Throw 节点 Run() 返回. 实现 node.Coded → 可被 region 失败出口截获.
type ThrowError struct {
	Message string
	Code    string // 用户填; 空 → CodeThrown
}

// Error 只返 message — 错误码经 ErrCode() 单独取 (dump 行渲染 err[code]=msg / Fail 出口 Code 字段),
// 不再带 "throw: " 前缀 (码已单列, 前缀成重复噪音)。
func (e *ThrowError) Error() string { return e.Message }
func (e *ThrowError) Unwrap() error { return errThrow }
func (e *ThrowError) ErrCode() node.ErrCode {
	if e.Code == "" {
		return node.CodeThrown
	}
	return node.ErrCode(e.Code)
}

func IsThrowRequested(err error) bool { return errors.Is(err, errThrow) }
