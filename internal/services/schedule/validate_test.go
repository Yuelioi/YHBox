package schedule

import (
	"strings"
	"testing"
)

func TestValidate_OK_Cron(t *testing.T) {
	s := &Schedule{
		SchemaVersion: CurrentSchemaVersion, ID: "x", Name: "ok",
		Targets: []TargetRef{{Kind: TargetWorkflow, ID: "c1"}},
		Trigger: Trigger{Kind: TriggerCron, SubKind: CronDaily, At: "05:00"},
		OnError: OnErrorStop,
	}
	if err := s.Validate(); err != nil {
		t.Errorf("expected valid: %v", err)
	}
}

func TestValidate_OK_Manual(t *testing.T) {
	s := &Schedule{
		SchemaVersion: CurrentSchemaVersion, ID: "x", Name: "ok",
		Targets: []TargetRef{{Kind: TargetWorkflow, ID: "c1"}},
		Trigger: Trigger{Kind: TriggerManual},
		OnError: OnErrorStop,
	}
	if err := s.Validate(); err != nil {
		t.Errorf("expected valid: %v", err)
	}
}

func TestValidate_EmptyTargets(t *testing.T) {
	s := &Schedule{SchemaVersion: CurrentSchemaVersion, ID: "x", Name: "ok", Trigger: Trigger{Kind: TriggerManual}, OnError: OnErrorStop}
	err := s.Validate()
	if err == nil || !strings.Contains(err.Error(), "targets") {
		t.Errorf("expected targets error: %v", err)
	}
}

func TestValidate_BadTriggerKind(t *testing.T) {
	s := &Schedule{
		SchemaVersion: CurrentSchemaVersion, ID: "x", Name: "ok",
		Targets: []TargetRef{{Kind: TargetWorkflow, ID: "c1"}},
		Trigger: Trigger{Kind: "weird"},
		OnError: OnErrorStop,
	}
	err := s.Validate()
	if err == nil || !strings.Contains(err.Error(), "trigger") {
		t.Errorf("expected trigger error: %v", err)
	}
}

func TestValidate_CronDailyBadTime(t *testing.T) {
	cases := []string{"5:00", "25:00", "5", "05:60", "abc", ""}
	for _, at := range cases {
		s := &Schedule{
			SchemaVersion: CurrentSchemaVersion, ID: "x", Name: "ok",
			Targets: []TargetRef{{Kind: TargetWorkflow, ID: "c1"}},
			Trigger: Trigger{Kind: TriggerCron, SubKind: CronDaily, At: at},
			OnError: OnErrorStop,
		}
		if err := s.Validate(); err == nil {
			t.Errorf("at=%q should fail", at)
		}
	}
}

func TestValidate_CronIntervalBadMinutes(t *testing.T) {
	s := &Schedule{
		SchemaVersion: CurrentSchemaVersion, ID: "x", Name: "ok",
		Targets: []TargetRef{{Kind: TargetWorkflow, ID: "c1"}},
		Trigger: Trigger{Kind: TriggerCron, SubKind: CronInterval, EveryMinutes: 0},
		OnError: OnErrorStop,
	}
	if err := s.Validate(); err == nil {
		t.Error("interval=0 should fail")
	}
}

func TestValidate_BadOnError(t *testing.T) {
	s := &Schedule{
		SchemaVersion: CurrentSchemaVersion, ID: "x", Name: "ok",
		Targets: []TargetRef{{Kind: TargetWorkflow, ID: "c1"}},
		Trigger: Trigger{Kind: TriggerManual},
		OnError: "weird",
	}
	if err := s.Validate(); err == nil {
		t.Error("bad onError should fail")
	}
}

func TestValidate_NegativeTargetInterval(t *testing.T) {
	s := &Schedule{
		SchemaVersion: CurrentSchemaVersion, ID: "x", Name: "ok",
		Targets:               []TargetRef{{Kind: TargetWorkflow, ID: "c1"}},
		Trigger:               Trigger{Kind: TriggerManual},
		TargetIntervalSeconds: -1,
		OnError:               OnErrorStop,
	}
	if err := s.Validate(); err == nil || !strings.Contains(err.Error(), "targetIntervalSeconds") {
		t.Fatalf("Validate error = %v, want target interval rejection", err)
	}
}
