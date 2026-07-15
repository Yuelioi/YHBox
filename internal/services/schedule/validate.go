package schedule

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var hhmmRE = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

func (s *Schedule) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return errors.New("schedule name 不能为空")
	}
	if s.SchemaVersion != CurrentSchemaVersion {
		return fmt.Errorf("schemaVersion %d 不支持", s.SchemaVersion)
	}
	if len(s.Targets) == 0 {
		return errors.New("targets 不能为空")
	}
	for i, t := range s.Targets {
		if t.Kind != "workflow" {
			return fmt.Errorf("targets[%d].kind 必须 \"workflow\"，got %q", i, t.Kind)
		}
		if t.ID == "" {
			return fmt.Errorf("targets[%d].id 不能为空", i)
		}
	}
	switch s.Trigger.Kind {
	case "cron":
		switch s.Trigger.SubKind {
		case "daily":
			if !hhmmRE.MatchString(s.Trigger.At) {
				return fmt.Errorf("trigger.at %q 不合法（必须 HH:MM 24h）", s.Trigger.At)
			}
		case "interval":
			if s.Trigger.EveryMinutes <= 0 {
				return errors.New("trigger.everyMinutes 必须 > 0")
			}
		default:
			return fmt.Errorf("trigger.subKind %q 不支持（cron 下必须 daily|interval）", s.Trigger.SubKind)
		}
	case "hotkey":
		if strings.TrimSpace(s.Trigger.Hotkey) == "" {
			return errors.New("trigger.hotkey 不能为空")
		}
	case "once", "manual":
	default:
		return fmt.Errorf("trigger.kind %q 不支持（必须 cron|hotkey|once|manual）", s.Trigger.Kind)
	}
	switch s.OnError {
	case "stop", "continue":
	default:
		return fmt.Errorf("onError %q 不支持（必须 stop|continue）", s.OnError)
	}
	if s.TimeoutMinutes < 0 {
		return errors.New("timeoutMinutes 不能为负")
	}
	return nil
}

func (s *Schedule) Normalize() {
	if s.SchemaVersion == 0 {
		s.SchemaVersion = CurrentSchemaVersion
	}
	if s.OnError == "" {
		s.OnError = "stop"
	}
}
