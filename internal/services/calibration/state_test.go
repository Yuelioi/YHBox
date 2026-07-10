package calibration

import "testing"

func TestResetClearsCalibrationSnapshot(t *testing.T) {
	currentState.addRelative(12, 34)
	currentState.setActive(true)
	t.Cleanup(func() {
		Reset()
		currentState.setActive(false)
	})

	Reset()
	state := Get()
	if state.AbsDx != 0 || state.AbsDy != 0 || !state.Active {
		t.Fatalf("Get() after Reset = %#v, want active with zero counts", state)
	}
}
