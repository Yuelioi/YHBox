package trace

import (
	"time"

	"github.com/yottaapp/yotta/internal/automation/target"
)

type Status string

const (
	StatusSuccess Status = "success"
	StatusError   Status = "error"
)

type CoordinateStep struct {
	From   target.CoordinateSpace
	To     target.CoordinateSpace
	Input  any
	Output any
}

type ExecutionSource struct {
	GraphID      string
	NodeID       string
	InvocationID string
	Attempt      int
}

type ActionRecord struct {
	Action          string
	Source          ExecutionSource
	Target          target.Target
	Backend         string
	Request         any
	Result          any
	Status          Status
	Error           string
	CoordinateSteps []CoordinateStep
	StartedAt       time.Time
	EndedAt         time.Time
}

func (r ActionRecord) Duration() time.Duration {
	if r.StartedAt.IsZero() || r.EndedAt.IsZero() {
		return 0
	}
	return r.EndedAt.Sub(r.StartedAt)
}

type Recorder interface {
	Record(ActionRecord)
}
