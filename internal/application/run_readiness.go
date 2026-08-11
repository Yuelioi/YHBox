package application

import (
	"errors"
	"strings"

	"github.com/yottaapp/yotta/internal/admission"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

type RunReadinessState string

const (
	RunReadinessStarted                RunReadinessState = "started"
	RunReadinessWorkflowInvalid        RunReadinessState = "workflow-invalid"
	RunReadinessTargetRequired         RunReadinessState = "target-required"
	RunReadinessCredentialRequired     RunReadinessState = "credential-required"
	RunReadinessEnvironmentUnavailable RunReadinessState = "environment-unavailable"
	RunReadinessNotStarted             RunReadinessState = "not-started"
	RunReadinessFailed                 RunReadinessState = "failed"
)

// RunReadiness is the application-level interpretation of a StartRun result.
// Every caller uses this classifier so a compiler diagnostic or admission
// requirement cannot be mistaken for a successfully queued Run.
type RunReadiness struct {
	State         RunReadinessState
	Code          string
	GraphID       string
	NodeID        string
	FromNodeID    string
	FromPortID    string
	ToNodeID      string
	ToPortID      string
	RequirementID string
	Slot          string
}

func ClassifyRunStart(result StartRunResult, startErr error) RunReadiness {
	if result.Record.Valid() {
		return RunReadiness{State: RunReadinessStarted}
	}
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Severity == schema.SeverityError {
			return RunReadiness{
				State:      RunReadinessWorkflowInvalid,
				Code:       diagnostic.Code,
				GraphID:    strings.Join(diagnostic.GraphPath, "/"),
				NodeID:     diagnostic.NodeID,
				FromNodeID: diagnosticStringParam(diagnostic, "fromNodeId"),
				FromPortID: diagnosticStringParam(diagnostic, "fromPortId"),
				ToNodeID:   diagnosticStringParam(diagnostic, "toNodeId"),
				ToPortID:   diagnosticStringParam(diagnostic, "toPortId"),
			}
		}
	}
	var admissionErr *admission.Error
	if errors.As(startErr, &admissionErr) {
		state := RunReadinessFailed
		switch admissionErr.Code {
		case admission.CodeTargetUnavailable, admission.CodeTargetAmbiguous:
			state = RunReadinessTargetRequired
		case admission.CodeCredentialUnavailable, admission.CodeCredentialAmbiguous:
			state = RunReadinessCredentialRequired
		case admission.CodeProviderIncompatible, admission.CodeUnsupportedHost:
			state = RunReadinessEnvironmentUnavailable
		}
		return RunReadiness{
			State: state, Code: admissionErr.Code, GraphID: admissionErr.GraphID,
			NodeID: admissionErr.NodeID, RequirementID: admissionErr.RequirementID, Slot: admissionErr.Slot,
		}
	}
	if startErr != nil {
		return RunReadiness{State: RunReadinessFailed}
	}
	return RunReadiness{State: RunReadinessNotStarted}
}

func diagnosticStringParam(diagnostic schema.Diagnostic, key string) string {
	value, _ := diagnostic.Params[key].(string)
	return value
}

func (r RunReadiness) UserFixable() bool {
	switch r.State {
	case RunReadinessWorkflowInvalid, RunReadinessTargetRequired, RunReadinessCredentialRequired,
		RunReadinessEnvironmentUnavailable, RunReadinessNotStarted:
		return true
	default:
		return false
	}
}
