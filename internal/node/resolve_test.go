// internal/node/resolve_test.go
package node

import (
	"context"
	"errors"
	"testing"
)

type sizeWin struct {
	w, h int
	err  error
}

func (s sizeWin) BringForeground() error { return nil }
func (s sizeWin) HWND() uintptr          { return 0 }
func (s sizeWin) ClientSize() (int, int, error) { return s.w, s.h, s.err }
func (s sizeWin) SetActive(context.Context, string, string, string, string) error { return nil }

func resolveCtx(w WindowService) Ctx {
	svc := StubServices()
	svc.Window = w
	return newCtx(context.Background(), svc, &Spec{Kind: "_test"}, nil)
}

func TestResolvePoint_Ratio_NoWindow(t *testing.T) {
	// UnitRatio: 原样返回, 不碰 Window (即使 ClientSize 会报错也不该调).
	ctx := resolveCtx(sizeWin{err: errors.New("should not be called")})
	x, y, err := ResolvePoint(ctx, Point{X: 0.25, Y: 0.75})
	if err != nil || x != 0.25 || y != 0.75 {
		t.Fatalf("got %v,%v,%v want 0.25,0.75,nil", x, y, err)
	}
}

func TestResolvePoint_Px_DividesByClientSize(t *testing.T) {
	ctx := resolveCtx(sizeWin{w: 1920, h: 1080})
	x, y, err := ResolvePoint(ctx, Point{X: 960, Y: 540, Unit: UnitPx})
	if err != nil || x != 0.5 || y != 0.5 {
		t.Fatalf("got %v,%v,%v want 0.5,0.5,nil", x, y, err)
	}
}

func TestResolvePoint_Px_SmallValueNotHeuristic(t *testing.T) {
	// 显式 px=1 必须除 (不走 |v|<=1 启发) → 1/1920.
	ctx := resolveCtx(sizeWin{w: 1920, h: 1080})
	x, _, _ := ResolvePoint(ctx, Point{X: 1, Y: 1, Unit: UnitPx})
	if x != 1.0/1920.0 {
		t.Fatalf("got %v want %v", x, 1.0/1920.0)
	}
}

func TestResolvePoint_Px_ClientSizeErr(t *testing.T) {
	ctx := resolveCtx(sizeWin{err: errors.New("no window")})
	if _, _, err := ResolvePoint(ctx, Point{X: 1, Y: 1, Unit: UnitPx}); err == nil {
		t.Fatal("want error when ClientSize fails")
	}
}
