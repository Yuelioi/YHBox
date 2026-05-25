// node_services.go — Phase 5.4
//
// 桥接 RuntimeContext (+ stopwatchTable + zerolog) 到 node.* service interfaces.
// 给 Phase 5.5 ContainerRunner.execNode 通过 node.RunNode dispatch 真节点用.
//
// 8 个 adapter: log / vars / sys / stopwatch / input / window / capture / vision.
// 全部 hold *RuntimeContext (live 读 rt.Window/Input/Capture, setupRuntime 后才 populate).
package runtime

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"strings"
	"time"

	"github.com/lxn/win"
	"github.com/rs/zerolog"

	"yhbox/internal/node"
	"yhbox/internal/services/expr"
	"yhbox/pkg/vision"
)

// ============================================================================
// LogAdapter — zerolog → node.LogService
// ============================================================================

type logAdapter struct{ log zerolog.Logger }

func (a logAdapter) Debug(format string, args ...any) { a.log.Debug().Msgf(format, args...) }
func (a logAdapter) Info(format string, args ...any)  { a.log.Info().Msgf(format, args...) }
func (a logAdapter) Warn(format string, args ...any)  { a.log.Warn().Msgf(format, args...) }

// NewLogAdapter wrap zerolog into node.LogService.
func NewLogAdapter(log zerolog.Logger) node.LogService { return logAdapter{log: log} }

// ============================================================================
// VarStoreAdapter — RuntimeContext.vars → node.VarStore
// Phase 5.5 加 RegionRunner frame.LocalVars 时改 — 当前只走 global rt.vars.
// ============================================================================

type varStoreAdapter struct {
	rt    *RuntimeContext
	state func() *ExecState // live ExecState provider (frame stack 跟 Run 周期).
}

func (a *varStoreAdapter) Get(name string) (any, bool) {
	v, ok := a.rt.Vars()[name]
	return v, ok
}

func (a *varStoreAdapter) Set(name string, value any) {
	a.rt.SetVar(name, expr.Value(value))
}

func (a *varStoreAdapter) Inc(name string, delta float64) float64 {
	_ = a.rt.IncVar(name, delta)
	cur, _ := expr.AsNumber(a.rt.Vars()[name])
	return cur
}

// scoped 变种: scope=global → rt.vars; scope=local → frame.LocalVars; scope=auto →
// frame.LocalVars 已有 → local, 否则 global. 镜像老 execSetVar/execIncVar/evalGetVar.
//
// Adapter 持 *RuntimeContext, RuntimeContext.State() 拿当前 ExecState (LocalVars 栈).

func (a *varStoreAdapter) GetScoped(name, scope string) (any, bool) {
	switch scope {
	case "local":
		if v, ok := a.state().GetLocalVarHere(name); ok {
			return v, true
		}
		return nil, false
	case "auto":
		if v, ok := a.state().GetLocalVarChain(name); ok {
			return v, true
		}
		if v, ok := a.rt.Vars()[name]; ok {
			return v, true
		}
		return nil, false
	default: // global, "", others fallthrough
		if v, ok := a.rt.Vars()[name]; ok {
			return v, true
		}
		return nil, false
	}
}

func (a *varStoreAdapter) SetScoped(name, scope string, value any) {
	switch scope {
	case "local":
		a.state().SetLocalVar(name, expr.Value(value))
	case "auto":
		if _, ok := a.state().GetLocalVarHere(name); ok {
			a.state().SetLocalVar(name, expr.Value(value))
		} else {
			a.rt.SetVar(name, expr.Value(value))
		}
	default: // global, ""
		a.rt.SetVar(name, expr.Value(value))
	}
}

func (a *varStoreAdapter) IncScoped(name, scope string, delta float64) float64 {
	cur, _ := expr.AsNumber(a.getScopedRaw(name, scope))
	newV := cur + delta
	a.SetScoped(name, scope, newV)
	return newV
}

// getScopedRaw — IncScoped helper. scope 同 GetScoped, 但缺值返 0 不返 bool.
func (a *varStoreAdapter) getScopedRaw(name, scope string) any {
	v, _ := a.GetScoped(name, scope)
	return v
}

