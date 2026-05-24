// internal/nodes/control/sentinels.go
// Sentinel errors for halt / loop control nodes.
//
// Stop/Break/Continue can't express their semantics through normal exec-out
// edges — they require runner cooperation. Phase 4 ships these as plain errors
// returned from Run; Phase 5 runner intercepts them:
//   - errStopRun           → halt graph dispatch cleanly (no container:error emit)
//   - errBreakRequested    → Loop region pops frame + walks to .complete out
//   - errContinueRequested → Loop region rewinds to header for next iter
//
// Until the Phase 5 runner integrates these, Break/Continue outside a Loop
// region just surface as node errors. That's acceptable: the old v3/v4 runtime
// rejects Break/Continue outside Loop with the same "not in Loop" error
// (see internal/services/container/runtime/nodes.go::execBreak/execContinue).
package control

import "errors"

var (
	errStopRun           = errors.New("graph stop requested")
	errBreakRequested    = errors.New("loop break requested")
	errContinueRequested = errors.New("loop continue requested")
)
