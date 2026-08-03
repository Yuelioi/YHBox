//go:build windows

package input

import (
	"strings"
	"testing"
	"time"

	"github.com/lxn/win"
)

func TestPostMessageClickHoldsBeforeUpAndSettlesAfter(t *testing.T) {
	var events []string
	err := performPostMessageClick(50*time.Millisecond, 30*time.Millisecond,
		func() error { events = append(events, "down"); return nil },
		func() error { events = append(events, "up"); return nil },
		func(duration time.Duration) { events = append(events, "sleep:"+duration.String()) },
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(events, ","); got != "down,sleep:50ms,up,sleep:30ms" {
		t.Fatalf("PostMessage click sequence = %q", got)
	}
}

func TestPostMessageActivationRefreshUsesKeepaliveWindow(t *testing.T) {
	now := time.Date(2026, 8, 1, 2, 0, 0, 0, time.UTC)
	if !postMessageActivationDue(time.Time{}, now) {
		t.Fatal("first input did not require activation")
	}
	if postMessageActivationDue(now.Add(-postMessageActivationKeepalive/2), now) {
		t.Fatal("activation refreshed inside its keepalive window")
	}
	if !postMessageActivationDue(now.Add(-postMessageActivationKeepalive), now) {
		t.Fatal("activation did not refresh after its keepalive window")
	}
}

func TestPostMessageBackend_NameAndCapabilities(t *testing.T) {
	b := newPostMessageBackend()
	if b.Name() != "postmessage" {
		t.Errorf("Name() = %q, want postmessage", b.Name())
	}
	caps := b.Capabilities()
	if !caps.BackgroundInput {
		t.Error("PostMessage should support BackgroundInput")
	}
	if caps.GlobalInput {
		t.Error("PostMessage should NOT be global input")
	}
}

func TestPostMessageBackend_ReleaseAllEmpty(t *testing.T) {
	b := newPostMessageBackend()
	// 没 down 过任何东西, ReleaseAll 不能 panic
	if err := b.ReleaseAll(); err != nil {
		t.Errorf("empty ReleaseAll should not error: %v", err)
	}
}

func TestPostMessageBackend_RejectsDeadTargetWithoutTrackingState(t *testing.T) {
	b := newPostMessageBackend()
	fakeHwnd := win.HWND(0xDEADBEEF) // 假 hwnd
	if err := b.KeyDown(fakeHwnd, "W"); err == nil {
		t.Fatal("KeyDown accepted a dead target")
	}
	b.mu.Lock()
	_, hasW := b.heldKeys[VK("W")]
	b.mu.Unlock()
	if hasW {
		t.Error("failed KeyDown must not track held state")
	}
}

// TypeText 走 PostMessage WM_CHAR (PostText) —— targeted hwnd, 不依赖前台焦点.
// 死 hwnd 必须返回错误，禁止节点把未投递的字符报告成成功。
func TestPostMessageBackend_TypeTextRejectsDeadTarget(t *testing.T) {
	b := newPostMessageBackend()
	fakeHwnd := win.HWND(0xDEADBEEF)
	// ASCII + CJK (BMP 内) + emoji (BMP 外, surrogate pair)
	if err := b.TypeText(fakeHwnd, "abc你好😀"); err == nil {
		t.Error("TypeText accepted a dead target")
	}
	// 空串也不能炸
	if err := b.TypeText(fakeHwnd, ""); err != nil {
		t.Errorf("TypeText(empty) returned err: %v", err)
	}
}

func TestNewBackend_PostMessage(t *testing.T) {
	b, err := NewBackend("postmessage")
	if err != nil {
		t.Fatalf("NewBackend postmessage err: %v", err)
	}
	if b.Name() != "postmessage" {
		t.Errorf("got %q", b.Name())
	}
}

func TestNewBackend_Unknown(t *testing.T) {
	if _, err := NewBackend("xxx"); err == nil {
		t.Error("unknown backend should err")
	}
}

func TestPostMessageRejectsInvalidKeysAndDeadWindowOperations(t *testing.T) {
	b := newPostMessageBackend()
	for _, invoke := range []func() error{
		func() error { return b.KeyDown(0, "missing") },
		func() error { return b.KeyUp(0, "missing") },
		func() error { return b.KeyDownCode(0, 0) },
		func() error { return b.KeyUpCode(0, 256) },
		func() error { return b.MouseUp(0, "left") },
		func() error { return b.MouseMoveRel(0, 1, -1, 0) },
		func() error { return b.MoveTo(0, 0.5, 0.5) },
		func() error { return b.Scroll(0, 0.5, 0.5, 1, false) },
		func() error { return b.Click(0, 0.5, 0.5, "left", 1) },
	} {
		if err := invoke(); err == nil {
			t.Fatal("dead window operation succeeded")
		}
	}
	if _, _, err := b.CursorRatio(0); err == nil {
		t.Fatal("CursorRatio accepted an empty client rect")
	}
}

func TestPostMessageButtonMappingsAreComplete(t *testing.T) {
	for _, test := range []struct {
		name        string
		button      MouseButton
		down, up    uint32
		downFlags   uintptr
		selectedKey string
	}{
		{name: "left", button: MouseLeft, down: WM_LBUTTONDOWN, up: WM_LBUTTONUP, downFlags: MK_LBUTTON, selectedKey: "unknown"},
		{name: "right", button: MouseRight, down: WM_RBUTTONDOWN, up: WM_RBUTTONUP, downFlags: MK_RBUTTON, selectedKey: "right"},
		{name: "middle", button: MouseMiddle, down: WM_MBUTTONDOWN, up: WM_MBUTTONUP, downFlags: MK_MBUTTON, selectedKey: "middle"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := pickButton(test.selectedKey); got != test.button {
				t.Fatalf("pickButton() = %v, want %v", got, test.button)
			}
			down, flags := postMessageButton(test.selectedKey, false)
			up, upFlags := postMessageButton(test.selectedKey, true)
			if down != test.down || up != test.up || flags != test.downFlags || upFlags != 0 {
				t.Fatalf("messages down=%#x/%#x up=%#x/%#x", down, flags, up, upFlags)
			}
		})
	}
	if _, _, err := checkedClientSize(0); err == nil || !strings.Contains(err.Error(), "GetClientRect") {
		t.Fatalf("checkedClientSize(0) error = %v", err)
	}
}
