package application

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/yottaapp/yotta/internal/capability"
	run "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

type runJob struct {
	workflowID string
	cancel     context.CancelFunc
	providers  map[string]run.InstalledProvider
	release    func()
}

// The private methods below own the single production worker and every
// in-process Run lifetime. runMu keeps worker state separate from the
// Application command and lifecycle locks.
func (a *Application) enqueue(runID, workflowID string, providers map[string]run.InstalledProvider, release func(), control *compiler.DebugController) {
	a.runMu.Lock()
	defer a.runMu.Unlock()
	a.jobs[runID] = &runJob{
		workflowID: workflowID,
		providers:  providers, release: release,
	}
	if control != nil {
		a.debug[runID] = control
	}
	a.queue = append(a.queue, runID)
	select {
	case a.wake <- struct{}{}:
	default:
	}
}

func (a *Application) validateDebugStart(breakpoints []compiler.DebugBreakpoint) error {
	a.runMu.Lock()
	defer a.runMu.Unlock()
	for runID, control := range a.debug {
		if control.Snapshot().Status == compiler.DebugCompleted {
			delete(a.debug, runID)
		}
	}
	if len(a.debug) >= MaxDebugSessions {
		return errors.New("debug session budget exceeded")
	}
	if len(breakpoints) > compiler.MaxDebugQueueEntries {
		return errors.New("debug breakpoint budget exceeded")
	}
	for _, breakpoint := range breakpoints {
		if breakpoint.GraphID == "" || breakpoint.NodeID == "" {
			return errors.New("debug breakpoint requires graph and node")
		}
	}
	return nil
}

func (a *Application) GetDebugSnapshot(runID string) (compiler.DebugSnapshot, error) {
	a.runMu.Lock()
	control := a.debug[runID]
	a.runMu.Unlock()
	if control == nil {
		return compiler.DebugSnapshot{}, errors.New("debug session does not exist")
	}
	return control.Snapshot(), nil
}

func (a *Application) ControlDebugRun(ctx context.Context, runID string, action DebugAction) (compiler.DebugSnapshot, error) {
	if ctx == nil {
		return compiler.DebugSnapshot{}, errors.New("debug control context is required")
	}
	a.runMu.Lock()
	control := a.debug[runID]
	a.runMu.Unlock()
	if control == nil {
		return compiler.DebugSnapshot{}, errors.New("debug session does not exist")
	}
	var err error
	switch action {
	case DebugContinue:
		err = control.Continue()
	case DebugPause:
		err = control.Pause()
	case DebugStep:
		err = control.Step()
	default:
		err = errors.New("unknown debug action")
	}
	return control.Snapshot(), err
}

func (a *Application) SetDebugBreakpoints(ctx context.Context, runID string, breakpoints []compiler.DebugBreakpoint) (compiler.DebugSnapshot, error) {
	if ctx == nil {
		return compiler.DebugSnapshot{}, errors.New("set debug breakpoints context is required")
	}
	a.runMu.Lock()
	control := a.debug[runID]
	a.runMu.Unlock()
	if control == nil {
		return compiler.DebugSnapshot{}, errors.New("debug session does not exist")
	}
	if err := control.SetBreakpoints(breakpoints); err != nil {
		return compiler.DebugSnapshot{}, err
	}
	return control.Snapshot(), nil
}

// ActiveSourceRuns returns the queued/running run IDs currently retaining one
// Workflow Source. Historical Run records are intentionally not references:
// they retain immutable Program and journal facts after Source deletion.
func (a *Application) ActiveSourceRuns(workflowID string) []string {
	a.runMu.Lock()
	defer a.runMu.Unlock()
	runIDs := make([]string, 0)
	for runID, job := range a.jobs {
		if job.workflowID == workflowID {
			runIDs = append(runIDs, runID)
		}
	}
	sort.Strings(runIDs)
	return runIDs
}

