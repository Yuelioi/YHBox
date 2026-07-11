package input

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/node"
)

// sizeWindow — WindowService stub with fixed client size for px-coordinate tests.
// Different from recordingWindow (which has ClientSize 0,0); named sizeWindow so
// Task 4/6 can import or mirror it clearly.
type sizeWindow struct{ w, h int }

func (s sizeWindow) BringForeground() error { return nil }
func (s sizeWindow) HWND() uintptr          { return 0 }
func (s sizeWindow) ClientSize() (int, int, error) {
	return s.w, s.h, nil
}
func (s sizeWindow) SetActive(_ context.Context, _, _, _, _ string) error { return nil }
func (s sizeWindow) Maximize() error                                      { return nil }
func (s sizeWindow) Minimize() error                                      { return nil }
func (s sizeWindow) Restore() error                                       { return nil }
func (s sizeWindow) BorderlessFullscreen() error                          { return nil }
func (s sizeWindow) RestoreBorders() error                                { return nil }
func (s sizeWindow) MoveResize(_, _, _, _ int) error                      { return nil }
func (s sizeWindow) Close() error                                         { return nil }
func (s sizeWindow) Snapshot() (node.Window, error)                       { return node.Window{}, nil }

func TestClickAt_HappyPath(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickAt{})
	rn, _ := node.Get("ClickAt")

	rec := &recordingInput{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{caInPoint: node.Point{X: 0.3, Y: 0.7}, caInButton: "right", caInDurationMs: 80},
		nil, withInput(rec), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != caOutDone {
		t.Errorf("exit = %q, want Done", r.ExitName)
	}
	if len(rec.calls) != 2 ||
		rec.calls[0] != "MoveTo:0.300:0.700" ||
		rec.calls[1] != "Click:0.300:0.700:right:80" {
		t.Errorf("calls = %v, want [MoveTo Click]", rec.calls)
	}
}

func TestClickAt_DefaultsApplied(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickAt{})
	rn, _ := node.Get("ClickAt")

	rec := &recordingInput{}
	// 不传任何 config — 全走 Default (0.5, 0.5, left, 50)
	r := node.RunNode(context.Background(), rn, nil, nil, nil, withInput(rec), false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if len(rec.calls) != 2 ||
		rec.calls[0] != "MoveTo:0.500:0.500" ||
		rec.calls[1] != "Click:0.500:0.500:left:50" {
		t.Errorf("calls = %v, want [MoveTo Click]", rec.calls)
	}
}

func TestClickAt_MissingWindowCapabilityRejectedBeforePxResolution(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickAt{})
	rn, _ := node.Get("ClickAt")
	result := node.RunNode(context.Background(), rn, nil,
		map[string]any{caInPoint: node.Point{X: 10, Y: 20, Unit: node.UnitPx}}, nil,
		node.ServiceBundle{Input: &recordingInput{}}, false)
	var assemblyErr *node.AssemblyError
	if !errors.As(result.Error, &assemblyErr) || assemblyErr.Capability != node.RuntimeCapabilityWindow {
		t.Fatalf("error = %v, want Window AssemblyError", result.Error)
	}
}

// ClickAt 走 Click (内部 down→hold→up 原子, 不可中途取消) → 取消语义在「滑动阶段」:
// 长 MoveMs 滑动途中取消 → 还没按下就返回, 不会发 Click, 不会有按键残留.
func TestClickAt_CtxCancel_AbortsBeforeClick(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickAt{})
	rn, _ := node.Get("ClickAt")

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(20 * time.Millisecond); cancel() }()

	rec := &recordingInput{}
	start := time.Now()
	r := node.RunNode(ctx, rn, nil,
		map[string]any{caInPoint: node.Point{X: 1, Y: 1}, caInButton: "left",
			caInMoveMs: 2000, caInDurationMs: 50},
		nil, withInput(rec), false)
	elapsed := time.Since(start)

	if elapsed > time.Second {
		t.Fatalf("elapsed %v — ClickAt 没在滑动途中响应 ctx 取消", elapsed)
	}
	if r.Error == nil || !errors.Is(r.Error, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", r.Error)
	}
	for _, c := range rec.calls {
		if strings.HasPrefix(c, "Click") {
			t.Errorf("取消应发生在按下前, 不该有 Click; calls = %v", rec.calls)
		}
	}
}

func TestClickAt_MoveMs_SlidesBeforeDown(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickAt{})
	rn, _ := node.Get("ClickAt")

	rec := &recordingInput{}
	// MoveMs=64 → 4 帧 MoveTo (起点 spy(0,0) → 终点 (1,1)), 再 MouseDown/Up
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{caInPoint: node.Point{X: 1, Y: 1}, caInButton: "left",
			caInMoveMs: 64, caInDurationMs: 10},
		nil, withInput(rec), false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if len(rec.calls) != 5 {
		t.Fatalf("calls = %v, want 5 (4 滑动帧 + Click)", rec.calls)
	}
	if rec.calls[3] != "MoveTo:1.000:1.000" {
		t.Errorf("末滑帧 = %q, want MoveTo:1.000:1.000", rec.calls[3])
	}
	if rec.calls[4] != "Click:1.000:1.000:left:10" {
		t.Errorf("click = %q, want Click:1.000:1.000:left:10", rec.calls[4])
	}
}

