package schedule

import (
	"encoding/json"
	"testing"
)

func TestSchedule_JSONRoundTrip(t *testing.T) {
	src := Schedule{
		SchemaVersion:  1,
		ID:             "s1",
		Name:           "日常",
		Enabled:        true,
		Targets:        []TargetRef{{Kind: "workflow", ID: "c-A"}, {Kind: "workflow", ID: "c-B"}},
		Trigger:        Trigger{Kind: "cron", SubKind: "daily", At: "05:00"},
		TimeoutMinutes: 30,
		OnError:        "stop",
	}
	b, err := json.Marshal(src)
	if err != nil {
		t.Fatal(err)
	}
	var got Schedule
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Name != "日常" || len(got.Targets) != 2 || got.Trigger.At != "05:00" {
		t.Errorf("round-trip lost: %+v", got)
	}
}
