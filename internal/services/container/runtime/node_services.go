// node_services.go
//
// 桥接 RuntimeContext (+ stopwatchTable + zerolog) 到 node.* service interfaces.
// 给 ContainerRunner.execNode 通过 node.RunNode dispatch 真节点用.
//
// 8 个 adapter: log / vars / sys / stopwatch / input / window / capture / vision.
// 全部 hold *RuntimeContext (live 读 rt.Window/Input/Capture, setupRuntime 后才 populate).
package runtime

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"strings"
	"time"

	"github.com/lxn/win"
	"github.com/rs/zerolog"

	"yhbox/internal/node"
	"yhbox/internal/services/expr"
	clipruntime "yhbox/internal/services/inputclip/runtime"
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
// frame.LocalVars 已有 → local, 否则 global.
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
// SysStoreAdapter — RuntimeContext.sys + special-cases → node.SysStore.
// 镜像 GetSys path 解析: now_ms / varLastChange.<name> live 读, 其余走 resolveSysPath.
// 注: PureData Evaluator 经 framework snapshot wrap 拿到的是 frozenSysStoreAdapter
// (持 tick-frozen SysState), 不是这个 live adapter. live adapter 给 Runnable 节点.
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
// ParamStoreAdapter — frame.LocalParams → node.ParamStore
// 当前 frame 是 dispatch live, 持 *ExecState getter (跟 varStoreAdapter 同模式).
// ============================================================================

type paramStoreAdapter struct {
	state func() *ExecState
}

func (a *paramStoreAdapter) Get(name string) (any, bool) {
	if a.state == nil {
		return nil, false
	}
	s := a.state()
	if s == nil || s.CurrentFrame == nil {
		return nil, false
	}
	v, ok := s.CurrentFrame.LocalParams[name]
	return v, ok
}

// NewParamStoreAdapter wrap *ExecState getter into node.ParamStore.
func NewParamStoreAdapter(state func() *ExecState) node.ParamStore {
	return &paramStoreAdapter{state: state}
}

// ============================================================================
// FrozenSysStoreAdapter — Snapshot.Sys 的 runtime-side wrapper.
// 持 frozen SysState 值 + live rt (for now_ms / varLastChange.X escape).
// PureData Evaluator 通过 ctx.Sys() 读, framework wrap 时把 services.Sys 换成此实例.
// ============================================================================

type frozenSysStoreAdapter struct {
	sys SysState        // frozen snapshot
	rt  *RuntimeContext // for now_ms / varLastChange.X live read
}

func (a *frozenSysStoreAdapter) Get(path string) (any, bool) {
	if path == "now_ms" {
		return float64(nowMillis()), true
	}
	if name, ok := strings.CutPrefix(path, "varLastChange."); ok {
		return float64(a.rt.VarLastChange(name)), true
	}
	v, err := resolveSysPath(a.sys, path)
	if err != nil {
		return nil, false
	}
	return v, true
}

// newFrozenSysStore wraps a frozen SysState + live rt closure into node.SysStore.
// Called by ContainerRunner when building bundle.Snapshot getter.
func newFrozenSysStore(sys SysState, rt *RuntimeContext) node.SysStore {
	return &frozenSysStoreAdapter{sys: sys, rt: rt}
}

// ============================================================================
// StopwatchAdapter — stopwatchTable → node.StopwatchStore
// 1:1 pass-through (start/stop/read).
// ============================================================================

type stopwatchAdapter struct{ tbl *stopwatchTable }

func (a *stopwatchAdapter) Start(key string)      { a.tbl.start(key) }
func (a *stopwatchAdapter) Stop(key string)       { a.tbl.stop(key) }
func (a *stopwatchAdapter) Read(key string) int64 { return a.tbl.read(key) }

// NewStopwatchAdapter wrap *stopwatchTable into node.StopwatchStore.
func NewStopwatchAdapter(tbl *stopwatchTable) node.StopwatchStore { return &stopwatchAdapter{tbl: tbl} }

// ============================================================================
// ClipPlayerAdapter — RuntimeContext (ClipResolver + InputBackend + InputBus +
// ClipPolicy + MouseCounts360 + Window) → node.ClipPlayer. PlayClip 节点用.
// ============================================================================

type clipPlayerAdapter struct{ rt *RuntimeContext }

