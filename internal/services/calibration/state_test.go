package calibration

import "testing"

func TestResetClearsCalibrationSnapshot(t *testing.T) {
	absDx.Store(12)
	absDy.Store(34)
	live.Store(true)
	t.Cleanup(func() {
		Reset()
		live.Store(false)
	})

	Reset()
	state := Get()
	if state.AbsDx != 0 || state.AbsDy != 0 || !state.Active {
		t.Fatalf("Get() after Reset = %#v, want active with zero counts", state)
	}
}
