package installed

import (
	"bytes"
	"context"
	"errors"
	"image/png"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/lxn/win"
	"github.com/yottaapp/yotta/internal/appcontrol"
	"github.com/yottaapp/yotta/internal/services/inputclip"
	"github.com/yottaapp/yotta/internal/services/recording"
	"github.com/yottaapp/yotta/pkg/winutil"
)

const (
	nativeFixtureTitle      = "Yotta Native Fixture  "
	nativeFixtureOtherTitle = "Yotta Native Fixture Other"
	nativeFixtureClass      = "YottaNativeAutomationFixture"
)

type nativeFixtureWindows struct {
	primary   win.HWND
	secondary win.HWND
	done      <-chan struct{}
}

type nativeFixtureEvent struct {
	message uint32
	wParam  uintptr
}

var nativeFixtureState struct {
	sync.Mutex
	events    []nativeFixtureEvent
	remaining atomic.Int32
}

func nativeFixtureWindowProc(hwnd win.HWND, message uint32, wParam, lParam uintptr) uintptr {
	switch message {
	case win.WM_KEYDOWN, win.WM_KEYUP, win.WM_CHAR,
		win.WM_MOUSEMOVE, win.WM_LBUTTONDOWN, win.WM_LBUTTONUP, win.WM_MOUSEWHEEL:
		// Publish the observation only after default handling completes. Otherwise
		// the test can inject a release while the fixture is still processing the
		// corresponding button-down message.
		result := win.DefWindowProc(hwnd, message, wParam, lParam)
		nativeFixtureState.Lock()
		nativeFixtureState.events = append(nativeFixtureState.events, nativeFixtureEvent{message: message, wParam: wParam})
		nativeFixtureState.Unlock()
		return result
	case win.WM_CLOSE:
		win.DestroyWindow(hwnd)
		return 0
	case win.WM_DESTROY:
		if nativeFixtureState.remaining.Add(-1) == 0 {
			win.PostQuitMessage(0)
		}
		return 0
	}
	return win.DefWindowProc(hwnd, message, wParam, lParam)
}

func startNativeFixture(t *testing.T) nativeFixtureWindows {
	t.Helper()
	type startResult struct {
		windows nativeFixtureWindows
		err     error
	}
	nativeFixtureState.Lock()
	nativeFixtureState.events = nil
	nativeFixtureState.Unlock()
	ready := make(chan startResult, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		className, err := syscall.UTF16PtrFromString(nativeFixtureClass)
		if err != nil {
			ready <- startResult{err: err}
			return
		}
		instance := win.GetModuleHandle(nil)
		windowClass := win.WNDCLASSEX{
			CbSize: uint32(unsafe.Sizeof(win.WNDCLASSEX{})), LpfnWndProc: syscall.NewCallback(nativeFixtureWindowProc),
			HInstance: instance, HbrBackground: win.HBRUSH(win.COLOR_WINDOW + 1), LpszClassName: className,
		}
		if win.RegisterClassEx(&windowClass) == 0 {
			ready <- startResult{err: errors.New("register native fixture window class")}
			return
		}
		defer win.UnregisterClass(className)
		create := func(title string, x int32) (win.HWND, error) {
			windowTitle, encodeErr := syscall.UTF16PtrFromString(title)
			if encodeErr != nil {
				return 0, encodeErr
			}
			hwnd := win.CreateWindowEx(0, className, windowTitle, win.WS_OVERLAPPEDWINDOW|win.WS_VISIBLE,
				x, 120, 640, 420, 0, 0, instance, nil)
			if hwnd == 0 {
				return 0, errors.New("create native fixture window")
			}
			win.ShowWindow(hwnd, win.SW_SHOW)
			win.UpdateWindow(hwnd)
			return hwnd, nil
		}
		primary, err := create(nativeFixtureTitle, 120)
		if err != nil {
			ready <- startResult{err: err}
			return
		}
		secondary, err := create(nativeFixtureOtherTitle, 220)
		if err != nil {
			win.DestroyWindow(primary)
			ready <- startResult{err: err}
			return
		}
		nativeFixtureState.remaining.Store(2)
		ready <- startResult{windows: nativeFixtureWindows{primary: primary, secondary: secondary, done: done}}
		var message win.MSG
		for win.GetMessage(&message, 0, 0, 0) > 0 {
			win.TranslateMessage(&message)
			win.DispatchMessage(&message)
		}
	}()
	select {
	case result := <-ready:
		if result.err != nil {
			t.Fatal(result.err)
		}
		t.Cleanup(func() {
			win.PostMessage(result.windows.primary, win.WM_CLOSE, 0, 0)
			win.PostMessage(result.windows.secondary, win.WM_CLOSE, 0, 0)
			select {
			case <-result.windows.done:
			case <-time.After(3 * time.Second):
				t.Error("native fixture did not close")
			}
		})
		return result.windows
	case <-time.After(3 * time.Second):
		t.Fatal("native fixture did not start")
		return nativeFixtureWindows{}
	}
}

