package schedule

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// HotkeyRegistrar 由 main.go 注入 services.HotkeyRegistry 适配。
// schedule 包不能 import services（cycle）。
//
// label: i18n key string. labelParams: 给 vue-i18n named interpolation (nil 表示无动态).
type HotkeyRegistrar interface {
	Register(key, source, label string, labelParams map[string]string, hotkeyStr, readonlyReason string, onFire func()) error
	Unregister(key string) error
}

type WorkflowRunner interface {
	StartWorkflow(context.Context, string) error
}

// Daemon 启动期注册所有 enabled schedules 到 cron / hotkey / once，
// trigger fire 时把 QueuedRun 扔进 execution queue。
type Daemon struct {
	store    *Store
	runner   WorkflowRunner
	hotkeys  HotkeyRegistrar
	cron     *cron.Cron
	mu       sync.Mutex
	cronIDs  map[string]cron.EntryID // schedule.ID → cron entry ID
	hotkeyKs map[string]string       // schedule.ID → hotkey registry key
	started  bool
	stopped  bool
	runCtx   context.Context
	cancel   context.CancelFunc
	fireWG   sync.WaitGroup
	stopDone chan struct{}
	stopErr  error
}

func NewDaemon(store *Store, runner WorkflowRunner, hotkeys HotkeyRegistrar) *Daemon {
	return &Daemon{
		store:    store,
		runner:   runner,
		hotkeys:  hotkeys,
		cron:     cron.New(),
		cronIDs:  map[string]cron.EntryID{},
		hotkeyKs: map[string]string{},
	}
}

// Start 注册所有 enabled schedules + 启动 cron。
func (d *Daemon) Start() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.started || d.stopped {
		return
	}
	d.started = true
	d.runCtx, d.cancel = context.WithCancel(context.Background())
	d.cron.Start()
	for _, s := range d.store.List() {
		if !s.Enabled {
			continue
		}
		if err := d.registerLocked(&s); err != nil {
			// 单条失败不影响其它（startup-friendly）
			fmt.Printf("schedule daemon: register %s failed: %v\n", s.ID, err)
		}
	}
}

// Stop 关 cron + 取消所有注册的 hotkey。
func (d *Daemon) Stop() {
	_ = d.StopContext(context.Background())
}

// StopContext initiates shutdown once and waits for cron jobs or context.
func (d *Daemon) StopContext(ctx context.Context) error {
	d.mu.Lock()
	if !d.stopped {
		d.stopped = true
		if d.cancel != nil {
			d.cancel()
		}
		cronDone := closedDone()
		if d.started {
			var cleanupErrs []error
			for sid, hk := range d.hotkeyKs {
				if err := d.hotkeys.Unregister(hk); err != nil {
					cleanupErrs = append(cleanupErrs, fmt.Errorf("unregister schedule %s hotkey: %w", sid, err))
				}
				delete(d.hotkeyKs, sid)
			}
			for sid, id := range d.cronIDs {
				d.cron.Remove(id)
				delete(d.cronIDs, sid)
			}
			d.stopErr = errors.Join(cleanupErrs...)
			cronDone = d.cron.Stop().Done()
		}
		d.stopDone = make(chan struct{})
		stopDone := d.stopDone
		go func() {
			<-cronDone
			d.fireWG.Wait()
			close(stopDone)
		}()
	}
	done := d.stopDone
	cleanupErr := d.stopErr
	d.mu.Unlock()
	select {
	case <-done:
		return cleanupErr
	case <-ctx.Done():
		return errors.Join(cleanupErr, ctx.Err())
	}
}

// Reload schedule.Update / Create / Delete 后调，重注册受影响的 schedule。
// 简单粗暴：unregister all → register all。
func (d *Daemon) Reload() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.started || d.stopped {
		return nil
	}
	var reloadErr error
	for sid, hk := range d.hotkeyKs {
		if err := d.hotkeys.Unregister(hk); err != nil {
			reloadErr = errors.Join(reloadErr, fmt.Errorf("unregister schedule %s hotkey: %w", sid, err))
			continue
		}
		delete(d.hotkeyKs, sid)
	}
	for sid, id := range d.cronIDs {
		d.cron.Remove(id)
		delete(d.cronIDs, sid)
	}
	for _, s := range d.store.List() {
		if !s.Enabled {
			continue
		}
		if _, oldBindingStillActive := d.hotkeyKs[s.ID]; oldBindingStillActive {
			continue
		}
		if err := d.registerLocked(&s); err != nil {
			reloadErr = errors.Join(reloadErr, fmt.Errorf("register schedule %s: %w", s.ID, err))
		}
	}
	return reloadErr
}

