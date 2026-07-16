package recording

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/automation/target"
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
		Meta:   inputclip.ClipMeta{},
	}, nil
}
func (*blockingStopRecorder) Cancel() {}

type countingClipSaver struct{ calls int }

func (s *countingClipSaver) Save(*inputclip.InputClip) error {
	s.calls++
	return nil
}
func (*countingClipSaver) List() []inputclip.ClipSummary { return nil }
func (*countingClipSaver) Delete(string) error           { return nil }

type resultRecorder struct {
	result    *StopResult
	cancelled bool
	hwnd      uintptr
	meta      inputclip.ClipMeta
	startErr  error
}

func (*resultRecorder) SetMouseCounts360Getter(func() int) {}
func (*resultRecorder) Active() bool                       { return true }
func (*resultRecorder) Pause()                             {}
func (*resultRecorder) Resume()                            {}
func (r *resultRecorder) Start(hwnd uintptr, meta inputclip.ClipMeta) (string, error) {
	r.hwnd, r.meta = hwnd, meta
	return "session", r.startErr
}
func (r *resultRecorder) Stop() (*StopResult, error) { return r.result, nil }
func (r *resultRecorder) Cancel()                    { r.cancelled = true }

type memoryClipStore struct {
	saved   *inputclip.InputClip
	clips   []inputclip.ClipSummary
	deleted []string
}

func (s *memoryClipStore) Save(clip *inputclip.InputClip) error { s.saved = clip; return nil }
func (s *memoryClipStore) List() []inputclip.ClipSummary        { return s.clips }
func (s *memoryClipStore) Delete(id string) error               { s.deleted = append(s.deleted, id); return nil }

type recordingTargetResolver struct {
	window      target.WindowHandle
	resolveErr  error
	activateErr error
	activated   string
}

func (r *recordingTargetResolver) ResolveWindow(context.Context, string) (target.WindowHandle, error) {
	return r.window, r.resolveErr
}

func (r *recordingTargetResolver) Activate(_ context.Context, slot string) error {
	r.activated = slot
	return r.activateErr
}

type recordingHotkeys struct{}

func (recordingHotkeys) GetStopHotkeyVK() uint32  { return 0x78 }
func (recordingHotkeys) GetPauseHotkeyVK() uint32 { return 0 }
func (recordingHotkeys) GetMouseMode() string     { return "absolute" }

