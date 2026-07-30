package schedule

import (
	"encoding/json"
	"testing"
)

func TestSchedule_JSONRoundTrip(t *testing.T) {
	src := Schedule{
		SchemaVersion:  CurrentSchemaVersion,
		ID:             "s1",
		Name:           "日常",
		Description:    "每天运行",
		Category:       "日常任务",
		Tags:           []string{"自动", "早晨"},
		Enabled:        true,
		Targets:        []TargetRef{{Kind: TargetWorkflow, ID: "c-A"}, {Kind: TargetWorkflow, ID: "c-B"}},
		Trigger:        Trigger{Kind: TriggerCron, SubKind: CronDaily, At: "05:00"},
		TimeoutMinutes: 30,
		OnError:        OnErrorStop,
	}
	b, err := json.Marshal(src)
	if err != nil {
		t.Fatal(err)
	}
	var got Schedule
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Name != "日常" || got.Description != "每天运行" || got.Category != "日常任务" ||
		len(got.Tags) != 2 || len(got.Targets) != 2 || got.Trigger.At != "05:00" {
		t.Errorf("round-trip lost: %+v", got)
	}
}
