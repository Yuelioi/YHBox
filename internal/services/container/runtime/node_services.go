// node_services.go
//
// 桥接 RuntimeContext (+ stopwatchTable + zerolog) 到 node.* service interfaces.
// 给 ContainerRunner.execNode 通过 node.RunNode dispatch 真节点用.
//
// adapter: log / vars / param / stopwatch / input / window / capture / vision / clip.
// 全部 hold *RuntimeContext (live 经 rt.ActiveHWND()/WindowHandle() 读当前活动窗口及 Input/Capture, setupRuntime 后才 populate).
package runtime

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"math"
	"sort"
	"time"

	"github.com/lxn/win"
	"github.com/rs/zerolog"

	"yotta/internal/automation/controller"
	"yotta/internal/automation/target"
	automationtrace "yotta/internal/automation/trace"
	"yotta/internal/node"
	"yotta/internal/services/container"
	"yotta/internal/services/expr"
	clipruntime "yotta/internal/services/inputclip/runtime"
	"yotta/pkg/vision"
	"yotta/pkg/winutil"
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

// Names — 当前已知变量名全集 (global; 含运行中动态建的)。Script 绑定层注入 $name
// live getter 用 (可选能力接口, 见 services/script.varNamer — 不进 node.VarStore 主接口,
// 免得十来个测试 fake 全陪绑)。
func (a *varStoreAdapter) Names() []string {
	vars := a.rt.Vars()
	out := make([]string, 0, len(vars))
	for k := range vars {
		out = append(out, k)
	}
	return out
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

// LastChange 透传 rt.VarLastChange — 该变量上次 Set/Inc 的 unix-ms; 没设过返 0.
func (a *varStoreAdapter) LastChange(name string) int64 { return a.rt.VarLastChange(name) }

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
			wh := rt.WindowHandle()
			if wh.HWND == 0 {
				return 0, 0 // 未解析窗口 → 不缩放
			}
			return wh.ClientW, wh.ClientH
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
// InputAdapter — pkginput.Backend (+ 当前活动窗口 hwnd) → node.InputService
// hwnd 每次方法调用经 rt.ActiveHWND() live 读, setupRuntime 后才有值.
// ============================================================================

type inputAdapter struct {
	rt          *RuntimeContext
	traceSource automationtrace.ActionSource
}

func (a *inputAdapter) hwnd() (win.HWND, error) {
	h, err := a.rt.ActiveHWND()
	return win.HWND(h), err
}

func (a *inputAdapter) ensure() error {
	if a.rt.Input == nil {
		return fmt.Errorf("input backend not initialised (setupRuntime not run)")
	}
	return nil
}

func (a *inputAdapter) inputController() (controller.Controller, error) {
	return a.rt.controllerForActiveTarget(a.traceSource, controllerNeed{Input: true})
}

func (a *inputAdapter) pointerController() (controller.PointerInput, error) {
	ctrl, err := a.inputController()
	if err != nil {
		return nil, err
	}
	pointer, ok := ctrl.(controller.PointerInput)
	if !ok {
		return nil, fmt.Errorf("active controller %T does not support pointer input", ctrl)
	}
	return pointer, nil
}

func (a *inputAdapter) keyboardController() (controller.KeyboardInput, error) {
	ctrl, err := a.inputController()
	if err != nil {
		return nil, err
	}
	keyboard, ok := ctrl.(controller.KeyboardInput)
	if !ok {
		return nil, fmt.Errorf("active controller %T does not support keyboard input", ctrl)
	}
	return keyboard, nil
}

func (a *inputAdapter) KeyDown(vk string) error {
	ctrl, err := a.keyboardController()
	if err != nil {
		return err
	}
	return ctrl.KeyDown(context.Background(), controller.KeyRequest{
		Key: vk,
		Policy: controller.ActionPolicy{
			ForegroundRequired: true,
		},
	})
}

func (a *inputAdapter) KeyUp(vk string) error {
	ctrl, err := a.keyboardController()
	if err != nil {
		return err
	}
	return ctrl.KeyUp(context.Background(), controller.KeyRequest{
		Key: vk,
		Policy: controller.ActionPolicy{
			ForegroundRequired: true,
		},
	})
}

func (a *inputAdapter) Click(xRatio, yRatio float64, button string, durationMs int) error {
	ctrl, err := a.pointerController()
	if err != nil {
		return err
	}
	return ctrl.Click(context.Background(), controller.ClickRequest{
		Point:      target.NewNormalizedPoint(xRatio, yRatio),
		Button:     button,
		DurationMs: durationMs,
		Policy: controller.ActionPolicy{
			ForegroundRequired: true,
		},
	})
}

func (a *inputAdapter) MouseMoveRel(dx, dy, durationMs int) error {
	ctrl, err := a.pointerController()
	if err != nil {
		return err
	}
	return ctrl.MoveRelative(context.Background(), controller.RelativeMoveRequest{
		Dx:         dx,
		Dy:         dy,
		DurationMs: durationMs,
		Policy: controller.ActionPolicy{
			ForegroundRequired: true,
		},
	})
}

func (a *inputAdapter) MoveTo(xRatio, yRatio float64) error {
	ctrl, err := a.pointerController()
	if err != nil {
		return err
	}
	return ctrl.Move(context.Background(), controller.MoveRequest{
		Point: target.NewNormalizedPoint(xRatio, yRatio),
		Policy: controller.ActionPolicy{
			ForegroundRequired: true,
		},
	})
}

func (a *inputAdapter) CursorRatio() (float64, float64, error) {
	if err := a.ensure(); err != nil {
		return 0, 0, err
	}
	h, err := a.hwnd()
	if err != nil {
		return 0, 0, err
	}
	return a.rt.Input.CursorRatio(h)
}

func (a *inputAdapter) Scroll(xRatio, yRatio float64, notches int, horizontal bool) error {
	ctrl, err := a.pointerController()
	if err != nil {
		return err
	}
	return ctrl.Scroll(context.Background(), controller.ScrollRequest{
		Point:      target.NewNormalizedPoint(xRatio, yRatio),
		Notches:    notches,
		Horizontal: horizontal,
		Policy: controller.ActionPolicy{
			ForegroundRequired: true,
		},
	})
}

func (a *inputAdapter) Drag(x1, y1, x2, y2 float64, button string, durationMs int) error {
	ctrl, err := a.pointerController()
	if err != nil {
		return err
	}
	return ctrl.Drag(context.Background(), controller.DragRequest{
		From:       target.NewNormalizedPoint(x1, y1),
		To:         target.NewNormalizedPoint(x2, y2),
		Button:     button,
		DurationMs: durationMs,
		Policy: controller.ActionPolicy{
			ForegroundRequired: true,
		},
	})
}

func (a *inputAdapter) MouseDown(xRatio, yRatio float64, button string) error {
	ctrl, err := a.pointerController()
	if err != nil {
		return err
	}
	return ctrl.MouseDown(context.Background(), controller.MouseButtonRequest{
		Point:  target.NewNormalizedPoint(xRatio, yRatio),
		Button: button,
		Policy: controller.ActionPolicy{
			ForegroundRequired: true,
		},
	})
}

func (a *inputAdapter) MouseUp(button string) error {
	ctrl, err := a.pointerController()
	if err != nil {
		return err
	}
	return ctrl.MouseUp(context.Background(), controller.MouseButtonRequest{
		Button: button,
		Policy: controller.ActionPolicy{
			ForegroundRequired: true,
		},
	})
}

func (a *inputAdapter) TypeText(s string) error {
	ctrl, err := a.keyboardController()
	if err != nil {
		return err
	}
	return ctrl.Text(context.Background(), controller.TextRequest{
		Text: s,
		Policy: controller.ActionPolicy{
			ForegroundRequired: true,
		},
	})
}

// NewInputAdapter wrap *RuntimeContext into node.InputService.
func NewInputAdapter(rt *RuntimeContext) node.InputService { return &inputAdapter{rt: rt} }

func newInputAdapterWithSource(rt *RuntimeContext, source automationtrace.ActionSource) node.InputService {
	return &inputAdapter{rt: rt, traceSource: source}
}

// ============================================================================
// WindowAdapter — 当前活动窗口（经 rt.WindowHandle()/SetActiveWindow() 访问）+ rt.Game → node.WindowService
// ============================================================================

type windowAdapter struct{ rt *RuntimeContext }

func (a *windowAdapter) BringForeground() error {
	if a.rt.Game == nil {
		return fmt.Errorf("game provider not initialised")
	}
	h, err := a.rt.ActiveHWND()
	if err != nil {
		return err
	}
	if !a.rt.Game.BringToForeground(h) {
		return fmt.Errorf("OS rejected BringToForeground")
	}
	return nil
}

func (a *windowAdapter) HWND() uintptr {
	h, _ := a.rt.ActiveHWND()
	return h
}

func (a *windowAdapter) ClientSize() (int, int, error) {
	wh := a.rt.WindowHandle()
	if wh.HWND == 0 {
		return 0, 0, ErrNoActiveWindow
	}
	return wh.ClientW, wh.ClientH, nil
}

// resolveWindowFn 测试可替换; 默认真 Win32 解析.
var resolveWindowFn = winutil.ResolveWindow

// clientSizeFn 测试可替换; 默认 live Win32 GetClientRect.
var clientSizeFn = func(h win.HWND) (int, int, error) { return winutil.ClientSize(h) }

func (a *windowAdapter) SetActive(ctx context.Context, title, class, processName, titleMatch string) error {
	spec := winutil.MatchSpec{Title: title, Class: class, ProcessName: processName, TitleMatch: titleMatch}
	wh, err := resolveWindowFn(ctx, spec, 3*time.Second, 500*time.Millisecond)
	if err != nil {
		return fmt.Errorf("WindowTarget resolve: %w", err)
	}
	a.rt.SetActiveWindow(wh) // 整体替换 + 清该 hwnd 帧缓存

	// SendInput 后端需前台焦点才能注入到目标窗口 → 解析窗口时拉到前台 (固定 150ms 有界等待).
	// PostMessage 按 hwnd 直发, 不激活. (原 RunMode=foreground 已并入 InputBackend=sendinput.)
	if a.rt.Container != nil && a.rt.Container.InputBackend == "sendinput" && a.rt.Game != nil {
		a.rt.Game.BringToForeground(wh.HWND)
		time.Sleep(150 * time.Millisecond)
	}

	// ROI 分辨率检查.
	a.rt.emitROIResolutionWarnings(wh.ClientW, wh.ClientH)
	return nil
}

func (a *windowAdapter) Snapshot() (node.Window, error) {
	wh := a.rt.WindowHandle()
	if wh.HWND == 0 {
		return node.Window{}, ErrNoActiveWindow
	}
	w := node.Window{
		HWND: wh.HWND, Title: wh.Title, Class: wh.Class,
		Process: wh.ProcessName, PID: wh.PID, ClientW: wh.ClientW, ClientH: wh.ClientH,
	}
	// 操作后按 live HWND 重读客户区尺寸 — maximize/move/resize 改了, Done.Window 必须反映新值;
	// 读失败 (窗口已关等) 退化保留解析时缓存值, 不让已成功的操作因事后读取失败而报错。
	if cw, ch, err := clientSizeFn(win.HWND(wh.HWND)); err == nil {
		w.ClientW, w.ClientH = cw, ch
	}
	return w, nil
}

func (a *windowAdapter) Maximize() error {
	h, err := a.rt.ActiveHWND()
	if err != nil {
		return err
	}
	return winutil.Maximize(h)
}

func (a *windowAdapter) Minimize() error {
	h, err := a.rt.ActiveHWND()
	if err != nil {
		return err
	}
	return winutil.Minimize(h)
}

func (a *windowAdapter) Restore() error {
	h, err := a.rt.ActiveHWND()
	if err != nil {
		return err
	}
	return winutil.Restore(h)
}

func (a *windowAdapter) MoveResize(x, y, w, h int) error {
	hwnd, err := a.rt.ActiveHWND()
	if err != nil {
		return err
	}
	return winutil.MoveResize(hwnd, x, y, w, h)
}

func (a *windowAdapter) Close() error {
	h, err := a.rt.ActiveHWND()
	if err != nil {
		return err
	}
	return winutil.CloseWindow(h)
}

func (a *windowAdapter) BorderlessFullscreen() error {
	h, err := a.rt.ActiveHWND()
	if err != nil {
		return err
	}
	saved, err := winutil.EnterBorderless(h)
	if err != nil {
		return err
	}
	a.rt.saveBorderless(h, saved)
	return nil
}

func (a *windowAdapter) RestoreBorders() error {
	h, err := a.rt.ActiveHWND()
	if err != nil {
		return err
	}
	saved, ok := a.rt.takeBorderless(h)
	if ok && winutil.WindowPID(h) != saved.PID {
		ok = false
	}
	if !ok {
		saved = winutil.SavedWindow{}
	}
	return winutil.ExitBorderless(h, saved)
}

// NewWindowAdapter wrap *RuntimeContext into node.WindowService.
func NewWindowAdapter(rt *RuntimeContext) node.WindowService { return &windowAdapter{rt: rt} }

// ============================================================================
// CaptureAdapter — pkgcapture.IBackend → node.CaptureService
// 抓帧 + png.Encode 返字节流 (跟 screenshot.go 一致).
// ============================================================================

type captureAdapter struct {
	rt          *RuntimeContext
	traceSource automationtrace.ActionSource
}

func (a *captureAdapter) controller() (controller.Screenshotter, error) {
	ctrl, err := a.rt.controllerForActiveTarget(a.traceSource, controllerNeed{Capture: true})
	if err != nil {
		return nil, err
	}
	screenshotter, ok := ctrl.(controller.Screenshotter)
	if !ok {
		return nil, fmt.Errorf("active controller %T does not support screenshots", ctrl)
	}
	return screenshotter, nil
}

func (a *captureAdapter) Capture() ([]byte, error) {
	frame, err := a.captureFrame()
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
	frame, err := a.captureFrame()
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

func (a *captureAdapter) captureFrame() (*image.RGBA, error) {
	ctrl, err := a.controller()
	if err != nil {
		return nil, err
	}
	frame, err := ctrl.Screenshot(context.Background(), controller.ScreenshotRequest{
		Space: target.SpaceWindowClient,
	})
	if err != nil {
		return nil, err
	}
	return frame.Image, nil
}

// NewCaptureAdapter wrap *RuntimeContext into node.CaptureService.
func NewCaptureAdapter(rt *RuntimeContext) node.CaptureService { return &captureAdapter{rt: rt} }

func newCaptureAdapterWithSource(rt *RuntimeContext, source automationtrace.ActionSource) node.CaptureService {
	return &captureAdapter{rt: rt, traceSource: source}
}

// ============================================================================
// VisionAdapter — rt.Matcher + rt.Capture → node.VisionService
// Match/WaitMatch 走 Matcher; DetectColor 走 rt.Capture + CaptureFrameCached (100ms 缓存);
// DetectColorHSV / ROIColorScan / DualBarTrack 自抓帧 + 复用包内 helper (countHSVInROI / scanClusters / vision.AnalyzeDualColorBar).
// ============================================================================

// visionWaitPollMs WaitMatch 默认轮询间隔 (ms).
const visionWaitPollMs = 100

type visionAdapter struct{ rt *RuntimeContext }

// scaleTolerance 读容器级模板缩放容差, 透传给 matcher.Detect (matcher 已不再 per-container).
func (a *visionAdapter) scaleTolerance() float64 {
	if a.rt.Container == nil {
		return container.DefaultScaleTolerance
	}
	return container.ReadWindowTargetScaleTolerance(a.rt.Container)
}

func (a *visionAdapter) Match(ctx context.Context, keys []string, threshold float64, roi node.Geometry) (node.MatchHit, error) {
	if a.rt.Matcher == nil || len(keys) == 0 {
		return node.MatchHit{}, nil
	}
	return a.matchOnce(ctx, keys, threshold, roi)
}

func (a *visionAdapter) WaitMatch(ctx context.Context, keys []string, threshold float64, roi node.Geometry, timeout time.Duration) (node.MatchHit, error) {
	if a.rt.Matcher == nil || len(keys) == 0 {
		return node.MatchHit{}, nil
	}
	if timeout <= 0 {
		return a.matchOnce(ctx, keys, threshold, roi)
	}
	deadline := time.Now().Add(timeout)
	bestConf := 0.0
	for {
		if err := ctx.Err(); err != nil {
			return node.MatchHit{}, err
		}
		hit, err := a.matchOnce(ctx, keys, threshold, roi)
		if err != nil {
			return node.MatchHit{}, err
		}
		if hit.Conf > bestConf {
			bestConf = hit.Conf
		}
		if hit.Found {
			return hit, nil
		}
		if time.Now().After(deadline) {
			return node.MatchHit{Conf: bestConf}, nil
		}
		select {
		case <-ctx.Done():
			return node.MatchHit{}, ctx.Err()
		case <-time.After(visionWaitPollMs * time.Millisecond):
		}
	}
}

// matchOnce 单帧多模板 OR (按 keys 序取首个命中)。roi 零值 → nil region (variant.BBox 快速定位);
// 非零 → 解析成比例搜索区下发。命中带 bbox。
func (a *visionAdapter) matchOnce(ctx context.Context, keys []string, threshold float64, roi node.Geometry) (node.MatchHit, error) {
	var frame *image.RGBA
	if a.rt.Capture != nil {
		h, err := a.rt.ActiveHWND()
		if err != nil {
			return node.MatchHit{}, err
		}
		f, err := a.rt.CaptureFrameCached(h)
		if err != nil {
			return node.MatchHit{}, err
		}
		frame = f
	}
	var region []float64
	if frame != nil && (roi.Pct.W > 0 && roi.Pct.H > 0 || len(roi.Overrides) > 0) {
		fw, fh := frame.Bounds().Dx(), frame.Bounds().Dy()
		rx, ry, rw, rh, _ := ResolveGeometry(roi, fw, fh)
		region = []float64{float64(rx) / float64(fw), float64(ry) / float64(fh), float64(rw) / float64(fw), float64(rh) / float64(fh)}
	}
	tol := a.scaleTolerance()
	bestConf := 0.0
	for _, guid := range keys {
		found, pt, bbox, conf, err := a.rt.Matcher.Detect(ctx, frame, guid, threshold, region, tol)
		if err != nil {
			return node.MatchHit{}, err
		}
		if conf > bestConf {
			bestConf = conf
		}
		if found {
			return node.MatchHit{Found: true, Point: node.Point{X: pt.X, Y: pt.Y}, BBox: bbox, Conf: conf}, nil
		}
	}
	return node.MatchHit{Conf: bestConf}, nil
}

func (a *visionAdapter) DualBarTrack(roi node.Geometry, inner, outer node.HSVRange, opts node.DualBarOptions) (node.DualColorBarResult, error) {
	if a.rt.Capture == nil {
		return node.DualColorBarResult{}, fmt.Errorf("capture backend not initialised")
	}
	h, err := a.rt.ActiveHWND()
	if err != nil {
		return node.DualColorBarResult{Found: false}, nil
	}
	frame, err := a.rt.Capture.Frame(win.HWND(h))
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
	return out, nil
}

func (a *visionAdapter) DetectColor(roi node.Geometry, mode string, rng [6]int) (int, float64, float64, error) {
	if a.rt.Capture == nil {
		return 0, 0, 0, nil
	}
	hwnd, err := a.rt.ActiveHWND()
	if err != nil {
		return 0, 0, 0, err
	}
	frame, err := a.rt.CaptureFrameCached(hwnd)
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
	return count, cx, cy, nil
}

func (a *visionAdapter) DetectColorHSV(roi node.Geometry, hsv node.HSVRange) (int, float64, error) {
	if a.rt.Capture == nil {
		return 0, 0, fmt.Errorf("capture backend not initialised")
	}
	h, err := a.rt.ActiveHWND()
	if err != nil {
		return 0, 0, err
	}
	frame, err := a.rt.Capture.Frame(win.HWND(h))
	if err != nil {
		return 0, 0, err
	}
	if frame == nil {
		return 0, 0, fmt.Errorf("capture: nil frame")
	}
	sub := cropFrameByGeometry(frame, roi)
	count, ratio := countHSVInROI(sub, hsvRangeFromNode(hsv))
	return count, ratio, nil
}

func (a *visionAdapter) DetectColorBlobs(roi node.Geometry, mode string, rng [6]int, minArea int) ([]node.BlobEntry, error) {
	if a.rt.Capture == nil {
		return nil, fmt.Errorf("capture backend not initialised")
	}
	hwnd, err := a.rt.ActiveHWND()
	if err != nil {
		return nil, err
	}
	frame, err := a.rt.Capture.Frame(win.HWND(hwnd)) // 直接抓新帧，绕开 100ms 缓存（轮询需要）
	if err != nil {
		return nil, err
	}
	if frame == nil {
		return nil, fmt.Errorf("capture: nil frame")
	}
	frameW, frameH := frame.Bounds().Dx(), frame.Bounds().Dy()
	x0, y0, w, h, _ := ResolveGeometry(roi, frameW, frameH)
	if w <= 0 || h <= 0 {
		return []node.BlobEntry{}, nil
	}
	pxBlobs := findColorBlobs(frame, x0, y0, x0+w, y0+h, mode, rng, minArea)
	out := make([]node.BlobEntry, len(pxBlobs))
	fw, fh := float64(frameW), float64(frameH)
	for i, b := range pxBlobs {
		out[i] = node.BlobEntry{
			CenterX: float64(b.CenterX) / fw,
			CenterY: float64(b.CenterY) / fh,
			X:       float64(b.X) / fw,
			Y:       float64(b.Y) / fh,
			W:       float64(b.W) / fw,
			H:       float64(b.H) / fh,
			Area:    b.Area,
		}
	}
	return out, nil
}

func (a *visionAdapter) ROIColorScan(roi node.Geometry, hsv node.HSVRange, axis string, minPx, maxPx int) ([]node.ClusterEntry, error) {
	if a.rt.Capture == nil {
		return nil, fmt.Errorf("capture backend not initialised")
	}
	h, err := a.rt.ActiveHWND()
	if err != nil {
		return nil, err
	}
	frame, err := a.rt.Capture.Frame(win.HWND(h))
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
	h, err := a.rt.ActiveHWND()
	if err != nil {
		return nil, err
	}
	frame, err := a.rt.Capture.Frame(win.HWND(h))
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

func (a *visionAdapter) FindColorSignature(roi node.Geometry, sig node.ColorSignature, defaultTol int) (bool, node.Point, error) {
	if a.rt.Capture == nil {
		return false, node.Point{}, fmt.Errorf("capture backend not initialised")
	}
	h, err := a.rt.ActiveHWND()
	if err != nil {
		return false, node.Point{}, err
	}
	frame, err := a.rt.CaptureFrameCached(h)
	if err != nil {
		return false, node.Point{}, err
	}
	if frame == nil {
		return false, node.Point{}, fmt.Errorf("capture: nil frame")
	}
	frameW, frameH := frame.Bounds().Dx(), frame.Bounds().Dy()
	// 复用 DetectColorBlobs 相同的 ResolveGeometry → 像素矩形解析法。
	rx, ry, rw, rh, _ := ResolveGeometry(roi, frameW, frameH)
	// ColorPoint.Tol *int → 具体 tol (nil 用 defaultTol)。
	pts := make([]vision.ColorSigPoint, len(sig.Points))
	for i, p := range sig.Points {
		tol := defaultTol
		if p.Tol != nil {
			tol = *p.Tol
		}
		pts[i] = vision.ColorSigPoint{DX: p.DX, DY: p.DY, R: p.R, G: p.G, B: p.B, Tol: tol}
	}
	found, ax, ay := vision.FindColorSignature(frame, rx, ry, rw, rh, pts)
	if !found {
		return false, node.Point{}, nil
	}
	return true, node.Point{X: float64(ax) / float64(frameW), Y: float64(ay) / float64(frameH)}, nil
}

// MatchAll 抓一次帧, 在 roi 内各 guid DetectAll (同帧) → 合并 → 统一跨模板 NMS。spec §节点2。
func (a *visionAdapter) MatchAll(ctx context.Context, keys []string, threshold float64, minDistance int, roi node.Geometry) ([]node.TemplateMatch, error) {
	if a.rt.Matcher == nil || len(keys) == 0 {
		return nil, nil
	}
	var frame *image.RGBA
	frameW := 0
	if a.rt.Capture != nil {
		h, err := a.rt.ActiveHWND()
		if err != nil {
			return nil, err
		}
		f, err := a.rt.CaptureFrameCached(h)
		if err != nil {
			return nil, err
		}
		frame = f
		if frame != nil {
			frameW = frame.Bounds().Dx()
		}
	}
	// roi 总作为显式比例搜索区下发: 绕开 variant.BBox 单点定位 (找全部要搜整片)。
	// 零值 Geometry → ResolveGeometry 返全帧 → region [0,0,1,1]。
	var region []float64
	if frame != nil {
		fw, fh := frame.Bounds().Dx(), frame.Bounds().Dy()
		rx, ry, rw, rh, _ := ResolveGeometry(roi, fw, fh)
		region = []float64{float64(rx) / float64(fw), float64(ry) / float64(fh), float64(rw) / float64(fw), float64(rh) / float64(fh)}
	}
	tol := a.scaleTolerance()
	var all []node.TemplateMatch
	for _, guid := range keys {
		hits, err := a.rt.Matcher.DetectAll(ctx, frame, guid, threshold, region, tol)
		if err != nil {
			return nil, err
		}
		all = append(all, hits...)
	}
	return nmsMatches(all, minDistance, frameW), nil
}

// nmsMatches 跨模板统一 NMS (归一化空间): conf 降序 (并列 y,x) 保留, 中心距 < radius 抑制低分。
// minDistance>0 → radius = minDistance/frameW (归一化); <=0 → 各命中 bbox 短边/2。
func nmsMatches(matches []node.TemplateMatch, minDistance, frameW int) []node.TemplateMatch {
	if len(matches) <= 1 {
		return matches
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Conf != matches[j].Conf {
			return matches[i].Conf > matches[j].Conf
		}
		if matches[i].Point.Y != matches[j].Point.Y {
			return matches[i].Point.Y < matches[j].Point.Y
		}
		return matches[i].Point.X < matches[j].Point.X
	})
	radius := func(m node.TemplateMatch) float64 {
		if minDistance > 0 && frameW > 0 {
			return float64(minDistance) / float64(frameW)
		}
		if m.BBox[2] < m.BBox[3] {
			return m.BBox[2] / 2
		}
		return m.BBox[3] / 2
	}
	kept := make([]node.TemplateMatch, 0, len(matches))
	for _, c := range matches {
		rc := radius(c)
		ok := true
		for _, k := range kept {
			lim := rc
			if rk := radius(k); rk < lim {
				lim = rk
			}
			if math.Hypot(c.Point.X-k.Point.X, c.Point.Y-k.Point.Y) < lim {
				ok = false
				break
			}
		}
		if ok {
			kept = append(kept, c)
		}
	}
	return kept
}

// DecodeQR 抓全帧 → 按 roi 裁子图 → 解码所有 QR → 定位点加 ROI 偏移后归一化到全帧。
// 返回按 bbox min-y 再 min-x 升序排列的 QRResult slice。解码失败 → 空 slice + nil error。
func (a *visionAdapter) DecodeQR(roi node.Geometry) ([]node.QRResult, error) {
	if a.rt.Capture == nil {
		return nil, fmt.Errorf("capture backend not initialised")
	}
	h, err := a.rt.ActiveHWND()
	if err != nil {
		return nil, err
	}
	frame, err := a.rt.CaptureFrameCached(h)
	if err != nil {
		return nil, err
	}
	if frame == nil {
		return nil, fmt.Errorf("capture: nil frame")
	}
	frameW, frameH := frame.Bounds().Dx(), frame.Bounds().Dy()
	rx, ry, rw, rh, _ := ResolveGeometry(roi, frameW, frameH)

	// 裁子图: 用已有像素版裁图 helper (cropFrameByGeometry 已裁好 image.RGBA 子图).
	sub := cropFrameByGeometry(frame, roi)

	hits, err := vision.DecodeQRFromImage(sub)
	if err != nil {
		return nil, err
	}
	if len(hits) == 0 {
		return nil, nil
	}

	results := make([]node.QRResult, 0, len(hits))
	for _, hit := range hits {
		pts := make([]node.Point, len(hit.Points))
		for i, p := range hit.Points {
			// 定位点像素坐标相对子图; 加 ROI 偏移后归一化到全帧.
			absX := rx + p[0]
			absY := ry + p[1]
			pts[i] = node.Point{
				X: float64(absX) / float64(frameW),
				Y: float64(absY) / float64(frameH),
			}
		}
		results = append(results, node.QRResult{Text: hit.Text, Points: pts})
	}

	// 按定位点外接 bbox 左上角 (min-x, min-y) 排序: 先 min-y 升序, 再 min-x 升序.
	sortQRResults(results)
	_ = rw
	_ = rh
	return results, nil
}

// sortQRResults 按每个 QRResult 的定位点外接 bbox 左上角 min-y 升序, 相同 min-y 再按 min-x 升序.
func sortQRResults(results []node.QRResult) {
	bboxMinXY := func(r node.QRResult) (float64, float64) {
		minX, minY := 1.0, 1.0
		for _, p := range r.Points {
			if p.X < minX {
				minX = p.X
			}
			if p.Y < minY {
				minY = p.Y
			}
		}
		return minX, minY
	}
	for i := 1; i < len(results); i++ {
		for j := i; j > 0; j-- {
			xi, yi := bboxMinXY(results[j])
			xj, yj := bboxMinXY(results[j-1])
			if yi < yj || (yi == yj && xi < xj) {
				results[j], results[j-1] = results[j-1], results[j]
			} else {
				break
			}
		}
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
// withTickSnapshot 写入, EvaluatePureData wrap 时调 Snapshot(ctx) 拿 frozen Vars view.
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
			}
		},
	}
}