func (a *Application) CancelRun(ctx context.Context, runID string) (run.Record, error) {
	if ctx == nil {
		return run.Record{}, errors.New("cancel Run context is required")
	}
	a.runMu.Lock()
	job := a.jobs[runID]
	if job != nil && job.cancel != nil {
		job.cancel()
		a.runMu.Unlock()
		return a.runs.Load(runID)
	}
	var releaseProviders func()
	if job != nil {
		releaseProviders = job.release
		delete(a.jobs, runID)
		a.removeQueuedLocked(runID)
	}
	a.runMu.Unlock()
	if releaseProviders != nil {
		releaseProviders()
	}
	current, err := a.runs.Load(runID)
	if err != nil || current.Status() != run.StatusQueued {
		return current, err
	}
	next, err := current.Cancel(a.transitionTime(current.Admission().QueuedAt))
	if err != nil {
		return run.Record{}, err
	}
	if err := a.runs.Update(ctx, current.Digest(), next); err != nil {
		return run.Record{}, err
	}
	a.runMu.Lock()
	control := a.debug[runID]
	a.runMu.Unlock()
	if control != nil {
		control.Complete(string(next.Status()))
	}
	a.emit(next, nil)
	return next, nil
}

func (a *Application) CancelAll(ctx context.Context) error {
	if ctx == nil {
		return errors.New("cancel all Runs context is required")
	}
	a.commandMu.RLock()
	defer a.commandMu.RUnlock()
	if err := a.requireRunning(); err != nil {
		return err
	}
	a.runMu.Lock()
	runIDs := make([]string, 0, len(a.jobs))
	for runID := range a.jobs {
		runIDs = append(runIDs, runID)
	}
	a.runMu.Unlock()
	var cancelErr error
	for _, runID := range runIDs {
		if _, err := a.CancelRun(ctx, runID); err != nil && !errors.Is(err, run.ErrRunConflict) {
			cancelErr = errors.Join(cancelErr, err)
		}
	}
	return cancelErr
}

func (a *Application) close(ctx context.Context) error {
	a.runMu.Lock()
	if a.cancel == nil {
		a.runMu.Unlock()
		return nil
	}
	queued := append([]string(nil), a.queue...)
	a.queue = nil
	for _, job := range a.jobs {
		if job.cancel != nil {
			job.cancel()
		}
	}
	a.cancel()
	a.runMu.Unlock()
	var closeErr error
	for _, runID := range queued {
		if _, err := a.CancelRun(ctx, runID); err != nil && !errors.Is(err, run.ErrRunConflict) {
			closeErr = errors.Join(closeErr, err)
		}
	}
	done := make(chan struct{})
	go func() { a.worker.Wait(); close(done) }()
	select {
	case <-done:
		return closeErr
	case <-ctx.Done():
		return errors.Join(closeErr, ctx.Err())
	}
}

func (a *Application) workerLoop() {
	defer a.worker.Done()
	for {
		runID, ctx, ok := a.nextJob()
		if !ok {
			return
		}
		err := a.execute(ctx, runID)
		a.runMu.Lock()
		job := a.jobs[runID]
		delete(a.jobs, runID)
		a.runMu.Unlock()
		if job != nil && job.release != nil {
			job.release()
		}
		record, loadErr := a.runs.Load(runID)
		if loadErr == nil {
			a.emit(record, err)
		} else if a.onRunEvent != nil {
			a.onRunEvent(RunEvent{RunID: runID, Err: errors.Join(err, loadErr)})
		}
	}
}

func (a *Application) nextJob() (string, context.Context, bool) {
	for {
		a.runMu.Lock()
		if len(a.queue) != 0 {
			runID := a.queue[0]
			a.queue = a.queue[1:]
			job := a.jobs[runID]
			if job == nil {
				a.runMu.Unlock()
				continue
			}
			jobCtx, cancel := context.WithCancel(a.ctx)
			job.cancel = cancel
			a.runMu.Unlock()
			return runID, jobCtx, true
		}
		ctx := a.ctx
		a.runMu.Unlock()
		select {
		case <-a.wake:
		case <-ctx.Done():
			return "", nil, false
		}
	}
}

