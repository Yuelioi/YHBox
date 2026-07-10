package main

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/google/uuid"

	"github.com/yottaapp/yotta/internal/services/container"
	containerruntime "github.com/yottaapp/yotta/internal/services/container/runtime"
)

type debugRunnerFactory func(containerID string) (*containerruntime.ContainerRunner, error)

type containerDebugManager struct {
	mu         sync.Mutex
	newRunner  debugRunnerFactory
	emit       func(string, any)
	workerBusy func() bool
	session    *containerDebugSession
}

type containerDebugSession struct {
	id          string
	containerID string
	mode        string
	startNodeID string
	status      string
	runner      *containerruntime.ContainerRunner
	ctx         context.Context
	cancel      context.CancelFunc
	warnings    []container.DebugWarning

	runningNodeID   string
	runningNodeKind string
	lastNodeID      string
	lastNodeKind    string
	lastExit        string
	lastOutput      map[string]any
	err             *container.DebugRunError
	workerActive    bool
}

func newContainerDebugManager(newRunner debugRunnerFactory, emit func(string, any), workerBusy func() bool) *containerDebugManager {
	return &containerDebugManager{
		newRunner:  newRunner,
		emit:       emit,
		workerBusy: workerBusy,
	}
}

func (m *containerDebugManager) DebugStart(containerID string, options container.DebugStartOptions) (container.DebugSessionState, error) {
	if len(options.GraphPath) > 0 {
		return container.DebugSessionState{}, errors.New("debug_unsupported_graph_path")
	}
	if m.workerBusy != nil && m.workerBusy() {
		return container.DebugSessionState{}, errors.New("debug_run_busy")
	}
	m.mu.Lock()
	if m.session != nil && !debugTerminal(m.session.status) {
		m.mu.Unlock()
		return container.DebugSessionState{}, errors.New("debug_session_busy")
	}
	if m.session != nil && m.session.workerActive {
		m.mu.Unlock()
		return container.DebugSessionState{}, errors.New("debug_session_busy")
	}
	if m.session != nil && debugTerminal(m.session.status) {
		m.session.runner.StopRuntime()
		m.session = nil
	}
	m.mu.Unlock()

	r, err := m.newRunner(containerID)
	if err != nil {
		return container.DebugSessionState{}, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	if err := r.StartRuntime(ctx); err != nil {
		cancel()
		return container.DebugSessionState{}, err
	}
	mode := container.DebugModeEntry
	var warnings []container.DebugWarning
	if options.StartNodeID == "" {
		err = r.SeedFromEntry()
	} else {
		mode = container.DebugModeFromNode
		err = r.SeedFromNode(options.StartNodeID)
		warnings = append(warnings, container.DebugWarning{
			Code:    "debug_skips_upstream_context",
			Message: "Upstream nodes are skipped; existing variables and active target state may be used.",
			NodeID:  options.StartNodeID,
		})
	}
	if err != nil {
		cancel()
		r.StopRuntime()
		return container.DebugSessionState{}, err
	}

	s := &containerDebugSession{
		id:          uuid.NewString(),
		containerID: containerID,
		mode:        mode,
		startNodeID: options.StartNodeID,
		status:      container.DebugStatusPaused,
		runner:      r,
		ctx:         ctx,
		cancel:      cancel,
		warnings:    warnings,
	}

	m.mu.Lock()
	m.session = s
	state := m.stateLocked(s)
	m.mu.Unlock()
	m.emitState(state)
	return state, nil
}

func (m *containerDebugManager) DebugStep(sessionID string) (container.DebugSessionState, error) {
	m.mu.Lock()
	s, err := m.requireSessionLocked(sessionID)
	if err != nil {
		m.mu.Unlock()
		return container.DebugSessionState{}, err
	}
	if s.status != container.DebugStatusPaused {
		m.mu.Unlock()
		return container.DebugSessionState{}, errors.New("debug_session_busy")
	}
	m.markRunningLocked(s, container.DebugStatusStepping)
	s.workerActive = true
	state := m.stateLocked(s)
	m.mu.Unlock()
	m.emitState(state)

	go m.runStep(s.id, false)
	return state, nil
}

func (m *containerDebugManager) DebugContinue(sessionID string) (container.DebugSessionState, error) {
	m.mu.Lock()
	s, err := m.requireSessionLocked(sessionID)
	if err != nil {
		m.mu.Unlock()
		return container.DebugSessionState{}, err
	}
	if s.status != container.DebugStatusPaused {
		m.mu.Unlock()
		return container.DebugSessionState{}, errors.New("debug_session_busy")
	}
	m.markRunningLocked(s, container.DebugStatusRunning)
	s.workerActive = true
	state := m.stateLocked(s)
	m.mu.Unlock()
	m.emitState(state)

	go m.runStep(s.id, true)
	return state, nil
}

func (m *containerDebugManager) DebugPause(sessionID string) (container.DebugSessionState, error) {
	m.mu.Lock()
	s, err := m.requireSessionLocked(sessionID)
	if err != nil {
		m.mu.Unlock()
		return container.DebugSessionState{}, err
	}
	if s.status == container.DebugStatusRunning {
		s.status = container.DebugStatusPauseRequested
	}
	state := m.stateLocked(s)
	m.mu.Unlock()
	m.emitState(state)
	return state, nil
}

func (m *containerDebugManager) DebugStop(sessionID string) (container.DebugSessionState, error) {
	m.mu.Lock()
	s, err := m.requireSessionLocked(sessionID)
	if err != nil {
		m.mu.Unlock()
		return container.DebugSessionState{}, err
	}
	s.cancel()
	s.status = container.DebugStatusStopped
	s.runningNodeID = ""
	s.runningNodeKind = ""
	state := m.stateLocked(s)
	if !s.workerActive {
		s.runner.StopRuntime()
		m.session = nil
	}
	m.mu.Unlock()
	m.emitState(state)
	return state, nil
}

func (m *containerDebugManager) DebugState(sessionID string) (container.DebugSessionState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, err := m.requireSessionLocked(sessionID)
	if err != nil {
		return container.DebugSessionState{}, err
	}
	return m.stateLocked(s), nil
}

func (m *containerDebugManager) IsActive() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.session != nil && (!debugTerminal(m.session.status) || m.session.workerActive)
}

