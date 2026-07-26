package recording

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/services/inputclip"
	inputdriver "github.com/yottaapp/yotta/pkg/input"
	"github.com/yottaapp/yotta/pkg/winutil"
)

func TestNativeRecorderProducesCanonicalEncodableInput(t *testing.T) {
	if os.Getenv("YOTTA_WINDOWS_NATIVE_SMOKE") != "1" {
		t.Skip("set YOTTA_WINDOWS_NATIVE_SMOKE=1 to run desktop input smoke")
	}
	hwnd := winutil.ForegroundWindow()
	if hwnd == 0 {
		t.Fatal("native recording feedback loop requires a foreground window")
	}
	window, err := winutil.WindowMetadata(hwnd)
	if err != nil || window.ClientW <= 0 || window.ClientH <= 0 {
		t.Fatalf("inspect foreground window: window=%#v error=%v", window, err)
	}
	recorder := NewRecorder()
	_, err = recorder.Start(hwnd, inputclip.ClipMeta{
		RecordingMode:  inputclip.RecordingModePrecise,
		MouseMode:      "absolute",
		BaseResolution: [2]int{window.ClientW, window.ClientH},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if recorder.Active() {
			recorder.Cancel()
		}
	}()

	backend, err := inputdriver.NewBackend("sendinput")
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()
	defer backend.ReleaseAll()
	const vkShift = uint32(0x10)
	for index := 0; index < 512; index++ {
		if err := backend.KeyDownCode(0, vkShift); err != nil {
			t.Fatalf("inject shift down: %v", err)
		}
		if err := backend.KeyUpCode(0, vkShift); err != nil {
			t.Fatalf("inject shift up: %v", err)
		}
	}
	time.Sleep(50 * time.Millisecond)
	result, err := recorder.Stop()
	if err != nil {
		t.Fatal(err)
	}
	if err := canonicalizeStopResult(result); err != nil {
		t.Fatalf("canonicalize native recording: %v", err)
	}
	if len(result.Events) == 0 || result.Events[0].TUs != 0 {
		t.Fatalf("canonical native events = %#v", result.Events)
	}
	clip := &inputclip.InputClip{Meta: result.Meta, Events: result.Events}
	clip.UpdateDuration()
	var carrier bytes.Buffer
	if err := inputclip.Encode(&carrier, clip); err != nil {
		t.Fatalf("encode native recording: %v", err)
	}
	if _, err := inputclip.Decode(bytes.NewReader(carrier.Bytes())); err != nil {
		t.Fatalf("decode native recording: %v", err)
	}
	root := t.TempDir()
	assets, _ := newRecordingAssetStore(t, root)
	clips := inputclip.NewService(assets)
	clip.ID = "clip-native-recorder"
	clip.Label = "Native recorder"
	if err := clips.Save(clip); err != nil {
		t.Fatalf("save native recording asset: %v", err)
	}
	loaded, err := clips.Get(clip.ID)
	if err != nil {
		t.Fatalf("reload native recording asset: %v", err)
	}
	if len(loaded.Events) != len(clip.Events) || loaded.Blob != clip.Blob || len(clips.List()) != 1 {
		t.Fatalf("reloaded native clip=%#v events=%d", loaded, len(loaded.Events))
	}
	for _, event := range loaded.Events[:min(2, len(loaded.Events))] {
		var replayErr error
		switch event.Type {
		case inputclip.EventTypeKeyDown:
			replayErr = backend.KeyDownCode(0, uint32(event.A))
		case inputclip.EventTypeKeyUp:
			replayErr = backend.KeyUpCode(0, uint32(event.A))
		}
		if replayErr != nil {
			t.Fatalf("replay reloaded native event: %v", replayErr)
		}
	}
	if err := backend.ReleaseAll(); err != nil {
		t.Fatalf("release reloaded native clip input: %v", err)
	}
	if _, err := assets.ReadBlob(context.Background(), loaded.Blob); err != nil {
		t.Fatalf("read reloaded native carrier: %v", err)
	}
}
