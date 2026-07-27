// Package nodeadapter defines the host ABI implemented by built-in and plugin
// node adapters. It contains no compiler, scheduler, or Program ownership.
package nodeadapter

import (
	"context"
	"time"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecatalog"
	run "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

type Adapter func(context.Context, Invocation) (AdapterResult, error)

type AdapterResult struct {
	Outputs     map[string]datatype.ValueEnvelope
	ExecOutputs []string
}

type NodeFailure struct {
	Code   string
	Output string
	Cause  error
}

func (failure *NodeFailure) Error() string {
	if failure == nil {
		return "node failure"
	}
	if failure.Cause != nil {
		return failure.Cause.Error()
	}
	return failure.Code
}

func (failure *NodeFailure) Unwrap() error {
	if failure == nil {
		return nil
	}
	return failure.Cause
}

type RoutedFailure struct {
	Code         string
	Category     string
	RetryHint    bool
	SourceNodeID string
	SourcePortID string
	Attempt      int
}

type SignalTrigger struct {
	Channel   schema.EdgeChannel
	InputPort string
	From      schema.Endpoint
	Failure   *RoutedFailure
}

type InstalledAdapter struct {
	Implementation nodecatalog.ImplementationLock
	Run            Adapter
}

type StateSnapshot struct {
	Value     datatype.ValueEnvelope
	Revision  int64
	ChangedAt time.Time
}

// StateBinding is invocation-scoped attenuated authority. Implementations are
// created by the Executor from the Program's immutable state access contract.
type StateBinding interface {
	Read() (StateSnapshot, error)
	Write(datatype.ValueEnvelope) (StateSnapshot, error)
	Update(func(datatype.ValueEnvelope) (datatype.ValueEnvelope, error)) (StateSnapshot, error)
}

type Invocation struct {
	InvocationID string
	Attempt      int
	GraphID      string
	NodeID       string
	Config       map[string]any
	Inputs       map[string]datatype.ValueEnvelope
	InputTypes   map[string]datatype.ResolvedType
	OutputTypes  map[string]datatype.ResolvedType
	Sessions     map[string]*run.Session
	State        map[string]StateBinding
	Trigger      *SignalTrigger
	ObservedAt   time.Time
	MonotonicNow func() time.Time
	ReadEntropy  func([]byte) error
	Wait         func(context.Context, time.Duration) error
	Spawn        func(func(context.Context) error) error
	RecordAction func(context.Context, AdapterAction) error
	EmitStatus   func(context.Context, string, map[string]int64) error
}

type AdapterAction struct {
	EffectID    string
	Action      string
	Outcome     run.ActionOutcome
	ErrorCode   string
	SummaryCode string
	Counters    map[string]int64
	Facts       map[string]string
}