// NewVarStoreAdapter wrap *RuntimeContext into node.VarStore.
// state 是可选 ExecState getter — adapter 持有 closure, 让 scope=local/auto 能访问 frame
// stack. 没有 frame 上下文的 caller (单元测试) 不传; scoped 方法降级成 global.
func NewVarStoreAdapter(rt *RuntimeContext, state ...func() *ExecState) node.VarStore {
	g := func() *ExecState { return nil }
	if len(state) > 0 && state[0] != nil {
		g = state[0]
	}
	return &varStoreAdapter{rt: rt, state: g}
}

// ============================================================================
// SysStoreAdapter — RuntimeContext.sys + special-cases → node.SysStore
// 镜像 evalGetSys: now_ms / varLastChange.<name> live 读, 其余走 resolveSysPath.
// Phase 5.5 接 tick snapshot 时再加 currentTick 路径.
// ============================================================================

type sysStoreAdapter struct{ rt *RuntimeContext }

func (a *sysStoreAdapter) Get(path string) (any, bool) {
	if path == "now_ms" {
		return float64(nowMillis()), true
	}
	if name, ok := strings.CutPrefix(path, "varLastChange."); ok {
		return float64(a.rt.VarLastChange(name)), true
	}
	v, err := resolveSysPath(a.rt.Sys(), path)
	if err != nil {
		return nil, false
	}
	return v, true
}

// NewSysStoreAdapter wrap *RuntimeContext into node.SysStore.
func NewSysStoreAdapter(rt *RuntimeContext) node.SysStore { return &sysStoreAdapter{rt: rt} }

// ============================================================================
// StopwatchAdapter — stopwatchTable → node.StopwatchStore
// 1:1 pass-through, 镜像老 start/stop/read 语义.
// ============================================================================

type stopwatchAdapter struct{ tbl *stopwatchTable }

func (a *stopwatchAdapter) Start(key string)      { a.tbl.start(key) }
func (a *stopwatchAdapter) Stop(key string)       { a.tbl.stop(key) }
func (a *stopwatchAdapter) Read(key string) int64 { return a.tbl.read(key) }

// NewStopwatchAdapter wrap *stopwatchTable into node.StopwatchStore.
func NewStopwatchAdapter(tbl *stopwatchTable) node.StopwatchStore { return &stopwatchAdapter{tbl: tbl} }

// ============================================================================
// InputAdapter — pkginput.Backend (+ rt.Window.HWND) → node.InputService
// hwnd 每次方法调用 live 读 rt.Window.HWND, setupRuntime 后才有值.
// ============================================================================

type inputAdapter struct{ rt *RuntimeContext }

func (a *inputAdapter) hwnd() win.HWND { return win.HWND(a.rt.Window.HWND) }

func (a *inputAdapter) ensure() error {
	if a.rt.Input == nil {
		return fmt.Errorf("input backend not initialised (setupRuntime not run)")
	}
	return nil
}

func (a *inputAdapter) KeyPress(vk string, durationMs int) error {
	if err := a.ensure(); err != nil {
		return err
	}
	return a.rt.Input.KeyPress(a.hwnd(), vk, durationMs)
}

func (a *inputAdapter) KeyDown(vk string) error {
	if err := a.ensure(); err != nil {
		return err
	}
	return a.rt.Input.KeyDown(a.hwnd(), vk)
}

func (a *inputAdapter) KeyUp(vk string) error {
	if err := a.ensure(); err != nil {
		return err
	}
	return a.rt.Input.KeyUp(a.hwnd(), vk)
}

func (a *inputAdapter) Click(xRatio, yRatio float64, button string, durationMs int) error {
	if err := a.ensure(); err != nil {
		return err
	}
	return a.rt.Input.Click(a.hwnd(), xRatio, yRatio, button, durationMs)
}

func (a *inputAdapter) MouseMoveRel(dx, dy, durationMs int) error {
	if err := a.ensure(); err != nil {
		return err
	}
	return a.rt.Input.MouseMoveRel(a.hwnd(), dx, dy, durationMs)
}

func (a *inputAdapter) Scroll(xRatio, yRatio float64, notches int) error {
	if err := a.ensure(); err != nil {
		return err
	}
	return a.rt.Input.Scroll(a.hwnd(), xRatio, yRatio, notches)
}

func (a *inputAdapter) MouseDown(xRatio, yRatio float64, button string) error {
	if err := a.ensure(); err != nil {
		return err
	}
	return a.rt.Input.MouseDown(a.hwnd(), xRatio, yRatio, button)
}