func setNativeFixtureTitle(t *testing.T, hwnd win.HWND, title string) {
	t.Helper()
	encoded, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		t.Fatal(err)
	}
	if result := win.SendMessage(hwnd, 0x000C, 0, uintptr(unsafe.Pointer(encoded))); result == 0 {
		metadata, metadataErr := winutil.WindowMetadata(uintptr(hwnd))
		if metadataErr != nil || metadata.Title != title {
			t.Fatalf("set native fixture title: metadata=%#v error=%v", metadata, metadataErr)
		}
	}
}

func nativeFixtureMark() int {
	nativeFixtureState.Lock()
	defer nativeFixtureState.Unlock()
	return len(nativeFixtureState.events)
}

func waitNativeFixtureEvents(t *testing.T, mark int, predicate func([]nativeFixtureEvent) bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	var observed []nativeFixtureEvent
	for time.Now().Before(deadline) {
		nativeFixtureState.Lock()
		events := append([]nativeFixtureEvent(nil), nativeFixtureState.events[mark:]...)
		nativeFixtureState.Unlock()
		observed = events
		if predicate(events) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("native fixture did not receive the expected input events; observed=%+v", observed)
}

func hasNativeFixtureEvent(events []nativeFixtureEvent, message uint32, wParam uintptr) bool {
	for _, event := range events {
		if event.message == message && (wParam == 0 || event.wParam == wParam) {
			return true
		}
	}
	return false
}

func nativeFixtureProfile(t *testing.T, inspection appcontrol.ExecutableInspection, title, titleMatch, selection string) Profile {
	t.Helper()
	profile, err := SealProfile(NewDesktopProfileDraft(DesktopProfilePayload{
		Application: appcontrol.ProfileDraft{Executable: inspection.Executable, Arguments: []string{}},
		WindowTitle: title, WindowTitleMatch: titleMatch, WindowSelection: selection, WindowClass: nativeFixtureClass,
		InputBackend: "sendinput", CaptureBackend: "gdi", ResolveTimeoutMilliseconds: 500,
	}))
	if err != nil {
		t.Fatal(err)
	}
	return profile
}

func waitNativeWindowState(t *testing.T, reader windowStateReader, want string) WindowStateResponse {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for {
		state, err := reader.WindowState(ctx)
		if err == nil && state.State == want {
			return state
		}
		select {
		case <-ctx.Done():
			t.Fatalf("wait window state %q: last error=%v", want, err)
			return WindowStateResponse{}
		case <-time.After(20 * time.Millisecond):
		}
	}
}

func TestNativeWindowsDriverEndToEnd(t *testing.T) {
	if os.Getenv("YOTTA_WINDOWS_NATIVE_SMOKE") != "1" {
		t.Skip("set YOTTA_WINDOWS_NATIVE_SMOKE=1 to run desktop input smoke")
	}
	windows := startNativeFixture(t)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	inspection, err := appcontrol.InspectExecutable(executable)
	if err != nil {
		t.Fatal(err)
	}
	profile := nativeFixtureProfile(t, inspection, nativeFixtureTitle, "exact", "unique")
	driver, err := newPlatformDriver(profile)
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resolved, err := driver.ResolveTarget(ctx)
	if err != nil || resolved.Ref.HWND != uintptr(windows.primary) {
		t.Fatalf("resolve exact trailing-space title: target=%#v error=%v", resolved, err)
	}
	metadata, err := winutil.WindowMetadata(uintptr(windows.primary))
	if err != nil || metadata.Title != nativeFixtureTitle || metadata.Class != nativeFixtureClass {
		t.Fatalf("exact native metadata=%#v error=%v", metadata, err)
	}

	setNativeFixtureTitle(t, windows.primary, "Yotta Native Fixture Dynamic  ")
	regexDriver, err := newPlatformDriver(nativeFixtureProfile(t, inspection, `^Yotta Native Fixture Dynamic  $`, "regex", "unique"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := regexDriver.ResolveTarget(ctx); err != nil {
		t.Fatalf("resolve dynamic regex title: %v", err)
	}
	if err := regexDriver.Close(); err != nil {
		t.Fatal(err)
	}
	setNativeFixtureTitle(t, windows.primary, nativeFixtureTitle)
	setNativeFixtureTitle(t, windows.secondary, nativeFixtureTitle)
	if _, err := driver.ResolveTarget(ctx); err == nil {
		t.Fatal("unique selection accepted two matching windows")
	} else {
		var failure *Failure
		if !errors.As(err, &failure) || failure.Code != CodeTargetAmbiguous {
			t.Fatalf("multi-window error=%v", err)
		}
	}
	setNativeFixtureTitle(t, windows.secondary, nativeFixtureOtherTitle)

	mark := nativeFixtureMark()
	if err := driver.Execute(ctx, OperationPressKeys, PressKeysRequest{Keys: []string{"F8"}, DurationMilliseconds: 10}); err != nil {
		t.Fatalf("press keys: %v", err)
	}
	waitNativeFixtureEvents(t, mark, func(events []nativeFixtureEvent) bool {
		return hasNativeFixtureEvent(events, win.WM_KEYDOWN, win.VK_F8) && hasNativeFixtureEvent(events, win.WM_KEYUP, win.VK_F8)
	})
	mark = nativeFixtureMark()
	if err := driver.Execute(ctx, OperationTypeText, TypeTextRequest{Text: "Y"}); err != nil {
		t.Fatalf("type text: %v", err)
	}
	waitNativeFixtureEvents(t, mark, func(events []nativeFixtureEvent) bool {
		return hasNativeFixtureEvent(events, win.WM_CHAR, 'Y')
	})
	mark = nativeFixtureMark()
	if err := driver.Execute(ctx, OperationMove, MoveRequest{Point: Point{X: 0.3, Y: 0.3, Unit: "ratio"}}); err != nil {
		t.Fatalf("move pointer: %v", err)
	}
	if err := driver.Execute(ctx, OperationClick, ClickRequest{Point: Point{X: 0.3, Y: 0.3, Unit: "ratio"}, Button: "left", DurationMilliseconds: 10}); err != nil {
		t.Fatalf("click pointer: %v", err)
	}
	waitNativeFixtureEvents(t, mark, func(events []nativeFixtureEvent) bool {
		return hasNativeFixtureEvent(events, win.WM_MOUSEMOVE, 0) &&
			hasNativeFixtureEvent(events, win.WM_LBUTTONDOWN, 0) && hasNativeFixtureEvent(events, win.WM_LBUTTONUP, 0)
	})
	if err := driver.Execute(ctx, OperationMoveRelative, RelativeMoveRequest{DeltaX: 8, DeltaY: 4, DurationMilliseconds: 10}); err != nil {
		t.Fatalf("relative move: %v", err)
	}
	if err := driver.Execute(ctx, OperationScroll, ScrollRequest{Point: Point{X: 0.4, Y: 0.4, Unit: "ratio"}, Notches: 1}); err != nil {
		t.Fatalf("scroll: %v", err)
	}

	recorder := recording.NewRecorder()
	_, err = recorder.Start(uintptr(windows.primary), inputclip.ClipMeta{
		RecordingMode: inputclip.RecordingModePrecise, MouseMode: "absolute", BaseResolution: [2]int{metadata.ClientW, metadata.ClientH},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := driver.Execute(ctx, OperationDrag, DragRequest{
		From: Point{X: 0.2, Y: 0.2, Unit: "ratio"}, To: Point{X: 0.7, Y: 0.7, Unit: "ratio"},
		Button: "left", DurationMilliseconds: 80,
	}); err != nil {
		recorder.Cancel()
		t.Fatalf("drag: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	recorded, err := recorder.Stop()
	if err != nil {
		t.Fatal(err)
	}
	var recordedDown, recordedUp bool
	for _, event := range recorded.Events {
		recordedDown = recordedDown || event.Type == inputclip.EventTypeMouseBtnDown
		recordedUp = recordedUp || event.Type == inputclip.EventTypeMouseBtnUp
	}
	if !recordedDown || !recordedUp {
		t.Fatalf("SendInput drag did not enter the native hook: %#v", recorded.Events)
	}

	heldOpener, ok := driver.(heldInputOpener)
	if !ok {
		t.Fatal("Win32 driver does not expose held input")
	}
	held, err := heldOpener.OpenHeldInput()
	if err != nil {
		t.Fatal(err)
	}
	mark = nativeFixtureMark()
	if err := held.Execute(ctx, OperationHoldKeys, HoldKeysRequest{Keys: []string{"SHIFT"}}); err != nil {
		t.Fatalf("hold keys: %v", err)
	}
	waitNativeFixtureEvents(t, mark, func(events []nativeFixtureEvent) bool {
		return hasNativeFixtureEvent(events, win.WM_KEYDOWN, win.VK_SHIFT)
	})
	if err := held.Close(); err != nil {
		t.Fatalf("release held keys: %v", err)
	}
	waitNativeFixtureEvents(t, mark, func(events []nativeFixtureEvent) bool {
		return hasNativeFixtureEvent(events, win.WM_KEYUP, win.VK_SHIFT)
	})
	held, err = heldOpener.OpenHeldInput()
	if err != nil {
		t.Fatal(err)
	}
	mark = nativeFixtureMark()
	if err := held.Execute(ctx, OperationHoldButton, HoldButtonRequest{Point: Point{X: 0.5, Y: 0.5, Unit: "ratio"}, Button: "left"}); err != nil {
		t.Fatalf("hold button: %v", err)
	}
	waitNativeFixtureEvents(t, mark, func(events []nativeFixtureEvent) bool {
		return hasNativeFixtureEvent(events, win.WM_LBUTTONDOWN, 0)
	})
	if err := held.Close(); err != nil {
		t.Fatalf("release held button: %v", err)
	}
	waitNativeFixtureEvents(t, mark, func(events []nativeFixtureEvent) bool {
		return hasNativeFixtureEvent(events, win.WM_LBUTTONUP, 0)
	})

	mark = nativeFixtureMark()
	if err := driver.PlayEvent(ctx, PlaybackEvent{Kind: PlaybackKeyDown, KeyCode: win.VK_F7}); err != nil {
		t.Fatal(err)
	}
	if err := driver.PlayEvent(ctx, PlaybackEvent{Kind: PlaybackKeyUp, KeyCode: win.VK_F7}); err != nil {
		t.Fatal(err)
	}
	if err := driver.ReleaseInput(); err != nil {
		t.Fatal(err)
	}
	waitNativeFixtureEvents(t, mark, func(events []nativeFixtureEvent) bool {
		return hasNativeFixtureEvent(events, win.WM_KEYDOWN, win.VK_F7) && hasNativeFixtureEvent(events, win.WM_KEYUP, win.VK_F7)
	})

	stateReader, ok := driver.(windowStateReader)
	if !ok {
		t.Fatal("Win32 driver does not expose window state")
	}
	if err := driver.Execute(ctx, OperationMoveResizeWindow, MoveResizeWindowRequest{X: 160, Y: 140, Width: 620, Height: 400}); err != nil {
		t.Fatalf("move/resize window: %v", err)
	}
	state := waitNativeWindowState(t, stateReader, "normal")
	if state.Width != 620 || state.Height != 400 {
		t.Fatalf("moved window state=%#v", state)
	}
	if err := driver.Execute(ctx, OperationSetWindowState, SetWindowStateRequest{State: "minimize"}); err != nil {
		t.Fatalf("minimize window: %v", err)
	}
	waitNativeWindowState(t, stateReader, "minimized")
	if err := driver.Execute(ctx, OperationSetWindowState, SetWindowStateRequest{State: "restore"}); err != nil {
		t.Fatalf("restore window: %v", err)
	}
	waitNativeWindowState(t, stateReader, "normal")
	if err := driver.Execute(ctx, OperationSetWindowState, SetWindowStateRequest{State: "maximize"}); err != nil {
		t.Fatalf("maximize window: %v", err)
	}
	waitNativeWindowState(t, stateReader, "maximized")
	if err := driver.Execute(ctx, OperationSetWindowState, SetWindowStateRequest{State: "restore"}); err != nil {
		t.Fatalf("restore maximized window: %v", err)
	}
	waitNativeWindowState(t, stateReader, "normal")

	captured, err := driver.Capture(ctx)
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	image, err := png.Decode(bytes.NewReader(captured))
	if err != nil || image.Bounds().Dx() <= 0 || image.Bounds().Dy() <= 0 {
		t.Fatalf("captured PNG bounds=%v error=%v", image.Bounds(), err)
	}
	waiter, ok := driver.(windowWaiter)
	if !ok {
		t.Fatal("Win32 driver does not expose window waiting")
	}
	if matched, err := waiter.WaitWindow(ctx, true, 250*time.Millisecond); err != nil || !matched {
		t.Fatalf("wait window matched=%v error=%v", matched, err)
	}
	if err := driver.Execute(ctx, OperationCloseWindow, struct{}{}); err != nil {
		t.Fatalf("close window: %v", err)
	}
	if matched, err := waiter.WaitWindow(ctx, false, time.Second); err != nil || !matched {
		t.Fatalf("wait window gone matched=%v error=%v", matched, err)
	}
}
