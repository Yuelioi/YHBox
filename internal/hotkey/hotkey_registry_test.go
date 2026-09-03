package hotkey

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/apperr"
)

func TestHotkeyValidationErrorsProjectStableProblems(t *testing.T) {
	tests := []struct {
		err error
		id  string
	}{
		{&HotkeyConflictError{Key: "action.a", ConflictKey: "action.b", Label: "hotkeys.label.workflow"}, "hotkey.conflict"},
		{&HotkeyReservedError{Hotkey: "Ctrl+C", Reason: "raw reason"}, "hotkey.reserved"},
		{&HotkeyInvalidError{Hotkey: "Nope", Reason: "raw reason"}, "hotkey.invalid"},
	}
	for _, test := range tests {
		envelope := apperr.From(test.err)
		if envelope.ID != test.id || envelope.Category != apperr.CategoryValidation || envelope.OperationID == "" {
			t.Fatalf("%T projected %#v", test.err, envelope)
		}
	}
}

func TestRegistry_RegisterBasicEntry(t *testing.T) {
	mgr := newTestHotkeyManager()
	r := NewHotkeyRegistry(mgr)

	err := r.Register("test.foo", HotkeySourceSystem, "测试 foo", nil, "", "", func() {})
	if err != nil {
		t.Fatalf("Register err=%v", err)
	}
	got, ok := r.Get("test.foo")
	if !ok {
		t.Fatal("Get 应找到 test.foo")
	}
	if got.Status != HotkeyStatusUnbound {
		t.Errorf("空 hotkeyStr 时 Status=%q, want unbound", got.Status)
	}
}

func TestRegistry_RegisterRejectsDuplicateKey(t *testing.T) {
	mgr := newTestHotkeyManager()
	r := NewHotkeyRegistry(mgr)
	_ = r.Register("dup", HotkeySourceSystem, "first", nil, "", "", func() {})
	err := r.Register("dup", HotkeySourceSystem, "second", nil, "", "", func() {})
	if !errors.Is(err, ErrDuplicateKey) {
		t.Errorf("第二次 Register 应返 ErrDuplicateKey, got %v", err)
	}
}

func TestRegistry_ListReturnsAll(t *testing.T) {
	mgr := newTestHotkeyManager()
	r := NewHotkeyRegistry(mgr)
	_ = r.Register("a", HotkeySourceSystem, "A", nil, "", "", func() {})
	_ = r.Register("b", HotkeySourceAction, "B", nil, "", "", func() {})
	list := r.List()
	if len(list) != 2 {
		t.Errorf("len=%d want 2", len(list))
	}
}

func TestRegistry_DebugDumpFormat(t *testing.T) {
	mgr := newTestHotkeyManager()
	r := NewHotkeyRegistry(mgr)
	_ = r.Register("x", HotkeySourceSystem, "测试 X", nil, "", "", func() {})
	dump := r.DebugDump()
	if !strings.Contains(dump, "x") || !strings.Contains(dump, "unbound") {
		t.Errorf("DebugDump 应含 key 和 status, got: %s", dump)
	}
}

func TestRegistry_UpdateRejectsReserved(t *testing.T) {
	mgr := newTestHotkeyManager()
	r := NewHotkeyRegistry(mgr)
	_ = r.Register("x", HotkeySourceAction, "X", nil, "", "", func() {})
	err := r.Update("x", "Ctrl+C")
	var rerr *HotkeyReservedError
	if !errors.As(err, &rerr) {
		t.Errorf("Ctrl+C 应返 ReservedError, got %v", err)
	}
}

func TestRegistry_UpdateRejectsConflict(t *testing.T) {
	mgr := newTestHotkeyManager()
	r := NewHotkeyRegistry(mgr)
	_ = r.Register("a", HotkeySourceAction, "A", nil, "Ctrl+Shift+Alt+F1", "", func() {})
	_ = r.Register("b", HotkeySourceAction, "B", nil, "", "", func() {})
	err := r.Update("b", "Ctrl+Shift+Alt+F1")
	var cerr *HotkeyConflictError
	if !errors.As(err, &cerr) {
		t.Errorf("撞 a 应返 ConflictError, got %v", err)
	}
	// cleanup
	_ = r.Unregister("a")
	_ = r.Unregister("b")
}

