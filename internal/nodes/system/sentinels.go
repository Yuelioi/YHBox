// internal/nodes/system/sentinels.go
// Sentinel errors for system stub / region nodes.
//
// SubgraphInput / SubgraphOutput / CollapsedNode can't express their semantics
// through normal exec-out edges — they require runner cooperation. Phase 5
// RegionRunner / sub-runner mechanism replaces these stubs and these errors
// disappear in favor of real frame push/pop semantics.
//
// CommentBox is render-only; framework gatekeep (Spec.IsVisualOnly) should
// prevent reaching its Run, but errVisualOnlyNotRunnable is defensive in case
// the Phase 5 runner forgets the IsVisualOnly check.
//
// errSubgraphMustUseRegion / errTryMustUseRegion — defensive sentinels returned
// by Subgraph.Run / Try.Run if the framework misses the RegionRunner dispatch
// path (RunNodeAsRegion). Normal execution goes through RunRegion which receives
// the body callback the runner constructs.
//
// ThrowError — typed error returned by Throw. Try (or any upstream catch) can
// errors.As to extract the user-provided message without string-parsing the
// error text. Unwrap → errThrow base sentinel so errors.Is(err, errThrow) also
// works for "is this a throw?" queries without caring about message.
package system

import "errors"

// errSubgraphNodeStub returned by SubgraphInput / SubgraphOutput / CollapsedNode Run.
// Phase 5 RegionRunner / sub-runner mechanism replaces these stubs.
var errSubgraphNodeStub = errors.New("subgraph node — Phase 5 runtime impl pending")

// errVisualOnlyNotRunnable returned by CommentBox Run (which framework gatekeep should
// prevent reaching, but defensive in case Phase 5 runner forgets the IsVisualOnly check).
var errVisualOnlyNotRunnable = errors.New("visual-only node — should not be executed")

// errSubgraphMustUseRegion — Subgraph 的 Run() 被框架直调时返此 error. 正常路径是
// RunNodeAsRegion → RunRegion + runner-provided body 回调.
var errSubgraphMustUseRegion = errors.New("Subgraph node must be invoked via RunNodeAsRegion, not RunNode")

// errTryMustUseRegion — Try 的 Run() 被框架直调时返此 error. 正常路径同 Subgraph.
var errTryMustUseRegion = errors.New("Try node must be invoked via RunNodeAsRegion, not RunNode")

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
