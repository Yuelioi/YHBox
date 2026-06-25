// Package runtime 把 Container 蓝图跑起来（token-based dispatch）。
//
// 节点 config 是 string 表达式 → expr.Parse → AST 缓存到 RuntimeContext。
// 节点之间通过 token 流转（不是 graph traversal，方便 Loop/Break/Continue 直接
// 跳转到目标 token）。
package runtime

import (
	"errors"
	"fmt"
	"image"
	"maps"
	"sync"
	"time"

	"github.com/lxn/win"
	"yotta/internal/services/container"
	"yotta/internal/services/execution"
	"yotta/internal/services/expr"
	"yotta/internal/services/inputclip/backends"
	clipruntime "yotta/internal/services/inputclip/runtime"
	pkgcapture "yotta/pkg/capture"
	pkginput "yotta/pkg/input"
	"yotta/pkg/winutil"
)

// frameCacheTTL: 同 hwnd 100ms 内复用一帧. 100ms 是 fishing v2 主循环 Sleep 下限,
// 单 iter 内多次 Detect 命中缓存; 跨 iter (Sleep>=100ms) 自然 miss → 重新抓帧.
const frameCacheTTL = 100 * time.Millisecond

// RuntimeContext 单 Container run 的状态。
//   - vars 通过 mutex 保护（Parallel/Race 子分支并发读写）
//   - inputBus 注入：每个 input 节点 Lock/Unlock 保证 OS input 串行
//   - templateMatcher 注入：Wait/Check/ClickTemplate 用
//   - emit 注入：Log/Toast 节点把消息推到前端
//
// Input / Window / Capture 由 ContainerRunner.setupRuntime 在 Run 启动期间解析
// WindowTarget 节点后 populate. Game 字段供 BringWindowForeground 节点用.
type RuntimeContext struct {
	Container *container.Container
	// Subgraphs 起跑时从全局池解析出的引用闭包快照 (2026-06-12 全局化 — 容器不再拥有子图).
	// 构造后由 caller 填 (main.go runFunc 解析闭包; 测试 fixture 直接赋); 跑中不回查 store,
	// 编辑不影响在跑实例 (快照语义同旧 Container.Subgraphs 内存切片).
	Subgraphs []container.Subgraph
	InputBus  *execution.InputBus
	Matcher   TemplateMatcher
	Input     pkginput.Backend // per-container 实例, setupRuntime 注入
	Game      GameProvider
	Emit      func(name string, data any)

	Capture pkgcapture.IBackend // per-container 实例, setupRuntime 注入

	// 粘性活动窗口寄存器. 运行时由 WindowTarget.Run 经 SetActiveWindow 改写;
	// 输入/检测节点 + EventTick listener goroutine 经 WindowHandle()/ActiveHWND() 并发读.
	windowMu sync.RWMutex
	window   winutil.WindowHandle // HWND==0 = 未解析
	// windowOverride 派发期 per-node 覆盖栈; 栈顶为当前有效窗口, 空则用粘性 window。windowMu 守护。
	windowOverride []winutil.WindowHandle

	// 帧缓存: 同一 iter 内模板/DetectColor 多次抓帧复用 (100ms TTL, 按 hwnd 分条).
	// DualBar/HSV/ROIScan/Grid 不走此缓存 (QTE 高频轮询要新帧).
	frameMu    sync.Mutex
	frameCache map[uintptr]frameCacheEntry

	// borderless 进入前布局快照, RestoreBorders 取出还原; 按 hwnd 分条。
	borderlessMu    sync.Mutex
	borderlessSaved map[uintptr]winutil.SavedWindow

	// PlayClip 节点用: InputClip 解析 + 注入后端 + 当前机器 mouse 360° counts.
	ClipResolver   ClipResolver
	InputBackend   backends.IInputBackend
	MouseCounts360 int
	ClipPolicy     clipruntime.PlaybackPolicy

	mu     sync.Mutex
	vars   map[string]expr.Value
	params map[string]expr.Value

	// varTimestamps: name → unix ms 上次 SetVar/IncVar 改写时间. Fishing v2 watchdog
	// 通过 VarLastChange 节点查 state 多久没变. live 读 (不进 snapshot).
	varTimestamps map[string]int64
}

