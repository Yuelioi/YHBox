package schedule

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var hhmmRE = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

func (s *Schedule) Validate() error {
	if err := validateID(s.ID); err != nil {
		return err
	}
	if strings.TrimSpace(s.Name) == "" {
		return errors.New("schedule name 不能为空")
	}
	if len([]rune(s.Name)) > 160 || len([]rune(s.Description)) > 1_000 ||
		len([]rune(s.Category)) > 128 || len(s.Tags) > 16 {
		return errors.New("schedule metadata 超出长度限制")
	}
	for index, tag := range s.Tags {
		if strings.TrimSpace(tag) == "" || len([]rune(tag)) > 128 {
			return fmt.Errorf("tags[%d] 不合法", index)
		}
	}
	if s.SchemaVersion != CurrentSchemaVersion {
		return fmt.Errorf("schemaVersion %q 不支持", s.SchemaVersion)
	}
	if len(s.Targets) == 0 {
		return errors.New("targets 不能为空")
	}
	for i, t := range s.Targets {
		if t.Kind != TargetWorkflow {
			return fmt.Errorf("targets[%d].kind 必须 \"workflow\"，got %q", i, t.Kind)
		}
		if t.ID == "" {
			return fmt.Errorf("targets[%d].id 不能为空", i)
		}
	}
	switch s.Trigger.Kind {
	case TriggerCron:
		switch s.Trigger.SubKind {
		case CronDaily:
			if !hhmmRE.MatchString(s.Trigger.At) {
				return fmt.Errorf("trigger.at %q 不合法（必须 HH:MM 24h）", s.Trigger.At)
			}
		case CronInterval:
			if s.Trigger.EveryMinutes <= 0 {
				return errors.New("trigger.everyMinutes 必须 > 0")
			}
		default:
			return fmt.Errorf("trigger.subKind %q 不支持（cron 下必须 daily|interval）", s.Trigger.SubKind)
		}
	case TriggerHotkey:
		if strings.TrimSpace(s.Trigger.Hotkey) == "" {
			return errors.New("trigger.hotkey 不能为空")
		}
	case TriggerOnce, TriggerManual:
	default:
		return fmt.Errorf("trigger.kind %q 不支持（必须 cron|hotkey|once|manual）", s.Trigger.Kind)
	}
	switch s.OnError {
	case OnErrorStop, OnErrorContinue:
	default:
		return fmt.Errorf("onError %q 不支持（必须 stop|continue）", s.OnError)
	}
	if s.TimeoutMinutes < 0 {
		return errors.New("timeoutMinutes 不能为负")
	}
	if s.LastStatus != "" && s.LastStatus != FireStatusQueued && s.LastStatus != FireStatusFailed {
		return fmt.Errorf("lastStatus %q 不支持（必须 queued|failed）", s.LastStatus)
	}
	if s.LastReadiness != nil {
		switch s.LastReadiness.State {
		case "started", "workflow-invalid", "target-required", "credential-required",
			"environment-unavailable", "not-started", "failed":
		default:
			return fmt.Errorf("lastReadiness.state %q 不支持", s.LastReadiness.State)
		}
	}
	return nil
}