func TestRegistry_UpdateSelfNoConflict(t *testing.T) {
	mgr := newTestHotkeyManager()
	r := NewHotkeyRegistry(mgr)
	_ = r.Register("a", HotkeySourceAction, "A", nil, "Ctrl+Shift+Alt+F2", "", func() {})
	// 改自己当前值 noop
	if err := r.Update("a", "Ctrl+Shift+Alt+F2"); err != nil {
		t.Errorf("改自己当前值应返 nil, got %v", err)
	}
	_ = r.Unregister("a")
}

func TestRegistry_UpdateNormalizationDetectsConflict(t *testing.T) {
	mgr := newTestHotkeyManager()
	r := NewHotkeyRegistry(mgr)
	_ = r.Register("a", HotkeySourceAction, "A", nil, "Ctrl+Shift+Alt+F3", "", func() {})
	_ = r.Register("b", HotkeySourceAction, "B", nil, "", "", func() {})
	err := r.Update("b", "Shift+Ctrl+Alt+F3") // 顺序不同同义
	var cerr *HotkeyConflictError
	if !errors.As(err, &cerr) {
		t.Errorf("Shift+Ctrl+Alt+F3 撞 Ctrl+Shift+Alt+F3 应识别为冲突, got %v", err)
	}
	_ = r.Unregister("a")
	_ = r.Unregister("b")
}

func TestRegistry_ClearViaEmptyString(t *testing.T) {
	mgr := newTestHotkeyManager()
	r := NewHotkeyRegistry(mgr)
	_ = r.Register("a", HotkeySourceAction, "A", nil, "Ctrl+Shift+Alt+F4", "", func() {})
	if err := r.Update("a", ""); err != nil {
		t.Fatal(err)
	}
	got, _ := r.Get("a")
	if got.Status != HotkeyStatusUnbound {
		t.Errorf("Clear 后 Status=%q want unbound", got.Status)
	}
	if got.HotkeyStr != "" {
		t.Errorf("Clear 后 HotkeyStr=%q want ''", got.HotkeyStr)
	}
}

func TestRegistry_UnregisterRemovesEntry(t *testing.T) {
	mgr := newTestHotkeyManager()
	r := NewHotkeyRegistry(mgr)
	_ = r.Register("a", HotkeySourceAction, "A", nil, "Ctrl+Shift+Alt+F5", "", func() {})
	if err := r.Unregister("a"); err != nil {
		t.Fatal(err)
	}
	if _, ok := r.Get("a"); ok {
		t.Error("Unregister 后 Get 应返 false")
	}
}

func TestRegistry_UnregisterFailureKeepsEntry(t *testing.T) {
	mgr := newTestHotkeyManager()
	r := NewHotkeyRegistry(mgr)
	if err := r.Register("a", HotkeySourceSystem, "A", nil, "Ctrl+Shift+Alt+F5", "", func() {}); err != nil {
		t.Fatalf("register a: %v", err)
	}
	if err := r.Register("b", HotkeySourceSystem, "B", nil, "Ctrl+Shift+Alt+F6", "", func() {}); err != nil {
		t.Fatalf("register b: %v", err)
	}
	unregisterFailure := errors.New("native unregister failed")
	mgr.runLoop = func(_ context.Context, _ []HotkeySpec, _ func(int), ready chan<- error, done chan<- struct{}) {
		ready <- unregisterFailure
		close(done)
	}

	if err := r.Unregister("a"); !errors.Is(err, unregisterFailure) {
		t.Fatalf("Unregister error=%v, want native failure", err)
	}
	if _, ok := r.Get("a"); !ok {
		t.Fatal("registry deleted entry after native unregister failed")
	}
}

