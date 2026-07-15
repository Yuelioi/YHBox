// Package schedule 持有触发器实体（model/store/validate）。cron daemon + hotkey 注册
// 在同包的 daemon.go。
package schedule

import "time"

type SchemaVersion string

const CurrentSchemaVersion SchemaVersion = "3.1"

type TargetKind string

const TargetWorkflow TargetKind = "workflow"

type TriggerKind string

const (
	TriggerCron   TriggerKind = "cron"
	TriggerHotkey TriggerKind = "hotkey"
	TriggerOnce   TriggerKind = "once"
	TriggerManual TriggerKind = "manual"
)

type CronSubKind string

const (
	CronDaily    CronSubKind = "daily"
	CronInterval CronSubKind = "interval"
)

type OnErrorMode string

const (
	OnErrorStop     OnErrorMode = "stop"
	OnErrorContinue OnErrorMode = "continue"
)

type FireStatus string

const (
	FireStatusQueued FireStatus = "queued"
	FireStatusFailed FireStatus = "failed"
)

// TargetRef Schedule 触发后要跑的 Workflow Source。
type TargetRef struct {
	Kind TargetKind `json:"kind"`
	ID   string     `json:"id"`
}

// Trigger 触发器配置。kind 决定哪些子字段有意义。
//
//	kind="cron"   + subKind="daily"     → at="HH:MM"
//	kind="cron"   + subKind="interval"  → everyMinutes=N
//	kind="hotkey" → hotkey="Ctrl+Shift+X"
//	kind="once"   → 启动后一次，无额外字段
//	kind="manual" → 只能 UI 手动按 ▶，无自动触发
type Trigger struct {
	Kind         TriggerKind `json:"kind"`
	SubKind      CronSubKind `json:"subKind,omitempty"`
	At           string      `json:"at,omitempty"`
	EveryMinutes int         `json:"everyMinutes,omitempty"`
	Hotkey       string      `json:"hotkey,omitempty"`
}

type Schedule struct {
	SchemaVersion  SchemaVersion `json:"schemaVersion"`
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	Enabled        bool          `json:"enabled"`
	Targets        []TargetRef   `json:"targets"`
	Trigger        Trigger       `json:"trigger"`
	TimeoutMinutes int           `json:"timeoutMinutes"`
	OnError        OnErrorMode   `json:"onError"`
	LastFiredAt    *time.Time    `json:"lastFiredAt,omitempty"`
	LastStatus     FireStatus    `json:"lastStatus,omitempty"`
	CreatedAt      time.Time     `json:"createdAt"`
	UpdatedAt      time.Time     `json:"updatedAt"`
}