func (a *inputAdapter) MouseUp(button string) error {
	if err := a.ensure(); err != nil {
		return err
	}
	return a.rt.Input.MouseUp(a.hwnd(), button)
}

// NewInputAdapter wrap *RuntimeContext into node.InputService.
func NewInputAdapter(rt *RuntimeContext) node.InputService { return &inputAdapter{rt: rt} }

// ============================================================================
// WindowAdapter — rt.Window + rt.Game → node.WindowService
// ============================================================================

type windowAdapter struct{ rt *RuntimeContext }

func (a *windowAdapter) BringForeground() error {
	if a.rt.Game == nil {
		return fmt.Errorf("game provider not initialised")
	}
	if !a.rt.Game.BringToForeground(a.rt.Window.HWND) {
		return fmt.Errorf("OS rejected BringToForeground")
	}
	return nil
}

func (a *windowAdapter) HWND() uintptr { return a.rt.Window.HWND }

func (a *windowAdapter) ClientSize() (int, int, error) {
	return a.rt.Window.ClientW, a.rt.Window.ClientH, nil
}

// NewWindowAdapter wrap *RuntimeContext into node.WindowService.
func NewWindowAdapter(rt *RuntimeContext) node.WindowService { return &windowAdapter{rt: rt} }

// ============================================================================
// CaptureAdapter — pkgcapture.IBackend → node.CaptureService
// 抓帧 + png.Encode 返字节流 (跟 screenshot.go 一致).
// ============================================================================

type captureAdapter struct{ rt *RuntimeContext }

func (a *captureAdapter) Capture() ([]byte, error) {
	if a.rt.Capture == nil {
		return nil, fmt.Errorf("capture backend not initialised")
	}
	frame, err := a.rt.Capture.Frame(win.HWND(a.rt.Window.HWND))
	if err != nil {
		return nil, err
	}
	if frame == nil {
		return nil, fmt.Errorf("capture: nil frame")
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, frame); err != nil {
		return nil, fmt.Errorf("png encode: %w", err)
	}
	return buf.Bytes(), nil
}

func (a *captureAdapter) CaptureROI(x, y, w, h int) ([]byte, error) {
	if a.rt.Capture == nil {
		return nil, fmt.Errorf("capture backend not initialised")
	}
	frame, err := a.rt.Capture.FrameROI(win.HWND(a.rt.Window.HWND), x, y, w, h)
	if err != nil {
		return nil, err
	}
	if frame == nil {
		return nil, fmt.Errorf("capture: nil frame")
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, frame); err != nil {
		return nil, fmt.Errorf("png encode: %w", err)
	}
	return buf.Bytes(), nil
}

// NewCaptureAdapter wrap *RuntimeContext into node.CaptureService.
func NewCaptureAdapter(rt *RuntimeContext) node.CaptureService { return &captureAdapter{rt: rt} }

// ============================================================================
// VisionAdapter — rt.Matcher + rt.Color + rt.Capture → node.VisionService
// Match/WaitMatch 走 Matcher; DetectColor 走 Color;
// DetectColorHSV / ROIColorScan / BarTrack 自抓帧 + 复用包内 helper (countHSVInROI / scanClusters / vision.AnalyzeBar).
// ============================================================================

// visionWaitPollMs WaitMatch 默认轮询间隔, 跟老 runtime defaultPollMs (100ms) 同值.
const visionWaitPollMs = 100

type visionAdapter struct{ rt *RuntimeContext }

func (a *visionAdapter) containerID() string {
	if a.rt.Container == nil {
		return ""
	}
	return a.rt.Container.ID
}

func (a *visionAdapter) Match(key string, threshold float64) (*node.Point, float64, error) {
	if a.rt.Matcher == nil {
		return nil, 0, nil
	}
	found, pt, _, err := a.rt.Matcher.Detect(context.Background(), a.containerID(), a.rt.Window.HWND, key, threshold, nil)
	if err != nil {
		return nil, 0, err
	}
	a.writeLastTemplate(found, pt)
	if !found {
		return nil, 0, nil
	}
	return &node.Point{X: pt.X, Y: pt.Y}, 1.0, nil
}

