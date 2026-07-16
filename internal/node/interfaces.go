// internal/node/interfaces.go
// Package node interfaces — Node + extension interfaces + Inputs/Outputs/Ctx contracts.
package node

import (
	"context"
	"fmt"
	"time"

	"github.com/yottaapp/yotta/internal/automation/target"
	"github.com/yottaapp/yotta/internal/services/llm"
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

// Validator — 节点自身静态校验.
type Validator interface {
	Validate(in Inputs) []ValidationError
}

// Dependencer — 子图分享 / library import 时 BFS 抽外部资产引用.
type Dependencer interface {
	Dependencies(in Inputs) []Dependency
}

// RegionRunner — 控制流节点 (Loop / Subgraph / CollapsedNode) 实现. body 回调
// 由 dispatch 构造, region 节点决定调用次数 (Loop 多次, Subgraph 单次). body error
// 裸透传, 由 dispatch 失败路由到 region 的 Fail 出口.
//
// body 第一返回值 = region 内部到达的出口 (Subgraph: callee OutputPins decl ID);
// "" 表示没有出口语义 (Loop 单轮迭代) 或没到达任何出口 — 节点据此决定 fire 哪个 exit.
type RegionRunner interface {
	Node
	RunRegion(ctx Ctx, in Inputs, body func(Ctx) (string, error)) (Outputs, error)
}

// Evaluator — pure-data 节点 (IsPureData=true) 实现, 走 EvaluatePureData 入口求 single value.
// 不返 Outputs (没 exit 出口), 直接返算出来的标量 — 给 data-edge 下游消费.
//
// 没实现 Evaluator → EvaluatePureData 返 error. 依赖 runtime state 的 pure-data 节点
// (GetVar) 看到的是 tick-frozen 快照: dispatch 入口 wrap services.Vars, 节点照常
// 写 ctx.Services().Vars 即可拿到一致视图 (见 EvaluatePureData 的 snapshot wrap).
type Evaluator interface {
	Evaluate(ctx Ctx, in Inputs) (any, error)
}

// Inputs — 节点 Run 收的输入. 取值优先级:
//  1. data-wire (纯数据 pin 上游) [最高]
//  2. Inspector config
//  3. exec-data wire (上游 exec 出口 Data 字段同名注入)
//  4. InputSpec.Default
//  5. Required + 缺 → framework 返 ValidationError (NOT panic)
//  6. Optional + 缺 → 零值, Has() = false
type Inputs interface {
	String(name string) string
	Float64(name string) float64
	Int(name string) int
	Bool(name string) bool
	StringList(name string) []string
	// List 读 List 型 pin ([]any). 非列表/nil → nil. 不把裸 string 当一元列表 (与 StringList 区别).
	List(name string) []any
	File(name string) (File, bool)
	Point(name string) Point
	Window(name string) (Window, bool)
	Rect(name string) Rect
	Geometry(name string) Geometry
	Color(name string) Color
	Duration(name string) time.Duration
	// JSONValue 读任意合法 JSON 值: object / array / scalar / nil.
	JSONValue(name string) any
	// JSON 兼容旧 object-only helper; 非 object 返回 nil.
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

// Ctx — Run 期间框架注入. 服务接口都返 nil-safe; 节点拿到 nil 调方法会 panic (清晰报错).
type Ctx interface {
	Context() context.Context
	Now() time.Time
	Out(exitName string) OutBuilder
	// Services returns the immutable per-dispatch service bundle. Keeping one
	// accessor makes Ctx stable as new optional runtime ports are introduced;
	// RuntimeCapabilities still controls which fields a node may require. The
	// returned value is a copy and omits the framework-internal Snapshot hook.
	Services() ServiceBundle

	// CaptureOutput 路径② (Spec C): region 节点每轮把迭代产出写进用户绑定变量。
	// field = 该节点 Body 出口声明的 Data 字段名; 框架查 config.capture[field], 非空则 SetScoped auto。
	// fire-time 节点不用调它 (框架在 dispatch routeResult 自动捕获出口 Data 字段)。
	CaptureOutput(field string, value any)
}

// OutBuilder — fluent builder.
//   - exitName 不在 Spec.Outputs → Fire panic
//   - Run 内调多次 Fire → 第 2 次 panic
type OutBuilder interface {
	Set(field string, value any) OutBuilder
	Fire() Outputs
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
	Scroll(xRatio, yRatio float64, notches int, horizontal bool) error

	MouseDown(xRatio, yRatio float64, button string) error
	MouseUp(button string) error

	// Drag 按住 button 从 (x1,y1) 拖到 (x2,y2), durationMs 毫秒. 坐标均为 0-1 客户区比例.
	Drag(x1, y1, x2, y2 float64, button string, durationMs int) error

	// TypeText 向当前窗口注入一串文字 (unicode, 逐 rune SendInput). 内部 backend 处理窗口激活.
	TypeText(s string) error
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
	// LastChange 返该变量上次 Set/Inc 的 unix-ms 时间戳；从未写过返 0。live 读，不受快照影响。
	LastChange(name string) int64
}

// ParamStore — GetParam 节点用. 读当前 frame 的 LocalParams (subgraph 入参).
// frame-private state, runtime 端 wire 时持 *ExecState getter; 节点端只 read.
// Snapshot wrap 不包 ParamStore (frame-state per-frame-private, 不需要 snapshot 语义).
type ParamStore interface {
	Get(name string) (value any, ok bool)
}

// WindowService — legacy Container window nodes 的 Win32 adapter seam.
type WindowService interface {
	BringForeground() error
	HWND() uintptr
	ClientSize() (w, h int, err error)
	// SetActive 运行时解析 title/class/processName 匹配的窗口, 设为当前活动窗口 (粘性).
	// 解析失败 (超时/取消) 返 error. ctx 用于取消等待.
	SetActive(ctx context.Context, title, class, processName, titleMatch string) error
	Maximize() error
	Minimize() error
	Restore() error
	BorderlessFullscreen() error
	RestoreBorders() error
	MoveResize(x, y, w, h int) error
	Close() error
	// Snapshot 返回当前活动窗口(含覆盖)的元数据快照, 给 Done.Window / Win32WindowTarget 用。
	Snapshot() (Window, error)
}

// TargetService — 切换/读取运行时活动自动化目标. Win32WindowTarget 仍走 WindowService;
// Android/Browser 等非窗口目标走这里, 后续输入/截图由 runtime controller factory 解析.
type TargetService interface {
	SetActive(target.Target) error
	Active() (target.Target, bool)
}

// AppLifecycleService — 目标内应用生命周期操作。Android ADB 走 package
// launch/force-stop；不支持该能力的 target 在 runtime adapter 中报错。
type AppLifecycleService interface {
	StartApp(packageName string) error
	StopApp(packageName string) error
}

// CaptureService — Capture 节点用. PNG 字节流; wire 连 pkg/capture
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

// SubgraphCaller — 脚本绑定层调子图用. runtime ContainerRunner 实现并注入 bundle;
// 非 runner 语境 (单节点测试 / StubServices) 为 nil — 绑定层据此决定是否暴露 Subgraph().
//
// CallSubgraph 同步跑完容器内 sgID 指定的子图 (push frame + 切 dispatch 表 + 跑到出口),
// params 直接 seed 为 callee 入参 (缺的声明入参补 Default). 返回到达出口的 decl Name
// (人读名, 非 decl ID — 脚本比对 r.exit 用). ctx 取消 / 递归超深 / 子图不存在 → error.
type SubgraphCaller interface {
	CallSubgraph(ctx context.Context, sgID string, params map[string]any) (exitName string, err error)
}

// ServiceBundle — RunNode 入参集合, 替代 8-arg signature. 全部字段 nullable;
// 节点 spec / 实测 wire 时决定哪些必填.
//
// Snapshot 是 framework-internal wrap hook: EvaluatePureData 入口调一次, nil → 跳过 wrap
// (StubServices 默认 nil, 测试不需要). 调用方 (runtime ContainerRunner) 负责 capture
// tick-frozen view.
type ServiceBundle struct {
	Log         LogService
	Input       InputService
	Vars        VarStore
	Params      ParamStore // frame.LocalParams getter, GetParam 用
	Window      WindowService
	Target      TargetService
	App         AppLifecycleService
	Capture     CaptureService
	Stopwatches StopwatchStore
	Subgraphs   SubgraphCaller                     // 脚本调子图用, runtime 端 wire (ContainerRunner 自身)
	AI          AIProviderService                  // AI 节点取按连接缓存的 llm.Provider, runtime 端从 settings wire
	Registry    RegistryReader                     // Script 只绑定本 runner registry generation
	Snapshot    func(ctx context.Context) Snapshot // tick snapshot getter, ctx 携带 runtime tickCtxKey value
}

// AssemblyError reports a missing runtime dependency before node code runs.
type AssemblyError struct {
	NodeKind   string
	Capability RuntimeCapability
}

func (e *AssemblyError) Error() string {
	return fmt.Sprintf("node %q requires runtime capability %q, but it is not wired", e.NodeKind, e.Capability)
}

// AIProviderService — AI 节点按 connectionID(空 = ai.default)取一个缓存的 llm.Provider。
// 实现持 settings getter, 取用时按连接指纹自愈; runtime 端注入。
type AIProviderService interface {
	Provider(connectionID string) (llm.Provider, error)
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