func TestRegistry_OnActionHotkeyChangeCallback(t *testing.T) {
	mgr := newTestHotkeyManager()
	var gotID, gotStr string
	r := NewHotkeyRegistryWithCallbacks(mgr, Callbacks{OnActionChange: func(id, str string) error { gotID, gotStr = id, str; return nil }})
	_ = r.Register("action.abc", HotkeySourceAction, "A", nil, "", "", func() {})
	if err := r.Update("action.abc", "Ctrl+Shift+Alt+F6"); err != nil {
		t.Fatal(err)
	}
	if gotID != "abc" || gotStr != "Ctrl+Shift+Alt+F6" {
		t.Errorf("callback got (%q, %q), want (abc, Ctrl+Shift+Alt+F6)", gotID, gotStr)
	}
	_ = r.Unregister("action.abc")
}

func TestRegistryTransientConflictRemainsVisibleAndDoesNotPersist(t *testing.T) {
	mgr := newTestHotkeyManager()
	persisted := false
	r := NewHotkeyRegistryWithCallbacks(mgr, Callbacks{OnActionChange: func(string, string) error {
		persisted = true
		return nil
	}})
	if err := r.Register("action.workflow", HotkeySourceAction, "workflow", nil, "Ctrl+Shift+1", "", nil); err != nil {
		t.Fatal(err)
	}
	if err := r.RegisterTransient("launcher.slot.1", "slot", nil, "Ctrl+Shift+1", nil); err != nil {
		t.Fatal(err)
	}
	entry, ok := r.Get("launcher.slot.1")
	if !ok || entry.Status != HotkeyStatusFailed || entry.LastError == "" {
		t.Fatalf("transient entry = %#v, exists=%v", entry, ok)
	}
	if persisted {
		t.Fatal("transient registration must not persist through action callback")
	}
}

func TestRegistry_UpdateRollsBackBindingWhenPersistenceFails(t *testing.T) {
	mgr := newTestHotkeyManager()
	wantErr := errors.New("settings save failed")
	r := NewHotkeyRegistryWithCallbacks(mgr, Callbacks{OnActionChange: func(string, string) error { return wantErr }})
	if err := r.Register("action.persist", HotkeySourceAction, "persist", nil,
		"Ctrl+Shift+Alt+F6", "", func() {}); err != nil {
		t.Fatal(err)
	}
	if err := r.Update("action.persist", "Ctrl+Shift+Alt+F7"); !errors.Is(err, wantErr) {
		t.Fatalf("Update error = %v", err)
	}
	entry, ok := r.Get("action.persist")
	if !ok || entry.HotkeyStr != "Ctrl+Shift+Alt+F6" || entry.Status != HotkeyStatusActive {
		t.Fatalf("entry after rollback = %+v, ok=%v", entry, ok)
	}
	bindings := mgr.Bindings()
	if len(bindings) != 1 || bindings[0].Spec.Name != "Ctrl+Shift+Alt+F6" {
		t.Fatalf("native bindings after rollback = %+v", bindings)
	}
	_ = r.Unregister("action.persist")
}

