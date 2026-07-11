package recording

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/services/inputclip"
)

type blockingStopRecorder struct {
	started chan struct{}
	release chan struct{}
}

func (*blockingStopRecorder) SetMouseCounts360Getter(func() int) {}
func (*blockingStopRecorder) Active() bool                       { return true }
func (*blockingStopRecorder) Pause()                             {}
func (*blockingStopRecorder) Resume()                            {}
func (*blockingStopRecorder) Start(uintptr, inputclip.ClipMeta) (string, error) {
	return "", nil
}
func (r *blockingStopRecorder) Stop() (*StopResult, error) {
	close(r.started)
	<-r.release
	return &StopResult{
		TempID: "partial",
		Meta:   inputclip.ClipMeta{FilterMode: "precise"},
	}, nil
}
func (*blockingStopRecorder) Cancel() {}

type countingClipSaver struct{ calls int }

func (s *countingClipSaver) Save(*inputclip.InputClip) error {
	s.calls++
	return nil
}

func TestServiceShutdownCancelsWithoutPersistingAndRejectsStart(t *testing.T) {
	s, _ := newTestService()
	s.setState(RecordingState{Phase: PhaseRecording, ContainerID: "container", TempID: "partial"})

	if err := Shutdown(context.Background(), s); err != nil {
		t.Fatal(err)
	}
	if err := Shutdown(context.Background(), s); err != nil {
		t.Fatalf("second Shutdown() error = %v", err)
	}
	if state := s.GetState(); state.Phase != PhaseIdle {
		t.Fatalf("state after shutdown = %+v", state)
	}
	if _, err := s.Start(StartArgs{ContainerID: "container"}); err == nil ||
		!strings.Contains(err.Error(), "closed") {
		t.Fatalf("Start() after shutdown error = %v, want closed", err)
	}
}

func TestShutdownDuringFinalizingDiscardsRecorderResult(t *testing.T) {
	recorder := &blockingStopRecorder{started: make(chan struct{}), release: make(chan struct{})}
	saver := &countingClipSaver{}
	s := NewService(recorder, nil, saver)
	s.setState(RecordingState{Phase: PhaseRecording, ContainerID: "container", TempID: "partial"})
	stopResult := make(chan error, 1)
	go func() {
		_, err := s.Stop()
		stopResult <- err
	}()
	select {
	case <-recorder.started:
	case <-time.After(time.Second):
		t.Fatal("Stop() did not enter recorder drain")
	}
	shutdownResult := make(chan error, 1)
	go func() { shutdownResult <- Shutdown(context.Background(), s) }()
	deadline := time.Now().Add(time.Second)
	for !s.closed.Load() {
		if time.Now().After(deadline) {
			t.Fatal("Shutdown() did not publish cancellation intent")
		}
		time.Sleep(time.Millisecond)
	}
	close(recorder.release)
	for name, result := range map[string]<-chan error{
		"stop": stopResult, "shutdown": shutdownResult,
	} {
		select {
		case err := <-result:
			if err != nil {
				t.Fatalf("%s error = %v", name, err)
			}
		case <-time.After(time.Second):
			t.Fatalf("%s did not return", name)
		}
	}
	if saver.calls != 0 {
		t.Fatalf("clip saves during shutdown = %d", saver.calls)
	}
	if state := s.GetState(); state.Phase != PhaseIdle {
		t.Fatalf("final state = %+v", state)
	}
}

// 状态机 + 幂等性单测. 不碰真 OS hook — 只验 Service 的状态转换/幂等/事件广播逻辑.

func newTestService() (*Service, *[]string) {
	s := NewService(NewRecorder(), nil, nil)
	var mu sync.Mutex
	var events []string
	ConfigureEmitter(s, func(name string, _ any) {
		mu.Lock()
		events = append(events, name)
		mu.Unlock()
	})
	return s, &events
}

func TestService_InitialStateIdle(t *testing.T) {
	s, _ := newTestService()
	if got := s.GetState().Phase; got != PhaseIdle {
		t.Fatalf("初始 phase = %q, want %q", got, PhaseIdle)
	}
}

func TestService_StopWhenIdle_IsNoOpNoError(t *testing.T) {
	s, events := newTestService()
	// 幂等核心: idle 时 Stop → (nil, nil), 不报 ErrRecorderNotActive, 不碰 recorder.
	payload, err := s.Stop()
	if err != nil {
		t.Fatalf("idle Stop 应无错, got %v", err)
	}
	if payload != nil {
		t.Fatalf("idle Stop 应返 nil payload, got %+v", payload)
	}
	if s.GetState().Phase != PhaseIdle {
		t.Fatalf("idle Stop 后 phase 应仍 idle, got %q", s.GetState().Phase)
	}
	if len(*events) != 0 {
		t.Fatalf("idle Stop 不该 emit 状态事件, got %v", *events)
	}
}

