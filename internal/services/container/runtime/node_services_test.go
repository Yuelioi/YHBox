package runtime

import (
	"context"
	"errors"
	"image"
	"testing"
	"time"

	"github.com/lxn/win"
	"github.com/rs/zerolog"

	"yotta/internal/automation/controller"
	"yotta/internal/automation/target"
	automationtrace "yotta/internal/automation/trace"
	"yotta/internal/node"
	"yotta/internal/services/container"
	"yotta/internal/services/execution"
	"yotta/internal/services/expr"
	pkginput "yotta/pkg/input"
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

func (m stubMatcher) Detect(_ context.Context, _ *image.RGBA, _ string, _ float64, _ []float64, _ float64) (bool, expr.Point, [4]float64, float64, error) {
	conf := m.conf
	if conf == 0 && m.found {
		conf = 1.0
	}
	return m.found, m.pt, [4]float64{}, conf, nil
}

func (m stubMatcher) DetectAll(_ context.Context, _ *image.RGBA, _ string, _ float64, _ []float64, _ float64) ([]node.TemplateMatch, error) {
	return nil, nil
}

func TestVisionAdapter_Match_Found(t *testing.T) {
	rt := newAdapterTestRT(t, nil)
	rt.Matcher = stubMatcher{found: true, pt: expr.Point{X: 0.42, Y: 0.13}}
	a := &visionAdapter{rt: rt}
	hit, err := a.Match(context.Background(), []string{"foo"}, 0.8, node.Geometry{})
	if err != nil {
		t.Fatal(err)
	}
	if !hit.Found {
		t.Fatal("hit.Found = false, want true after Match found")
	}
	if hit.Point.X != 0.42 || hit.Point.Y != 0.13 {
		t.Errorf("hit.Point = %v, want {0.42, 0.13}", hit.Point)
	}
}

func TestVisionAdapter_WaitMatch_TimeoutReportsBestConf(t *testing.T) {
	rt := newAdapterTestRT(t, nil)
	rt.Matcher = stubMatcher{found: false, conf: 0.62} // 没过阈值, 但真实匹配度 0.62
	a := &visionAdapter{rt: rt}
	hit, err := a.WaitMatch(context.Background(), []string{"foo"}, 0.85, node.Geometry{}, 60*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if hit.Found {
		t.Errorf("hit.Found = true, want false on timeout")
	}
	if hit.Conf < 0.61 || hit.Conf > 0.63 {
		t.Errorf("timeout conf = %.3f, want ~0.62 (best seen during poll)", hit.Conf)
	}
}

// mapMatcher 按 guid 返不同的 found/bbox/conf, 用于多模板 OR 和 bbox 测试.
type mapMatcher struct {
	results map[string]struct {
		found bool
		pt    expr.Point
		bbox  [4]float64
		conf  float64
	}
}

func (m mapMatcher) Detect(_ context.Context, _ *image.RGBA, guid string, _ float64, _ []float64, _ float64) (bool, expr.Point, [4]float64, float64, error) {
	if r, ok := m.results[guid]; ok {
		return r.found, r.pt, r.bbox, r.conf, nil
	}
	return false, expr.Point{}, [4]float64{}, 0, nil
}

func (m mapMatcher) DetectAll(_ context.Context, _ *image.RGBA, _ string, _ float64, _ []float64, _ float64) ([]node.TemplateMatch, error) {
	return nil, nil
}

func TestVisionAdapter_Match_ReturnsBBox(t *testing.T) {
	rt := newAdapterTestRT(t, nil)
	wantBBox := [4]float64{0.1, 0.2, 0.3, 0.4}
	rt.Matcher = mapMatcher{results: map[string]struct {
		found bool
		pt    expr.Point
		bbox  [4]float64
		conf  float64
	}{
		"tmpl-a": {found: true, pt: expr.Point{X: 0.25, Y: 0.4}, bbox: wantBBox, conf: 0.9},
	}}
	a := &visionAdapter{rt: rt}
	hit, err := a.Match(context.Background(), []string{"tmpl-a"}, 0.8, node.Geometry{})
	if err != nil {
		t.Fatal(err)
	}
	if !hit.Found {
		t.Fatal("hit.Found = false, want true")
	}
	if hit.BBox != wantBBox {
		t.Errorf("hit.BBox = %v, want %v", hit.BBox, wantBBox)
	}
	if hit.Point.X != 0.25 || hit.Point.Y != 0.4 {
		t.Errorf("hit.Point = %v, want {0.25, 0.4}", hit.Point)
	}
}

func TestVisionAdapter_Match_MultiTemplate_OR(t *testing.T) {
	rt := newAdapterTestRT(t, nil)
	rt.Matcher = mapMatcher{results: map[string]struct {
		found bool
		pt    expr.Point
		bbox  [4]float64
		conf  float64
	}{
		"tmpl-a": {found: false, conf: 0.5},
		"tmpl-b": {found: true, pt: expr.Point{X: 0.6, Y: 0.7}, bbox: [4]float64{0.5, 0.6, 0.2, 0.2}, conf: 0.95},
	}}
	a := &visionAdapter{rt: rt}
	// keys=[a,b]; a miss, b hit → OR 语义: hit.Found==true
	hit, err := a.Match(context.Background(), []string{"tmpl-a", "tmpl-b"}, 0.8, node.Geometry{})
	if err != nil {
		t.Fatal(err)
	}
	if !hit.Found {
		t.Fatal("hit.Found = false, want true (OR: b hit)")
	}
	if hit.Conf != 0.95 {
		t.Errorf("hit.Conf = %.3f, want 0.95 (from tmpl-b)", hit.Conf)
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

func (f fakeCapture) Name() string                          { return "fake" }
func (f fakeCapture) Frame(_ win.HWND) (*image.RGBA, error) { return f.img, nil }
func (f fakeCapture) FrameROI(_ win.HWND, _, _, _, _ int) (*image.RGBA, error) {
	return f.img, nil
}
func (f fakeCapture) ClientSize(_ win.HWND) (int, int, error) {
	return f.img.Bounds().Dx(), f.img.Bounds().Dy(), nil
}
func (f fakeCapture) Close() error { return nil }

type recordingRuntimeInput struct {
	clickHWND        []uintptr
	clickX           []float64
	clickY           []float64
	clickButton      []string
	clickDuration    []int
	keyDownHWND      []uintptr
	keyDownKeys      []string
	keyUpHWND        []uintptr
	keyUpKeys        []string
	moveHWND         []uintptr
	moveX            []float64
	moveY            []float64
	scrollHWND       []uintptr
	scrollX          []float64
	scrollY          []float64
	scrollNotches    []int
	scrollHorizontal []bool
	mouseDownHWND    []uintptr
	mouseDownX       []float64
	mouseDownY       []float64
	mouseDownButton  []string
	mouseUpHWND      []uintptr
	mouseUpButton    []string
	dragHWND         []uintptr
	dragX1           []float64
	dragY1           []float64
	dragX2           []float64
	dragY2           []float64
	dragButton       []string
	dragDurationMs   []int
	moveRelHWND      []uintptr
	moveRelDx        []int
	moveRelDy        []int
	moveRelDuration  []int
	textHWND         []uintptr
	textValues       []string
}

func (r *recordingRuntimeInput) Name() string { return "sendinput" }
func (r *recordingRuntimeInput) Capabilities() pkginput.Capabilities {
	return pkginput.Capabilities{}
}
func (r *recordingRuntimeInput) Click(hwnd win.HWND, xRatio, yRatio float64, button string, durMs int) error {
	r.clickHWND = append(r.clickHWND, uintptr(hwnd))
	r.clickX = append(r.clickX, xRatio)
	r.clickY = append(r.clickY, yRatio)
	r.clickButton = append(r.clickButton, button)
	r.clickDuration = append(r.clickDuration, durMs)
	return nil
}
func (r *recordingRuntimeInput) KeyDown(hwnd win.HWND, key string) error {
	r.keyDownHWND = append(r.keyDownHWND, uintptr(hwnd))
	r.keyDownKeys = append(r.keyDownKeys, key)
	return nil
}
func (r *recordingRuntimeInput) KeyUp(hwnd win.HWND, key string) error {
	r.keyUpHWND = append(r.keyUpHWND, uintptr(hwnd))
	r.keyUpKeys = append(r.keyUpKeys, key)
	return nil
}
func (r *recordingRuntimeInput) MouseDown(hwnd win.HWND, xRatio, yRatio float64, button string) error {
	r.mouseDownHWND = append(r.mouseDownHWND, uintptr(hwnd))
	r.mouseDownX = append(r.mouseDownX, xRatio)
	r.mouseDownY = append(r.mouseDownY, yRatio)
	r.mouseDownButton = append(r.mouseDownButton, button)
	return nil
}
func (r *recordingRuntimeInput) MouseUp(hwnd win.HWND, button string) error {
	r.mouseUpHWND = append(r.mouseUpHWND, uintptr(hwnd))
	r.mouseUpButton = append(r.mouseUpButton, button)
	return nil
}
func (r *recordingRuntimeInput) MouseMoveRel(hwnd win.HWND, dx, dy, durationMs int) error {
	r.moveRelHWND = append(r.moveRelHWND, uintptr(hwnd))
	r.moveRelDx = append(r.moveRelDx, dx)
	r.moveRelDy = append(r.moveRelDy, dy)
	r.moveRelDuration = append(r.moveRelDuration, durationMs)
	return nil
}
func (r *recordingRuntimeInput) Scroll(hwnd win.HWND, xRatio, yRatio float64, notches int, horizontal bool) error {
	r.scrollHWND = append(r.scrollHWND, uintptr(hwnd))
	r.scrollX = append(r.scrollX, xRatio)
	r.scrollY = append(r.scrollY, yRatio)
	r.scrollNotches = append(r.scrollNotches, notches)
	r.scrollHorizontal = append(r.scrollHorizontal, horizontal)
	return nil
}
func (r *recordingRuntimeInput) Drag(hwnd win.HWND, x1Ratio, y1Ratio, x2Ratio, y2Ratio float64, button string, durationMs int) error {
	r.dragHWND = append(r.dragHWND, uintptr(hwnd))
	r.dragX1 = append(r.dragX1, x1Ratio)
	r.dragY1 = append(r.dragY1, y1Ratio)
	r.dragX2 = append(r.dragX2, x2Ratio)
	r.dragY2 = append(r.dragY2, y2Ratio)
	r.dragButton = append(r.dragButton, button)
	r.dragDurationMs = append(r.dragDurationMs, durationMs)
	return nil
}
func (r *recordingRuntimeInput) TypeText(hwnd win.HWND, text string) error {
	r.textHWND = append(r.textHWND, uintptr(hwnd))
	r.textValues = append(r.textValues, text)
	return nil
}
func (r *recordingRuntimeInput) MoveTo(hwnd win.HWND, xRatio, yRatio float64) error {
	r.moveHWND = append(r.moveHWND, uintptr(hwnd))
	r.moveX = append(r.moveX, xRatio)
	r.moveY = append(r.moveY, yRatio)
	return nil
}
func (r *recordingRuntimeInput) CursorRatio(win.HWND) (float64, float64, error) {
	return 0, 0, nil
}
func (r *recordingRuntimeInput) ReleaseAll() error { return nil }
func (r *recordingRuntimeInput) Close() error      { return nil }

type fakeRuntimeControllerFactory struct {
	ctrl   controller.Controller
	target target.Target
	trace  automationtrace.Recorder
}

func (f *fakeRuntimeControllerFactory) NewController(tg target.Target, trace automationtrace.Recorder) (controller.Controller, error) {
	f.target = tg
	f.trace = trace
	return f.ctrl, nil
}

type fakeRuntimeController struct {
	target target.Target
	clicks []controller.ClickRequest
}

func (f *fakeRuntimeController) Target() target.Target { return f.target }

func (f *fakeRuntimeController) Capabilities(context.Context) controller.CapabilitySet {
	return controller.CapabilitySet{Click: true}
}

func (f *fakeRuntimeController) HealthCheck(context.Context) controller.HealthReport {
	return controller.HealthReport{OK: true}
}

func (f *fakeRuntimeController) Click(_ context.Context, req controller.ClickRequest) error {
	f.clicks = append(f.clicks, req)
	return nil
}

func (f *fakeRuntimeController) Move(context.Context, controller.MoveRequest) error { return nil }

func (f *fakeRuntimeController) Scroll(context.Context, controller.ScrollRequest) error { return nil }

func (f *fakeRuntimeController) MouseDown(context.Context, controller.MouseButtonRequest) error {
	return nil
}

func (f *fakeRuntimeController) MouseUp(context.Context, controller.MouseButtonRequest) error {
	return nil
}

func (f *fakeRuntimeController) Drag(context.Context, controller.DragRequest) error { return nil }

func (f *fakeRuntimeController) MoveRelative(context.Context, controller.RelativeMoveRequest) error {
	return nil
}

func TestInputAdapter_ClickRoutesThroughControllerTrace(t *testing.T) {
	rt := newAdapterTestRT(t, nil)
	rt.SetActiveWindow(winutil.WindowHandle{HWND: 99, Title: "After Effects", ClientW: 1920, ClientH: 1080})
	input := &recordingRuntimeInput{}
	rt.Input = input

	err := NewInputAdapter(rt).Click(0.25, 0.75, "right", 80)
	if err != nil {
		t.Fatalf("Click error = %v", err)
	}
	if len(input.clickHWND) != 1 || input.clickHWND[0] != 99 || input.clickX[0] != 0.25 || input.clickY[0] != 0.75 || input.clickButton[0] != "right" || input.clickDuration[0] != 80 {
		t.Fatalf("backend Click = hwnds %#v x %#v y %#v buttons %#v durations %#v", input.clickHWND, input.clickX, input.clickY, input.clickButton, input.clickDuration)
	}
	records := rt.TraceRecords()
	if len(records) != 1 {
		t.Fatalf("trace len = %d, want 1", len(records))
	}
	if records[0].Action != "click" || records[0].Target.ID != "win32:99" || records[0].Backend != "sendinput" {
		t.Fatalf("trace record = %#v", records[0])
	}
}

func TestInputAdapter_RejectsNonWin32ActiveTarget(t *testing.T) {
	rt := newAdapterTestRT(t, nil)
	rt.SetActiveTarget(target.Target{
		ID:         "android:emulator-5554",
		Kind:       target.KindAndroidADB,
		Ref:        target.TargetRef{ADBSerial: "emulator-5554"},
		Resolution: target.Size{W: 1080, H: 1920},
	})
	rt.Input = &recordingRuntimeInput{}

	err := NewInputAdapter(rt).Click(0.25, 0.75, "left", 0)
	if err == nil {
		t.Fatal("expected non-win32 active target error")
	}
	if got := err.Error(); got != "no controller factory for active target kind android-adb" {
		t.Fatalf("error = %q", got)
	}
}

func TestInputAdapter_ClickRoutesThroughInjectedControllerFactory(t *testing.T) {
	rt := newAdapterTestRT(t, nil)
	tg := target.Target{
		ID:         "android:emulator-5554",
		Kind:       target.KindAndroidADB,
		Ref:        target.TargetRef{ADBSerial: "emulator-5554"},
		Resolution: target.Size{W: 1080, H: 1920},
	}
	rt.SetActiveTarget(tg)
	fakeCtrl := &fakeRuntimeController{target: tg}
	factory := &fakeRuntimeControllerFactory{ctrl: fakeCtrl}
	rt.ControllerFactory = factory

	err := NewInputAdapter(rt).Click(0.25, 0.75, "left", 0)
	if err != nil {
		t.Fatalf("Click error = %v", err)
	}
	if factory.target.ID != tg.ID {
		t.Fatalf("factory target = %#v, want %#v", factory.target, tg)
	}
	if len(fakeCtrl.clicks) != 1 {
		t.Fatalf("fake controller clicks = %#v", fakeCtrl.clicks)
	}
	if got := fakeCtrl.clicks[0].Point; got.X != 0.25 || got.Y != 0.75 || got.Space != target.SpaceNormalized {
		t.Fatalf("click point = %#v", got)
	}
}

func TestTargetAdapter_SetActiveValidatesAndUpdatesRuntime(t *testing.T) {
	rt := &RuntimeContext{}
	svc := NewTargetAdapter(rt)
	tg := target.Target{
		ID:         "android:emulator-5554",
		Kind:       target.KindAndroidADB,
		Ref:        target.TargetRef{ADBSerial: "emulator-5554"},
		Resolution: target.Size{W: 1080, H: 1920},
	}
	if err := svc.SetActive(tg); err != nil {
		t.Fatalf("SetActive() error = %v", err)
	}
	got, ok := rt.ActiveTarget()
	if !ok {
		t.Fatal("runtime active target missing")
	}
	if got.ID != tg.ID || got.Kind != tg.Kind || got.Ref.ADBSerial != tg.Ref.ADBSerial {
		t.Fatalf("runtime active target = %#v, want %#v", got, tg)
	}

	if err := svc.SetActive(target.Target{Kind: target.KindAndroidADB}); err == nil {
		t.Fatal("SetActive() expected validation error")
	}
}

func TestInputAdapter_MoveToRoutesThroughControllerTrace(t *testing.T) {
	rt := newAdapterTestRT(t, nil)
	rt.SetActiveWindow(winutil.WindowHandle{HWND: 66, Title: "After Effects", ClientW: 1920, ClientH: 1080})
	input := &recordingRuntimeInput{}
	rt.Input = input

	err := NewInputAdapter(rt).MoveTo(0.4, 0.6)
	if err != nil {
		t.Fatalf("MoveTo error = %v", err)
	}
	if len(input.moveHWND) != 1 || input.moveHWND[0] != 66 || input.moveX[0] != 0.4 || input.moveY[0] != 0.6 {
		t.Fatalf("backend MoveTo = hwnds %#v x %#v y %#v", input.moveHWND, input.moveX, input.moveY)
	}
	records := rt.TraceRecords()
	if len(records) != 1 {
		t.Fatalf("trace len = %d, want 1", len(records))
	}
	if records[0].Action != "move" || records[0].Target.ID != "win32:66" || records[0].Backend != "sendinput" {
		t.Fatalf("trace record = %#v", records[0])
	}
	if len(records[0].CoordinateSteps) != 1 {
		t.Fatalf("coordinate steps len = %d, want 1", len(records[0].CoordinateSteps))
	}
}

func TestInputAdapter_ScrollRoutesThroughControllerTrace(t *testing.T) {
	rt := newAdapterTestRT(t, nil)
	rt.SetActiveWindow(winutil.WindowHandle{HWND: 55, Title: "After Effects", ClientW: 1920, ClientH: 1080})
	input := &recordingRuntimeInput{}
	rt.Input = input

	err := NewInputAdapter(rt).Scroll(0.2, 0.8, -3, true)
	if err != nil {
		t.Fatalf("Scroll error = %v", err)
	}
	if len(input.scrollHWND) != 1 || input.scrollHWND[0] != 55 || input.scrollX[0] != 0.2 || input.scrollY[0] != 0.8 || input.scrollNotches[0] != -3 || !input.scrollHorizontal[0] {
		t.Fatalf("backend Scroll = hwnds %#v x %#v y %#v notches %#v horizontal %#v", input.scrollHWND, input.scrollX, input.scrollY, input.scrollNotches, input.scrollHorizontal)
	}
	records := rt.TraceRecords()
	if len(records) != 1 {
		t.Fatalf("trace len = %d, want 1", len(records))
	}
	if records[0].Action != "scroll" || records[0].Target.ID != "win32:55" || records[0].Backend != "sendinput" {
		t.Fatalf("trace record = %#v", records[0])
	}
	if len(records[0].CoordinateSteps) != 1 {
		t.Fatalf("coordinate steps len = %d, want 1", len(records[0].CoordinateSteps))
	}
}

func TestInputAdapter_KeyDownRoutesThroughControllerTrace(t *testing.T) {
	rt := newAdapterTestRT(t, nil)
	rt.SetActiveWindow(winutil.WindowHandle{HWND: 77, Title: "After Effects", ClientW: 1920, ClientH: 1080})
	input := &recordingRuntimeInput{}
	rt.Input = input

	err := NewInputAdapter(rt).KeyDown("ctrl")
	if err != nil {
		t.Fatalf("KeyDown error = %v", err)
	}
	if len(input.keyDownHWND) != 1 || input.keyDownHWND[0] != 77 || input.keyDownKeys[0] != "ctrl" {
		t.Fatalf("backend KeyDown = hwnds %#v keys %#v, want hwnd 77 key ctrl", input.keyDownHWND, input.keyDownKeys)
	}
	records := rt.TraceRecords()
	if len(records) != 1 {
		t.Fatalf("trace len = %d, want 1", len(records))
	}
	if records[0].Action != "key-down" || records[0].Target.ID != "win32:77" || records[0].Backend != "sendinput" {
		t.Fatalf("trace record = %#v", records[0])
	}
}

func TestInputAdapter_KeyUpRoutesThroughControllerTrace(t *testing.T) {
	rt := newAdapterTestRT(t, nil)
	rt.SetActiveWindow(winutil.WindowHandle{HWND: 88, Title: "After Effects", ClientW: 1920, ClientH: 1080})
	input := &recordingRuntimeInput{}
	rt.Input = input

	err := NewInputAdapter(rt).KeyUp("n")
	if err != nil {
		t.Fatalf("KeyUp error = %v", err)
	}
	if len(input.keyUpHWND) != 1 || input.keyUpHWND[0] != 88 || input.keyUpKeys[0] != "n" {
		t.Fatalf("backend KeyUp = hwnds %#v keys %#v, want hwnd 88 key n", input.keyUpHWND, input.keyUpKeys)
	}
	records := rt.TraceRecords()
	if len(records) != 1 {
		t.Fatalf("trace len = %d, want 1", len(records))
	}
	if records[0].Action != "key-up" || records[0].Target.ID != "win32:88" || records[0].Backend != "sendinput" {
		t.Fatalf("trace record = %#v", records[0])
	}
}

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

func TestWindowAdapter_Snapshot_ReReadsLiveSize(t *testing.T) {
	orig := clientSizeFn
	defer func() { clientSizeFn = orig }()
	clientSizeFn = func(win.HWND) (int, int, error) { return 1920, 1080, nil }

	rt := &RuntimeContext{}
	rt.SetActiveWindow(winutil.WindowHandle{HWND: 7, Title: "X", ClientW: 100, ClientH: 50})
	a := NewWindowAdapter(rt)
	w, err := a.Snapshot()
	if err != nil || w.HWND != 7 || w.Title != "X" || w.ClientW != 1920 || w.ClientH != 1080 {
		t.Fatalf("Snapshot 应重读 live 尺寸 1920x1080 (Title 仍快照): %+v %v", w, err)
	}
}

func TestWindowAdapter_Snapshot_NoActiveWindow(t *testing.T) {
	rt := &RuntimeContext{}
	if _, err := NewWindowAdapter(rt).Snapshot(); err != ErrNoActiveWindow {
		t.Fatalf("无活动窗口应返 ErrNoActiveWindow, got %v", err)
	}
}