func (a *visionAdapter) WaitMatch(ctx context.Context, key string, threshold float64, timeout time.Duration) (*node.Point, float64, error) {
	if a.rt.Matcher == nil {
		return nil, 0, nil
	}
	if timeout <= 0 {
		// 单次 (interface 注: timeout<=0 视为 0).
		found, pt, _, err := a.rt.Matcher.Detect(ctx, a.containerID(), a.rt.Window.HWND, key, threshold, nil)
		if err != nil {
			return nil, 0, err
		}
		a.writeLastTemplate(found, pt)
		if !found {
			return nil, 0, nil
		}
		return &node.Point{X: pt.X, Y: pt.Y}, 1.0, nil
	}
	deadline := time.Now().Add(timeout)
	for {
		if err := ctx.Err(); err != nil {
			return nil, 0, err
		}
		found, pt, _, err := a.rt.Matcher.Detect(ctx, a.containerID(), a.rt.Window.HWND, key, threshold, nil)
		if err != nil {
			return nil, 0, err
		}
		if found {
			a.writeLastTemplate(true, pt)
			return &node.Point{X: pt.X, Y: pt.Y}, 1.0, nil
		}
		if time.Now().After(deadline) {
			a.writeLastTemplate(false, expr.Point{})
			return nil, 0, nil
		}
		select {
		case <-ctx.Done():
			return nil, 0, ctx.Err()
		case <-time.After(visionWaitPollMs * time.Millisecond):
		}
	}
}

// writeLastTemplate 写 SysState.LastFound/LastPoint — Match/WaitMatch 复刻老 runtime
// execCheckTemplate/execWaitTemplate 的副作用, 让 GetSys path=lastTemplate.{found,point}
// 下游节点能读. fishing-v2 watchdog_check / inspect_phase 等子图依赖.
func (a *visionAdapter) writeLastTemplate(found bool, pt expr.Point) {
	a.rt.UpdateSys(func(s *SysState) {
		s.LastFound = found
		if found {
			s.LastPoint = pt
		} else {
			s.LastPoint = expr.Point{}
		}
	})
}

func (a *visionAdapter) BarTrack(roi node.Rect) (node.BarTrackResult, error) {
	if a.rt.Capture == nil {
		return node.BarTrackResult{}, fmt.Errorf("capture backend not initialised")
	}
	hwnd := win.HWND(a.rt.Window.HWND)
	frame, err := a.rt.Capture.FrameROI(hwnd, int(roi.X), int(roi.Y), int(roi.W), int(roi.H))
	if err != nil || frame == nil {
		// 抓帧失败 (常见: HWND 失效 / 截图后台权限丢失) → 视 Missing 不冒泡 error.
		// 复刻老 runtime execColorBarTrack 行为.
		return node.BarTrackResult{Found: false}, nil
	}
	result := vision.AnalyzeBar(frame)
	if result == nil {
		return node.BarTrackResult{Found: false}, nil
	}
	out := node.BarTrackResult{
		CursorX:    result.CursorX,
		TargetX:    result.TargetX,
		TargetW:    result.TargetW,
		Confidence: result.Confidence,
		YellowPx:   result.YellowPx,
		GreenPx:    result.GreenPx,
	}
	out.Found = result.CursorX >= 0 && result.TargetX >= 0 && result.Confidence >= confBarV2

	// 老 runtime execColorBarTrack 写 SysState.LastBarTrack — state_FISHING 等子图通过
	// GetSys path=lastBarTrack.{cursorX,targetX,targetW,...} 读. atomic #5 拆老后这写回
	// 责任落到 VisionAdapter (新框架 ColorBarTrack 只 emit exec-data, 不知 SysState).
	// P1.3: SysState.LastBarTrack 直接 hold node.BarTrackResult, 不再字段拷贝.
	a.rt.UpdateSys(func(s *SysState) {
		s.LastBarTrack = out
	})
	return out, nil
}

func (a *visionAdapter) DetectColor(region [4]float64, mode string, rng [6]int) (int, float64, float64, error) {
	if a.rt.Color == nil {
		return 0, 0, 0, nil
	}
	count, cx, cy, err := a.rt.Color.Detect(context.Background(), a.rt.Window.HWND, region, mode, rng)
	if err == nil {
		// 老 runtime execDetectColor 写 LastColorCount/LastColorCenter — GetSys
		// path=lastColor.count/cx/cy 下游读.
		a.rt.UpdateSys(func(s *SysState) {
			s.LastColorCount = int64(count)
			s.LastColorCenter = expr.Point{X: cx, Y: cy}
		})
	}
	return count, cx, cy, err
}

