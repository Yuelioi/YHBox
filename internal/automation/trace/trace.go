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

type ActionSource struct {
	ContainerID string
	NodeID      string
	NodeKind    string
	InPin       string
}

type ActionRecord struct {
	Action          string
	Source          ActionSource
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
