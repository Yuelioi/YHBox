// internal/nodes/detect/click_common_test.go
// Task 2.4 — parseMods + clickWithMods 序列测试.
package detect

import (
	"context"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
)

// ─── 序列记录 InputService stub ──────────────────────────────────────────────

// recInput 记录 KeyDown / Click / KeyUp 调用序列，供 clickWithMods 序列断言。
type recInput struct {
	seq []string
}

func (r *recInput) KeyDown(vk string) error {
	r.seq = append(r.seq, "KeyDown:"+vk)
	return nil
}
func (r *recInput) KeyUp(vk string) error {
	r.seq = append(r.seq, "KeyUp:"+vk)
	return nil
}
func (r *recInput) Click(x, y float64, button string, durationMs int) error {
	r.seq = append(r.seq, "Click")
	return nil
}
func (r *recInput) MouseMoveRel(dx, dy, durationMs int) error                        { return nil }
func (r *recInput) MoveTo(x, y float64) error                                        { return nil }
func (r *recInput) CursorRatio() (float64, float64, error)                           { return 0, 0, nil }
func (r *recInput) Scroll(x, y float64, notches int, horizontal bool) error          { return nil }
func (r *recInput) MouseDown(x, y float64, button string) error                      { return nil }
func (r *recInput) MouseUp(button string) error                                      { return nil }
func (r *recInput) Drag(x1, y1, x2, y2 float64, button string, durationMs int) error { return nil }
func (r *recInput) TypeText(s string) error                                          { return nil }

func (r *recInput) matches(want []string) bool {
	if len(r.seq) != len(want) {
		return false
	}
	for i, s := range want {
		if r.seq[i] != s {
			return false
		}
	}
	return true
}

// ─── TestParseMods ────────────────────────────────────────────────────────────
// parseMods 已提升为 node.ParseMods (Task 3.2); 这里保留端到端覆盖确认导出 API 可用.

func TestParseMods(t *testing.T) {
	if mods, ok := node.ParseMods("ctrl+shift"); !ok || len(mods) != 2 || mods[0] != "ctrl" || mods[1] != "shift" {
		t.Fatalf("ctrl+shift → %v ok=%v", mods, ok)
	}
	if mods, ok := node.ParseMods(""); !ok || len(mods) != 0 {
		t.Fatalf("空 → %v ok=%v", mods, ok)
	}
	if _, ok := node.ParseMods("ctrl+foo"); ok {
		t.Fatalf("非法修饰键应 ok=false")
	}
}

// ─── TestClickWithMods_Sequence ───────────────────────────────────────────────
// 通过 ClickTemplate RunNode + recInput 验证 clickWithMods 的 KeyDown→Click×N→KeyUp 序列。

func TestClickWithMods_Sequence(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickTemplate{})
	rn, _ := node.Get("ClickTemplate")

	pt := node.Point{X: 0.5, Y: 0.5}
	// bbox center = pt
	vision := &mockVision{point: &pt, bbox: [4]float64{0.5, 0.5, 0, 0}, conf: 0.93, hitOnCall: 1}
	rec := &recInput{}

	svc := node.StubServices()
	svc.Vision = vision
	svc.Input = rec

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			clkInTemplates:  []string{"tpl.mods"},
			clkInTimeoutMs:  200,
			clkInThreshold:  0.85,
			clkInKeys:       "ctrl+shift",
			clkInClickCount: 2,
		},
		nil, svc, false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != clkOutDone {
		t.Fatalf("exit=%q want Done", r.ExitName)
	}
	// 期望: KeyDown(ctrl) KeyDown(shift) Click Click KeyUp(shift) KeyUp(ctrl)
	want := []string{"KeyDown:ctrl", "KeyDown:shift", "Click", "Click", "KeyUp:shift", "KeyUp:ctrl"}
	if !rec.matches(want) {
		t.Fatalf("序列=%v want %v", rec.seq, want)
	}
}

// ─── 零回归: 默认 Keys=""/ClickCount=1 = 单击无修饰 ─────────────────────────

func TestClickWithMods_DefaultNoMods(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickTemplate{})
	rn, _ := node.Get("ClickTemplate")

	pt := node.Point{X: 0.5, Y: 0.5}
	vision := &mockVision{point: &pt, bbox: [4]float64{0.5, 0.5, 0, 0}, conf: 0.93, hitOnCall: 1}
	rec := &recInput{}

	svc := node.StubServices()
	svc.Vision = vision
	svc.Input = rec

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			clkInTemplates: []string{"tpl.default"},
			clkInTimeoutMs: 200,
			clkInThreshold: 0.85,
			// Keys/ClickCount 不传 → 默认 ""/1
		},
		nil, svc, false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != clkOutDone {
		t.Fatalf("exit=%q want Done", r.ExitName)
	}
	// 无修饰、单击: 只有一个 Click, 无 KeyDown/KeyUp
	want := []string{"Click"}
	if !rec.matches(want) {
		t.Fatalf("序列=%v want %v", rec.seq, want)
	}
}

// ─── Validate: 非法修饰键 + ClickCount<1 ─────────────────────────────────────

func TestClickTemplate_InvalidModifierKey(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickTemplate{})
	rn, _ := node.Get("ClickTemplate")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			clkInTemplates: []string{"tpl.val"},
			clkInKeys:      "ctrl+foo",
		},
		nil, withVisionAndInput(&mockVision{}, &recordingInput{}), false)

	if len(r.Validation) == 0 {
		t.Fatal("expected validation error for INVALID_MODIFIER_KEY")
	}
	found := false
	for _, e := range r.Validation {
		if e.Code == "INVALID_MODIFIER_KEY" {
			found = true
		}
	}
	if !found {
		t.Errorf("validation=%v, want INVALID_MODIFIER_KEY", r.Validation)
	}
}

func TestClickTemplate_InvalidClickCount(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickTemplate{})
	rn, _ := node.Get("ClickTemplate")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			clkInTemplates:  []string{"tpl.val"},
			clkInClickCount: 0,
		},
		nil, withVisionAndInput(&mockVision{}, &recordingInput{}), false)

	if len(r.Validation) == 0 {
		t.Fatal("expected validation error for INVALID_CLICK_COUNT")
	}
	found := false
	for _, e := range r.Validation {
		if e.Code == "INVALID_CLICK_COUNT" {
			found = true
		}
	}
	if !found {
		t.Errorf("validation=%v, want INVALID_CLICK_COUNT", r.Validation)
	}
}
