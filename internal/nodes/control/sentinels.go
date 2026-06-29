// internal/nodes/control/sentinels.go
// Sentinel errors for halt / loop control nodes.
//
// Stop/Break/Continue can't express their semantics through normal exec-out
// edges — they require runner cooperation. They are returned as plain errors
// from Run, and the runner intercepts them:
//   - errStopRun           → halt graph dispatch cleanly (no container:error emit)
//   - errBreakRequested    → Loop region pops frame + walks to .complete out
//   - errContinueRequested → Loop region rewinds to header for next iter
//
// Break/Continue used outside a Loop region just surface as node errors. That's
// acceptable.
package control

import "errors"

var (
	errStopRun           = errors.New("graph stop requested")
	errBreakRequested    = errors.New("loop break requested")
	errContinueRequested = errors.New("loop continue requested")
)

// IsStopRequested 报 err 是否是 Stop 节点的 sentinel — runtime ContainerRunner.Run 用这个
// 决定是否 graceful halt (no container:error emit).
func IsStopRequested(err error) bool { return errors.Is(err, errStopRun) }

// IsBreakRequested 报 err 是否是 Break 节点 sentinel — 给 region runner 检测.
func IsBreakRequested(err error) bool { return errors.Is(err, errBreakRequested) }

// IsContinueRequested 报 err 是否是 Continue 节点 sentinel.
func IsContinueRequested(err error) bool { return errors.Is(err, errContinueRequested) }
