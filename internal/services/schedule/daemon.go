package schedule

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"yotta/internal/services/execution"
)

// HotkeyRegistrar 由 main.go 注入 services.HotkeyRegistry 适配。
// schedule 包不能 import services（cycle）。
//
// label: i18n key string. labelParams: 给 vue-i18n named interpolation (nil 表示无动态).
type HotkeyRegistrar interface {
	Register(key, source, label string, labelParams map[string]string, hotkeyStr, readonlyReason string, onFire func()) error
	Unregister(key string) error
}

// Daemon 启动期注册所有 enabled schedules 到 cron / hotkey / once，
// trigger fire 时把 QueuedRun 扔进 execution queue。
type Daemon struct {
	store    *Store
	queue    *execution.ExecutionQueue
	hotkeys  HotkeyRegistrar
	cron     *cron.Cron
	mu       sync.Mutex
	cronIDs  map[string]cron.EntryID // schedule.ID → cron entry ID
	hotkeyKs map[string]string       // schedule.ID → hotkey registry key
}

func NewDaemon(store *Store, queue *execution.ExecutionQueue, hotkeys HotkeyRegistrar) *Daemon {
	return &Daemon{
		store:    store,
		queue:    queue,
		hotkeys:  hotkeys,
		cron:     cron.New(),
		cronIDs:  map[string]cron.EntryID{},
		hotkeyKs: map[string]string{},
	}
}

// Start 注册所有 enabled schedules + 启动 cron。
func (d *Daemon) Start() {
	d.cron.Start()
	d.mu.Lock()
	defer d.mu.Unlock()
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
	d.mu.Lock()
	defer d.mu.Unlock()
	for sid, hk := range d.hotkeyKs {
		_ = d.hotkeys.Unregister(hk)
		delete(d.hotkeyKs, sid)
	}
	for sid, id := range d.cronIDs {
		d.cron.Remove(id)
		delete(d.cronIDs, sid)
	}
	ctx := d.cron.Stop()
	<-ctx.Done()
}

// Reload schedule.Update / Create / Delete 后调，重注册受影响的 schedule。
// 简单粗暴：unregister all → register all。
func (d *Daemon) Reload() {
	d.mu.Lock()
	defer d.mu.Unlock()
	for sid, hk := range d.hotkeyKs {
		_ = d.hotkeys.Unregister(hk)
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
		_ = d.registerLocked(&s)
	}
}

func (d *Daemon) registerLocked(s *Schedule) error {
	switch s.Trigger.Kind {
	case "cron":
		spec, err := buildCronSpec(s.Trigger)
		if err != nil {
			return err
		}
		id, err := d.cron.AddFunc(spec, func() { d.fire(s.ID, execution.SourceSchedule) })
		if err != nil {
			return fmt.Errorf("cron AddFunc: %w", err)
		}
		d.cronIDs[s.ID] = id
	case "hotkey":
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
			func() { d.fire(s.ID, execution.SourceHotkey) }); err != nil {
			return fmt.Errorf("hotkey register: %w", err)
		}
		d.hotkeyKs[s.ID] = key
	case "once":
		// 启动后立即一次。fire 异步避免持锁。
		go d.fire(s.ID, execution.SourceSchedule)
	case "manual":
		// 不注册自动触发
	default:
		return fmt.Errorf("unknown trigger kind %q", s.Trigger.Kind)
	}
	return nil
}

// FireManual UI 上手动按 ▶ 跑某 schedule。
func (d *Daemon) FireManual(scheduleID string) error {
	return d.fire(scheduleID, execution.SourceManual)
}

func (d *Daemon) fire(scheduleID string, source execution.TriggerSource) error {
	s, ok := d.store.Get(scheduleID)
	if !ok {
		return fmt.Errorf("schedule %s not found", scheduleID)
	}
	targets := make([]execution.TargetRef, len(s.Targets))
	for i, t := range s.Targets {
		targets[i] = execution.TargetRef{Kind: t.Kind, ID: t.ID}
	}
	var dl *time.Time
	if s.TimeoutMinutes > 0 {
		t := time.Now().Add(time.Duration(s.TimeoutMinutes) * time.Minute)
		dl = &t
	}
	onErr := execution.OnErrorMode(s.OnError)
	if onErr == "" {
		onErr = execution.OnErrorStop
	}
	run := execution.QueuedRun{
		Targets:  targets,
		OnError:  onErr,
		Deadline: dl,
		Source:   source,
	}
	if _, ok := d.queue.Enqueue(run); !ok {
		return errors.New("execution queue closed")
	}
	// 记录 fire 时间
	now := time.Now()
	if cur, ok := d.store.Get(scheduleID); ok {
		cur.LastFiredAt = &now
		cur.LastStatus = "queued"
		_ = d.store.Save(&cur)
	}
	return nil
}

// buildCronSpec subKind=daily + at="HH:MM" → "M H * * *"
//                subKind=interval + everyMinutes=N → "@every Nm"
func buildCronSpec(t Trigger) (string, error) {
	switch t.SubKind {
	case "daily":
		hh, mm, err := parseHHMM(t.At)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%d %d * * *", mm, hh), nil
	case "interval":
		if t.EveryMinutes <= 0 {
			return "", errors.New("interval everyMinutes 必须 > 0")
		}
		return fmt.Sprintf("@every %dm", t.EveryMinutes), nil
	}
	return "", fmt.Errorf("unknown cron subKind %q", t.SubKind)
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
