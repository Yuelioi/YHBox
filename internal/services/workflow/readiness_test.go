package workflow

import (
	"errors"
	"testing"

	"github.com/yottaapp/yotta/internal/admission"
	appcore "github.com/yottaapp/yotta/internal/application"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

func TestRunReadinessViewSeparatesUserFixesFromSystemFailures(t *testing.T) {
	invalid := runReadinessView(appcore.StartRunResult{Diagnostics: []schema.Diagnostic{{
		Code: "INVALID_CONFIG", Severity: schema.SeverityError, GraphPath: []string{"main"}, NodeID: "click",
	}}}, nil)
	if invalid.State != "workflow-invalid" || invalid.Code != "INVALID_CONFIG" ||
		invalid.GraphID != "main" || invalid.NodeID != "click" {
		t.Fatalf("invalid readiness = %#v", invalid)
	}

	target := runReadinessView(appcore.StartRunResult{}, &admission.Error{
		Code: admission.CodeTargetUnavailable, Slot: "game-window", NodeID: "activate",
	})
	if target.State != "target-required" || target.Code != admission.CodeTargetUnavailable ||
		target.Slot != "game-window" || target.NodeID != "activate" {
		t.Fatalf("target readiness = %#v", target)
	}
	if !readinessErrorHandled(target) {
		t.Fatal("target readiness should return as a user-fixable result")
	}

	failed := runReadinessView(appcore.StartRunResult{}, errors.New("disk unavailable"))
	if failed.State != "failed" || readinessErrorHandled(failed) {
		t.Fatalf("system failure readiness = %#v", failed)
	}

	persistenceFailed := runReadinessView(appcore.StartRunResult{}, &admission.Error{
		Code: admission.CodePersistenceFailed,
	})
	if persistenceFailed.State != "failed" || readinessErrorHandled(persistenceFailed) {
		t.Fatalf("persistence failure readiness = %#v", persistenceFailed)
	}

	policyInvalid := runReadinessView(appcore.StartRunResult{}, &admission.Error{
		Code: admission.CodePolicyInvalid,
	})
	if policyInvalid.State != "failed" || readinessErrorHandled(policyInvalid) {
		t.Fatalf("invalid policy readiness = %#v", policyInvalid)
	}
}
