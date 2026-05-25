// internal/nodes/system/sentinels.go
// System package 仅余 Throw 相关 typed error.
//
// ThrowError — typed error returned by Throw. Try (或任意上游 catch) 用 errors.As
// 抽 message; Unwrap → errThrow 让 errors.Is(err, errThrow) 仅判断 "是否 throw".
package system

import "errors"

// errThrow — Throw 节点用的 base sentinel. ThrowError.Unwrap() 返此值, 调用方
// errors.Is(err, errThrow) 可识别 "是否 Throw 抛出" 而不关心具体 message.
var errThrow = errors.New("throw")

// ThrowError 由 Throw 节点 Run() 返回. Try region 可用 errors.As(err, &te) 抽取
// 原始 message; 也可 errors.Is(err, errThrow) 仅判断"是否 throw".
type ThrowError struct {
	Message string
}

func (e *ThrowError) Error() string { return "throw: " + e.Message }
func (e *ThrowError) Unwrap() error { return errThrow }

// IsThrowRequested 报 err 是否是 Throw 节点 sentinel — runtime routeResult defensive
// 用这个识别 Throw 漏到顶层 (Try.RunRegion 应该已截获).
func IsThrowRequested(err error) bool { return errors.Is(err, errThrow) }
