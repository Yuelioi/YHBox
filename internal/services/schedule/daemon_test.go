package schedule

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type fakeWorkflowRunner struct {
	mu  sync.Mutex
	ids []string
	err error
}

type blockingWorkflowRunner struct {
	started chan struct{}
	ended   chan struct{}
}

func (r *blockingWorkflowRunner) StartInstallation(ctx context.Context, _ string) error {
	close(r.started)
	<-ctx.Done()
	close(r.ended)
	return ctx.Err()
}

func (f *fakeWorkflowRunner) StartInstallation(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ids = append(f.ids, id)
	return f.err
}

func (f *fakeWorkflowRunner) started() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.ids...)
}

type fakeRegistrar struct {
	mu            sync.Mutex
	callbacks     map[string]func()
	unregisterErr error
}

func newFakeRegistrar() *fakeRegistrar {
	return &fakeRegistrar{callbacks: map[string]func(){}}
}

func (f *fakeRegistrar) Register(key, source, label string, _ map[string]string, hotkey, readonly string, onFire func()) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.callbacks[key] = onFire
	return nil
}

func (f *fakeRegistrar) Unregister(key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.unregisterErr != nil {
		return f.unregisterErr
	}
	delete(f.callbacks, key)
	return nil
}

func (f *fakeRegistrar) Fire(key string) {
	f.mu.Lock()
	cb := f.callbacks[key]
	f.mu.Unlock()
	if cb != nil {
		cb()
	}
}

func TestParseHHMM(t *testing.T) {
	hh, mm, err := parseHHMM("05:00")
	if err != nil || hh != 5 || mm != 0 {
		t.Errorf("parse 05:00: %d:%d err=%v", hh, mm, err)
	}
	if _, _, err := parseHHMM("bad"); err == nil {
		t.Error("bad input must error")
	}
	if _, _, err := parseHHMM("25:00"); err == nil {
		t.Error("hour OOB must error")
	}
}

func TestBuildCronSpec(t *testing.T) {
	s, err := buildCronSpec(Trigger{SubKind: CronDaily, At: "05:00"})
	if err != nil || s != "0 5 * * *" {
		t.Errorf("daily spec: %q err=%v", s, err)
	}
	s, err = buildCronSpec(Trigger{SubKind: CronInterval, EveryMinutes: 30})
	if err != nil || s != "@every 30m" {
		t.Errorf("interval spec: %q err=%v", s, err)
	}
	if _, err := buildCronSpec(Trigger{SubKind: "bad"}); err == nil {
		t.Error("bad subKind must error")
	}
}

func TestDaemonHotkeyFire(t *testing.T) {
	tmp := t.TempDir()
	store, err := NewStore(tmp)
	if err != nil {
		t.Fatal(err)
	}
	runner := &fakeWorkflowRunner{}
	registrar := newFakeRegistrar()
	daemon := NewDaemon(store, runner, registrar)

	// 注册一个 hotkey schedule
	sc := &Schedule{
		SchemaVersion: CurrentSchemaVersion,
		ID:            "test-1",
		Name:          "test",
		Enabled:       true,
		Targets:       []TargetRef{{Kind: TargetWorkflowInstallation, ID: "C1"}},
		Trigger:       Trigger{Kind: TriggerHotkey, Hotkey: "Ctrl+Shift+1"},
		OnError:       OnErrorStop,
	}
	if err := store.Save(sc); err != nil {
		t.Fatal(err)
	}
	daemon.Start()
	defer daemon.Stop()

	registrar.Fire("schedule.test-1")
	if started := runner.started(); len(started) != 1 || started[0] != "C1" {
		t.Fatalf("started workflows = %#v", started)
	}
}

func TestDaemonFireManual(t *testing.T) {
	tmp := t.TempDir()
	store, _ := NewStore(tmp)
	runner := &fakeWorkflowRunner{}
	daemon := NewDaemon(store, runner, newFakeRegistrar())

	sc := &Schedule{
		SchemaVersion: CurrentSchemaVersion,
		ID:            "m-1",
		Name:          "manual",
		Enabled:       true,
		Targets:       []TargetRef{{Kind: TargetWorkflowInstallation, ID: "X"}},
		Trigger:       Trigger{Kind: TriggerManual},
		OnError:       OnErrorStop,
	}
	_ = store.Save(sc)
	daemon.Start()
	defer daemon.Stop()

	if err := daemon.FireManual("m-1"); err != nil {
		t.Fatal(err)
	}
	if started := runner.started(); len(started) != 1 || started[0] != "X" {
		t.Fatalf("manual fire = %#v", started)
	}
}

