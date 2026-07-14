package main

import "testing"

func TestContainerChangeListenerRefreshesRuntimeAndNotifiesWebviews(t *testing.T) {
	refreshed := 0
	emitted := ""
	listener := containerChangeListener(
		func() { refreshed++ },
		func(name string, _ any) { emitted = name },
	)

	listener()

	if refreshed != 1 {
		t.Fatalf("refresh count = %d, want 1", refreshed)
	}
	if emitted != "container:changed" {
		t.Fatalf("emitted event = %q, want container:changed", emitted)
	}
}
