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

// withVision 把 ServiceBundle 的 Vision 字段换成给定 mock, 其余 stub.
func withVision(v node.VisionService) node.ServiceBundle {
	b := node.StubServices()
	b.Vision = v
	return b
}

// recVars 记录式 VarStore — 跨 detect 包测试文件复用, 验证节点捕获到变量.
// 只把 SetScoped 写进 m (capture 走 SetScoped); 其余方法满足接口即可.
// 定义在 mockVision 同文件 (字母序最早) 以免重复定义.
type recVars struct {
	m map[string]any
}

func newRecVars() *recVars { return &recVars{m: map[string]any{}} }

func (r *recVars) Get(name string) (any, bool) { v, ok := r.m[name]; return v, ok }
func (r *recVars) Set(name string, v any)      { r.m[name] = v }
func (r *recVars) Inc(string, float64) float64 { return 0 }
func (r *recVars) GetScoped(name, _ string) (any, bool) {
	v, ok := r.m[name]
	return v, ok
}
func (r *recVars) SetScoped(name, _ string, v any)           { r.m[name] = v }
func (r *recVars) IncScoped(string, string, float64) float64 { return 0 }
func (r *recVars) LastChange(string) int64                   { return 0 }

// withVisionVars 同 withVision 但额外注入记录式 VarStore, 验证捕获.
func withVisionVars(v node.VisionService, vars node.VarStore) node.ServiceBundle {
	b := node.StubServices()
	b.Vision = v
	b.Vars = vars
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

func TestCheckTemplate_Capture_Hit(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&CheckTemplate{})
	rn, _ := node.Get("CheckTemplate")

	pt := node.Point{X: 0.5, Y: 0.6}
	vision := &mockVision{point: &pt, conf: 0.92}
	vars := newRecVars()
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			ctInTemplates: []string{"fishing.hook_icon"},
			ctInThreshold: 0.85,
			ctCapFound:    "f",
			ctCapPoint:    "p",
		},
		nil, withVisionVars(vision, vars), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != ctOutFound {
		t.Fatalf("exit = %q, want Found", r.ExitName)
	}
	if got, ok := vars.Get("f"); !ok || got != true {
		t.Errorf("capture f = %v (ok=%v), want true", got, ok)
	} else if _, isBool := got.(bool); !isBool {
		// CaptureType=bool: 断言写入值的 Go 类型是 bool.
		t.Errorf("capture f Go type = %T, want bool", got)
	}
	gp, ok := vars.Get("p")
	if !ok {
		t.Fatal("capture p not written")
	}
	if gp != pt {
		t.Errorf("capture p = %v, want %v", gp, pt)
	}
	// CaptureType=point: 断言写入值的 Go 类型是 node.Point.
	if _, isPoint := gp.(node.Point); !isPoint {
		t.Errorf("capture p Go type = %T, want node.Point", gp)
	}
}

func TestCheckTemplate_Capture_Miss(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&CheckTemplate{})
	rn, _ := node.Get("CheckTemplate")

	vision := &mockVision{point: nil, conf: 0.3}
	vars := newRecVars()
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			ctInTemplates: []string{"fishing.hook_icon"},
			ctInThreshold: 0.85,
			ctCapFound:    "f",
			ctCapPoint:    "p",
		},
		nil, withVisionVars(vision, vars), false)

	if r.ExitName != ctOutNotFound {
		t.Fatalf("exit = %q, want NotFound", r.ExitName)
	}
	if got, ok := vars.Get("f"); !ok || got != false {
		t.Errorf("capture f = %v (ok=%v), want false", got, ok)
	}
	// miss exit 不带 Point → 不写.
	if _, ok := vars.Get("p"); ok {
		t.Error("capture p written on NotFound exit, want unwritten")
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
