package main

import (
	"reflect"
	"testing"
)

func TestWorkflowEditorUIFailures(t *testing.T) {
	t.Run("accepts the dark non-overlapping Nuxt UI flow", func(t *testing.T) {
		failures := workflowEditorUIFailures(
			pageState{GraphChromeDark: true, RunStarted: true},
			pageState{ConfirmDialog: true},
			pageState{SaveInlineFeedback: true},
		)
		if len(failures) != 0 {
			t.Fatalf("unexpected failures: %v", failures)
		}
	})

	t.Run("reports every guarded regression", func(t *testing.T) {
		failures := workflowEditorUIFailures(
			pageState{HandleOverlaps: 8},
			pageState{NativeConfirmCalls: 1},
			pageState{SaveToast: true},
		)
		want := []string{
			"Vue Flow controls or minimap use a light background",
			"8 workflow handles overlap their labels",
			"new workflow omitted the RunStarted root",
			"workflow navigation called window.confirm",
			"workflow navigation did not open the shared confirm dialog",
			"workflow save displayed a success toast",
			"workflow save omitted inline success feedback",
		}
		if !reflect.DeepEqual(failures, want) {
			t.Fatalf("failures = %v, want %v", failures, want)
		}
	})
}
