package schema

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

const (
	CodeUnsupportedWorkflowFormat = "UNSUPPORTED_WORKFLOW_FORMAT"
	CodeInvalidWorkflowJSON       = "INVALID_WORKFLOW_JSON"
	CodeDuplicateField            = "DUPLICATE_FIELD"
	CodeUnknownField              = "UNKNOWN_FIELD"
	CodeMissingRequiredField      = "MISSING_REQUIRED_FIELD"
	CodeInvalidField              = "INVALID_FIELD"
	CodeDuplicateID               = "DUPLICATE_ID"
	CodeMissingEntryGraph         = "MISSING_ENTRY_GRAPH"
)

type Diagnostic struct {
	Code      string         `json:"code"`
	Severity  Severity       `json:"severity" jsonschema:"required,enum=error,enum=warning,enum=info"`
	GraphPath []string       `json:"graphPath,omitempty"`
	NodeID    string         `json:"nodeId,omitempty"`
	FieldPath []string       `json:"fieldPath,omitempty"`
	Params    map[string]any `json:"params,omitempty"`
	Fix       *DiagnosticFix `json:"fix,omitempty"`
	Message   string         `json:"message,omitempty"`
}

type DiagnosticFix struct {
	Kind      string   `json:"kind" jsonschema:"required,enum=set_field,enum=remove_field"`
	GraphPath []string `json:"graphPath,omitempty"`
	NodeID    string   `json:"nodeId,omitempty"`
	FieldPath []string `json:"fieldPath"`
	Value     any      `json:"value,omitempty"`
}
