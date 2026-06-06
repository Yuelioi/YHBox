package runtime

import (
	"context"
	"errors"
	"image"
	"testing"
	"time"

	"github.com/lxn/win"
	"github.com/rs/zerolog"

	"yotta/internal/node"
	"yotta/internal/services/container"
	"yotta/internal/services/execution"
	"yotta/internal/services/expr"
	"yotta/pkg/winutil"
)

// newAdapterTestRT 构造一个最小 RuntimeContext 供 adapter 测试用.
// vars 预填 — 镜像 Container.Vars 装载流程.
func newAdapterTestRT(t *testing.T, vars []container.VarDecl) *RuntimeContext {
	t.Helper()
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "test-adapters",
		Name:          "test-adapters",
		Vars:          vars,
		Graph: container.Graph{
			Nodes: []container.GraphNode{{ID: "start", Kind: "Start"}},
		},
	}
	return NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, nil, nil, nil, 0)
}

// ============================================================================
// VarStoreAdapter
// ============================================================================

func TestVarStoreAdapter_GetUnset(t *testing.T) {
	rt := newAdapterTestRT(t, nil)
	a := NewVarStoreAdapter(rt)
	if _, ok := a.Get("missing"); ok {
		t.Errorf("Get unknown var should be (nil, false), got ok=true")
	}
}

func TestVarStoreAdapter_GetDeclared(t *testing.T) {
	rt := newAdapterTestRT(t, []container.VarDecl{
		{Name: "count", Type: "number", Default: 7.0},
		{Name: "flag", Type: "bool", Default: true},
	})
	a := NewVarStoreAdapter(rt)
	if v, ok := a.Get("count"); !ok || v.(float64) != 7.0 {
		t.Errorf("Get count = %v, %v; want 7.0, true", v, ok)
	}
	if v, ok := a.Get("flag"); !ok || v.(bool) != true {
		t.Errorf("Get flag = %v, %v; want true, true", v, ok)
	}
}

func TestVarStoreAdapter_SetGetRoundTrip(t *testing.T) {
	rt := newAdapterTestRT(t, nil)
	a := NewVarStoreAdapter(rt)
	a.Set("x", "hello")
	a.Set("n", 42.5)
	if v, ok := a.Get("x"); !ok || v.(string) != "hello" {
		t.Errorf("Get x = %v, %v; want \"hello\", true", v, ok)
	}
	if v, ok := a.Get("n"); !ok || v.(float64) != 42.5 {
		t.Errorf("Get n = %v, %v; want 42.5, true", v, ok)
	}
}

func TestVarStoreAdapter_SetUpdatesTimestamp(t *testing.T) {
	rt := newAdapterTestRT(t, nil)
	a := NewVarStoreAdapter(rt)
	before := time.Now().UnixMilli()
	a.Set("x", 1.0)
	ts := rt.VarLastChange("x")
	if ts < before {
		t.Errorf("VarLastChange = %d, want >= %d (live time)", ts, before)
	}
}

func TestVarStoreAdapter_LastChange(t *testing.T) {
	rt := newAdapterTestRT(t, nil)
	a := NewVarStoreAdapter(rt)
	// 未设过的变量返 0.
	if got := a.LastChange("foo"); got != 0 {
		t.Errorf("LastChange unset = %d, want 0", got)
	}
	// SetVar 后返 >0, 且与 rt.VarLastChange 同步.
	rt.SetVar("foo", expr.Value("bar"))
	got := a.LastChange("foo")
	if got <= 0 {
		t.Errorf("LastChange after Set = %d, want > 0", got)
	}
	if got != rt.VarLastChange("foo") {
		t.Errorf("LastChange = %d, want %d (rt.VarLastChange)", got, rt.VarLastChange("foo"))
	}
}

func TestVarStoreAdapter_IncFromZero(t *testing.T) {
	rt := newAdapterTestRT(t, nil)
	a := NewVarStoreAdapter(rt)
	got := a.Inc("counter", 3.0)
	if got != 3.0 {
		t.Errorf("Inc from unset = %v, want 3.0", got)
	}
	if v, ok := a.Get("counter"); !ok || v.(float64) != 3.0 {
		t.Errorf("after Inc Get = %v, %v; want 3.0, true", v, ok)
	}
}

func TestVarStoreAdapter_IncTwice(t *testing.T) {
	rt := newAdapterTestRT(t, []container.VarDecl{{Name: "n", Type: "number", Default: 10.0}})
	a := NewVarStoreAdapter(rt)
	if got := a.Inc("n", 5.0); got != 15.0 {
		t.Errorf("first Inc = %v, want 15.0", got)
	}
	if got := a.Inc("n", -3.0); got != 12.0 {
		t.Errorf("second Inc = %v, want 12.0", got)
	}
}

// ============================================================================
// StopwatchAdapter
// ============================================================================

