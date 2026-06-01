// internal/node/interfaces.go
// Package node interfaces — Node + extension interfaces + Inputs/Outputs/Ctx contracts.
package node

import (
	"context"
	"time"
)

// Node — minimal contract. 只描述 "我是什么", 不描述 "我怎么跑".
// 三 capability interfaces (Runnable / RegionRunner / Evaluator) 表执行语义.
// Register 验证 exactly-one capability (IsVisualOnly / IsGraphMarker 例外).
type Node interface {
	Spec() Spec
}

// Runnable — exec 节点, 框架同步调一次.
type Runnable interface {
	Run(ctx Ctx, in Inputs) (Outputs, error)
}

// 以下是扩展接口, 节点选择性实现. Framework type assertion 探测.

// Displayer — 节点想自定义日志面板显示就实现. 不实现 → 节点不出现在 node-log.
type Displayer interface {
	Display(in Inputs, exitName string, out OutputData) string
}

// Validator — 节点自身静态校验.
type Validator interface {
	Validate(in Inputs) []ValidationError
}

// Dependencer — 子图分享 / library import 时 BFS 抽外部资产引用.
type Dependencer interface {
	Dependencies(in Inputs) []Dependency
}

// RegionRunner — 控制流节点 (Loop / Try / Subgraph / CollapsedNode) 实现. body 回调
// 由 dispatch 构造, region 节点决定调用次数 (Loop 多次, Try 单次截获 error).
type RegionRunner interface {
	Node
	RunRegion(ctx Ctx, in Inputs, body func(Ctx) error) (Outputs, error)
}

// Evaluator — pure-data 节点 (IsPureData=true) 实现, 走 EvaluatePureData 入口求 single value.
// 不返 Outputs (没 exit 出口), 直接返算出来的标量 — 给 data-edge 下游消费.
//
// 节点不实现 Evaluator → EvaluatePureData 返 error; dispatch 可探测后 fallback 老路径.
// 设计目的: GetVar/GetSys/GetParam 依赖 runtime state (frame/snapshot), 这几个不实现 Evaluator,
// dispatch 走 fallback. Add/Sub/.../Select 等 22 purefunc 自包含, 实现即可.
type Evaluator interface {
	Evaluate(ctx Ctx, in Inputs) (any, error)
}

// Inputs — 节点 Run 收的输入. 取值优先级:
//   1. data-wire (纯数据 pin 上游) [最高]
//   2. Inspector config
//   3. exec-data wire (上游 exec 出口 Data 字段同名注入)
//   4. InputSpec.Default
//   5. Required + 缺 → framework 返 ValidationError (NOT panic)
//   6. Optional + 缺 → 零值, Has() = false
type Inputs interface {
	String(name string) string
	Float64(name string) float64
	Int(name string) int
	Bool(name string) bool
	StringList(name string) []string
	Point(name string) Point
	Rect(name string) Rect
	Geometry(name string) Geometry
	Color(name string) Color
	Duration(name string) time.Duration
	JSON(name string) map[string]any
	Raw(name string) any
	Has(name string) bool
	// Keys 返 merged map 的所有 key. 给 dynamic-input 节点 (Expr) 遍历用.
	Keys() []string
}

// Outputs — Run 返回值. 节点不该直接构造, 通过 ctx.Out(...).Fire() 拿.
type Outputs interface {
	exit() (name string, data map[string]any)
}

// OutputData — Display 接收的出口数据 (immutable snapshot).
type OutputData interface {
	String(field string) string
	Float64(field string) float64
	Int(field string) int
	Bool(field string) bool
	Point(field string) Point
	Raw(field string) any
	Has(field string) bool
}

// Ctx — Run 期间框架注入. 服务接口都返 nil-safe; 节点拿到 nil 调方法会 panic (清晰报错).
type Ctx interface {
	Context() context.Context
	Now() time.Time
	Out(exitName string) OutBuilder

	// Services. 全部 nullable — 节点用之前应当假设非 nil (有 service 的节点都
	// 在 spec / build 阶段已经 wire 真 backend); 测试时 stub.
	Vision() VisionService
	Log() LogService
	Input() InputService
	Vars() VarStore
	Sys() SysStore
	Params() ParamStore
	Window() WindowService
	Capture() CaptureService
	Stopwatches() StopwatchStore
	Clip() ClipPlayer
}