func TestClickAt_InvalidButton_ValidationError(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickAt{})
	rn, _ := node.Get("ClickAt")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{caInButton: "side1"},
		nil, withInput(&recordingInput{}), false)

	if len(r.Validation) != 1 || r.Validation[0].Code != "INVALID_MOUSE_BUTTON" {
		t.Errorf("validation = %v, want INVALID_MOUSE_BUTTON", r.Validation)
	}
}

// ─── Task 3.2: Keys / ClickCount 新增测试 ────────────────────────────────────

// TestClickAt_Keys_ClickCount: ctrl+shift 双击 → KeyDown×2 + Click×2 + KeyUp×2.
func TestClickAt_Keys_ClickCount(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickAt{})
	rn, _ := node.Get("ClickAt")

	rec := &recordingInput{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			caInPoint:      node.Point{X: 0.5, Y: 0.5},
			caInButton:     "left",
			caInDurationMs: 50,
			caInKeys:       "ctrl+shift",
			caInClickCount: 2,
		},
		nil, withInput(rec), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != caOutDone {
		t.Fatalf("exit=%q want Done", r.ExitName)
	}
	// Expected: MoveTo + KeyDown:ctrl + KeyDown:shift + Click×2 + KeyUp:shift + KeyUp:ctrl
	want := []string{
		"MoveTo:0.500:0.500",
		"KeyDown:ctrl",
		"KeyDown:shift",
		"Click:0.500:0.500:left:50",
		"Click:0.500:0.500:left:50",
		"KeyUp:shift",
		"KeyUp:ctrl",
	}
	if len(rec.calls) != len(want) {
		t.Fatalf("calls=%v want %v", rec.calls, want)
	}
	for i, s := range want {
		if rec.calls[i] != s {
			t.Errorf("calls[%d]=%q want %q", i, rec.calls[i], s)
		}
	}
}

// TestClickAt_DefaultRegression: Keys="" / ClickCount=1 = 原单击行为 (零回归).
func TestClickAt_DefaultRegression(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickAt{})
	rn, _ := node.Get("ClickAt")

	rec := &recordingInput{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{caInPoint: node.Point{X: 0.3, Y: 0.7}, caInDurationMs: 80},
		nil, withInput(rec), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	// MoveTo then one plain Click — no KeyDown/KeyUp
	if len(rec.calls) != 2 ||
		rec.calls[0] != "MoveTo:0.300:0.700" ||
		rec.calls[1] != "Click:0.300:0.700:left:80" {
		t.Errorf("calls=%v want [MoveTo Click]", rec.calls)
	}
}

// TestClickAt_DurationMs_Preserved: DurationMs 传给 ClickWithMods (长按不回归).
func TestClickAt_DurationMs_Preserved(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickAt{})
	rn, _ := node.Get("ClickAt")

	rec := &recordingInput{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{caInPoint: node.Point{X: 0.5, Y: 0.5}, caInDurationMs: 300},
		nil, withInput(rec), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	// Click must carry durationMs=300
	if len(rec.calls) != 2 || rec.calls[1] != "Click:0.500:0.500:left:300" {
		t.Errorf("calls=%v want Click with dur=300", rec.calls)
	}
}

// TestClickAt_InvalidModifierKey_ValidationError: Keys="ctrl+bad" → INVALID_MODIFIER_KEY.
func TestClickAt_InvalidModifierKey_ValidationError(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickAt{})
	rn, _ := node.Get("ClickAt")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{caInKeys: "ctrl+bad"},
		nil, withInput(&recordingInput{}), false)

	found := false
	for _, e := range r.Validation {
		if e.Code == "INVALID_MODIFIER_KEY" {
			found = true
		}
	}
	if !found {
		t.Errorf("validation=%v want INVALID_MODIFIER_KEY", r.Validation)
	}
}

// TestClickAt_InvalidClickCount_ValidationError: ClickCount=0 → INVALID_CLICK_COUNT.
func TestClickAt_InvalidClickCount_ValidationError(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickAt{})
	rn, _ := node.Get("ClickAt")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{caInClickCount: 0},
		nil, withInput(&recordingInput{}), false)

	found := false
	for _, e := range r.Validation {
		if e.Code == "INVALID_CLICK_COUNT" {
			found = true
		}
	}
	if !found {
		t.Errorf("validation=%v want INVALID_CLICK_COUNT", r.Validation)
	}
}

// TestClickAt_PxPoint_ResolvesToRatio: px 坐标经 ResolvePoint 换算为比例.
func TestClickAt_PxPoint_ResolvesToRatio(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickAt{})
	rn, _ := node.Get("ClickAt")

	rec := &recordingInput{}
	b := node.StubServices()
	b.Input = rec
	b.Window = sizeWindow{w: 1920, h: 1080}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{caInPoint: node.Point{X: 960, Y: 540, Unit: node.UnitPx}, caInDurationMs: 50},
		nil, b, false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	// 960/1920=0.5, 540/1080=0.5
	if len(rec.calls) != 2 || rec.calls[1] != "Click:0.500:0.500:left:50" {
		t.Errorf("calls=%v want Click:0.500:0.500", rec.calls)
	}
}