func TestStopwatchAdapter_ReadUnknown(t *testing.T) {
	a := NewStopwatchAdapter(newStopwatchTable())
	if got := a.Read("never-started"); got != 0 {
		t.Errorf("Read unknown = %d, want 0", got)
	}
}

func TestStopwatchAdapter_StartReadRunning(t *testing.T) {
	a := NewStopwatchAdapter(newStopwatchTable())
	a.Start("k")
	time.Sleep(15 * time.Millisecond)
	got := a.Read("k")
	if got < 10 {
		t.Errorf("Read after 15ms = %d, want >= 10", got)
	}
}

func TestStopwatchAdapter_StopFreezesElapsed(t *testing.T) {
	a := NewStopwatchAdapter(newStopwatchTable())
	a.Start("k")
	time.Sleep(15 * time.Millisecond)
	a.Stop("k")
	first := a.Read("k")
	time.Sleep(15 * time.Millisecond)
	second := a.Read("k")
	if first != second {
		t.Errorf("Read after Stop should be stable: first=%d second=%d", first, second)
	}
}

func TestStopwatchAdapter_RestartResets(t *testing.T) {
	a := NewStopwatchAdapter(newStopwatchTable())
	a.Start("k")
	time.Sleep(20 * time.Millisecond)
	a.Start("k") // restart
	got := a.Read("k")
	if got >= 20 {
		t.Errorf("Read after restart = %d, want < 20 (reset)", got)
	}
}

// ============================================================================
// ServiceBundle 构造 smoke
// ============================================================================

func TestNewServiceBundleFor_AllSlotsFilled(t *testing.T) {
	rt := newAdapterTestRT(t, nil)
	bundle := NewServiceBundleFor(rt, newStopwatchTable(), zerolog.Nop(), nil)
	if bundle.Vision == nil {
		t.Error("bundle.Vision is nil")
	}
	if bundle.Log == nil {
		t.Error("bundle.Log is nil")
	}
	if bundle.Input == nil {
		t.Error("bundle.Input is nil")
	}
	if bundle.Vars == nil {
		t.Error("bundle.Vars is nil")
	}
	if bundle.Window == nil {
		t.Error("bundle.Window is nil")
	}
	if bundle.Capture == nil {
		t.Error("bundle.Capture is nil")
	}
	if bundle.Stopwatches == nil {
		t.Error("bundle.Stopwatches is nil")
	}
}

// ============================================================================
// VisionAdapter Match
// ============================================================================

// stubMatcher 控 found / point / conf 让 test 验 Match 返回值.
type stubMatcher struct {
	found bool
	pt    expr.Point
	conf  float64 // 0 = 按 found 取 (found→1.0); 非 0 = 显式真实匹配度
}

func (m stubMatcher) Detect(_ context.Context, _ string, _ *image.RGBA, _ string, _ float64, _ []float64) (bool, expr.Point, [4]float64, float64, error) {
	conf := m.conf
	if conf == 0 && m.found {
		conf = 1.0
	}
	return m.found, m.pt, [4]float64{}, conf, nil
}

func TestVisionAdapter_Match_Found(t *testing.T) {
	rt := newAdapterTestRT(t, nil)
	rt.Matcher = stubMatcher{found: true, pt: expr.Point{X: 0.42, Y: 0.13}}
	a := &visionAdapter{rt: rt}
	pt, _, err := a.Match(context.Background(), []string{"foo"}, 0.8, "any")
	if err != nil {
		t.Fatal(err)
	}
	if pt == nil {
		t.Fatal("pt = nil, want non-nil after Match found")
	}
	if pt.X != 0.42 || pt.Y != 0.13 {
		t.Errorf("pt = %v, want {0.42, 0.13}", pt)
	}
}

