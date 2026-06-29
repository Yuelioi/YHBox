// internal/node/click_mods_test.go
// TDD tests for ParseMods / ClickWithMods (Task 3.2).
package node

import (
	"context"
	"fmt"
	"testing"
)

// ─── recInputMods ─────────────────────────────────────────────────────────────

// recInputMods records KeyDown / Click / KeyUp sequences for ClickWithMods tests.
type recInputMods struct {
	seq []string
	err error
}

func (r *recInputMods) KeyDown(vk string) error {
	r.seq = append(r.seq, "KeyDown:"+vk)
	return r.err
}
func (r *recInputMods) KeyUp(vk string) error {
	r.seq = append(r.seq, "KeyUp:"+vk)
	return r.err
}
func (r *recInputMods) Click(x, y float64, button string, durationMs int) error {
	r.seq = append(r.seq, fmt.Sprintf("Click:%d", durationMs))
	return r.err
}
func (r *recInputMods) MouseMoveRel(dx, dy, durationMs int) error   { return nil }
func (r *recInputMods) MoveTo(x, y float64) error                   { return nil }
func (r *recInputMods) CursorRatio() (float64, float64, error)      { return 0, 0, nil }
func (r *recInputMods) Scroll(x, y float64, notches int, horizontal bool) error { return nil }
func (r *recInputMods) MouseDown(x, y float64, button string) error                       { return nil }
func (r *recInputMods) MouseUp(button string) error                                       { return nil }
func (r *recInputMods) Drag(x1, y1, x2, y2 float64, button string, durationMs int) error { return nil }
func (r *recInputMods) TypeText(s string) error                                           { return nil }

func (r *recInputMods) matches(want []string) bool {
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

// makeModsCtx builds a Ctx with the given InputService (no spec — ClickWithMods doesn't use ctx.Out).
func makeModsCtx(rec InputService) Ctx {
	svc := StubServices()
	svc.Input = rec
	return newCtx(context.Background(), svc, &Spec{Kind: "_test"}, nil)
}

// ─── ParseMods unit tests ─────────────────────────────────────────────────────

func TestParseMods_CtrlShift(t *testing.T) {
	mods, ok := ParseMods("ctrl+shift")
	if !ok {
		t.Fatal("ctrl+shift should be ok=true")
	}
	if len(mods) != 2 || mods[0] != "ctrl" || mods[1] != "shift" {
		t.Fatalf("got %v, want [ctrl shift]", mods)
	}
}

func TestParseMods_Empty(t *testing.T) {
	mods, ok := ParseMods("")
	if !ok {
		t.Fatal("empty should be ok=true")
	}
	if len(mods) != 0 {
		t.Fatalf("got %v, want empty", mods)
	}
}

func TestParseMods_Invalid(t *testing.T) {
	_, ok := ParseMods("ctrl+foo")
	if ok {
		t.Fatal("ctrl+foo should be ok=false")
	}
}

// ─── ClickWithMods sequence tests ────────────────────────────────────────────

func TestClickWithMods_CtrlShift_2Click(t *testing.T) {
	rec := &recInputMods{}
	ctx := makeModsCtx(rec)
	pt := Point{X: 0.5, Y: 0.5}
	err := ClickWithMods(ctx, pt, "left", "ctrl+shift", 2, 50)
	if err != nil {
		t.Fatal(err)
	}
	// KeyDown(ctrl) KeyDown(shift) Click(50) Click(50) KeyUp(shift) KeyUp(ctrl)
	want := []string{"KeyDown:ctrl", "KeyDown:shift", "Click:50", "Click:50", "KeyUp:shift", "KeyUp:ctrl"}
	if !rec.matches(want) {
		t.Fatalf("seq=%v want %v", rec.seq, want)
	}
}

func TestClickWithMods_NoMods_SingleClick(t *testing.T) {
	rec := &recInputMods{}
	ctx := makeModsCtx(rec)
	pt := Point{X: 0.3, Y: 0.7}
	err := ClickWithMods(ctx, pt, "right", "", 1, 80)
	if err != nil {
		t.Fatal(err)
	}
	// No mods: just one Click with the provided durationMs
	want := []string{"Click:80"}
	if !rec.matches(want) {
		t.Fatalf("seq=%v want %v", rec.seq, want)
	}
}

func TestClickWithMods_DurationMs_PassedToClick(t *testing.T) {
	rec := &recInputMods{}
	ctx := makeModsCtx(rec)
	pt := Point{X: 0.5, Y: 0.5}
	err := ClickWithMods(ctx, pt, "left", "", 1, 120)
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.seq) != 1 || rec.seq[0] != "Click:120" {
		t.Fatalf("seq=%v want [Click:120]", rec.seq)
	}
}
