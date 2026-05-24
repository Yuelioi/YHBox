// internal/nodes/system/sentinels.go
// Sentinel errors for stub system nodes.
//
// SubgraphInput / SubgraphOutput / CollapsedNode can't express their semantics
// through normal exec-out edges — they require runner cooperation. Phase 5
// RegionRunner / sub-runner mechanism replaces these stubs and these errors
// disappear in favor of real frame push/pop semantics.
//
// CommentBox is render-only; framework gatekeep (Spec.IsVisualOnly) should
// prevent reaching its Run, but errVisualOnlyNotRunnable is defensive in case
// the Phase 5 runner forgets the IsVisualOnly check.
package system

import "errors"

// errSubgraphNodeStub returned by SubgraphInput / SubgraphOutput / CollapsedNode Run.
// Phase 5 RegionRunner / sub-runner mechanism replaces these stubs.
var errSubgraphNodeStub = errors.New("subgraph node — Phase 5 runtime impl pending")

// errVisualOnlyNotRunnable returned by CommentBox Run (which framework gatekeep should
// prevent reaching, but defensive in case Phase 5 runner forgets the IsVisualOnly check).
var errVisualOnlyNotRunnable = errors.New("visual-only node — should not be executed")