// OutBuilder — fluent builder.
//   - exitName 不在 Spec.Outputs → Fire panic
//   - Run 内调多次 Fire → 第 2 次 panic
type OutBuilder interface {
	Set(field string, value any) OutBuilder
	Fire() Outputs
}

// VisionService — template + 颜色 + bar-track 检测. wire_container.go::templateMatcherAdapter
// + pkg/vision.* 实现.
//
// 单帧 Match (CheckTemplate) / WaitMatch (WaitTemplate) / BarTrack (ColorBarTrack) +
// 颜色: DetectColor (RGB/HSV 全 ROI 像素计数) / DetectColorHSV (HSV 比例阈值) /
// ROIColorScan (沿 axis 找连续 cluster).
type VisionService interface {
	// Match 单帧多模板匹配. keys 是模板 key 列表, mode = "any" | "all":
	//   - any: 任一 key 命中即返该 key 的命中点 (按列表序取首个命中). nil pt = 全 miss.
	//   - all: 全部 key 同帧命中才返 (点取列表首个 key 的命中点); 任一 miss → nil pt.
	// 空 keys → nil pt. ctx 透传容器 cancel — 长 Match (磁盘读模板 + CPU 比对) 配合 Stop 瞬停.
	Match(ctx context.Context, keys []string, threshold float64, mode string) (pt *Point, conf float64, err error)

	// WaitMatch 阻塞轮询直到 (按 mode) 命中或 timeout. timeout<=0 视为 0 (单帧).
	// mode 语义同 Match. 命中: pt!=nil. timeout: pt=nil.
	WaitMatch(ctx context.Context, keys []string, threshold float64, mode string, timeout time.Duration) (pt *Point, conf float64, err error)

	// DualBarTrack 抓全帧后按 roi Geometry 裁子区 + 双色 HSV cluster → 算 inner 在 outer 区域里的位置.
	// 通用双色条算法 (适用: 血条/进度条/QTE 双色条/钓鱼 cursor-target).
	// roi 经 ResolveGeometry 解析像素区 (override-by-resolution / pct / 全帧). Found=false 即 missing.
	// opts 零值字段走 vision 包默认 (fishing 实测出来的; 通用 case 可能要调).
	DualBarTrack(roi Geometry, inner, outer HSVRange, opts DualBarOptions) (result DualColorBarResult, err error)

	// DetectColor 按 roi (ratio Geometry, 享 per-resolution override) 裁区后统计落在 rng 内的像素数.
	// Geometry 零值 = 全帧. mode = "hsv" | "rgb". rng = 6 元 [aMin,aMax,bMin,bMax,cMin,cMax].
	// 返 count + 命中像素中心客户区比例坐标 (cx,cy). 无命中 cx/cy = 0.
	DetectColor(roi Geometry, mode string, rng [6]int) (count int, cx, cy float64, err error)

	// DetectColorHSV 按 roi (ratio Geometry) 裁子帧后统计落在 hsv 区间的像素数 + 比例.
	// Geometry 零值 = 全帧.
	DetectColorHSV(roi Geometry, hsv HSVRange) (count int, ratio float64, err error)

	// ROIColorScan 按 roi (ratio Geometry) 裁子帧后沿 axis ("x"|"y") 扫描 HSV 命中像素,
	// 合并为连续 cluster; 只返长度 ∈ [minPx, maxPx] 的段.
	// maxPx<=0 → adapter 默认 = 子帧宽/3 (axis x) 或 高/3 (axis y).
	ROIColorScan(roi Geometry, hsv HSVRange, axis string, minPx, maxPx int) (clusters []ClusterEntry, err error)

	// GridSignature 抓全帧后按 roi Geometry 裁子区 → box-average 降采样成
	// gridSize×gridSize RGB 签名 (flat, 行主序, len = gridSize²×3). 每调一次抓
	// 一张新帧 (无缓存). Geometry 零值 = 全帧; adapter 内经 ResolveGeometry 解析像素区.
	// 给帧差节点 (WaitStable/WaitChange) 跨 poll 比对用.
	GridSignature(roi Geometry, gridSize int) (sig []uint8, err error)
}