func TestServiceStopCreatesPendingThenFinalizePersistsMetadata(t *testing.T) {
	recorder := &resultRecorder{result: &StopResult{
		TempID: "session", Meta: inputclip.ClipMeta{},
		Events: []inputclip.Event{{TUs: 0}, {TUs: 250_000}},
	}}
	clips := &memoryClipStore{}
	s := NewService(recorder, nil, clips, nil)
	s.setState(RecordingState{Phase: PhaseRecording, TargetSlot: "editor"})

	pending, err := s.Stop()
	if err != nil {
		t.Fatal(err)
	}
	if pending == nil || pending.PendingID != "pending-session" || pending.DurationUs != 250_000 {
		t.Fatalf("pending = %+v", pending)
	}
	if clips.saved != nil {
		t.Fatal("Stop persisted a clip before user confirmation")
	}
	result, err := s.Finalize(FinalizeArgs{
		PendingID: pending.PendingID, Label: "  Boss 战  ", Category: " 战斗 ", Tags: []string{"循环", "循环", " "},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ClipID != "clip-session" || clips.saved == nil {
		t.Fatalf("result=%+v saved=%+v", result, clips.saved)
	}
	if clips.saved.Label != "Boss 战" || clips.saved.Category != "战斗" || len(clips.saved.Tags) != 1 {
		t.Fatalf("saved metadata = %+v", clips.saved)
	}
}

func TestServiceCancelDiscardsActiveSession(t *testing.T) {
	recorder := &resultRecorder{}
	s := NewService(recorder, nil, &memoryClipStore{}, nil)
	s.setState(RecordingState{Phase: PhasePaused, TargetSlot: "editor"})
	if err := s.Cancel(); err != nil {
		t.Fatal(err)
	}
	if !recorder.cancelled || s.GetState().Phase != PhaseIdle {
		t.Fatalf("cancelled=%v state=%+v", recorder.cancelled, s.GetState())
	}
}

func TestServiceStartUsesResolvedWindowAndSettingsSnapshot(t *testing.T) {
	recorder := &resultRecorder{}
	targets := &recordingTargetResolver{window: target.WindowHandle{HWND: 42, ClientW: 1280, ClientH: 720}}
	service := NewService(recorder, recordingHotkeys{}, &memoryClipStore{}, targets)
	id, err := service.Start(StartArgs{TargetSlot: "editor"})
	if err != nil {
		t.Fatal(err)
	}
	if id != "session" || recorder.hwnd != 42 || recorder.meta.MouseMode != "absolute" || recorder.meta.StopHotkeyVK != 0x78 || recorder.meta.BaseResolution != [2]int{1280, 720} {
		t.Fatalf("id=%q hwnd=%d meta=%+v", id, recorder.hwnd, recorder.meta)
	}
	if state := service.GetState(); state.Phase != PhaseRecording || state.TargetSlot != "editor" || state.TempID != "session" {
		t.Fatalf("recording state = %+v", state)
	}
	if err := service.Cancel(); err != nil {
		t.Fatal(err)
	}
}

func TestServiceRejectsUntrustedOrUnusableStartTargets(t *testing.T) {
	for _, test := range []struct {
		name    string
		targets TargetResolver
		want    string
	}{
		{name: "missing resolver", targets: nil, want: "resolver"},
		{name: "resolution failure", targets: &recordingTargetResolver{resolveErr: errors.New("gone")}, want: "RECORDING_TARGET_UNAVAILABLE"},
		{name: "empty client", targets: &recordingTargetResolver{window: target.WindowHandle{HWND: 42}}, want: "0x0"},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := NewService(&resultRecorder{}, nil, nil, test.targets)
			_, err := service.Start(StartArgs{TargetSlot: "editor"})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Start() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestServiceValidateTargetActivatesExactResolvedSlot(t *testing.T) {
	targets := &recordingTargetResolver{window: target.WindowHandle{HWND: 42}}
	service := NewService(&resultRecorder{}, nil, nil, targets)
	if err := service.ValidateTarget("editor"); err != nil || targets.activated != "editor" {
		t.Fatalf("ValidateTarget() error=%v activated=%q", err, targets.activated)
	}
	targets.activateErr = errors.New("activation denied")
	if err := service.ValidateTarget("editor"); !errors.Is(err, targets.activateErr) {
		t.Fatalf("ValidateTarget() error = %v", err)
	}
}

func TestServiceFinalizeRejectsInvalidMetadataAndDiscardRemovesPending(t *testing.T) {
	service := NewService(&resultRecorder{}, nil, &memoryClipStore{}, nil)
	service.pending["pending"] = pendingRecording{result: &StopResult{TempID: "session"}}
	for _, args := range []FinalizeArgs{
		{PendingID: "pending", Label: " "},
		{PendingID: "pending", Label: strings.Repeat("界", 81)},
		{PendingID: "missing", Label: "Valid"},
	} {
		if _, err := service.Finalize(args); err == nil {
			t.Fatalf("Finalize(%+v) succeeded", args)
		}
	}
	if err := service.Discard("pending"); err != nil {
		t.Fatal(err)
	}
	if _, ok := service.pending["pending"]; ok {
		t.Fatal("Discard retained pending recording")
	}
}

func TestServiceShutdownCancelsWithoutPersistingAndRejectsStart(t *testing.T) {
	s, _ := newTestService()
	s.setState(RecordingState{Phase: PhaseRecording, TargetSlot: "editor", TempID: "partial"})

	if err := Shutdown(context.Background(), s); err != nil {
		t.Fatal(err)
	}
	if err := Shutdown(context.Background(), s); err != nil {
		t.Fatalf("second Shutdown() error = %v", err)
	}
	if state := s.GetState(); state.Phase != PhaseIdle {
		t.Fatalf("state after shutdown = %+v", state)
	}
	if _, err := s.Start(StartArgs{TargetSlot: "editor"}); err == nil ||
		!strings.Contains(err.Error(), "closed") {
		t.Fatalf("Start() after shutdown error = %v, want closed", err)
	}
}

func TestShutdownDuringFinalizingDiscardsRecorderResult(t *testing.T) {
	recorder := &blockingStopRecorder{started: make(chan struct{}), release: make(chan struct{})}
	saver := &countingClipSaver{}
	s := NewService(recorder, nil, saver, nil)
	s.setState(RecordingState{Phase: PhaseRecording, TargetSlot: "editor", TempID: "partial"})
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
	var mu sync.Mutex
	var events []string
	s := NewService(NewRecorder(), nil, nil, nil, func(name string, _ any) {
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
	s.setState(RecordingState{Phase: PhaseRecording, TargetSlot: "editor", TempID: "t1"})
	// Start 撞非 idle → 返当前 tempID, 不报错, 不重启.
	id, err := s.Start(StartArgs{TargetSlot: "other"})
	if err != nil {
		t.Fatalf("非 idle Start 应无错 (幂等 no-op), got %v", err)
	}
	if id != "t1" {
		t.Fatalf("非 idle Start 应返当前 tempID t1, got %q", id)
	}
	if s.GetState().TargetSlot != "editor" {
		t.Fatalf("non-idle Start changed target slot, got %q", s.GetState().TargetSlot)
	}
}

func TestService_GetStateAndEmit(t *testing.T) {
	s, events := newTestService()
	s.setState(RecordingState{Phase: PhaseRecording, TargetSlot: "editor", TempID: "tt"})
	st := s.GetState()
	if st.Phase != PhaseRecording || st.TargetSlot != "editor" || st.TempID != "tt" {
		t.Fatalf("GetState 没反映 setState: %+v", st)
	}
	if len(*events) != 1 || (*events)[0] != "recording:state" {
		t.Fatalf("setState 应 emit 一次 recording:state, got %v", *events)
	}
}

func TestService_PauseFromRecording(t *testing.T) {
	s, _ := newTestService()
	s.setState(RecordingState{Phase: PhaseRecording, TargetSlot: "editor", StartedAtMs: 1000})
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
	if st.TargetSlot != "editor" {
		t.Fatalf("Pause lost target slot, got %q", st.TargetSlot)
	}
}

func TestService_ResumeAccumulatesPausedMs(t *testing.T) {
	s, _ := newTestService()
	// 造已暂停 500ms 的态 (PausedMs 之前已累计 1000).
	s.setState(RecordingState{
		Phase: PhasePaused, TargetSlot: "editor",
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
	s.setState(RecordingState{Phase: PhaseRecording, TargetSlot: "editor"})
	if err := s.Resume(); err != nil {
		t.Fatalf("recording Resume 应无错, got %v", err)
	}
	if s.GetState().Phase != PhaseRecording {
		t.Fatalf("recording Resume 应 no-op, phase = %q", s.GetState().Phase)
	}
}

func TestService_ValidateTarget_Guards(t *testing.T) {
	s, _ := newTestService()
	// Empty target slot is rejected.
	if err := s.ValidateTarget(""); err == nil {
		t.Fatal("empty target slot should return an error")
	}
	// installed target resolver 未注入 (newTestService 不注入) → error, 不 panic.
	if err := s.ValidateTarget("cA"); err == nil {
		t.Fatal("missing installed target resolver should return an error")
	}
}

func TestService_StopFromPaused_GuardOpens(t *testing.T) {
	s, _ := newTestService()
	// paused 态 Stop 守卫放开 (不必先 resume). recorder 非 active + resolver 未注入 → 内部 err,
	// 但 defer 必收敛到 idle; 关键是没在守卫处提前 no-op 留在 paused.
	s.setState(RecordingState{Phase: PhasePaused, TargetSlot: "editor"})
	_, _ = s.Stop()
	if got := s.GetState().Phase; got != PhaseIdle {
		t.Fatalf("paused Stop 后 phase = %q, want idle (守卫应放行 paused → finalizing → idle)", got)
	}
}