func newClipPlayerAdapter(rt *RuntimeContext) node.ClipPlayer { return &clipPlayerAdapter{rt: rt} }

// Play 阻塞回放 clipID. 抢 InputBus 独占锁 hold 整段, ctx 取消即 Cancel + 释放按下键.
// keepRanges 暂传 nil — 整段播, 片段裁剪未实装.
func (a *clipPlayerAdapter) Play(ctx context.Context, clipID string) error {
	rt := a.rt
	if rt.ClipResolver == nil {
		return errors.New("PlayClip: ClipResolver 未注入 (main.go SetClipResolver?)")
	}
	clip, ok := rt.ClipResolver.Resolve(clipID)
	if !ok {
		return fmt.Errorf("PlayClip: clip %q 未找到", clipID)
	}
	if rt.InputBackend == nil {
		return errors.New("PlayClip: InputBackend 未注入")
	}

	rt.InputBus.Lock()
	defer rt.InputBus.Unlock()

	p := clipruntime.NewClipPlayer(clip, nil, rt.InputBackend, rt.ClipPolicy, rt.MouseCounts360,
		func() (int, int) {
			if rt.Window.HWND == 0 {
				return 0, 0 // 未解析窗口 → 不缩放
			}
			return rt.Window.ClientW, rt.Window.ClientH
		},
	)
	p.Start(ctx)

	select {
	case <-p.Done():
		// ErrCancelled (Cancel 触发) 视为正常收尾; ctx.Canceled 透传给 dispatch 优雅 halt;
		// 其它是真回放错 (backend.Send 失败等) → 业务 error 出口.
		if err := p.Wait(); err != nil && !errors.Is(err, clipruntime.ErrCancelled) {
			return err
		}
		return nil
	case <-ctx.Done():
		p.Cancel()
		<-p.Done()
		return ctx.Err()
	}
}

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

func (a *inputAdapter) MoveTo(xRatio, yRatio float64) error {
	if err := a.ensure(); err != nil {
		return err
	}
	return a.rt.Input.MoveTo(a.hwnd(), xRatio, yRatio)
}