func TestVisionAdapter_WaitMatch_TimeoutReportsBestConf(t *testing.T) {
	rt := newAdapterTestRT(t, nil)
	rt.Matcher = stubMatcher{found: false, conf: 0.62} // 没过阈值, 但真实匹配度 0.62
	a := &visionAdapter{rt: rt}
	pt, conf, err := a.WaitMatch(context.Background(), []string{"foo"}, 0.85, "any", 60*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if pt != nil {
		t.Errorf("pt = %v, want nil on timeout", pt)
	}
	if conf < 0.61 || conf > 0.63 {
		t.Errorf("timeout conf = %.3f, want ~0.62 (best seen during poll)", conf)
	}
}

func TestVisionAdapter_DetectColor_Counts(t *testing.T) {
	// 2x1 帧: 全红. 检测全帧 rgb, 应命中 2 像素.
	img := image.NewRGBA(image.Rect(0, 0, 2, 1))
	img.Pix[0], img.Pix[3] = 255, 255 // x=0 红
	img.Pix[4], img.Pix[7] = 255, 255 // x=1 红
	rt := newAdapterTestRT(t, nil)
	rt.Capture = fakeCapture{img: img}
	rt.SetActiveWindow(winutil.WindowHandle{HWND: 1})
	a := &visionAdapter{rt: rt}
	count, _, _, err := a.DetectColor(node.Geometry{}, "rgb", [6]int{200, 255, 0, 50, 0, 50})
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

// ============================================================================
// Adapter interface compliance (compile-time)
// ============================================================================

var (
	_ node.VarStore       = (*varStoreAdapter)(nil)
	_ node.StopwatchStore = (*stopwatchAdapter)(nil)
	_ node.InputService   = (*inputAdapter)(nil)
	_ node.WindowService  = (*windowAdapter)(nil)
	_ node.CaptureService = (*captureAdapter)(nil)
	_ node.VisionService  = (*visionAdapter)(nil)
	_ node.LogService     = logAdapter{}
)

// ============================================================================
// DetectColor override test
// ============================================================================

type fakeCapture struct{ img *image.RGBA }

func (f fakeCapture) Name() string                                            { return "fake" }
func (f fakeCapture) Frame(_ win.HWND) (*image.RGBA, error)                  { return f.img, nil }
func (f fakeCapture) FrameROI(_ win.HWND, _, _, _, _ int) (*image.RGBA, error) {
	return f.img, nil
}
func (f fakeCapture) ClientSize(_ win.HWND) (int, int, error) {
	return f.img.Bounds().Dx(), f.img.Bounds().Dy(), nil
}
func (f fakeCapture) Close() error { return nil }

func TestDetectColor_UsesGeometryOverride(t *testing.T) {
	// 4x4 帧: 左上 2x2 红, 其余黑.
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			i := y*img.Stride + x*4
			img.Pix[i], img.Pix[i+3] = 255, 255
		}
	}
	rt, _ := newTestRunner(t)
	rt.Capture = fakeCapture{img: img}
	rt.SetActiveWindow(winutil.WindowHandle{HWND: 1})
	va := &visionAdapter{rt: rt}

	// override 精确匹配 4x4 → 只数左上 2x2 像素 rect, 应得 4 红.
	geo := node.Geometry{
		Pct: node.Rect{X: 0.9, Y: 0.9, W: 0.05, H: 0.05}, // pct 故意指到右下空区
		Overrides: []node.GeoOverride{{
			Resolution: node.Resolution{W: 4, H: 4},
			Px:         node.PixelRect{X: 0, Y: 0, W: 2, H: 2},
		}},
	}
	count, cx, cy, err := va.DetectColor(geo, "rgb", [6]int{200, 255, 0, 50, 0, 50})
	if err != nil {
		t.Fatal(err)
	}
	if count != 4 {
		t.Fatalf("count = %d, want 4 (override rect, 非 pct)", count)
	}
	// 中心还原成全帧比例: 命中像素 (0,0)(1,0)(0,1)(1,1) → sumX=sumY=2, count=4, 帧 4x4
	// → cx=cy=2/4/4=0.125. 验 ResolveGeometry rect 偏移没丢 + 全帧坐标系映射对.
	if cx != 0.125 || cy != 0.125 {
		t.Fatalf("center = (%v,%v), want (0.125,0.125)", cx, cy)
	}
}

// ============================================================================
// WindowAdapter.SetActive
// ============================================================================

func TestWindowAdapter_SetActive_SetsStickyWindow(t *testing.T) {
	rt := &RuntimeContext{Container: &container.Container{}}
	rt.initFrameCache()
	orig := resolveWindowFn
	defer func() { resolveWindowFn = orig }()
	resolveWindowFn = func(ctx context.Context, spec winutil.MatchSpec, timeout, interval time.Duration) (winutil.WindowHandle, error) {
		return winutil.WindowHandle{HWND: 99, ClientW: 1280, ClientH: 720}, nil
	}
	a := &windowAdapter{rt: rt}
	if err := a.SetActive(context.Background(), "Game", "", "", "exact"); err != nil {
		t.Fatalf("SetActive err: %v", err)
	}
	if rt.WindowHandle().HWND != 99 {
		t.Fatalf("active hwnd = %d, want 99", rt.WindowHandle().HWND)
	}
}

func TestWindowAdapter_SetActive_PropagatesResolveError(t *testing.T) {
	rt := &RuntimeContext{Container: &container.Container{}}
	rt.initFrameCache()
	orig := resolveWindowFn
	defer func() { resolveWindowFn = orig }()
	resolveWindowFn = func(ctx context.Context, spec winutil.MatchSpec, timeout, interval time.Duration) (winutil.WindowHandle, error) {
		return winutil.WindowHandle{}, errors.New("窗口未找到")
	}
	a := &windowAdapter{rt: rt}
	if err := a.SetActive(context.Background(), "Nope", "", "", "exact"); err == nil {
		t.Fatal("want error propagated, got nil")
	}
}
