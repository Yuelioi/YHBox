package recording

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/automation/target"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/services/asset"
	"github.com/yottaapp/yotta/internal/services/inputclip"
	"github.com/yottaapp/yotta/internal/services/macro"
	"github.com/yottaapp/yotta/internal/storage"
	"github.com/yottaapp/yotta/internal/storage/catalog"
)

type blockingStopRecorder struct {
	started chan struct{}
	release chan struct{}
}

func (*blockingStopRecorder) Active() bool { return true }
func (*blockingStopRecorder) Pause()       {}
func (*blockingStopRecorder) Resume()      {}
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

func (*resultRecorder) Active() bool { return true }
func (*resultRecorder) Pause()       {}
func (*resultRecorder) Resume()      {}
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

type memoryMacroStore struct {
	saved *macro.Macro
}

func (s *memoryMacroStore) Save(value *macro.Macro) (*macro.Macro, error) {
	s.saved = value
	value.Blob = blob.BlobRef{
		MediaType: macro.MediaType,
		Digest:    "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Size:      128,
	}
	return value, nil
}

type recordingTargetResolver struct {
	window     target.WindowHandle
	resolveErr error
	counts360  int
	released   int
}

func (r *recordingTargetResolver) AcquireRecordingTarget(context.Context, string) (target.WindowHandle, int, func(), error) {
	if r.resolveErr != nil {
		return target.WindowHandle{}, 0, nil, r.resolveErr
	}
	return r.window, r.counts360, func() { r.released++ }, nil
}

type recordingHotkeys struct{}

func (recordingHotkeys) GetStartHotkeyVK() uint32 { return 0x79 }
func (recordingHotkeys) GetStopHotkeyVK() uint32  { return 0x78 }
func (recordingHotkeys) GetPauseHotkeyVK() uint32 { return 0 }
func (recordingHotkeys) GetMouseMode() string     { return "absolute" }

type fakeStartHotkeyWatch struct {
	callback func()
	stopped  bool
}

func (*fakeStartHotkeyWatch) Start() error { return nil }
func (watch *fakeStartHotkeyWatch) Stop()  { watch.stopped = true }

