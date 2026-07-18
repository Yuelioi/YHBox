package compiler

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

const (
	MaxDebugQueueEntries = 128
	MaxDebugValueEntries = 256
)

type DebugStatus string

const (
	DebugRunning      DebugStatus = "running"
	DebugPausePending DebugStatus = "pause-pending"
	DebugPaused       DebugStatus = "paused"
	DebugCompleted    DebugStatus = "completed"
)

type DebugBreakpoint struct {
	GraphPath []string `json:"graphPath,omitempty"`
	GraphID   string   `json:"graphId"`
	NodeID    string   `json:"nodeId"`
}

type DebugQueueEntry struct {
	GraphPath []string `json:"graphPath"`
	GraphID   string   `json:"graphId"`
	NodeID    string   `json:"nodeId"`
}

type DebugBlobView struct {
	Digest    artifact.Digest `json:"digest"`
	MediaType string          `json:"mediaType"`
	Size      int64           `json:"size"`
}

// DebugValueView deliberately exposes only immutable type and carrier metadata.
// Inline values, resource handles, credentials, and other payload bytes never
// cross the debugger seam.
type DebugValueView struct {
	Type           datatype.ResolvedType       `json:"type"`
	Representation datatype.RepresentationKind `json:"representation"`
	Digest         artifact.Digest             `json:"digest,omitempty"`
	Size           int                         `json:"size"`
	Blob           *DebugBlobView              `json:"blob,omitempty"`
	Redacted       bool                        `json:"redacted"`
}

type DebugStateView struct {
	Value     DebugValueView `json:"value"`
	Revision  int64          `json:"revision"`
	ChangedAt time.Time      `json:"changedAt"`
}

type DebugSnapshot struct {
	Status            DebugStatus                          `json:"status"`
	RunStatus         string                               `json:"runStatus,omitempty"`
	Generation        uint64                               `json:"generation"`
	GraphPath         []string                             `json:"graphPath,omitempty"`
	GraphID           string                               `json:"graphId,omitempty"`
	NodeID            string                               `json:"nodeId,omitempty"`
	PreviousGraphPath []string                             `json:"previousGraphPath,omitempty"`
	PreviousGraphID   string                               `json:"previousGraphId,omitempty"`
	PreviousNodeID    string                               `json:"previousNodeId,omitempty"`
	Attempt           int                                  `json:"attempt,omitempty"`
	Queue             []DebugQueueEntry                    `json:"queue"`
	Inputs            map[string]DebugValueView            `json:"inputs"`
	Outputs           map[string]map[string]DebugValueView `json:"outputs"`
	State             map[string]DebugStateView            `json:"state"`
	QueueTrimmed      bool                                 `json:"queueTrimmed,omitempty"`
	ValuesTrimmed     bool                                 `json:"valuesTrimmed,omitempty"`
}

type DebugControllerOptions struct {
	StartPaused bool
	Breakpoints []DebugBreakpoint
	OnChanged   func(DebugSnapshot)
}

type debugMode uint8

const (
	debugModeRunning debugMode = iota
	debugModePaused
	debugModeStepping
	debugModePausePending
	debugModeCompleted
)

// DebugController is the concurrency-safe control module used by the one
// scheduler. Its public interface is also the test surface for pause, step,
// continue, breakpoints, and bounded snapshots.
type DebugController struct {
	mu          sync.Mutex
	cond        *sync.Cond
	mode        debugMode
	breakpoints map[string]struct{}
	snapshot    DebugSnapshot
	onChanged   func(DebugSnapshot)
}

func NewDebugController(options DebugControllerOptions) (*DebugController, error) {
	controller := &DebugController{
		mode:        debugModeRunning,
		breakpoints: make(map[string]struct{}, len(options.Breakpoints)),
		onChanged:   options.OnChanged,
		snapshot: DebugSnapshot{
			Status: DebugRunning, Queue: []DebugQueueEntry{}, Inputs: map[string]DebugValueView{},
			Outputs: map[string]map[string]DebugValueView{}, State: map[string]DebugStateView{},
		},
	}
	controller.cond = sync.NewCond(&controller.mu)
	if options.StartPaused {
		controller.mode = debugModePaused
		controller.snapshot.Status = DebugPaused
	}
	if err := controller.replaceBreakpoints(options.Breakpoints); err != nil {
		return nil, err
	}
	return controller, nil
}

func (c *DebugController) Snapshot() DebugSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	return cloneDebugSnapshot(c.snapshot)
}

func (c *DebugController) Continue() error {
	return c.resume(debugModeRunning)
}

func (c *DebugController) Step() error {
	return c.resume(debugModeStepping)
}

func (c *DebugController) Pause() error {
	c.mu.Lock()
	if c.mode == debugModeCompleted {
		c.mu.Unlock()
		return errors.New("debug Run is complete")
	}
	if c.mode == debugModePaused {
		c.mu.Unlock()
		return nil
	}
	c.mode = debugModePausePending
	c.snapshot.Status = DebugPausePending
	snapshot, notify := c.changedLocked()
	c.mu.Unlock()
	if notify != nil {
		notify(snapshot)
	}
	return nil
}

func (c *DebugController) SetBreakpoints(values []DebugBreakpoint) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.replaceBreakpoints(values)
}

func (c *DebugController) Complete(runStatus string) {
	c.mu.Lock()
	if c.mode == debugModeCompleted {
		c.mu.Unlock()
		return
	}
	c.mode = debugModeCompleted
	c.snapshot.Status = DebugCompleted
	c.snapshot.RunStatus = runStatus
	snapshot, notify := c.changedLocked()
	c.cond.Broadcast()
	c.mu.Unlock()
	if notify != nil {
		notify(snapshot)
	}
}