func TestRegistry_PersistenceRollbackFailureMarksEntryFailed(t *testing.T) {
	mgr := newTestHotkeyManager()
	loopCalls := 0
	mgr.runLoop = func(ctx context.Context, specs []HotkeySpec, handler func(int), ready chan<- error, done chan<- struct{}) {
		loopCalls++
		if loopCalls == 5 {
			ready <- errors.New("native rollback unregister failed")
			close(done)
			return
		}
		runTestHotkeyLoop(ctx, specs, handler, ready, done)
	}
	r := NewHotkeyRegistryWithCallbacks(mgr, Callbacks{OnActionChange: func(string, string) error { return errors.New("settings save failed") }})
	if err := r.Register("action.persist", HotkeySourceAction, "persist", nil,
		"Ctrl+Shift+Alt+F6", "", func() {}); err != nil {
		t.Fatal(err)
	}
	if err := r.Register("system.keep", HotkeySourceSystem, "keep", nil,
		"Ctrl+Shift+Alt+F8", "", func() {}); err != nil {
		t.Fatal(err)
	}
	if err := r.Update("action.persist", "Ctrl+Shift+Alt+F7"); err == nil {
		t.Fatal("Update error = nil")
	}
	entry, _ := r.Get("action.persist")
	if entry.Status != HotkeyStatusFailed || !strings.Contains(entry.LastError, "native rollback unregister failed") {
		t.Fatalf("entry after failed rollback = %+v", entry)
	}
	_ = r.Unregister("action.persist")
	_ = r.Unregister("system.keep")
}

func TestRegistry_RegisterEditorBasic(t *testing.T) {
	mgr := newTestHotkeyManager()
	r := NewHotkeyRegistry(mgr)

	err := r.RegisterEditor("editor.foo", "hotkeys.label.editor.foo", "Ctrl+Shift+Alt+F7", "hotkeys.readonly.editorBuiltin")
	if err != nil {
		t.Fatalf("RegisterEditor err=%v", err)
	}
	got, ok := r.Get("editor.foo")
	if !ok {
		t.Fatal("Get 应找到 editor.foo")
	}
	if got.Source != HotkeySourceEditor {
		t.Errorf("Source=%q, want editor", got.Source)
	}
	if got.Mechanism != HotkeyMechanismEditorInApp {
		t.Errorf("Mechanism=%q, want editor-inapp", got.Mechanism)
	}
	if got.ReadonlyReason == "" {
		t.Error("ReadonlyReason 应保留")
	}
	if got.Status != HotkeyStatusActive {
		t.Errorf("Status=%q, want active (editor-inapp 机制跳 OS, 直接 active)", got.Status)
	}
	_ = r.Unregister("editor.foo")
}

func TestRegistry_RegisterEditorSkipsOSBinding(t *testing.T) {
	mgr := newTestHotkeyManager()
	r := NewHotkeyRegistry(mgr)

	if err := r.RegisterEditor("editor.bar", "hotkeys.label.editor.bar", "Ctrl+Shift+Alt+F8", "hotkeys.readonly.editorBuiltin"); err != nil {
		t.Fatalf("RegisterEditor err=%v", err)
	}
	r.mu.RLock()
	e := r.entries["editor.bar"]
	r.mu.RUnlock()
	if e.bindingID != 0 {
		t.Errorf("editor-inapp entry bindingID=%d, want 0 (跳 OS register)", e.bindingID)
	}
	_ = r.Unregister("editor.bar")
}

func TestRegistry_RegisterEditorConflictKeepsEntryAsFailed(t *testing.T) {
	mgr := newTestHotkeyManager()
	r := NewHotkeyRegistry(mgr)

	_ = r.Register("system.taken", HotkeySourceSystem, "占用", nil, "Ctrl+Shift+Alt+F9", "", func() {})
	err := r.RegisterEditor("editor.collide", "hotkeys.label.editor.collide", "Ctrl+Shift+Alt+F9", "hotkeys.readonly.editorBuiltin")
	if err != nil {
		t.Fatalf("RegisterEditor 撞冲突时应返 nil (registry-level success), got %v", err)
	}
	got, ok := r.Get("editor.collide")
	if !ok {
		t.Fatal("conflict 后 entry 应保留 (RegisterEditor 跟老 Register 路径区别), got 删了")
	}
	if got.Status != HotkeyStatusFailed {
		t.Errorf("Status=%q, want failed", got.Status)
	}
	if got.LastError == "" {
		t.Error("LastError 应填冲突描述")
	}
	_ = r.Unregister("system.taken")
	_ = r.Unregister("editor.collide")
}

