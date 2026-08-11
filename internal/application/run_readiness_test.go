package application

import (
	"errors"
	"testing"

	"github.com/yottaapp/yotta/internal/admission"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

func TestClassifyRunStartSeparatesBlockedRunsFromSystemFailures(t *testing.T) {
	invalid := ClassifyRunStart(StartRunResult{Diagnostics: []schema.Diagnostic{{
		Severity: schema.SeverityError, Code: "UNKNOWN_PORT", NodeID: "click",
		Params: map[string]any{
			"fromNodeId": "wait", "fromPortId": "found",
			"toNodeId": "click", "toPortId": "in",
		},
	}}}, nil)
	if invalid.State != RunReadinessWorkflowInvalid || invalid.Code != "UNKNOWN_PORT" || !invalid.UserFixable() ||
		invalid.FromNodeID != "wait" || invalid.FromPortID != "found" ||
		invalid.ToNodeID != "click" || invalid.ToPortID != "in" {
		t.Fatalf("invalid readiness = %#v", invalid)
	}

	target := ClassifyRunStart(StartRunResult{}, &admission.Error{
		Code: admission.CodeTargetUnavailable, Slot: "game-window",
	})
	if target.State != RunReadinessTargetRequired || target.Slot != "game-window" || !target.UserFixable() {
		t.Fatalf("target readiness = %#v", target)
	}

	persistence := ClassifyRunStart(StartRunResult{}, &admission.Error{Code: admission.CodePersistenceFailed})
	if persistence.State != RunReadinessFailed || persistence.UserFixable() {
		t.Fatalf("persistence readiness = %#v", persistence)
	}

	failed := ClassifyRunStart(StartRunResult{}, errors.New("disk unavailable"))
	if failed.State != RunReadinessFailed || failed.UserFixable() {
		t.Fatalf("system readiness = %#v", failed)
	}
}
