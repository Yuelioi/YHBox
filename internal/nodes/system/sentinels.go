// internal/nodes/system/sentinels.go
// Sentinel errors for system stub / region nodes.
//
// errSubgraphNodeStub — SubgraphInput / SubgraphOutput Run 永不该被框架调; runner
// 在 dispatchInRegion / runRegionBody 内特殊路由跳过 Run.
//
// errVisualOnlyNotRunnable — CommentBox Run 防御 (Spec.IsVisualOnly gatekeep 应阻止).
//
// Region 节点 (Subgraph/Try/CollapsedNode) 的 "must use RunNodeAsRegion" defensive sentinel
// 已统一为 node.ErrMustUseRegion (P2.2c).
//
// ThrowError — typed error returned by Throw. Try (或任意上游 catch) 用 errors.As
// 抽 message; Unwrap → errThrow 让 errors.Is(err, errThrow) 仅判断 "是否 throw".
package system

import "errors"

// errSubgraphNodeStub returned by SubgraphInput / SubgraphOutput Run.
// 实际 runner 在 dispatchInRegion / runRegionBody 特殊路由 (sub-runner frame push/pop) 跳过 Run.
var errSubgraphNodeStub = errors.New("subgraph node — Phase 5 runtime impl pending")

// errVisualOnlyNotRunnable returned by CommentBox Run (framework gatekeep Spec.IsVisualOnly
// 应阻止, defensive 防 runner 忘检查).
var errVisualOnlyNotRunnable = errors.New("visual-only node — should not be executed")

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
