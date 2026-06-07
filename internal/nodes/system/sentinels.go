package system

import (
	"errors"

	"yotta/internal/node"
)

var errThrow = errors.New("throw")

// ThrowError 由 Throw 节点 Run() 返回. 实现 node.Coded → 可被 region 失败出口截获.
type ThrowError struct {
	Message string
	Code    string // 用户填; 空 → CodeThrown
}

func (e *ThrowError) Error() string { return "throw: " + e.Message }
func (e *ThrowError) Unwrap() error { return errThrow }
func (e *ThrowError) ErrCode() node.ErrCode {
	if e.Code == "" {
		return node.CodeThrown
	}
	return node.ErrCode(e.Code)
}

func IsThrowRequested(err error) bool { return errors.Is(err, errThrow) }