func TestRegisterLLHook(t *testing.T) {
	mgr := newTestHotkeyManager()
	r := NewHotkeyRegistry(mgr)

	err := r.RegisterLLHook("recording.stop", HotkeySourceRecording, "hotkeys.label.recording.stop", "F12", "")
	if err != nil {
		t.Fatalf("RegisterLLHook err=%v", err)
	}
	got, ok := r.Get("recording.stop")
	if !ok {
		t.Fatal("Get 应找到 recording.stop")
	}
	if got.Mechanism != HotkeyMechanismLLHook {
		t.Errorf("Mechanism=%q, want ll-hook", got.Mechanism)
	}
	if got.Status != HotkeyStatusActive {
		t.Errorf("Status=%q, want active", got.Status)
	}
	if got.HotkeyStr != "F12" {
		t.Errorf("HotkeyStr=%q, want F12", got.HotkeyStr)
	}
	r.mu.RLock()
	e := r.entries["recording.stop"]
	r.mu.RUnlock()
	if e.bindingID != 0 {
		t.Errorf("ll-hook entry bindingID=%d, want 0 (不占 OS)", e.bindingID)
	}
	_ = r.Unregister("recording.stop")
}

func TestRegisterLLHook_Rebind(t *testing.T) {
	mgr := newTestHotkeyManager()
	r := NewHotkeyRegistry(mgr)

	if err := r.RegisterLLHook("recording.stop", HotkeySourceRecording, "hotkeys.label.recording.stop", "F12", ""); err != nil {
		t.Fatalf("RegisterLLHook err=%v", err)
	}
	if err := r.Update("recording.stop", "F9"); err != nil {
		t.Fatalf("Update err=%v", err)
	}
	got, _ := r.Get("recording.stop")
	if got.HotkeyStr != "F9" {
		t.Errorf("HotkeyStr=%q, want F9", got.HotkeyStr)
	}
	if got.Status != HotkeyStatusActive {
		t.Errorf("Status=%q, want active", got.Status)
	}
	r.mu.RLock()
	e := r.entries["recording.stop"]
	r.mu.RUnlock()
	if e.bindingID != 0 {
		t.Errorf("rebind 后 ll-hook entry bindingID=%d, want 0 (不占 OS)", e.bindingID)
	}
	_ = r.Unregister("recording.stop")
}

func TestRegisterLLHook_ConflictAcrossSource(t *testing.T) {
	mgr := newTestHotkeyManager()
	r := NewHotkeyRegistry(mgr)

	_ = r.Register("system.execution-stop", HotkeySourceSystem, "停止", nil, "F9", "", func() {})
	err := r.RegisterLLHook("recording.stop", HotkeySourceRecording, "hotkeys.label.recording.stop", "F9", "")
	var cerr *HotkeyConflictError
	if !errors.As(err, &cerr) {
		t.Errorf("撞 system.execution-stop 应返 ConflictError, got %v", err)
	}
	if _, ok := r.Get("recording.stop"); ok {
		t.Error("冲突时 entry 不应注册进 (Get 应返 false)")
	}
	_ = r.Unregister("system.execution-stop")
}

func TestResumeSkipsLLHook(t *testing.T) {
	mgr := newTestHotkeyManager()
	r := NewHotkeyRegistry(mgr)

	if err := r.RegisterLLHook("recording.stop", HotkeySourceRecording, "hotkeys.label.recording.stop", "F12", ""); err != nil {
		t.Fatalf("RegisterLLHook err=%v", err)
	}
	if err := r.Pause(); err != nil {
		t.Fatalf("Pause err=%v", err)
	}
	if err := r.Resume(); err != nil {
		t.Fatalf("Resume err=%v", err)
	}
	got, _ := r.Get("recording.stop")
	if got.Status != HotkeyStatusActive {
		t.Errorf("Resume 后 Status=%q, want active (ll-hook 不被 OS 抢键)", got.Status)
	}
	r.mu.RLock()
	e := r.entries["recording.stop"]
	r.mu.RUnlock()
	if e.bindingID != 0 {
		t.Errorf("Resume 后 ll-hook entry bindingID=%d, want 0 (不重注册 OS)", e.bindingID)
	}
	_ = r.Unregister("recording.stop")
}