func (m *containerDebugManager) sessionID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.session == nil {
		return ""
	}
	return m.session.id
}

func (m *containerDebugManager) runStep(sessionID string, keepGoing bool) {
	for {
		m.mu.Lock()
		s, err := m.requireSessionLocked(sessionID)
		if err != nil {
			m.mu.Unlock()
			return
		}
		ctx := s.ctx
		runner := s.runner
		m.mu.Unlock()

		res, stepErr := runner.StepOnce(ctx)

		m.mu.Lock()
		s, err = m.requireSessionLocked(sessionID)
		if err != nil {
			m.mu.Unlock()
			return
		}
		s.runningNodeID = ""
		s.runningNodeKind = ""
		if s.status == container.DebugStatusStopped {
			s.workerActive = false
			s.runner.StopRuntime()
			state := m.stateLocked(s)
			m.session = nil
			m.mu.Unlock()
			m.emitState(state)
			return
		}
		s.lastNodeID = res.NodeID
		s.lastNodeKind = res.NodeKind
		s.lastExit = res.Exit
		s.lastOutput = res.Output
		if stepErr != nil {
			s.workerActive = false
			s.status = container.DebugStatusFailed
			s.err = &container.DebugRunError{Message: stepErr.Error()}
			s.runner.StopRuntime()
			state := m.stateLocked(s)
			m.mu.Unlock()
			m.emitState(state)
			return
		}
		if res.Finished {
			s.workerActive = false
			s.status = container.DebugStatusFinished
			s.runner.StopRuntime()
			state := m.stateLocked(s)
			m.mu.Unlock()
			m.emitState(state)
			return
		}
		if !keepGoing || s.status == container.DebugStatusPauseRequested {
			s.workerActive = false
			s.status = container.DebugStatusPaused
			state := m.stateLocked(s)
			m.mu.Unlock()
			m.emitState(state)
			return
		}
		m.markRunningLocked(s, container.DebugStatusRunning)
		state := m.stateLocked(s)
		m.mu.Unlock()
		m.emitState(state)
	}
}

func (m *containerDebugManager) markRunningLocked(s *containerDebugSession, status string) {
	s.status = status
	q := s.runner.QueueSnapshot()
	if len(q) == 0 {
		s.runningNodeID = ""
		s.runningNodeKind = ""
		return
	}
	s.runningNodeID = q[0].NodeID
	s.runningNodeKind = s.runner.NodeKind(q[0].NodeID)
}

func (m *containerDebugManager) requireSessionLocked(sessionID string) (*containerDebugSession, error) {
	if m.session == nil || m.session.id != sessionID {
		return nil, errors.New("debug_session_not_found")
	}
	return m.session, nil
}

func (m *containerDebugManager) stateLocked(s *containerDebugSession) container.DebugSessionState {
	q := []container.DebugTokenSummary(nil)
	if s.status == container.DebugStatusPaused || debugTerminal(s.status) {
		q = m.queueSummaryLocked(s)
	}
	state := container.DebugSessionState{
		SessionID:       s.id,
		ContainerID:     s.containerID,
		Status:          s.status,
		Mode:            s.mode,
		StartNodeID:     s.startNodeID,
		RunningNodeID:   s.runningNodeID,
		RunningNodeKind: s.runningNodeKind,
		LastNodeID:      s.lastNodeID,
		LastNodeKind:    s.lastNodeKind,
		LastExit:        s.lastExit,
		LastOutput:      copyDebugMap(s.lastOutput),
		Vars:            s.runner.VarSnapshot(),
		Queue:           q,
		Error:           s.err,
		Warnings:        append([]container.DebugWarning(nil), s.warnings...),
	}
	if len(q) > 0 {
		state.CurrentNodeID = q[0].NodeID
		state.CurrentNodeKind = q[0].NodeKind
	}
	return state
}

func (m *containerDebugManager) queueSummaryLocked(s *containerDebugSession) []container.DebugTokenSummary {
	tokens := s.runner.QueueSnapshot()
	out := make([]container.DebugTokenSummary, 0, len(tokens))
	for _, tok := range tokens {
		out = append(out, container.DebugTokenSummary{
			NodeID:       tok.NodeID,
			NodeKind:     s.runner.NodeKind(tok.NodeID),
			InPin:        tok.InPin,
			LoopDepth:    len(tok.LoopStack),
			ExecDataKeys: execDataKeys(tok.ExecData),
		})
	}
	return out
}

func (m *containerDebugManager) emitState(state container.DebugSessionState) {
	if m.emit != nil {
		m.emit("debug:state", state)
	}
}

func debugTerminal(status string) bool {
	switch status {
	case container.DebugStatusFinished, container.DebugStatusFailed, container.DebugStatusStopped:
		return true
	default:
		return false
	}
}

func execDataKeys(data map[string]any) []string {
	if len(data) == 0 {
		return nil
	}
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func copyDebugMap(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func errDebugUnavailable() error {
	return fmt.Errorf("debug manager not injected")
}