func TestServiceStopCreatesPendingThenFinalizePersistsMetadata(t *testing.T) {
	recorder := &resultRecorder{result: &StopResult{
		TempID: "session", Meta: inputclip.ClipMeta{RecordingMode: inputclip.RecordingModeSimple, MouseMode: "absolute", BaseResolution: [2]int{1280, 720}},
		Events: []inputclip.Event{
			{TUs: 100, Type: inputclip.EventTypeKeyDown, A: 'A'},
			{TUs: 250_100, Type: inputclip.EventTypeKeyUp, A: 'A'},
		},
	}}
	macros := &memoryMacroStore{}
	s := NewService(recorder, nil, nil, macros, nil)
	s.setState(RecordingState{Phase: PhaseRecording, TargetSlot: "editor"})

	pending, err := s.Stop()
	if err != nil {
		t.Fatal(err)
	}
	if pending == nil || pending.PendingID != "pending-session" || pending.DurationUs != 250_000 {
		t.Fatalf("pending = %+v", pending)
	}
	if macros.saved != nil {
		t.Fatal("Stop persisted a macro before user confirmation")
	}
	result, err := s.Finalize(FinalizeArgs{
		PendingID: pending.PendingID, Label: "  Boss 战  ", Category: " 战斗 ", Tags: []string{"循环", "循环", " "},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.AssetID != "macro-session" || result.AssetKind != asset.KindMacro || macros.saved == nil {
		t.Fatalf("result=%+v saved=%+v", result, macros.saved)
	}
	if macros.saved.Label != "Boss 战" || macros.saved.Category != "战斗" || len(macros.saved.Tags) != 1 {
		t.Fatalf("saved metadata = %+v", macros.saved)
	}
	if len(macros.saved.Document.Actions) != 3 || macros.saved.Document.Actions[0].Kind != macro.ActionKeyDown || macros.saved.Document.Actions[1].Kind != macro.ActionSleep || macros.saved.Document.Actions[2].Kind != macro.ActionKeyUp {
		t.Fatalf("saved atomic macro = %+v", macros.saved.Document.Actions)
	}
}

func TestServiceStopAsyncEmitsCompletePreviewPayload(t *testing.T) {
	recorder := &resultRecorder{result: &StopResult{
		TempID: "session",
		Meta:   inputclip.ClipMeta{RecordingMode: inputclip.RecordingModeSimple, MouseMode: "absolute", BaseResolution: [2]int{1280, 720}},
		Events: []inputclip.Event{
			{TUs: 0, Type: inputclip.EventTypeKeyDown, A: 'A'},
			{TUs: 25_000, Type: inputclip.EventTypeKeyUp, A: 'A'},
		},
	}}
	completed := make(chan map[string]any, 1)
	service := NewService(recorder, nil, &memoryClipStore{}, nil, nil, func(name string, data any) {
		if name != "recording:completed" {
			return
		}
		payload, ok := data.(map[string]any)
		if ok {
			completed <- payload
		}
	})
	service.setState(RecordingState{Phase: PhaseRecording, TargetSlot: "editor"})
	service.StopAsync()

	select {
	case payload := <-completed:
		preview, ok := payload["preview"].(RecordingPreview)
		if !ok || preview.Mode != "simple" || preview.KeyActions != 2 || payload["mode"] != inputclip.RecordingModeSimple {
			t.Fatalf("completed payload = %#v", payload)
		}
	case <-time.After(time.Second):
		t.Fatal("StopAsync did not emit recording:completed")
	}
}

func TestServiceCancelDiscardsActiveSession(t *testing.T) {
	recorder := &resultRecorder{}
	s := NewService(recorder, nil, &memoryClipStore{}, nil, nil)
	s.setState(RecordingState{Phase: PhasePaused, TargetSlot: "editor"})
	if err := s.Cancel(); err != nil {
		t.Fatal(err)
	}
	if !recorder.cancelled || s.GetState().Phase != PhaseIdle {
		t.Fatalf("cancelled=%v state=%+v", recorder.cancelled, s.GetState())
	}
}

func TestServiceStartArmsThenCountsDownBeforeRecorderCapture(t *testing.T) {
	recorder := &resultRecorder{}
	targets := &recordingTargetResolver{window: target.WindowHandle{HWND: 42, ClientW: 1280, ClientH: 720}}
	service := NewService(recorder, recordingHotkeys{}, &memoryClipStore{}, nil, targets)
	var watch *fakeStartHotkeyWatch
	service.startHotkeyFactory = func(_ uint32, callback func()) startHotkeyWatch {
		watch = &fakeStartHotkeyWatch{callback: callback}
		return watch
	}
	id, err := service.Start(StartArgs{TargetSlot: "editor", Mode: inputclip.RecordingModeSimple})
	if err != nil {
		t.Fatal(err)
	}
	if id != "" || recorder.hwnd != 0 || watch == nil {
		t.Fatalf("armed id=%q hwnd=%d watch=%+v", id, recorder.hwnd, watch)
	}
	if state := service.GetState(); state.Phase != PhaseArmed || state.TargetSlot != "editor" || state.CountdownEndsAtMs != 0 {
		t.Fatalf("armed state = %+v", state)
	}
	watch.callback()
	if state := service.GetState(); state.Phase != PhaseCountdown || state.CountdownEndsAtMs <= time.Now().UnixMilli() {
		t.Fatalf("countdown state = %+v", state)
	}
	service.finishCountdown(service.countdownGeneration.Load())
	if recorder.hwnd != 42 || recorder.meta.RecordingMode != inputclip.RecordingModeSimple || recorder.meta.MouseMode != "absolute" || recorder.meta.StopHotkeyVK != 0x78 || recorder.meta.BaseResolution != [2]int{1280, 720} {
		t.Fatalf("recording hwnd=%d meta=%+v", recorder.hwnd, recorder.meta)
	}
	if state := service.GetState(); state.Phase != PhaseRecording || state.TargetSlot != "editor" || state.TempID != "session" || state.StartedAtMs <= 0 {
		t.Fatalf("recording state = %+v", state)
	}
	if err := service.Cancel(); err != nil {
		t.Fatal(err)
	}
	if targets.released != 1 {
		t.Fatalf("recording target releases = %d, want 1", targets.released)
	}
}

func TestServiceCancelDuringCountdownNeverStartsRecorder(t *testing.T) {
	recorder := &resultRecorder{}
	targets := &recordingTargetResolver{window: target.WindowHandle{HWND: 42, ClientW: 1280, ClientH: 720}}
	service := NewService(recorder, recordingHotkeys{}, &memoryClipStore{}, nil, targets)
	var watch *fakeStartHotkeyWatch
	service.startHotkeyFactory = func(_ uint32, callback func()) startHotkeyWatch {
		watch = &fakeStartHotkeyWatch{callback: callback}
		return watch
	}
	if _, err := service.Start(StartArgs{TargetSlot: "editor", Mode: inputclip.RecordingModePrecise}); err != nil {
		t.Fatal(err)
	}
	watch.callback()
	generation := service.countdownGeneration.Load()
	if err := service.Cancel(); err != nil {
		t.Fatal(err)
	}
	service.finishCountdown(generation)
	if recorder.hwnd != 0 || targets.released != 1 || service.GetState().Phase != PhaseIdle {
		t.Fatalf("hwnd=%d released=%d state=%+v", recorder.hwnd, targets.released, service.GetState())
	}
}

func TestServiceStopReleasesExactRecordingTarget(t *testing.T) {
	recorder := &resultRecorder{result: &StopResult{
		TempID: "session",
		Meta: inputclip.ClipMeta{
			RecordingMode: inputclip.RecordingModeSimple,
			MouseMode:     "absolute", BaseResolution: [2]int{1280, 720},
		},
		Events: []inputclip.Event{
			{TUs: 10, Type: inputclip.EventTypeKeyDown, A: 'A'},
			{TUs: 20, Seq: 1, Type: inputclip.EventTypeKeyUp, A: 'A'},
		},
	}}
	targets := &recordingTargetResolver{window: target.WindowHandle{HWND: 42, ClientW: 1280, ClientH: 720}}
	service := NewService(recorder, recordingHotkeys{}, &memoryClipStore{}, nil, targets)
	service.startHotkeyFactory = func(_ uint32, callback func()) startHotkeyWatch {
		return &fakeStartHotkeyWatch{callback: callback}
	}
	if _, err := service.Start(StartArgs{TargetSlot: "editor", Mode: inputclip.RecordingModeSimple}); err != nil {
		t.Fatal(err)
	}
	if err := service.BeginCountdown(); err != nil {
		t.Fatal(err)
	}
	service.finishCountdown(service.countdownGeneration.Load())
	if targets.released != 0 {
		t.Fatalf("recording target released before Stop: %d", targets.released)
	}
	if _, err := service.Stop(); err != nil {
		t.Fatal(err)
	}
	if targets.released != 1 || service.GetState().Phase != PhasePending {
		t.Fatalf("released=%d state=%+v", targets.released, service.GetState())
	}
}

func TestRecordingSessionPersistsCanonicalCarrierAndReloadsDraftReference(t *testing.T) {
	root := t.TempDir()
	assets := newRecordingAssetStore(t, root)
	clips := inputclip.NewService(assets)
	recorder := &resultRecorder{result: &StopResult{
		TempID: "roundtrip",
		Meta: inputclip.ClipMeta{
			RecordingMode: inputclip.RecordingModePrecise,
			MouseMode:     "mixed", BaseResolution: [2]int{1280, 720}, MouseCounts360: 800,
		},
		Events: []inputclip.Event{
			{TUs: 800, Type: inputclip.EventTypeKeyDown, A: 'W'},
			{TUs: 1_800, Seq: 1, Type: inputclip.EventTypeRawDelta, B: 12, C: -4},
			{TUs: 2_800, Seq: 2, Type: inputclip.EventTypeKeyUp, A: 'W'},
		},
	}}
	service := NewService(recorder, nil, clips, nil, nil)
	service.setState(RecordingState{Phase: PhaseRecording, Mode: inputclip.RecordingModePrecise, TargetSlot: "editor"})
	pending, err := service.Stop()
	if err != nil {
		t.Fatal(err)
	}
	if pending == nil || pending.Mode != inputclip.RecordingModePrecise || service.GetState().Pending == nil {
		t.Fatalf("pending=%+v state=%+v", pending, service.GetState())
	}
	finalized, err := service.Finalize(FinalizeArgs{PendingID: pending.PendingID, Label: "Round trip"})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := clips.Get(finalized.AssetID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Meta.RecordingMode != inputclip.RecordingModePrecise || len(loaded.Events) != 3 || loaded.Events[0].TUs != 0 || loaded.Events[2].TUs != 2_000 {
		t.Fatalf("loaded clip = %+v events=%+v", loaded, loaded.Events)
	}
	if finalized.AssetKind != asset.KindClip || finalized.Blob != loaded.Blob {
		t.Fatalf("finalized=%+v blob=%+v", finalized, loaded.Blob)
	}
}

func newRecordingAssetStore(t *testing.T, root string) *asset.Store {
	t.Helper()
	roots, err := storage.Resolve(root)
	if err != nil {
		t.Fatal(err)
	}
	foundation, err := catalog.Open(context.Background(), roots)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = foundation.Close() })
	blobs, err := blob.Open(
		roots.Objects,
		blob.Limits{MaxBlobBytes: 1 << 20, MaxTotalBytes: 4 << 20},
		foundation.Objects(),
	)
	if err != nil {
		t.Fatal(err)
	}
	store, err := asset.NewStore(
		foundation.Assets(),
		foundation.Objects(),
		blobs,
		asset.WithGCGracePeriod(0),
	)
	if err != nil {
		t.Fatal(err)
	}
	return store
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
			service := NewService(&resultRecorder{}, nil, nil, nil, test.targets)
			_, err := service.Start(StartArgs{TargetSlot: "editor", Mode: inputclip.RecordingModeSimple})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Start() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestServiceValidateTargetAcquiresAndReleasesExactTarget(t *testing.T) {
	targets := &recordingTargetResolver{window: target.WindowHandle{HWND: 42}}
	service := NewService(&resultRecorder{}, nil, nil, nil, targets)
	if err := service.ValidateTarget("editor"); err != nil || targets.released != 1 {
		t.Fatalf("ValidateTarget() error=%v released=%d", err, targets.released)
	}
	targets.resolveErr = errors.New("activation denied")
	if err := service.ValidateTarget("editor"); err == nil || !strings.Contains(err.Error(), "RECORDING_TARGET_UNAVAILABLE") {
		t.Fatalf("ValidateTarget() error = %v", err)
	}
}

func TestServiceFinalizeRejectsInvalidMetadataAndDiscardRemovesPending(t *testing.T) {
	service := NewService(&resultRecorder{}, nil, &memoryClipStore{}, nil, nil)
	service.pending = &pendingRecording{result: &StopResult{TempID: "session", Meta: inputclip.ClipMeta{RecordingMode: inputclip.RecordingModeSimple}}}
	for _, args := range []FinalizeArgs{
		{PendingID: "pending-session", Label: " "},
		{PendingID: "pending-session", Label: strings.Repeat("界", 81)},
		{PendingID: "missing", Label: "Valid"},
	} {
		if _, err := service.Finalize(args); err == nil {
			t.Fatalf("Finalize(%+v) succeeded", args)
		}
	}
	if err := service.Discard("pending-session"); err != nil {
		t.Fatal(err)
	}
	if service.pending != nil {
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
	if _, err := s.Start(StartArgs{TargetSlot: "editor", Mode: inputclip.RecordingModeSimple}); err == nil ||
		!strings.Contains(err.Error(), "closed") {
		t.Fatalf("Start() after shutdown error = %v, want closed", err)
	}
}

func TestShutdownDuringFinalizingDiscardsRecorderResult(t *testing.T) {
	recorder := &blockingStopRecorder{started: make(chan struct{}), release: make(chan struct{})}
	saver := &countingClipSaver{}
	s := NewService(recorder, nil, saver, nil, nil)
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
	s := NewService(NewRecorder(), nil, nil, nil, nil, func(name string, _ any) {
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

func TestService_StartWhenSameSessionIsIdempotent(t *testing.T) {
	s, _ := newTestService()
	// 模拟已在录 (不走真 hook): 直接置 recording 态.
	s.setState(RecordingState{Phase: PhaseRecording, TargetSlot: "editor", Mode: inputclip.RecordingModeSimple, TempID: "t1"})
	// Start 撞非 idle → 返当前 tempID, 不报错, 不重启.
	id, err := s.Start(StartArgs{TargetSlot: "editor", Mode: inputclip.RecordingModeSimple})
	if err != nil {
		t.Fatalf("非 idle Start 应无错 (幂等 no-op), got %v", err)
	}
	if id != "t1" {
		t.Fatalf("非 idle Start 应返当前 tempID t1, got %q", id)
	}
	if s.GetState().TargetSlot != "editor" {
		t.Fatalf("non-idle Start changed target slot, got %q", s.GetState().TargetSlot)
	}
	if _, err := s.Start(StartArgs{TargetSlot: "other", Mode: inputclip.RecordingModeSimple}); err == nil || !strings.Contains(err.Error(), "RECORDING_SESSION_BUSY") {
		t.Fatalf("different active session error = %v", err)
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