func TestRegistryShutdownReleasesBindingsAndRejectsNewWork(t *testing.T) {
	mgr := newTestHotkeyManager()
	r := NewHotkeyRegistry(mgr)
	if err := r.Register("system.shutdown-test", HotkeySourceSystem, "shutdown", nil,
		"Ctrl+Shift+Alt+F10", "", func() {}); err != nil {
		t.Fatal(err)
	}

	if err := r.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := r.Shutdown(context.Background()); err != nil {
		t.Fatalf("second Shutdown() error = %v", err)
	}
	r.mu.RLock()
	bindingID := r.entries["system.shutdown-test"].bindingID
	r.mu.RUnlock()
	if bindingID != 0 {
		t.Fatalf("binding remains after shutdown: %d", bindingID)
	}

	for name, err := range map[string]error{
		"register": r.Register("late", HotkeySourceSystem, "late", nil, "", "", nil),
		"update":   r.Update("system.shutdown-test", ""),
		"pause":    r.Pause(),
		"resume":   r.Resume(),
	} {
		if !errors.Is(err, ErrRegistryClosed) {
			t.Fatalf("%s error = %v, want ErrRegistryClosed", name, err)
		}
	}
}

func TestRegistryShutdownRetriesBindingsLeftByPartialPauseFailure(t *testing.T) {
	mgr := newTestHotkeyManager()
	loopCalls := 0
	mgr.runLoop = func(ctx context.Context, specs []HotkeySpec, handler func(int), ready chan<- error, done chan<- struct{}) {
		loopCalls++
		if loopCalls == 3 {
			ready <- errors.New("transient native rebuild failure")
			close(done)
			return
		}
		runTestHotkeyLoop(ctx, specs, handler, ready, done)
	}
	r := NewHotkeyRegistry(mgr)
	for _, entry := range []struct{ key, hotkey string }{
		{"system.first", "Ctrl+Shift+Alt+F10"},
		{"system.second", "Ctrl+Shift+Alt+F11"},
	} {
		if err := r.Register(entry.key, HotkeySourceSystem, entry.key, nil, entry.hotkey, "", nil); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.Pause(); err == nil {
		t.Fatal("Pause() error = nil, want transient unregister failure")
	}
	if err := r.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() retry error = %v", err)
	}
	if bindings := mgr.Bindings(); len(bindings) != 0 {
		t.Fatalf("manager bindings after shutdown = %+v", bindings)
	}
}

func TestRegistryResumeRestoresBindingsAfterPartialPauseFailure(t *testing.T) {
	mgr := newTestHotkeyManager()
	loopCalls := 0
	mgr.runLoop = func(ctx context.Context, specs []HotkeySpec, handler func(int), ready chan<- error, done chan<- struct{}) {
		loopCalls++
		if loopCalls == 3 {
			ready <- errors.New("transient native rebuild failure")
			close(done)
			return
		}
		runTestHotkeyLoop(ctx, specs, handler, ready, done)
	}
	r := NewHotkeyRegistry(mgr)
	for _, entry := range []struct{ key, hotkey string }{
		{"system.first", "Ctrl+Shift+Alt+F10"},
		{"system.second", "Ctrl+Shift+Alt+F11"},
	} {
		if err := r.Register(entry.key, HotkeySourceSystem, entry.key, nil, entry.hotkey, "", nil); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.Pause(); err == nil {
		t.Fatal("Pause() error = nil, want transient unregister failure")
	}
	if err := r.Resume(); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	if bindings := mgr.Bindings(); len(bindings) != 2 {
		t.Fatalf("manager bindings after Resume = %+v", bindings)
	}
	if err := r.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}