func TestDaemonStartStopAreIdempotent(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	daemon := NewDaemon(store, &fakeWorkflowRunner{}, nil)

	daemon.Start()
	daemon.Start()
	daemon.Stop()
	daemon.Stop()
	daemon.Reload()
}

func TestDaemonStopBeforeStartPreventsLateStart(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	daemon := NewDaemon(store, &fakeWorkflowRunner{}, nil)

	daemon.Stop()
	daemon.Start()
	daemon.Stop()

	daemon.mu.Lock()
	defer daemon.mu.Unlock()
	if daemon.started {
		t.Fatal("daemon started after it had already stopped")
	}
}

func TestDaemonReloadBeforeStartIsNoop(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	registrar := newFakeRegistrar()
	daemon := NewDaemon(store, &fakeWorkflowRunner{}, registrar)
	daemon.Reload()

	registrar.mu.Lock()
	defer registrar.mu.Unlock()
	if len(registrar.callbacks) != 0 {
		t.Fatalf("pre-start Reload registered %d hotkeys", len(registrar.callbacks))
	}
}

func TestDaemonReloadReportsHotkeyCleanupErrors(t *testing.T) {
	store, _ := NewStore(t.TempDir())
	schedule := &Schedule{SchemaVersion: CurrentSchemaVersion, ID: "reload", Name: "reload", Enabled: true,
		Targets: []TargetRef{{Kind: TargetWorkflowInstallation, ID: "C1"}}, Trigger: Trigger{Kind: TriggerHotkey, Hotkey: "Ctrl+Shift+1"}, OnError: OnErrorStop}
	if err := store.Save(schedule); err != nil {
		t.Fatal(err)
	}
	want := errors.New("unregister failed")
	registrar := newFakeRegistrar()
	daemon := NewDaemon(store, &fakeWorkflowRunner{}, registrar)
	daemon.Start()
	registrar.unregisterErr = want
	if err := daemon.Reload(); !errors.Is(err, want) {
		t.Fatalf("Reload error = %v, want cleanup failure", err)
	}
	if got := daemon.hotkeyKs["reload"]; got != "schedule.reload" {
		t.Fatalf("failed binding was no longer tracked: %q", got)
	}
	registrar.mu.Lock()
	_, stillActive := registrar.callbacks["schedule.reload"]
	registrar.mu.Unlock()
	if !stillActive {
		t.Fatal("failed unregister binding was lost instead of retained for retry")
	}
}

func TestDaemonStopAggregatesHotkeyCleanupErrors(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	schedule := &Schedule{
		SchemaVersion: CurrentSchemaVersion,
		ID:            "cleanup",
		Name:          "cleanup",
		Enabled:       true,
		Targets:       []TargetRef{{Kind: TargetWorkflowInstallation, ID: "C1"}},
		Trigger:       Trigger{Kind: TriggerHotkey, Hotkey: "Ctrl+Shift+1"},
		OnError:       OnErrorStop,
	}
	if err := store.Save(schedule); err != nil {
		t.Fatalf("Save: %v", err)
	}
	cleanupFailure := errors.New("unregister failed")
	registrar := newFakeRegistrar()
	registrar.unregisterErr = cleanupFailure
	daemon := NewDaemon(store, &fakeWorkflowRunner{}, registrar)
	daemon.Start()

	if err := daemon.StopContext(context.Background()); !errors.Is(err, cleanupFailure) {
		t.Fatalf("StopContext error=%v, want cleanup failure", err)
	}
}

func TestDaemonStopCancelsAndWaitsForOwnedOnceFire(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	schedule := validSchedule("owned-once")
	schedule.Enabled = true
	schedule.Trigger = Trigger{Kind: TriggerOnce}
	if err := store.Save(schedule); err != nil {
		t.Fatal(err)
	}
	runner := &blockingWorkflowRunner{started: make(chan struct{}), ended: make(chan struct{})}
	daemon := NewDaemon(store, runner, nil)
	daemon.Start()
	select {
	case <-runner.started:
	case <-time.After(time.Second):
		t.Fatal("once fire did not start")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := daemon.StopContext(ctx); err != nil {
		t.Fatalf("StopContext: %v", err)
	}
	select {
	case <-runner.ended:
	default:
		t.Fatal("StopContext returned before owned fire ended")
	}
}
