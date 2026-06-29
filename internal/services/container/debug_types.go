package container

const (
	DebugStatusPaused         = "paused"
	DebugStatusStepping       = "stepping"
	DebugStatusRunning        = "running"
	DebugStatusPauseRequested = "pause_requested"
	DebugStatusFinished       = "finished"
	DebugStatusFailed         = "failed"
	DebugStatusStopped        = "stopped"

	DebugModeEntry    = "entry"
	DebugModeFromNode = "from_node"
)

type DebugStartOptions struct {
	StartNodeID string   `json:"startNodeId,omitempty"`
	GraphPath   []string `json:"graphPath,omitempty"`
}

type DebugSessionState struct {
	SessionID       string              `json:"sessionId"`
	ContainerID     string              `json:"containerId"`
	Status          string              `json:"status"`
	Mode            string              `json:"mode"`
	StartNodeID     string              `json:"startNodeId,omitempty"`
	CurrentNodeID   string              `json:"currentNodeId,omitempty"`
	CurrentNodeKind string              `json:"currentNodeKind,omitempty"`
	RunningNodeID   string              `json:"runningNodeId,omitempty"`
	RunningNodeKind string              `json:"runningNodeKind,omitempty"`
	LastNodeID      string              `json:"lastNodeId,omitempty"`
	LastNodeKind    string              `json:"lastNodeKind,omitempty"`
	LastExit        string              `json:"lastExit,omitempty"`
	LastOutput      map[string]any      `json:"lastOutput,omitempty"`
	Vars            map[string]any      `json:"vars,omitempty"`
	Queue           []DebugTokenSummary `json:"queue,omitempty"`
	Error           *DebugRunError      `json:"error,omitempty"`
	Warnings        []DebugWarning      `json:"warnings,omitempty"`
}

type DebugTokenSummary struct {
	NodeID       string   `json:"nodeId"`
	NodeKind     string   `json:"nodeKind"`
	InPin        string   `json:"inPin"`
	GraphPath    []string `json:"graphPath,omitempty"`
	LoopDepth    int      `json:"loopDepth,omitempty"`
	ExecDataKeys []string `json:"execDataKeys,omitempty"`
}

type DebugWarning struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	NodeID  string         `json:"nodeId,omitempty"`
	Params  map[string]any `json:"params,omitempty"`
}

type DebugRunError struct {
	Message string            `json:"message,omitempty"`
	Errors  []ValidationError `json:"errors,omitempty"`
	Code    string            `json:"code,omitempty"`
	Params  map[string]any    `json:"params,omitempty"`
}