func (d *Daemon) registerLocked(s *Schedule) error {
	switch s.Trigger.Kind {
	case TriggerCron:
		spec, err := buildCronSpec(s.Trigger)
		if err != nil {
			return err
		}
		id, err := d.cron.AddFunc(spec, func() { _ = d.fireOwned(s.ID) })
		if err != nil {
			return fmt.Errorf("cron AddFunc: %w", err)
		}
		d.cronIDs[s.ID] = id
	case TriggerHotkey:
		if s.Trigger.Hotkey == "" {
			return errors.New("hotkey trigger missing hotkey")
		}
		if d.hotkeys == nil {
			return errors.New("hotkey registrar not injected")
		}
		key := "schedule." + s.ID
		if err := d.hotkeys.Register(key, "schedule",
			"hotkeys.label.schedule", map[string]string{"name": s.Name},
			s.Trigger.Hotkey, "",
			func() { _ = d.fireOwned(s.ID) }); err != nil {
			return fmt.Errorf("hotkey register: %w", err)
		}
		d.hotkeyKs[s.ID] = key
	case TriggerOnce:
		// 启动后立即一次。任务先登记到 daemon owner，再异步执行以避免持锁。
		d.launchFireLocked(s.ID)
	case TriggerManual:
		// 不注册自动触发
	default:
		return fmt.Errorf("unknown trigger kind %q", s.Trigger.Kind)
	}
	return nil
}

// FireManual UI 上手动按 ▶ 跑某 schedule。
func (d *Daemon) FireManual(scheduleID string) error {
	return d.fireOwned(scheduleID)
}

func (d *Daemon) fireOwned(scheduleID string) error {
	d.mu.Lock()
	if !d.started {
		d.mu.Unlock()
		return errors.New("schedule daemon not started")
	}
	if d.stopped {
		d.mu.Unlock()
		return errors.New("schedule daemon stopped")
	}
	ctx := d.runCtx
	d.fireWG.Add(1)
	d.mu.Unlock()
	defer d.fireWG.Done()
	return d.fire(ctx, scheduleID)
}

// launchFireLocked starts an owned fire while d.mu is held. StopContext marks
// the daemon stopped under the same lock before waiting, so no Add can race
// with fireWG.Wait.
func (d *Daemon) launchFireLocked(scheduleID string) {
	ctx := d.runCtx
	d.fireWG.Add(1)
	go func() {
		defer d.fireWG.Done()
		_ = d.fire(ctx, scheduleID)
	}()
}

func (d *Daemon) fire(ctx context.Context, scheduleID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s, ok := d.store.Get(scheduleID)
	if !ok {
		return fmt.Errorf("schedule %s not found", scheduleID)
	}
	if d.runner == nil {
		return errors.New("workflow runner not injected")
	}
	runCtx := ctx
	var cancel context.CancelFunc
	if s.TimeoutMinutes > 0 {
		runCtx, cancel = context.WithTimeout(ctx, time.Duration(s.TimeoutMinutes)*time.Minute)
		defer cancel()
	}
	var runErr error
	started := 0
	for index, target := range s.Targets {
		if target.Kind != TargetWorkflow {
			err := fmt.Errorf("schedule target %d has unsupported kind %q", index, target.Kind)
			runErr = errors.Join(runErr, err)
			if s.OnError != OnErrorContinue {
				break
			}
			continue
		}
		if err := d.runner.StartWorkflow(runCtx, target.ID); err != nil {
			runErr = errors.Join(runErr, fmt.Errorf("start workflow %q: %w", target.ID, err))
			if s.OnError != OnErrorContinue {
				break
			}
			continue
		}
		started++
	}
	// StartWorkflow returns only after the durable QUEUED Run exists, so the
	// schedule status never claims a notification that admission did not commit.
	now := time.Now()
	if cur, ok := d.store.Get(scheduleID); ok {
		cur.LastFiredAt = &now
		if started > 0 {
			cur.LastStatus = FireStatusQueued
		} else {
			cur.LastStatus = FireStatusFailed
		}
		if err := d.store.Save(&cur); err != nil {
			runErr = errors.Join(runErr, fmt.Errorf("persist schedule %s fired status: %w", scheduleID, err))
		}
	}
	return runErr
}

// buildCronSpec subKind=daily + at="HH:MM" → "M H * * *"
//
//	subKind=interval + everyMinutes=N → "@every Nm"
func buildCronSpec(t Trigger) (string, error) {
	switch t.SubKind {
	case CronDaily:
		hh, mm, err := parseHHMM(t.At)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%d %d * * *", mm, hh), nil
	case CronInterval:
		if t.EveryMinutes <= 0 {
			return "", errors.New("interval everyMinutes 必须 > 0")
		}
		return fmt.Sprintf("@every %dm", t.EveryMinutes), nil
	}
	return "", fmt.Errorf("unknown cron subKind %q", t.SubKind)
}

func closedDone() <-chan struct{} {
	done := make(chan struct{})
	close(done)
	return done
}

func parseHHMM(s string) (hh, mm int, err error) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("at %q 必须 HH:MM 格式", s)
	}
	hh, err = strconv.Atoi(parts[0])
	if err != nil || hh < 0 || hh > 23 {
		return 0, 0, fmt.Errorf("at %q hour 不合法", s)
	}
	mm, err = strconv.Atoi(parts[1])
	if err != nil || mm < 0 || mm > 59 {
		return 0, 0, fmt.Errorf("at %q minute 不合法", s)
	}
	return hh, mm, nil
}
