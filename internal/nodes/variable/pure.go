// internal/nodes/variable/pure.go
// errPureDataNotEvaluatable returned by Run() of pure-data nodes (Phase 4 stub).
// Phase 5 adds pull-based eval; these nodes' Run is never called directly by graph runner.
package variable

import "errors"

var errPureDataNotEvaluatable = errors.New("pure-data node — evaluated by pull mechanism (Phase 5+)")
