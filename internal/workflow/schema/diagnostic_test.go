package schema

import "testing"

func TestHasErrorsDoesNotTurnAuthoringWarningsIntoAdmissionFailures(t *testing.T) {
	if HasErrors([]Diagnostic{{Code: "UNREACHABLE_EXECUTION", Severity: SeverityWarning}}) {
		t.Fatal("warning blocked Program admission")
	}
	if !HasErrors([]Diagnostic{{Code: "INVALID_CONFIG", Severity: SeverityError}}) {
		t.Fatal("error did not block Program admission")
	}
}
