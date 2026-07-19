package macro

import (
	"fmt"
	"testing"

	"github.com/yottaapp/yotta/internal/services/inputclip"
)

func TestAnalyzePreservesOverlappingHeldKeys(t *testing.T) {
	document := Document{SchemaVersion: SchemaVersion, BaseResolution: [2]int{1920, 1080}, Actions: []Action{
		{ID: "w-down", Kind: ActionKeyDown, Key: "W"},
		{ID: "gap-1", Kind: ActionSleep, DurationUs: 50_000},
		{ID: "d-down", Kind: ActionKeyDown, Key: "D"},
		{ID: "gap-2", Kind: ActionSleep, DurationUs: 450_000},
		{ID: "w-up", Kind: ActionKeyUp, Key: "W"},
		{ID: "gap-3", Kind: ActionSleep, DurationUs: 100_000},
		{ID: "d-up", Kind: ActionKeyUp, Key: "D"},
	}}
	analysis := Analyze(document)
	if len(analysis.Issues) != 0 || analysis.DurationUs != 600_000 {
		t.Fatalf("analysis = %+v", analysis)
	}
	if got := fmt.Sprint(analysis.HeldAfter[2].Keys); got != "[D W]" {
		t.Fatalf("held after D down = %s", got)
	}
	if got := fmt.Sprint(analysis.HeldAfter[4].Keys); got != "[D]" {
		t.Fatalf("held after W up = %s", got)
	}
}

func TestAnalyzeRejectsBrokenHeldInput(t *testing.T) {
	document := Document{SchemaVersion: SchemaVersion, BaseResolution: [2]int{100, 100}, Actions: []Action{
		{ID: "up", Kind: ActionKeyUp, Key: "W"},
		{ID: "down", Kind: ActionKeyDown, Key: "D"},
		{ID: "down-again", Kind: ActionKeyDown, Key: "D"},
	}}
	analysis := Analyze(document)
	codes := map[string]bool{}
	for _, issue := range analysis.Issues {
		codes[issue.Code] = true
	}
	for _, code := range []string{"macro.key-not-down", "macro.key-already-down", "macro.held-at-end"} {
		if !codes[code] {
			t.Fatalf("missing %s in %+v", code, analysis.Issues)
		}
	}
}

func TestFromInputEventsKeepsAtomicOverlapAndSleeps(t *testing.T) {
	document, err := FromInputEvents([]inputclip.Event{
		{TUs: 0, Type: inputclip.EventTypeKeyDown, A: 'W'},
		{TUs: 50_000, Type: inputclip.EventTypeKeyDown, A: 'D'},
		{TUs: 500_000, Type: inputclip.EventTypeKeyUp, A: 'W'},
		{TUs: 600_000, Type: inputclip.EventTypeKeyUp, A: 'D'},
	}, [2]int{1280, 720})
	if err != nil {
		t.Fatal(err)
	}
	want := []ActionKind{ActionKeyDown, ActionSleep, ActionKeyDown, ActionSleep, ActionKeyUp, ActionSleep, ActionKeyUp}
	if len(document.Actions) != len(want) {
		t.Fatalf("actions = %+v", document.Actions)
	}
	for index, kind := range want {
		if document.Actions[index].Kind != kind {
			t.Fatalf("action %d = %s, want %s", index, document.Actions[index].Kind, kind)
		}
	}
	if document.Actions[1].DurationUs != 50_000 || document.Actions[3].DurationUs != 450_000 || document.Actions[5].DurationUs != 100_000 {
		t.Fatalf("sleeps = %+v", document.Actions)
	}
}

func TestAnalyzeRejectsInvalidTaggedUnionShape(t *testing.T) {
	document := Document{SchemaVersion: SchemaVersion, BaseResolution: [2]int{100, 100}, Actions: []Action{
		{ID: "bad", Kind: ActionSleep, Key: "A", DurationUs: 10},
	}}
	if err := Validate(document); err == nil {
		t.Fatal("invalid sleep shape was accepted")
	}
}