func (a *inputAdapter) CursorRatio() (float64, float64, error) {
	if err := a.ensure(); err != nil {
		return 0, 0, err
	}
	return a.rt.Input.CursorRatio(a.hwnd())
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

func (a *captureAdapter) CaptureROI(roi node.Geometry) ([]byte, error) {
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
	sub := cropFrameByGeometry(frame, roi)
	var buf bytes.Buffer
	if err := png.Encode(&buf, sub); err != nil {
		return nil, fmt.Errorf("png encode: %w", err)
	}
	return buf.Bytes(), nil
}

// NewCaptureAdapter wrap *RuntimeContext into node.CaptureService.
func NewCaptureAdapter(rt *RuntimeContext) node.CaptureService { return &captureAdapter{rt: rt} }

// ============================================================================
// VisionAdapter — rt.Matcher + rt.Capture → node.VisionService
// Match/WaitMatch 走 Matcher; DetectColor 走 rt.Capture + CaptureFrameCached (100ms 缓存);
// DetectColorHSV / ROIColorScan / DualBarTrack 自抓帧 + 复用包内 helper (countHSVInROI / scanClusters / vision.AnalyzeDualColorBar).
// ============================================================================

// visionWaitPollMs WaitMatch 默认轮询间隔 (ms).
const visionWaitPollMs = 100

type visionAdapter struct{ rt *RuntimeContext }

func (a *visionAdapter) containerID() string {
	if a.rt.Container == nil {
		return ""
	}
	return a.rt.Container.ID
}

func (a *visionAdapter) Match(ctx context.Context, keys []string, threshold float64, mode string) (*node.Point, float64, error) {
	if a.rt.Matcher == nil || len(keys) == 0 {
		return nil, 0, nil
	}
	return a.matchOnce(ctx, keys, threshold, mode)
}

func (a *visionAdapter) WaitMatch(ctx context.Context, keys []string, threshold float64, mode string, timeout time.Duration) (*node.Point, float64, error) {
	if a.rt.Matcher == nil || len(keys) == 0 {
		return nil, 0, nil
	}
	if timeout <= 0 {
		return a.matchOnce(ctx, keys, threshold, mode) // 单帧 (interface 注: timeout<=0 视为 0).
	}
	deadline := time.Now().Add(timeout)
	bestConf := 0.0 // 轮询期间见过的最高匹配度; 超时时返回它供诊断 (「差多少」)
	for {
		if err := ctx.Err(); err != nil {
			return nil, 0, err
		}
		pt, conf, err := a.matchOnce(ctx, keys, threshold, mode)
		if err != nil {
			return nil, 0, err
		}
		if conf > bestConf {
			bestConf = conf
		}
		if pt != nil {
			return pt, conf, nil
		}
		if time.Now().After(deadline) {
			return nil, bestConf, nil
		}
		select {
		case <-ctx.Done():
			return nil, 0, ctx.Err()
		case <-time.After(visionWaitPollMs * time.Millisecond):
		}
	}
}

// matchOnce 单帧多模板判定. mode="all": 全部 key 同帧命中才算命中 (点取首个 key); 否则 "any":
// 按列表序取首个命中. 写 SysState.LastFound/LastPoint (整体命中与否 + 命中点).
func (a *visionAdapter) matchOnce(ctx context.Context, keys []string, threshold float64, mode string) (*node.Point, float64, error) {
	var frame *image.RGBA
	if a.rt.Capture != nil {
		f, err := a.rt.CaptureFrameCached(a.rt.Window.HWND)
		if err != nil {
			return nil, 0, err
		}
		frame = f
	}
	if mode == "all" {
		var firstPt expr.Point
		minConf := 1.0 // all 命中 = 全部 ≥ 阈值; 报最弱那个 (瓶颈) 的真实匹配度
		for idx, key := range keys {
			found, pt, _, conf, err := a.rt.Matcher.Detect(ctx, a.containerID(), frame, key, threshold, nil)
			if err != nil {
				return nil, 0, err
			}
			if conf < minConf {
				minConf = conf
			}
			if !found {
				a.writeLastTemplate(false, expr.Point{})
				return nil, conf, nil // 报这个没过的 key 的真实匹配度
			}
			if idx == 0 {
				firstPt = pt
			}
		}
		a.writeLastTemplate(true, firstPt)
		return &node.Point{X: firstPt.X, Y: firstPt.Y}, minConf, nil
	}
	// any (默认)
	bestConf := 0.0 // 全 miss 时报见过的最高匹配度 (诊断: 差多少)
	for _, key := range keys {
		found, pt, _, conf, err := a.rt.Matcher.Detect(ctx, a.containerID(), frame, key, threshold, nil)
		if err != nil {
			return nil, 0, err
		}
		if conf > bestConf {
			bestConf = conf
		}
		if found {
			a.writeLastTemplate(true, pt)
			return &node.Point{X: pt.X, Y: pt.Y}, conf, nil // 命中: 报真实匹配度 (不再写死 1.0)
		}
	}
	a.writeLastTemplate(false, expr.Point{})
	return nil, bestConf, nil
}

// writeLastTemplate 写 SysState.LastFound/LastPoint (Match/WaitMatch), 让 GetSys
// path=lastTemplate.{found,point} 下游节点能读. fishing-v2 watchdog_check / inspect_phase
// 等子图依赖.
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

func (a *visionAdapter) DualBarTrack(roi node.Geometry, inner, outer node.HSVRange, opts node.DualBarOptions) (node.DualColorBarResult, error) {
	if a.rt.Capture == nil {
		return node.DualColorBarResult{}, fmt.Errorf("capture backend not initialised")
	}
	hwnd := win.HWND(a.rt.Window.HWND)
	frame, err := a.rt.Capture.Frame(hwnd)
	if err != nil || frame == nil {
		// 抓帧失败 (常见: HWND 失效 / 截图后台权限丢失) → 视 Missing 不冒泡 error.
		return node.DualColorBarResult{Found: false}, nil
	}
	sub := cropFrameByGeometry(frame, roi)
	vInner := vision.HSVRange{HMin: inner.HMin, HMax: inner.HMax, SMin: inner.SMin, SMax: inner.SMax, VMin: inner.VMin, VMax: inner.VMax}
	vOuter := vision.HSVRange{HMin: outer.HMin, HMax: outer.HMax, SMin: outer.SMin, SMax: outer.SMax, VMin: outer.VMin, VMax: outer.VMax}
	vOpts := vision.DualBarOptions{
		InnerMinPx:      opts.InnerMinPx,
		InnerMaxPx:      opts.InnerMaxPx,
		OuterMinPx:      opts.OuterMinPx,
		BandRatioH:      opts.BandRatioH,
		BandRatioInner:  opts.BandRatioInner,
		ConfInnerWeight: opts.ConfInnerWeight,
		ConfOuterWeight: opts.ConfOuterWeight,
	}
	result := vision.AnalyzeDualColorBar(sub, vInner, vOuter, vOpts)
	if result == nil {
		return node.DualColorBarResult{Found: false}, nil
	}
	out := node.DualColorBarResult{
		Found:      result.Found && result.Confidence >= confBarV2,
		InnerX:     result.InnerX,
		OuterX:     result.OuterX,
		OuterWidth: result.OuterWidth,
		Confidence: result.Confidence,
		InnerPx:    result.InnerPx,
		OuterPx:    result.OuterPx,
	}

	// 写 SysState.LastDualBarTrack — fishing-v2 state_FISHING 子图通过 GetSys
	// path=lastDualBarTrack.{innerX,outerX,outerWidth,...} 读. 写回责任落在 VisionAdapter
	// (节点只 emit exec-data, 不碰 SysState).
	a.rt.UpdateSys(func(s *SysState) {
		s.LastDualBarTrack = out
	})
	return out, nil
}

func (a *visionAdapter) DetectColor(roi node.Geometry, mode string, rng [6]int) (int, float64, float64, error) {
	if a.rt.Capture == nil {
		return 0, 0, 0, nil
	}
	frame, err := a.rt.CaptureFrameCached(a.rt.Window.HWND)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("capture: %w", err)
	}
	if frame == nil {
		return 0, 0, 0, fmt.Errorf("capture: nil frame")
	}
	frameW, frameH := frame.Bounds().Dx(), frame.Bounds().Dy()
	// ResolveGeometry 给全帧上的像素 rect (override 优先 > pct×帧 > 全帧), 已 clamp.
	// 在全帧坐标系数像素并累加中心 → 中心还原成全帧比例.
	x0, y0, w, h, _ := ResolveGeometry(roi, frameW, frameH)
	count, sumX, sumY := countColorPixels(frame, x0, y0, x0+w, y0+h, mode, rng)
	var cx, cy float64
	if count > 0 {
		cx = float64(sumX) / float64(count) / float64(frameW)
		cy = float64(sumY) / float64(count) / float64(frameH)
	}
	// 写 LastColorCount/LastColorCenter — GetSys path=lastColor.count/cx/cy 下游读.
	a.rt.UpdateSys(func(s *SysState) {
		s.LastColorCount = int64(count)
		s.LastColorCenter = expr.Point{X: cx, Y: cy}
	})
	return count, cx, cy, nil
}

