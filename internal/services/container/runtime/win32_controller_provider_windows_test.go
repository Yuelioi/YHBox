//go:build windows

package runtime

import "testing"

type closeTrackingInput struct {
	*fakeInputBackend
	events *[]string
}

func (b *closeTrackingInput) ReleaseAll() error {
	*b.events = append(*b.events, "input.release-all")
	return nil
}

func (b *closeTrackingInput) Close() error {
	*b.events = append(*b.events, "input.close")
	return nil
}

type closeTrackingCapture struct {
	*mockCaptureBackend
	events *[]string
}

func (b *closeTrackingCapture) Close() error {
	*b.events = append(*b.events, "capture.close")
	return nil
}

func TestNativeWin32ControllerProviderCloseIsOrderedAndIdempotent(t *testing.T) {
	events := []string{}
	provider := &nativeWin32ControllerProvider{
		input: &closeTrackingInput{
			fakeInputBackend: &fakeInputBackend{},
			events:           &events,
		},
		capture: &closeTrackingCapture{
			mockCaptureBackend: &mockCaptureBackend{},
			events:             &events,
		},
	}

	if err := provider.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := provider.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
	want := []string{"input.release-all", "input.close", "capture.close"}
	if len(events) != len(want) {
		t.Fatalf("close events = %v, want %v", events, want)
	}
	for i := range want {
		if events[i] != want[i] {
			t.Fatalf("close events = %v, want %v", events, want)
		}
	}
}