func (a *visionAdapter) DetectColorHSV(roi node.Rect, hsv node.HSVRange) (int, float64, error) {
	if a.rt.Capture == nil {
		return 0, 0, fmt.Errorf("capture backend not initialised")
	}
	hwnd := win.HWND(a.rt.Window.HWND)
	frame, err := a.rt.Capture.FrameROI(hwnd, int(roi.X), int(roi.Y), int(roi.W), int(roi.H))
	if err != nil {
		return 0, 0, err
	}
	if frame == nil {
		return 0, 0, fmt.Errorf("capture: nil frame")
	}
	count, ratio := countHSVInROI(frame, hsvRangeFromNode(hsv))
	// 老 runtime execDetectColorHSV 写 LastDetect — GetSys path=lastDetect.pixelCount/pixelRatio 读.
	a.rt.UpdateSys(func(s *SysState) {
		s.LastDetect.PixelCount = count
		s.LastDetect.PixelRatio = ratio
	})
	return count, ratio, nil
}

func (a *visionAdapter) ROIColorScan(roi node.Rect, hsv node.HSVRange, axis string, minPx, maxPx int) ([]node.ClusterEntry, error) {
	if a.rt.Capture == nil {
		return nil, fmt.Errorf("capture backend not initialised")
	}
	hwnd := win.HWND(a.rt.Window.HWND)
	frame, err := a.rt.Capture.FrameROI(hwnd, int(roi.X), int(roi.Y), int(roi.W), int(roi.H))
	if err != nil {
		return nil, err
	}
	if frame == nil {
		return nil, fmt.Errorf("capture: nil frame")
	}
	internal := scanClusters(frame, hsvRangeFromNode(hsv), axis, minPx, maxPx)
	out := make([]node.ClusterEntry, len(internal))
	for i, c := range internal {
		out[i] = node.ClusterEntry{
			StartPx:  c.StartPx,
			EndPx:    c.EndPx,
			CenterPx: c.CenterPx,
			PxCount:  c.PxCount,
		}
	}
	// 老 runtime execROIColorScan 写 LastROIScan — GetSys path=lastROIScan.{clusterCount,clusters} 读.
	a.rt.UpdateSys(func(s *SysState) {
		s.LastROIScan.Clusters = internal
		s.LastROIScan.ClusterCount = len(internal)
	})
	return out, nil
}

// hsvRangeFromNode 转 node.HSVRange (导出字段) → 包内 hsvRange (非导出字段).
func hsvRangeFromNode(h node.HSVRange) hsvRange {
	return hsvRange{
		hMin: h.HMin, hMax: h.HMax,
		sMin: h.SMin, sMax: h.SMax,
		vMin: h.VMin, vMax: h.VMax,
	}
}

// NewVisionAdapter wrap *RuntimeContext into node.VisionService.
func NewVisionAdapter(rt *RuntimeContext) node.VisionService { return &visionAdapter{rt: rt} }

// ============================================================================
// ServiceBundle 构造 helper
// ============================================================================

// NewServiceBundleFor 用 RuntimeContext + stopwatchTable + logger 构造完整 ServiceBundle.
// Phase 5.5 ContainerRunner.execNode 拿这个走 node.RunNode dispatch.
//
// stateGetter — live ExecState 入口 (frame.LocalVars scope). ContainerRunner 构造时传
// func() *ExecState { return r.state }. nil 兜底 — adapter scoped 方法降级 global-only.
func NewServiceBundleFor(rt *RuntimeContext, stopwatches *stopwatchTable, log zerolog.Logger, stateGetter func() *ExecState) node.ServiceBundle {
	return node.ServiceBundle{
		Vision:      NewVisionAdapter(rt),
		Log:         NewLogAdapter(log),
		Input:       NewInputAdapter(rt),
		Vars:        NewVarStoreAdapter(rt, stateGetter),
		Sys:         NewSysStoreAdapter(rt),
		Window:      NewWindowAdapter(rt),
		Capture:     NewCaptureAdapter(rt),
		Stopwatches: NewStopwatchAdapter(stopwatches),
	}
}