func (a *visionAdapter) DetectColorHSV(roi node.Geometry, hsv node.HSVRange) (int, float64, error) {
	if a.rt.Capture == nil {
		return 0, 0, fmt.Errorf("capture backend not initialised")
	}
	hwnd := win.HWND(a.rt.Window.HWND)
	frame, err := a.rt.Capture.Frame(hwnd)
	if err != nil {
		return 0, 0, err
	}
	if frame == nil {
		return 0, 0, fmt.Errorf("capture: nil frame")
	}
	sub := cropFrameByGeometry(frame, roi)
	count, ratio := countHSVInROI(sub, hsvRangeFromNode(hsv))
	// 写 LastDetect — GetSys path=lastDetect.pixelCount/pixelRatio 读.
	a.rt.UpdateSys(func(s *SysState) {
		s.LastDetect.PixelCount = count
		s.LastDetect.PixelRatio = ratio
	})
	return count, ratio, nil
}

func (a *visionAdapter) ROIColorScan(roi node.Geometry, hsv node.HSVRange, axis string, minPx, maxPx int) ([]node.ClusterEntry, error) {
	if a.rt.Capture == nil {
		return nil, fmt.Errorf("capture backend not initialised")
	}
	hwnd := win.HWND(a.rt.Window.HWND)
	frame, err := a.rt.Capture.Frame(hwnd)
	if err != nil {
		return nil, err
	}
	if frame == nil {
		return nil, fmt.Errorf("capture: nil frame")
	}
	sub := cropFrameByGeometry(frame, roi)
	// maxPx<=0: 按子帧实际尺寸算默认上限 (axis x → 宽/3, y → 高/3), 与节点解耦.
	if maxPx <= 0 {
		if axis == "x" {
			maxPx = sub.Bounds().Dx() / 3
		} else {
			maxPx = sub.Bounds().Dy() / 3
		}
	}
	internal := scanClusters(sub, hsvRangeFromNode(hsv), axis, minPx, maxPx)
	out := make([]node.ClusterEntry, len(internal))
	for i, c := range internal {
		out[i] = node.ClusterEntry{
			StartPx:  c.StartPx,
			EndPx:    c.EndPx,
			CenterPx: c.CenterPx,
			PxCount:  c.PxCount,
		}
	}
	// 写 LastROIScan — GetSys path=lastROIScan.{clusterCount,clusters} 读.
	a.rt.UpdateSys(func(s *SysState) {
		s.LastROIScan.Clusters = internal
		s.LastROIScan.ClusterCount = len(internal)
	})
	return out, nil
}

