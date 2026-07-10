package hotkey

import "context"

func newTestHotkeyManager() *HotkeyManager {
	manager := NewHotkeyManager()
	manager.runLoop = runTestHotkeyLoop
	return manager
}

func runTestHotkeyLoop(ctx context.Context, _ []HotkeySpec, _ func(int), ready chan<- error, done chan<- struct{}) {
	defer close(done)
	ready <- nil
	<-ctx.Done()
}