func (a *Application) execute(ctx context.Context, runID string) error {
	a.runMu.Lock()
	job := a.jobs[runID]
	a.runMu.Unlock()
	if job == nil || job.providers == nil {
		return errors.New("run has no leased execution environment")
	}
	record, err := a.runs.Load(runID)
	if err != nil || record.Status() != run.StatusQueued {
		return errors.Join(err, errors.New("worker requires queued Run"))
	}
	program, bootstrapErr := a.programs.Load(record.Admission().ProgramHash)
	var grant capability.RunGrant
	if bootstrapErr == nil {
		grant, bootstrapErr = capability.OpenRunGrant(record.GrantArtifact(), program.CapabilityPlan(), a.catalog)
	}
	if ctx.Err() != nil {
		next, cancelErr := record.Cancel(a.transitionTime(record.Admission().QueuedAt))
		if cancelErr == nil {
			cancelErr = a.runs.Update(context.WithoutCancel(ctx), record.Digest(), next)
		}
		return errors.Join(ctx.Err(), cancelErr)
	}
	running, err := record.Start(a.transitionTime(record.Admission().QueuedAt))
	if err != nil {
		return err
	}
	if err := a.runs.Update(context.WithoutCancel(ctx), record.Digest(), running); err != nil {
		return err
	}
	journal, err := a.runs.OpenJournal(runID)
	if err != nil {
		return err
	}
	if ctx.Err() != nil {
		_, terminalErr := journal.Cancel(context.WithoutCancel(ctx), a.transitionTime(running.Admission().QueuedAt))
		return errors.Join(ctx.Err(), terminalErr)
	}
	if bootstrapErr != nil {
		_, terminalErr := journal.Fail(context.WithoutCancel(ctx), a.transitionTime(running.Admission().QueuedAt), run.RunError{
			Code: "runtime.bootstrap_failed", Category: run.ErrorCategoryInfrastructure,
		})
		return errors.Join(bootstrapErr, terminalErr)
	}
	owner, err := run.NewOwner(ctx, grant, job.providers, a.resourceOptions)
	if err != nil {
		if ctx.Err() != nil {
			_, terminalErr := journal.Cancel(context.WithoutCancel(ctx), a.transitionTime(running.Admission().QueuedAt))
			return errors.Join(ctx.Err(), err, terminalErr)
		}
		_, terminalErr := journal.Fail(context.WithoutCancel(ctx), a.transitionTime(running.Admission().QueuedAt), run.RunError{
			Code: "runtime.owner_failed", Category: run.ErrorCategoryInfrastructure,
		})
		return errors.Join(err, terminalErr)
	}
	a.runMu.Lock()
	control := a.debug[runID]
	a.runMu.Unlock()
	var executionErr error
	if control == nil {
		_, executionErr = a.executor.Run(ctx, program, owner, journal)
	} else {
		_, executionErr = a.executor.RunDebug(ctx, program, owner, journal, control)
	}
	closeCtx, cancel := context.WithTimeout(context.Background(), a.ownerCloseTimeout)
	closeErr := owner.Close(closeCtx)
	cancel()
	if control != nil {
		status := "UNKNOWN"
		if current := journal.Current(); current.Valid() {
			status = string(current.Status())
		}
		control.Complete(status)
	}
	return errors.Join(executionErr, closeErr)
}

func (a *Application) transitionTime(notBefore time.Time) time.Time {
	now := a.now().UTC()
	if now.Before(notBefore) {
		return notBefore
	}
	return now
}

func (a *Application) removeQueuedLocked(runID string) {
	for index, queuedID := range a.queue {
		if queuedID == runID {
			a.queue = append(a.queue[:index], a.queue[index+1:]...)
			return
		}
	}
}

func (a *Application) emit(record run.Record, err error) {
	if a.onRunEvent == nil || !record.Valid() {
		return
	}
	a.onRunEvent(RunEvent{
		RunID: record.Admission().RunID, Status: record.Status(), Generation: record.Generation(), Digest: record.Digest(), Err: err,
	})
}