func NewRuntimeContext(
	c *container.Container,
	bus *execution.InputBus,
	matcher TemplateMatcher,
	game GameProvider,
	emit func(name string, data any),
	clipResolver ClipResolver,
	mouseCounts360 int,
) *RuntimeContext {
	rt := &RuntimeContext{
		Container:      c,
		InputBus:       bus,
		Matcher:        matcher,
		Game:           game,
		Emit:           emit,
		ClipResolver:   clipResolver,
		MouseCounts360: mouseCounts360,
		ClipPolicy:     clipruntime.DefaultPlaybackPolicy(),
		vars:           make(map[string]expr.Value),
		params:         make(map[string]expr.Value),
		varTimestamps:  make(map[string]int64),
	}
	rt.initVars()
	rt.initFrameCache()
	return rt
}

// GameProvider 返回注入的 GameProvider（可能为 nil，1.22 前占位）。
func (rt *RuntimeContext) GameProvider() GameProvider { return rt.Game }

func (rt *RuntimeContext) initVars() {
	for _, v := range rt.Container.Vars {
		rt.vars[v.Name] = v.Default
	}
}

// Vars 拿当前快照（拷贝，调用方修改不影响 runtime）。
func (rt *RuntimeContext) Vars() map[string]expr.Value {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	cp := make(map[string]expr.Value, len(rt.vars))
	maps.Copy(cp, rt.vars)
	return cp
}

// SetVar 单字段写。线程安全。Fishing v2 watchdog 通过 varTimestamps 查上次改写时间.
func (rt *RuntimeContext) SetVar(name string, val expr.Value) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.vars[name] = val
	rt.varTimestamps[name] = nowMillis()
}

// IncVar 单字段增量。如果当前不是 number → 报错。
func (rt *RuntimeContext) IncVar(name string, delta float64) error {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	cur, _ := expr.AsNumber(rt.vars[name])
	rt.vars[name] = cur + delta
	rt.varTimestamps[name] = nowMillis()
	return nil
}

// VarLastChange 返 var 上次 SetVar/IncVar 时间 (unix ms). 未设过返 0.
// Fishing v2 watchdog 用 — 通过 VarLastChange 节点读.
func (rt *RuntimeContext) VarLastChange(name string) int64 {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.varTimestamps[name]
}

// nowMillis 当前 unix 毫秒. 包内 helper, 跟 Now 节点同源.
func nowMillis() int64 { return time.Now().UnixMilli() }

// SetParam 仅 ContainerRunner 启动时设（Container 目前无 params）。
func (rt *RuntimeContext) SetParam(name string, val expr.Value) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.params[name] = val
}

// Variables / params are accessed via dedicated pure-data nodes (GetVar / GetParam)
// that read from per-exec-tick snapshots (see snapshot.go + getvar.go).

// ErrNoActiveWindow 窗口类节点在任何 WindowTarget 点之前执行时返回. 走标准节点错误路.
var ErrNoActiveWindow = errors.New("NO_ACTIVE_WINDOW — 当前无活动窗口, 先放并执行一个 WindowTarget 节点")

// WindowHandle 并发安全读取当前有效窗口(覆盖栈非空返栈顶, 否则返粘性 window).
func (rt *RuntimeContext) WindowHandle() winutil.WindowHandle {
	rt.windowMu.RLock()
	defer rt.windowMu.RUnlock()
	if n := len(rt.windowOverride); n > 0 {
		return rt.windowOverride[n-1]
	}
	return rt.window
}

// ActiveHWND 并发安全读取 HWND. HWND==0 时返回 ErrNoActiveWindow.
func (rt *RuntimeContext) ActiveHWND() (uintptr, error) {
	rt.windowMu.RLock()
	defer rt.windowMu.RUnlock()
	wh := rt.window
	if n := len(rt.windowOverride); n > 0 {
		wh = rt.windowOverride[n-1]
	}
	if wh.HWND == 0 {
		return 0, ErrNoActiveWindow
	}
	return wh.HWND, nil
}

func (rt *RuntimeContext) PushWindowOverride(wh winutil.WindowHandle) {
	rt.windowMu.Lock()
	defer rt.windowMu.Unlock()
	rt.windowOverride = append(rt.windowOverride, wh)
}

func (rt *RuntimeContext) PopWindowOverride() {
	rt.windowMu.Lock()
	defer rt.windowMu.Unlock()
	if n := len(rt.windowOverride); n > 0 {
		rt.windowOverride = rt.windowOverride[:n-1]
	}
}

// SetActiveWindow 整体替换活动窗口 (粘性) + 清该 hwnd 帧缓存 + prune 过期.
func (rt *RuntimeContext) SetActiveWindow(wh winutil.WindowHandle) {
	rt.windowMu.Lock()
	rt.window = wh
	rt.windowMu.Unlock()
	rt.invalidateFrameCacheFor(wh.HWND)
}

