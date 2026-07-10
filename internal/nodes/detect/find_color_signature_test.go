// internal/nodes/detect/find_color_signature_test.go
package detect

import (
	"context"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
)

// ─── parse 层测试 ────────────────────────────────────────────────────

func TestFindColorSignature_ParseValid(t *testing.T) {
	raw := []any{
		map[string]any{"dx": 0.0, "dy": 0.0, "r": 200.0, "g": 30.0, "b": 30.0},
		map[string]any{"dx": 2.0, "dy": 0.0, "r": 255.0, "g": 255.0, "b": 255.0, "tol": 8.0},
	}
	sig, err := parseColorSignature(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sig.Points) != 2 {
		t.Fatalf("want 2 points, got %d", len(sig.Points))
	}
	// 锚点 tol 缺省 → nil
	if sig.Points[0].Tol != nil {
		t.Fatal("anchor tol should be nil (default)")
	}
	// 第二点 tol=8 → *int 非 nil
	if sig.Points[1].Tol == nil || *sig.Points[1].Tol != 8 {
		t.Fatalf("want tol=8, got %v", sig.Points[1].Tol)
	}
}

func TestFindColorSignature_ValidateAnchorNonZero(t *testing.T) {
	raw := []any{map[string]any{"dx": 1.0, "dy": 0.0, "r": 1.0, "g": 1.0, "b": 1.0}}
	if _, err := parseColorSignature(raw); err == nil {
		t.Fatal("anchor dx≠0 should error")
	}
}

func TestFindColorSignature_ValidatePointCount(t *testing.T) {
	var raw []any
	for i := 0; i < 65; i++ {
		raw = append(raw, map[string]any{"dx": 0.0, "dy": 0.0, "r": 1.0, "g": 1.0, "b": 1.0})
	}
	if _, err := parseColorSignature(raw); err == nil {
		t.Fatal("65 points should error")
	}
}

// ─── 路由测试 (用 mockVision) ─────────────────────────────────────────

func runFCS(t *testing.T, vision *mockVision, cfg map[string]any) node.RunResult {
	t.Helper()
	node.ResetRegistryForTest()
	node.Register(&FindColorSignature{})
	rn, _ := node.Get("FindColorSignature")
	return node.RunNode(context.Background(), rn, nil, cfg, nil, withVision(vision), false)
}

func TestFindColorSignature_Found(t *testing.T) {
	pt := node.Point{X: 0.3, Y: 0.4}
	v := &mockVision{sigFound: true, sigPoint: pt}
	r := runFCS(t, v, map[string]any{
		fcsInSignature: []any{
			map[string]any{"dx": 0.0, "dy": 0.0, "r": 200.0, "g": 0.0, "b": 0.0},
		},
	})
	if r.Error != nil {
		t.Fatalf("error: %v", r.Error)
	}
	if r.ExitName != fcsOutFound {
		t.Fatalf("exit=%q, want Found", r.ExitName)
	}
	got, ok := r.OutputData[fcsDataPoint]
	if !ok {
		t.Fatal("Point missing from output data")
	}
	gotPt, ok2 := got.(node.Point)
	if !ok2 || gotPt != pt {
		t.Fatalf("Point=%v, want %v", got, pt)
	}
}

func TestFindColorSignature_NotFound(t *testing.T) {
	v := &mockVision{sigFound: false}
	r := runFCS(t, v, map[string]any{
		fcsInSignature: []any{
			map[string]any{"dx": 0.0, "dy": 0.0, "r": 200.0, "g": 0.0, "b": 0.0},
		},
	})
	if r.Error != nil {
		t.Fatalf("error: %v", r.Error)
	}
	if r.ExitName != fcsOutNotFound {
		t.Fatalf("exit=%q, want NotFound", r.ExitName)
	}
}
