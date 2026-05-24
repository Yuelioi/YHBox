// internal/nodes/detect/check_template_test.go
package detect

import (
	"context"
	"errors"
	"testing"
	"time"

	"yhbox/internal/node"
)

type mockVision struct {
	point *node.Point
	conf  float64
	err   error
}

func (m mockVision) Match(key string, threshold float64) (*node.Point, float64, error) {
	return m.point, m.conf, m.err
}

// Phase 4 stub-conformance: VisionService 加了 WaitMatch + BarTrack, mock 用不到也得给 stub.
func (m mockVision) WaitMatch(ctx context.Context, key string, threshold float64, timeout time.Duration) (*node.Point, float64, error) {
	return m.point, m.conf, m.err
}
func (m mockVision) BarTrack(roi node.Rect) (node.BarTrackResult, error) {
	return node.BarTrackResult{}, nil
}

// withVision 把 ServiceBundle 的 Vision 字段换成给定 mock, 其余 stub.
func withVision(v node.VisionService) node.ServiceBundle {
	b := node.StubServices()
	b.Vision = v
	return b
}

func TestCheckTemplate_Hit(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&CheckTemplate{})
	rn, _ := node.Get("CheckTemplate")

	pt := node.Point{X: 0.5, Y: 0.5}
	vision := mockVision{point: &pt, conf: 0.92}
	r := node.RunNode(context.Background(), rn,
		nil,
		map[string]any{ctInTemplate: "fishing.hook_icon", ctInThreshold: 0.85},
		nil, withVision(vision))

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != ctOutFound {
		t.Errorf("exit = %q, want Found", r.ExitName)
	}
	if r.DisplayText == "" {
		t.Error("Display should emit on Found")
	}
}

func TestCheckTemplate_Miss(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&CheckTemplate{})
	rn, _ := node.Get("CheckTemplate")

	vision := mockVision{point: nil, conf: 0.3}
	r := node.RunNode(context.Background(), rn,
		nil,
		map[string]any{ctInTemplate: "fishing.hook_icon", ctInThreshold: 0.85},
		nil, withVision(vision))

	if r.ExitName != ctOutNotFound {
		t.Errorf("exit = %q, want NotFound", r.ExitName)
	}
}

func TestCheckTemplate_Error(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&CheckTemplate{})
	rn, _ := node.Get("CheckTemplate")

	vision := mockVision{err: errors.New("window closed")}
	r := node.RunNode(context.Background(), rn,
		nil,
		map[string]any{ctInTemplate: "fishing.hook_icon"},
		nil, withVision(vision))

	if r.Error == nil {
		t.Error("expected error propagation")
	}
}

func TestCheckTemplate_InvalidKey_ValidationError(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&CheckTemplate{})
	rn, _ := node.Get("CheckTemplate")

	r := node.RunNode(context.Background(), rn,
		nil,
		map[string]any{ctInTemplate: "no_dot"},
		nil, withVision(mockVision{}))

	if len(r.Validation) != 1 || r.Validation[0].Code != "INVALID_TEMPLATE_KEY" {
		t.Errorf("validation = %v, want INVALID_TEMPLATE_KEY", r.Validation)
	}
}

func TestCheckTemplate_RequiredMissing(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&CheckTemplate{})
	rn, _ := node.Get("CheckTemplate")

	r := node.RunNode(context.Background(), rn, nil, nil, nil, withVision(mockVision{}))
	if len(r.Validation) == 0 {
		t.Error("expected REQUIRED_FIELD_MISSING ValidationError for missing Template")
	}
}