// HSVRange HSV 阈值区间. H ∈ [0,360], S/V ∈ [0,255]. 给 DetectColorHSV / ROIColorScan 用.
type HSVRange struct {
	HMin int `json:"hMin"`
	HMax int `json:"hMax"`
	SMin int `json:"sMin"`
	SMax int `json:"sMax"`
	VMin int `json:"vMin"`
	VMax int `json:"vMax"`
}

// ClusterEntry ROIColorScan 沿 axis 找出的一个连续命中段, 像素坐标 (相对 roi 起点).
type ClusterEntry struct {
	StartPx  int `json:"startPx"`
	EndPx    int `json:"endPx"`
	CenterPx int `json:"centerPx"`
	PxCount  int `json:"pxCount"`
}

// DualColorBarResult DualBarTrack 算法单次运行结果. 跟 pkg/vision.DualColorBarResult
// 形态一致 (重复定义避免 internal/node 反向 import pkg/vision).
//
// Found = true 表 inner + outer 都找到; false 表至少有一个 miss (Inner/OuterX = -1).
// Inner/OuterPx 即使 miss 也填 HSV 命中像素总数 (调试用).
type DualColorBarResult struct {
	Found      bool    `json:"found"`
	InnerX     int     `json:"innerX"`
	OuterX     int     `json:"outerX"`
	OuterWidth int     `json:"outerWidth"`
	Confidence float64 `json:"confidence"`
	InnerPx    int     `json:"innerPx"`
	OuterPx    int     `json:"outerPx"`
}

// DualBarOptions DualBarTrack 算法可调参数. 0 值字段走 vision 默认 (fishing UI 实测).
// 跟 pkg/vision.DualBarOptions 形态一致.
type DualBarOptions struct {
	InnerMinPx      int     `json:"innerMinPx,omitempty"`
	InnerMaxPx      int     `json:"innerMaxPx,omitempty"`
	OuterMinPx      int     `json:"outerMinPx,omitempty"`
	BandRatioH      float64 `json:"bandRatioH,omitempty"`
	BandRatioInner  float64 `json:"bandRatioInner,omitempty"`
	ConfInnerWeight float64 `json:"confInnerWeight,omitempty"`
	ConfOuterWeight float64 `json:"confOuterWeight,omitempty"`
}

// LogService — 节点日志. 实际接 zerolog (main.go wire), 测试用 stdout stub.
type LogService interface {
	Debug(format string, args ...any)
	Info(format string, args ...any)
	Warn(format string, args ...any)
}

// InputService — KeyDown/KeyUp / Click / Scroll / MouseMoveRel + Hold start/stop.
// 镜像 pkg/input.Backend, 但 hwnd 由 Ctx 隐式提供 (内层 backend wire 时塞 hwnd).
// xRatio/yRatio 是 0-1 客户区比例.
type InputService interface {
	KeyDown(vk string) error
	KeyUp(vk string) error

	Click(xRatio, yRatio float64, button string, durationMs int) error
	MouseMoveRel(dx, dy, durationMs int) error
	MoveTo(xRatio, yRatio float64) error
	CursorRatio() (xRatio, yRatio float64, err error)
	Scroll(xRatio, yRatio float64, notches int) error

	MouseDown(xRatio, yRatio float64, button string) error
	MouseUp(button string) error
}

// VarStore — SetVar / GetVar / IncVar 节点用. Scope=auto/local/global 由 framework
// (RegionRunner / subgraph frame) 解析后传到这里; stub 是单层 map.
type VarStore interface {
	Get(name string) (value any, ok bool)
	Set(name string, value any)
	// Inc number 增量. value 不是 number → 视为 0 起步; 返回 newValue.
	Inc(name string, delta float64) (newValue float64)
	// GetScoped / SetScoped / IncScoped: scope = "auto" | "local" | "global".
	// auto: 当前 frame.LocalVars 有 → local; 否则 global.
	GetScoped(name, scope string) (value any, ok bool)
	SetScoped(name, scope string, value any)
	IncScoped(name, scope string, delta float64) (newValue float64)
}