func TestService_StartWhenNotIdle_IsNoOp(t *testing.T) {
	s, _ := newTestService()
	// 模拟已在录 (不走真 hook): 直接置 recording 态.
	s.setState(RecordingState{Phase: PhaseRecording, ContainerID: "cA", TempID: "t1"})
	// Start 撞非 idle → 返当前 tempID, 不报错, 不重启.
	id, err := s.Start(StartArgs{FilterMode: "precise", ContainerID: "cB"})
	if err != nil {
		t.Fatalf("非 idle Start 应无错 (幂等 no-op), got %v", err)
	}
	if id != "t1" {
		t.Fatalf("非 idle Start 应返当前 tempID t1, got %q", id)
	}
	if s.GetState().ContainerID != "cA" {
		t.Fatalf("非 idle Start 不该改容器, got %q", s.GetState().ContainerID)
	}
}

func TestService_GetStateAndEmit(t *testing.T) {
	s, events := newTestService()
	s.setState(RecordingState{Phase: PhaseRecording, ContainerID: "cX", FilterMode: "simple", TempID: "tt"})
	st := s.GetState()
	if st.Phase != PhaseRecording || st.ContainerID != "cX" || st.FilterMode != "simple" || st.TempID != "tt" {
		t.Fatalf("GetState 没反映 setState: %+v", st)
	}
	if len(*events) != 1 || (*events)[0] != "recording:state" {
		t.Fatalf("setState 应 emit 一次 recording:state, got %v", *events)
	}
}

func TestService_PauseFromRecording(t *testing.T) {
	s, _ := newTestService()
	s.setState(RecordingState{Phase: PhaseRecording, ContainerID: "cA", StartedAtMs: 1000})
	if err := s.Pause(); err != nil {
		t.Fatalf("Pause 应无错, got %v", err)
	}
	st := s.GetState()
	if st.Phase != PhasePaused {
		t.Fatalf("Pause 后 phase = %q, want paused", st.Phase)
	}
	if st.PausedAtMs <= 0 {
		t.Fatalf("Pause 后应记 PausedAtMs, got %d", st.PausedAtMs)
	}
	if st.ContainerID != "cA" {
		t.Fatalf("Pause 不该丢容器, got %q", st.ContainerID)
	}
}

func TestService_ResumeAccumulatesPausedMs(t *testing.T) {
	s, _ := newTestService()
	// 造已暂停 500ms 的态 (PausedMs 之前已累计 1000).
	s.setState(RecordingState{
		Phase: PhasePaused, ContainerID: "cA",
		PausedMs: 1000, PausedAtMs: time.Now().UnixMilli() - 500,
	})
	if err := s.Resume(); err != nil {
		t.Fatalf("Resume 应无错, got %v", err)
	}
	st := s.GetState()
	if st.Phase != PhaseRecording {
		t.Fatalf("Resume 后 phase = %q, want recording", st.Phase)
	}
	if st.PausedAtMs != 0 {
		t.Fatalf("Resume 后应清 PausedAtMs, got %d", st.PausedAtMs)
	}
	if st.PausedMs < 1400 || st.PausedMs > 1700 {
		t.Fatalf("PausedMs = %d, want ≈1500 (1000 + ~500ms 本次暂停)", st.PausedMs)
	}
}

func TestService_PauseResumeIdempotent(t *testing.T) {
	s, _ := newTestService()
	// idle 时 Pause → no-op (phase 不变).
	if err := s.Pause(); err != nil {
		t.Fatalf("idle Pause 应无错, got %v", err)
	}
	if s.GetState().Phase != PhaseIdle {
		t.Fatalf("idle Pause 应 no-op, phase = %q", s.GetState().Phase)
	}
	// recording 时 Resume → no-op.
	s.setState(RecordingState{Phase: PhaseRecording, ContainerID: "cA"})
	if err := s.Resume(); err != nil {
		t.Fatalf("recording Resume 应无错, got %v", err)
	}
	if s.GetState().Phase != PhaseRecording {
		t.Fatalf("recording Resume 应 no-op, phase = %q", s.GetState().Phase)
	}
}

func TestService_ValidateTarget_Guards(t *testing.T) {
	s, _ := newTestService()
	// 空 containerID → error.
	if err := s.ValidateTarget(""); err == nil {
		t.Fatal("空 containerID 应返 error")
	}
	// containerGet 未注入 (newTestService 不注入) → error, 不 panic.
	if err := s.ValidateTarget("cA"); err == nil {
		t.Fatal("ContainerGetter 未注入应返 error")
	}
}

func TestService_StopFromPaused_GuardOpens(t *testing.T) {
	s, _ := newTestService()
	// paused 态 Stop 守卫放开 (不必先 resume). recorder 非 active + containers nil → 内部 err,
	// 但 defer 必收敛到 idle; 关键是没在守卫处提前 no-op 留在 paused.
	s.setState(RecordingState{Phase: PhasePaused, ContainerID: "cA"})
	_, _ = s.Stop()
	if got := s.GetState().Phase; got != PhaseIdle {
		t.Fatalf("paused Stop 后 phase = %q, want idle (守卫应放行 paused → finalizing → idle)", got)
	}
}