func (rt *RuntimeContext) saveBorderless(hwnd uintptr, s winutil.SavedWindow) {
	rt.borderlessMu.Lock()
	defer rt.borderlessMu.Unlock()
	if rt.borderlessSaved == nil {
		rt.borderlessSaved = map[uintptr]winutil.SavedWindow{}
	}
	rt.borderlessSaved[hwnd] = s
}

func (rt *RuntimeContext) takeBorderless(hwnd uintptr) (winutil.SavedWindow, bool) {
	rt.borderlessMu.Lock()
	defer rt.borderlessMu.Unlock()
	s, ok := rt.borderlessSaved[hwnd]
	if ok {
		delete(rt.borderlessSaved, hwnd)
	}
	return s, ok
}

// frameCacheEntry 单条帧缓存项.
type frameCacheEntry struct {
	frame *image.RGBA
	at    time.Time
}

func (rt *RuntimeContext) initFrameCache() {
	rt.frameMu.Lock()
	defer rt.frameMu.Unlock()
	if rt.frameCache == nil {
		rt.frameCache = map[uintptr]frameCacheEntry{}
	}
}

func (rt *RuntimeContext) peekFrameCache(hwnd uintptr) *image.RGBA {
	rt.frameMu.Lock()
	defer rt.frameMu.Unlock()
	e, ok := rt.frameCache[hwnd]
	if !ok || time.Since(e.at) >= frameCacheTTL {
		return nil
	}
	return e.frame
}

func (rt *RuntimeContext) putFrameCache(hwnd uintptr, f *image.RGBA) {
	rt.frameMu.Lock()
	defer rt.frameMu.Unlock()
	if rt.frameCache == nil {
		rt.frameCache = map[uintptr]frameCacheEntry{}
	}
	rt.frameCache[hwnd] = frameCacheEntry{frame: f, at: time.Now()}
}

// invalidateFrameCacheFor 清指定 hwnd 条目 + prune 全部过期.
func (rt *RuntimeContext) invalidateFrameCacheFor(hwnd uintptr) {
	rt.frameMu.Lock()
	defer rt.frameMu.Unlock()
	if rt.frameCache == nil {
		return
	}
	delete(rt.frameCache, hwnd)
	for h, e := range rt.frameCache {
		if time.Since(e.at) >= frameCacheTTL {
			delete(rt.frameCache, h)
		}
	}
}

// CaptureFrameCached 走 rt.Capture 抓帧, 100ms TTL 缓存 (按 hwnd 分条). 仅模板匹配 + DetectColor 用.
func (rt *RuntimeContext) CaptureFrameCached(hwnd uintptr) (*image.RGBA, error) {
	if f := rt.peekFrameCache(hwnd); f != nil {
		return f, nil
	}
	f, err := rt.Capture.Frame(win.HWND(hwnd))
	if err != nil {
		return nil, err
	}
	rt.putFrameCache(hwnd, f)
	return f, nil
}

// emitROIResolutionWarnings 遍历主图+子图, 节点 _capturedAtResolution 与当前窗口 clientW/H 不符则发 warning.
// rt.Emit 为 nil 时直接返回. (原在 setupRuntime, 多窗口后挪到每次切窗调用.)
func (rt *RuntimeContext) emitROIResolutionWarnings(clientW, clientH int) {
	if rt.Emit == nil {
		return
	}
	checkROINode := func(n *container.GraphNode) {
		rawCap, ok := n.Config["_capturedAtResolution"].([]any)
		if !ok || len(rawCap) != 2 {
			return
		}
		cw, _ := rawCap[0].(float64)
		ch, _ := rawCap[1].(float64)
		if int(cw) != clientW || int(ch) != clientH {
			rt.Emit("container:warning", map[string]any{
				"nodeId":  n.ID,
				"code":    "ROI_RESOLUTION_MISMATCH",
				"message": fmt.Sprintf("node %q ROI captured at %vx%v but window is %dx%d — accuracy may degrade", n.ID, cw, ch, clientW, clientH),
			})
		}
	}
	for i := range rt.Container.Graph.Nodes {
		checkROINode(&rt.Container.Graph.Nodes[i])
	}
	for i := range rt.Subgraphs {
		for j := range rt.Subgraphs[i].Graph.Nodes {
			checkROINode(&rt.Subgraphs[i].Graph.Nodes[j])
		}
	}
}