// cropFrameByGeometry 全帧抓到后按 Geometry 解析的像素区裁子图.
// fullFrame=true (Geometry 零值/无匹配分辨率且 Pct 全 0) 直接返原帧, 避免无谓拷贝.
// 所有 override / pct 路径统一经 ResolveGeometry, 是通用 adapter crop 的规范范式.
func cropFrameByGeometry(frame *image.RGBA, roi node.Geometry) *image.RGBA {
	x, y, w, h, fullFrame := ResolveGeometry(roi, frame.Bounds().Dx(), frame.Bounds().Dy())
	if fullFrame || w <= 0 || h <= 0 {
		return frame
	}
	sub := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(sub, sub.Bounds(), frame, image.Point{X: x, Y: y}, draw.Src)
	return sub
}

// GridSignature 抓全帧后按 roi Geometry 裁子区 → box-average 降采样成
// gridSize×gridSize RGB 签名. Geometry 零值 = 全帧; 每调一次新抓一帧 (无缓存).
func (a *visionAdapter) GridSignature(roi node.Geometry, gridSize int) ([]uint8, error) {
	if a.rt.Capture == nil {
		return nil, fmt.Errorf("capture backend not initialised")
	}
	hwnd := win.HWND(a.rt.Window.HWND)
	frame, err := a.rt.Capture.Frame(hwnd)
	if err != nil {
		return nil, err
	}
	if frame == nil {
		return nil, fmt.Errorf("capture: nil frame")
	}
	sub := cropFrameByGeometry(frame, roi)
	return vision.Downsample(sub, gridSize), nil
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
// ContainerRunner.execNode 拿这个走 node.RunNode dispatch.
//
// stateGetter — live ExecState 入口 (frame.LocalVars / LocalParams scope). ContainerRunner
// 构造时传 func() *ExecState { return r.state }. nil 兜底 — adapter scoped 方法降级.
//
// Snapshot — per-tick view 从 ctx (tickCtxKey) 读. dispatchInRegion 入口
// withTickSnapshot 写入, EvaluatePureData wrap 时调 Snapshot(ctx) 拿 frozen Vars/Sys view.
// ctx 无 value → 返空 Snapshot{}, 等价跳过 wrap.
func NewServiceBundleFor(
	rt *RuntimeContext,
	stopwatches *stopwatchTable,
	log zerolog.Logger,
	stateGetter func() *ExecState,
) node.ServiceBundle {
	return node.ServiceBundle{
		Vision:      NewVisionAdapter(rt),
		Log:         NewLogAdapter(log),
		Input:       NewInputAdapter(rt),
		Vars:        NewVarStoreAdapter(rt, stateGetter),
		Sys:         NewSysStoreAdapter(rt),
		Params:      NewParamStoreAdapter(stateGetter),
		Window:      NewWindowAdapter(rt),
		Capture:     NewCaptureAdapter(rt),
		Stopwatches: NewStopwatchAdapter(stopwatches),
		Clip:        newClipPlayerAdapter(rt),
		Snapshot: func(ctx context.Context) node.Snapshot {
			tick := tickFromCtx(ctx)
			if tick == nil {
				return node.Snapshot{}
			}
			return node.Snapshot{
				Vars: tick.Vars,
				Sys:  newFrozenSysStore(tick.Sys, rt),
			}
		},
	}
}