func (c *DebugController) resume(mode debugMode) error {
	c.mu.Lock()
	if c.mode == debugModeCompleted {
		c.mu.Unlock()
		return errors.New("debug Run is complete")
	}
	if c.mode != debugModePaused {
		c.mu.Unlock()
		return errors.New("debug Run is not paused")
	}
	c.mode = mode
	c.snapshot.Status = DebugRunning
	snapshot, notify := c.changedLocked()
	c.cond.Broadcast()
	c.mu.Unlock()
	if notify != nil {
		notify(snapshot)
	}
	return nil
}

func (c *DebugController) checkpoint(ctx context.Context, snapshot DebugSnapshot) error {
	if ctx == nil {
		return errors.New("debug checkpoint requires context")
	}
	c.mu.Lock()
	snapshot.Status = DebugRunning
	snapshot.RunStatus = ""
	c.snapshot = cloneDebugSnapshot(snapshot)
	_, exactBreakpoint := c.breakpoints[debugBreakpointKey(DebugBreakpoint{GraphPath: snapshot.GraphPath, GraphID: snapshot.GraphID, NodeID: snapshot.NodeID})]
	_, graphBreakpoint := c.breakpoints[debugBreakpointKey(DebugBreakpoint{GraphID: snapshot.GraphID, NodeID: snapshot.NodeID})]
	breakpoint := exactBreakpoint || graphBreakpoint
	if c.mode == debugModeStepping || c.mode == debugModePausePending || breakpoint {
		c.mode = debugModePaused
	}
	if c.mode == debugModePaused {
		c.snapshot.Status = DebugPaused
	}
	changed, notify := c.changedLocked()
	c.mu.Unlock()
	if notify != nil {
		notify(changed)
	}

	stopWake := context.AfterFunc(ctx, func() {
		c.mu.Lock()
		c.cond.Broadcast()
		c.mu.Unlock()
	})
	defer stopWake()

	c.mu.Lock()
	defer c.mu.Unlock()
	for c.mode == debugModePaused && ctx.Err() == nil {
		c.cond.Wait()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if c.mode == debugModeCompleted {
		return errors.New("debug Run completed while awaiting checkpoint")
	}
	return nil
}

func (c *DebugController) replaceBreakpoints(values []DebugBreakpoint) error {
	if len(values) > MaxDebugQueueEntries {
		return errors.New("debug breakpoint budget exceeded")
	}
	next := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value.GraphID == "" || value.NodeID == "" {
			return errors.New("debug breakpoint requires graph and node")
		}
		if len(value.GraphPath) > schema.MaxGraphPath || len(value.GraphPath) > 0 && value.GraphPath[len(value.GraphPath)-1] != value.GraphID {
			return errors.New("debug breakpoint has an invalid graph path")
		}
		for _, segment := range value.GraphPath {
			if !programStateNamePattern.MatchString(segment) {
				return errors.New("debug breakpoint has an invalid graph path")
			}
		}
		next[debugBreakpointKey(value)] = struct{}{}
	}
	c.breakpoints = next
	return nil
}

func (c *DebugController) changedLocked() (DebugSnapshot, func(DebugSnapshot)) {
	c.snapshot.Generation++
	return cloneDebugSnapshot(c.snapshot), c.onChanged
}

func cloneDebugSnapshot(source DebugSnapshot) DebugSnapshot {
	clone := source
	clone.GraphPath = append([]string(nil), source.GraphPath...)
	clone.PreviousGraphPath = append([]string(nil), source.PreviousGraphPath...)
	clone.Queue = make([]DebugQueueEntry, len(source.Queue))
	for index, entry := range source.Queue {
		clone.Queue[index] = entry
		clone.Queue[index].GraphPath = append([]string(nil), entry.GraphPath...)
	}
	clone.Inputs = cloneDebugValues(source.Inputs)
	clone.Outputs = make(map[string]map[string]DebugValueView, len(source.Outputs))
	for nodeID, outputs := range source.Outputs {
		clone.Outputs[nodeID] = cloneDebugValues(outputs)
	}
	clone.State = make(map[string]DebugStateView, len(source.State))
	for name, state := range source.State {
		state.Value = cloneDebugValue(state.Value)
		clone.State[name] = state
	}
	return clone
}

func debugBreakpointKey(value DebugBreakpoint) string {
	return strings.Join(value.GraphPath, "\x00") + "\x01" + value.GraphID + "\x00" + value.NodeID
}

func cloneDebugValues(source map[string]DebugValueView) map[string]DebugValueView {
	clone := make(map[string]DebugValueView, len(source))
	for name, value := range source {
		clone[name] = cloneDebugValue(value)
	}
	return clone
}

func cloneDebugValue(source DebugValueView) DebugValueView {
	clone := source
	clone.Type = cloneResolvedType(source.Type)
	if source.Blob != nil {
		blob := *source.Blob
		clone.Blob = &blob
	}
	return clone
}

func cloneResolvedType(source datatype.ResolvedType) datatype.ResolvedType {
	clone := datatype.ResolvedType{Kind: source.Kind}
	if source.Ref != nil {
		ref := *source.Ref
		clone.Ref = &ref
	}
	if source.Element != nil {
		element := cloneResolvedType(*source.Element)
		clone.Element = &element
	}
	return clone
}
