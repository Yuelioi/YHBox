// internal/nodes/detect/check_template_test.go
package detect

import (
	"context"
	"errors"
	"testing"
	"time"

	"yotta/internal/node"
)

// mockVision 实现 VisionService — 跨 detect 包测试文件复用 (定义在字母序最早的
// 文件里以免重复定义). 各字段按需 set, 用不到的方法返零值.
type mockVision struct {
	// Match / WaitMatch 用
	point *node.Point
	conf  float64
	err   error

	// WaitMatch 专属: 进每次 polling 把 callCount 累加; 当 callCount == hitOnCall 时返 point/conf,
	// 否则返 nil. hitOnCall = 0 → 第一次就 hit. hitOnCall < 0 → 永不 hit (timeout 路径).
	hitOnCall int
	callCount int
	// missAfterCall>0: callCount 超过它后 WaitMatch 返 nil (模拟"模板被点掉后消失") —— 测点完验证/重试用; 0=禁用.
	missAfterCall int

	// DualBarTrack 用
	barResult node.DualColorBarResult
	barErr    error

	// DetectColor (老 RGB/HSV 模式) 用
	colorCount int
	colorCX    float64
	colorCY    float64
	colorErr   error

	// DetectColorHSV 用
	hsvCount int
	hsvRatio float64
	hsvErr   error

	// ROIColorScan 用
	clusters    []node.ClusterEntry
	clustersErr error

	// DetectColorBlobs 用
	blobs    []node.BlobEntry
	blobsErr error

	// GridSignature 用: 按调用次序返 gridSigs[i] (越界返最后一个); gridErr 非 nil 直接返错.
	gridSigs [][]uint8
	gridIdx  int
	gridErr  error

	// FindColorSignature 用
	sigFound bool
	sigPoint node.Point
	sigErr   error
}

func (m *mockVision) Match(ctx context.Context, keys []string, threshold float64, mode string) (*node.Point, float64, error) {
	return m.point, m.conf, m.err
}

func (m *mockVision) WaitMatch(ctx context.Context, keys []string, threshold float64, mode string, timeout time.Duration) (*node.Point, float64, error) {
	// 模拟 framework 真接的语义: 一次性 (timeout<=0 也算一次). 节点的 WaitTemplate
	// 是直接调 WaitMatch (服务内部轮询), 这里 hitOnCall 控制返不返命中.
	m.callCount++
	if m.err != nil {
		return nil, m.conf, m.err
	}
	// 模板被点掉后消失: 前 missAfterCall 次照常命中, 之后返 nil。
	if m.missAfterCall > 0 && m.callCount > m.missAfterCall {
		return nil, m.conf, nil
	}
	if m.hitOnCall >= 0 && m.callCount >= m.hitOnCall && m.point != nil {
		return m.point, m.conf, nil
	}
	// timeout 路径: ctx 真等 timeout 模拟服务侧 (节点测试期望 timeout 行为).
	if timeout > 0 {
		select {
		case <-ctx.Done():
			return nil, m.conf, nil
		case <-time.After(timeout):
		}
	}
	return nil, m.conf, nil
}

func (m *mockVision) DualBarTrack(roi node.Geometry, inner, outer node.HSVRange, opts node.DualBarOptions) (node.DualColorBarResult, error) {
	return m.barResult, m.barErr
}

func (m *mockVision) DetectColor(roi node.Geometry, mode string, rng [6]int) (int, float64, float64, error) {
	return m.colorCount, m.colorCX, m.colorCY, m.colorErr
}

func (m *mockVision) DetectColorHSV(roi node.Geometry, hsv node.HSVRange) (int, float64, error) {
	return m.hsvCount, m.hsvRatio, m.hsvErr
}

func (m *mockVision) ROIColorScan(roi node.Geometry, hsv node.HSVRange, axis string, minPx, maxPx int) ([]node.ClusterEntry, error) {
	return m.clusters, m.clustersErr
}

func (m *mockVision) DetectColorBlobs(roi node.Geometry, mode string, rng [6]int, minArea int) ([]node.BlobEntry, error) {
	return m.blobs, m.blobsErr
}

func (m *mockVision) GridSignature(roi node.Geometry, gridSize int) ([]uint8, error) {
	if m.gridErr != nil {
		return nil, m.gridErr
	}
	if len(m.gridSigs) == 0 {
		return nil, nil
	}
	i := m.gridIdx
	if i >= len(m.gridSigs) {
		i = len(m.gridSigs) - 1
	}
	m.gridIdx++
	return m.gridSigs[i], nil
}

func (m *mockVision) FindColorSignature(roi node.Geometry, sig node.ColorSignature, defaultTol int) (bool, node.Point, error) {
	return m.sigFound, m.sigPoint, m.sigErr
}

func (m *mockVision) DecodeQR(_ node.Geometry) ([]node.QRResult, error) {
	return nil, nil
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
	vision := &mockVision{point: &pt, conf: 0.92}
	r := node.RunNode(context.Background(), rn,
		nil,
		map[string]any{ctInTemplates: []string{"fishing.hook_icon"}, ctInThreshold: 0.85},
		nil, withVision(vision), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != ctOutFound {
		t.Errorf("exit = %q, want Found", r.ExitName)
	}
	if r.OutputData[ctDataMatched] != true {
		t.Errorf("Matched = %v, want true", r.OutputData[ctDataMatched])
	}
}

func TestCheckTemplate_Miss(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&CheckTemplate{})
	rn, _ := node.Get("CheckTemplate")

	vision := &mockVision{point: nil, conf: 0.3}
	r := node.RunNode(context.Background(), rn,
		nil,
		map[string]any{ctInTemplates: []string{"fishing.hook_icon"}, ctInThreshold: 0.85},
		nil, withVision(vision), false)

	if r.ExitName != ctOutNotFound {
		t.Errorf("exit = %q, want NotFound", r.ExitName)
	}
	if r.OutputData[ctDataMatched] != false {
		t.Errorf("Matched = %v, want false", r.OutputData[ctDataMatched])
	}
}

func TestCheckTemplate_Error(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&CheckTemplate{})
	rn, _ := node.Get("CheckTemplate")

	vision := &mockVision{err: errors.New("window closed")}
	r := node.RunNode(context.Background(), rn,
		nil,
		map[string]any{ctInTemplates: []string{"fishing.hook_icon"}},
		nil, withVision(vision), false)

	if r.Error == nil {
		t.Error("expected error propagation")
	}
}

func TestCheckTemplate_RequiredMissing(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&CheckTemplate{})
	rn, _ := node.Get("CheckTemplate")

	r := node.RunNode(context.Background(), rn, nil, nil, nil, withVision(&mockVision{}), false)
	if len(r.Validation) == 0 {
		t.Error("expected REQUIRED_FIELD_MISSING ValidationError for missing Template")
	}
}