// SysStore — GetSys 节点用 (read-only). path 形如 "now_ms" / "lastDualBarTrack.innerX" /
// "lastTemplate.found" — schema 见 services/container/sys/schema.go.
type SysStore interface {
	Get(path string) (value any, ok bool)
}

// ParamStore — GetParam 节点用. 读当前 frame 的 LocalParams (subgraph 入参).
// frame-private state, runtime 端 wire 时持 *ExecState getter; 节点端只 read.
// Snapshot wrap 不包 ParamStore (frame-state per-frame-private, 不需要 snapshot 语义).
type ParamStore interface {
	Get(name string) (value any, ok bool)
}

// WindowService — BringGameForeground 节点 + 任何节点想知道 hwnd/ClientSize.
type WindowService interface {
	BringForeground() error
	HWND() uintptr
	ClientSize() (w, h int, err error)
}

// CaptureService — Screenshot 节点用. PNG 字节流; wire 连 pkg/capture
// IBackend.Frame + png.Encode.
type CaptureService interface {
	Capture() (pngData []byte, err error)
	// CaptureROI 按 roi (ratio Geometry) 裁子帧后 png 编码返回. Geometry 零值 = 全帧.
	CaptureROI(roi Geometry) (pngData []byte, err error)
}

// StopwatchStore — StopwatchStart / Stop / Read 节点用. Per-key 多 stopwatch
// (跟 $vars.* 独立命名空间, 同名 key/var 不冲突).
//
// 语义:
//   - Start: 已存在 key 视为 reset.
//   - Stop:  不存在 key 视为 no-op (validator static-warn).
//   - Read:  不存在 key 返 0; running 返 now-start; stopped 返 stoppedAt-start.
type StopwatchStore interface {
	Start(key string)
	Stop(key string)
	Read(key string) (elapsedMs int64)
}

// ClipPlayer — PlayClip 节点用. 阻塞回放一整条录制的 InputClip (内部抢 InputBus 独占),
// ctx 取消即中断并释放已按下的键/键. runtime 端 wire 时持 ClipResolver + InputBackend +
// PlaybackPolicy + MouseCounts360 + Window 缩放; 节点端只调 Play.
//
// ctx 显式入参 (不藏在 bundle): bundle 是 per-runner 构造一次, ctx 是 per-Run tick —
// 回放取消必须接当前 Run 的 ctx, 跟 VisionService.Match 同模式.
type ClipPlayer interface {
	Play(ctx context.Context, clipID string) error
}

// ServiceBundle — RunNode 入参集合, 替代 8-arg signature. 全部字段 nullable;
// 节点 spec / 实测 wire 时决定哪些必填.
//
// Snapshot 是 framework-internal wrap hook: EvaluatePureData 入口调一次, nil → 跳过 wrap
// (StubServices 默认 nil, 测试不需要). 调用方 (runtime ContainerRunner) 负责 capture
// tick-frozen view; node 包不知 SysState 内部结构, 用 SysStore interface 承载.
type ServiceBundle struct {
	Vision      VisionService
	Log         LogService
	Input       InputService
	Vars        VarStore
	Sys         SysStore
	Params      ParamStore // frame.LocalParams getter, GetParam 用
	Window      WindowService
	Capture     CaptureService
	Stopwatches StopwatchStore
	Clip        ClipPlayer // PlayClip 用, runtime 端 wire
	Snapshot    func(ctx context.Context) Snapshot // tick snapshot getter, ctx 携带 runtime tickCtxKey value
}

type ValidationError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Field   string         `json:"field,omitempty"`
	Params  map[string]any `json:"params,omitempty"`
}

type Dependency struct {
	Kind string `json:"kind"`
	Key  string `json:"key"`
}
