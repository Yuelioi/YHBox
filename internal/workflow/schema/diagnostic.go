package schema

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

const (
	CodeUnsupportedWorkflowFormat    = "UNSUPPORTED_WORKFLOW_FORMAT"
	CodeInvalidWorkflowJSON          = "INVALID_WORKFLOW_JSON"
	CodeDuplicateField               = "DUPLICATE_FIELD"
	CodeUnknownField                 = "UNKNOWN_FIELD"
	CodeMissingRequiredField         = "MISSING_REQUIRED_FIELD"
	CodeInvalidField                 = "INVALID_FIELD"
	CodeDuplicateID                  = "DUPLICATE_ID"
	CodeMissingEntryGraph            = "MISSING_ENTRY_GRAPH"
	CodeUnknownNodeKind              = "UNKNOWN_NODE_KIND"
	CodeUnsupportedNodeContract      = "UNSUPPORTED_NODE_CONTRACT"
	CodeUnsupportedGraphContract     = "UNSUPPORTED_GRAPH_CONTRACT"
	CodeInvalidGraphEntry            = "INVALID_GRAPH_ENTRY"
	CodeMissingGraphOutput           = "MISSING_GRAPH_OUTPUT"
	CodeInvalidGraphBoundaryEdge     = "INVALID_GRAPH_BOUNDARY_EDGE"
	CodeUnknownCalleeGraph           = "UNKNOWN_CALLEE_GRAPH"
	CodeInvalidCalleeGraphKind       = "INVALID_CALLEE_GRAPH_KIND"
	CodeSubgraphCallCycle            = "SUBGRAPH_CALL_CYCLE"
	CodeCallPinTypeMismatch          = "CALL_PIN_TYPE_MISMATCH"
	CodeMissingCapabilityDeclaration = "MISSING_CAPABILITY_DECLARATION"
	CodeUnusedCapabilityDeclaration  = "UNUSED_CAPABILITY_DECLARATION"
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
