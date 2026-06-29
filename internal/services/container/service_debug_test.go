package container

import "testing"

type fakeDebugRunner struct {
	startID      string
	startOptions DebugStartOptions
	stepID       string
	continueID   string
	pauseID      string
	stopID       string
	stateID      string
}

func (f *fakeDebugRunner) RunOnce(string) error { return nil }
func (f *fakeDebugRunner) StopAll() error       { return nil }

func (f *fakeDebugRunner) DebugStart(id string, options DebugStartOptions) (DebugSessionState, error) {
	f.startID = id
	f.startOptions = options
	return DebugSessionState{SessionID: "s1", ContainerID: id, Status: DebugStatusPaused}, nil
}

func (f *fakeDebugRunner) DebugStep(sessionID string) (DebugSessionState, error) {
	f.stepID = sessionID
	return DebugSessionState{SessionID: sessionID, Status: DebugStatusStepping}, nil
}

func (f *fakeDebugRunner) DebugContinue(sessionID string) (DebugSessionState, error) {
	f.continueID = sessionID
	return DebugSessionState{SessionID: sessionID, Status: DebugStatusRunning}, nil
}

func (f *fakeDebugRunner) DebugPause(sessionID string) (DebugSessionState, error) {
	f.pauseID = sessionID
	return DebugSessionState{SessionID: sessionID, Status: DebugStatusPauseRequested}, nil
}

func (f *fakeDebugRunner) DebugStop(sessionID string) (DebugSessionState, error) {
	f.stopID = sessionID
	return DebugSessionState{SessionID: sessionID, Status: DebugStatusStopped}, nil
}

func (f *fakeDebugRunner) DebugState(sessionID string) (DebugSessionState, error) {
	f.stateID = sessionID
	return DebugSessionState{SessionID: sessionID, Status: DebugStatusPaused}, nil
}

func TestServiceDebugStartDelegates(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewStore(dir)
	svc := NewService(store)
	c, err := svc.Create("debug")
	if err != nil {
		t.Fatal(err)
	}
	runner := &fakeDebugRunner{}
	svc.SetRunner(runner)

	state, err := svc.DebugStart(c.ID, DebugStartOptions{StartNodeID: "n1", GraphPath: []string{"sg"}})
	if err != nil {
		t.Fatalf("DebugStart: %v", err)
	}
	if state.SessionID != "s1" || runner.startID != c.ID {
		t.Fatalf("state=%+v startID=%q", state, runner.startID)
	}
	if runner.startOptions.StartNodeID != "n1" || len(runner.startOptions.GraphPath) != 1 {
		t.Fatalf("options = %+v", runner.startOptions)
	}
}

func TestServiceDebugCommandsDelegateSessionID(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewStore(dir)
	svc := NewService(store)
	runner := &fakeDebugRunner{}
	svc.SetRunner(runner)

	if _, err := svc.DebugStep("s1"); err != nil {
		t.Fatalf("DebugStep: %v", err)
	}
	if _, err := svc.DebugContinue("s1"); err != nil {
		t.Fatalf("DebugContinue: %v", err)
	}
	if _, err := svc.DebugPause("s1"); err != nil {
		t.Fatalf("DebugPause: %v", err)
	}
	if _, err := svc.DebugStop("s1"); err != nil {
		t.Fatalf("DebugStop: %v", err)
	}
	if _, err := svc.DebugState("s1"); err != nil {
		t.Fatalf("DebugState: %v", err)
	}

	if runner.stepID != "s1" || runner.continueID != "s1" || runner.pauseID != "s1" || runner.stopID != "s1" || runner.stateID != "s1" {
		t.Fatalf("delegated ids: %+v", runner)
	}
}
