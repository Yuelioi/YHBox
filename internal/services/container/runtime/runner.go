package runtime

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"

	"github.com/rs/zerolog"

	"yotta/internal/node"
	"yotta/internal/services/container"
	"yotta/internal/services/inputclip/backends"
	pkgcapture "yotta/pkg/capture"
	pkginput "yotta/pkg/input"
)

// LoopFrame Loop body 期间的"我在哪个 Loop 里"上下文。Break/Continue 跳目标。
type LoopFrame struct {
	LoopNodeID string
	Iter       int64
	// 边查找：完成（complete）走 LoopNodeID.complete 出 pin，
	// 继续（continue）回到 LoopNodeID.in。
}

// ExecToken 待执行控制流令牌。
type ExecToken struct {
	NodeID    string
	InPin     string       // 进入哪个 in pin（默认 "in"）
	LoopStack []*LoopFrame // 最深的 frame 在尾
	// ExecData carry — 上游节点 ctx.Out("exit").Set("k", v).Fire() 推下来的 OutputData
	// 字段, 下游 build inputs 时通过 buildExecDataFor 读. 比 data-edge 更轻 (不走 spec 声明,
	// 跟 exit pin 绑定; 多下游 fanout 共享同一份 map).
	ExecData map[string]any
}

// edgeIndex 把 graph 边按 from-pin 索引一遍，dispatch 时 O(1) 查下游。
type edgeIndex struct {
	// from key: "<nodeId>.<pin>" → list of "<nodeId>.<pin>" downstream entry pins
	out map[string][]string
}

func buildEdgeIndex(g container.Graph) *edgeIndex {
	idx := &edgeIndex{out: make(map[string][]string)}
	for _, e := range g.Edges {
		idx.out[e.From] = append(idx.out[e.From], e.To)
	}
	return idx
}

// next 走 from "<nodeId>.<pin>" 输出，返下游 token 列表（一般 1 个；Parallel 可能 N 个）。
func (idx *edgeIndex) next(from string, currentLoops []*LoopFrame) []ExecToken {
	return idx.nextWithData(from, currentLoops, nil)
}

// nextWithData 跟 next 一样查下游, 但额外把 execData 挂到每个 ExecToken.ExecData.
// 多 fanout 下游共享同一份 map (immutable 视角; 下游不该改). data 经 build inputs 进
// merged map 不会改原 map, 安全.
func (idx *edgeIndex) nextWithData(from string, currentLoops []*LoopFrame, execData map[string]any) []ExecToken {
	tos := idx.out[from]
	if len(tos) == 0 {
		return nil
	}
	out := make([]ExecToken, 0, len(tos))
	for _, to := range tos {
		parts := strings.SplitN(to, ".", 2)
		if len(parts) != 2 {
			continue
		}
		out = append(out, ExecToken{
			NodeID:    parts[0],
			InPin:     parts[1],
			LoopStack: copyLoops(currentLoops),
			ExecData:  execData,
		})
	}
	return out
}

func copyLoops(src []*LoopFrame) []*LoopFrame {
	cp := make([]*LoopFrame, len(src))
	copy(cp, src)
	return cp
}

// ContainerRunner 跑一个 container 节点图 (token dispatch loop).
type ContainerRunner struct {
	rt        *RuntimeContext
	compiled  *CompiledContainer // 主图 + 所有 subgraphs 的 CompiledGraph 一次性产物.
	currentSG *CompiledGraph     // subgraph swap 时设, runRegionBody 用来识 entry/output marker.
	nodesByID map[string]*container.GraphNode
	edges     *edgeIndex
	dataEdges *dataEdgeIndex
	state     *ExecState
	stopwatches *stopwatchTable

	// bundle 是 node.ServiceBundle (LogService / VarStore / VisionService 等 8 个 adapter),
	// execNode 走 node.RunNode dispatch 时消费. 默认 Log 是 zerolog.Nop, main.go 启动后
	// SetLogger 注入真 logger.
	//
	// per-tick snapshot 不是 instance 字段, 而是 ctx (tickCtxKey) — dispatchInRegion 入口
	// withTickSnapshot 写, bundle.Snapshot 闭包从 ctx 读, per-goroutine 独立.
	bundle node.ServiceBundle
}

func NewContainerRunner(rt *RuntimeContext) *ContainerRunner {
	// 防御性 normalize — 兜底 in-memory 构造 (test fixture / 工具脚本) 没走 Store.Save 路径的
	// container, 保证 sg.Entry / OutputPins[*].NodeID 不空. Store-loaded container 已 normalize 过, 幂等.
	rt.Container.Normalize()
	cc := CompileContainer(rt.Container)
	// PlayClip 回放 target (rt.MouseCounts360) 用主图 MouseCalibration 节点值 — 容器自包含,
	// 跟 MouseMoveRel 的 state.CalibCounts 同源 (都来自 snapshotMainCalibCounts). 无节点 (=0)
	// 时保留构造期传入的 settings 本机值兜底.
	if cc.MainCalibCounts > 0 {
		rt.MouseCounts360 = cc.MainCalibCounts
	}
	r := &ContainerRunner{
		rt:          rt,
		compiled:    cc,
		nodesByID:   cc.Main.NodesByID,
		edges:       cc.Main.Edges,
		dataEdges:   cc.Main.DataEdges,
		stopwatches: newStopwatchTable(),
	}
	r.state = NewExecState(rt.Container.ID, cc.MainCalibCounts)
	// 默认 LogService 是 zerolog.Nop (沉默). main.go SetLogger 注入真 logger.
	// stateGetter — closure 让 VarStoreAdapter scope=local/auto 拿到 frame.LocalVars 栈.
	// tick snapshot 走 ctx (tickCtxKey) 不传 getter — bundle.Snapshot 闭包内部读 ctx.
	r.bundle = NewServiceBundleFor(
		rt,
		r.stopwatches,
		zerolog.Nop(),
		func() *ExecState { return r.state },
	)
	return r
}

// SetLogger 替换 bundle 里的 LogService 为真 zerolog logger.
// main.go runFunc 在 NewContainerRunner 后调一次.
func (r *ContainerRunner) SetLogger(log zerolog.Logger) {
	r.bundle.Log = NewLogAdapter(log)
}

// Bundle 返当前 ContainerRunner 持有的 ServiceBundle (dispatch 用).
func (r *ContainerRunner) Bundle() node.ServiceBundle { return r.bundle }

// snapshotMainCalibCounts 从主图找 MouseCalibration 节点 config.counts360 当启动 snapshot.
// 没节点 / counts360=0 → 返 0 (runtime 不缩放).
func snapshotMainCalibCounts(c *container.Container) int {
	for i := range c.Graph.Nodes {
		n := &c.Graph.Nodes[i]
		if n.Kind == "MouseCalibration" {
			if v, ok := container.PinInt(n, "Counts360"); ok {
				return v
			}
		}
	}
	return 0
}

// Run 启动 token dispatch：找 Start → 入队 → 主循环。
// 同时为每个 listener-driven 节点 (OnEvent / EventTick) 起后台 listener goroutine。
// 返回时机:
//   - 无 listener: 主流程 (queue 清空) 即返回。
//   - 有 listener: 等外层 ctx 取消/超时 → defer cancelChild → listenerWG.Wait → 返回
//     (后台 listener 要活到容器被外部停掉, 不能因主流程跑完就被秒杀)。
func (r *ContainerRunner) Run(ctx context.Context) error {
	// resolve WindowTarget → 活动窗口/Input/Capture (per-run state).
	// 必须最先做 — 后续 startNode/listener 都假设 rt.window(经 accessor)/Input/Capture 已 populate.
	if err := r.setupRuntime(); err != nil {
		return err
	}
	defer r.teardownRuntime() // LIFO 内部顺序保证 ReleaseAll → Input.Close → Capture.Close

	startNode := r.findStart()
	if startNode == nil {
		return errors.New("container: no Start node")
	}

	// 子 ctx：监听 goroutine 生命周期由 childCtx 控制。
	// cancelChild 在 Run 退出时调用（defer）确保 ctx 资源回收。
	// listener 跑完后 listenerWG 归零，Run 才真正返回。
	//
	// 生命周期语义:
	//   - 主 dispatch 正常结束 (队列空) 且有 listener: 等外层 ctx 超时/取消后再 cancel + wait.
	//   - 提前退出 (Stop / error / ctx.Err): cancelChild() 先发, listener 收到立即退; Wait 随后解锁.
	childCtx, cancelChild := context.WithCancel(ctx)
	var listenerWG sync.WaitGroup
	// LIFO: cancelChild 后注册先执行 → listener 收到 cancel → listenerWG.Wait() 解锁 → Run 返回.
	// 正常队列结束路径会先等外层 ctx (见下方), 此 defer 只作兜底资源释放.
	defer listenerWG.Wait()
	defer cancelChild()

	hasListeners := false
	for i := range r.rt.Container.Graph.Nodes {
		n := &r.rt.Container.Graph.Nodes[i]
		if !isListenerDriven(n.Kind) {
			continue
		}
		hasListeners = true
		l := newEventListener(r, n)
		listenerWG.Add(1)
		go func() {
			defer listenerWG.Done()
			l.run(childCtx)
		}()
	}

	queue := []ExecToken{}
	// Start 输出 → 入第一批 token
	queue = append(queue, r.edges.next(startNode.ID+".Done", nil)...)

	dispatchCount := 0
	for len(queue) > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		dispatchCount++
		if dispatchCount%1000 == 0 {
			// 每 1k token 让出一次 OS 线程，避免纯非 yield 节点组成的 hot loop 占满 CPU
			// 把当前 goroutine 让给 scheduler 跑别的（GC、listener、UI emit 等）
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			runtime.Gosched()
		}
		tok := queue[0]
		queue = queue[1:]

		node, ok := r.nodesByID[tok.NodeID]
		if !ok {
			return fmt.Errorf("container: token references unknown node %q", tok.NodeID)
		}

		// 节点级事件：让前端高亮"当前正在跑哪个节点"。
		if r.rt.Emit != nil {
			r.rt.Emit("container:node-enter", map[string]any{
				"containerId": r.rt.Container.ID,
				"nodeId":      node.ID,
				"nodeKind":    node.Kind,
			})
		}

		// per-exec-tick snapshot 由 dispatchInRegion 入口统一抓 (单一抓点), 这里不再抓.
		// passthroughDisabled / IsVisualOnly / IsPureData reject 路径都不需要 snapshot
		// (consumer GetVar/GetSys.Evaluate 经 framework snapshot wrap 在 data pull 阶段读).
		out, err := r.execNode(ctx, node, tok)
		if err != nil {
			if errors.Is(err, errStopRun) {
				return nil
			}
			// Break/Continue/Throw sentinel 漏到顶层 dispatch — Loop/Try 没截获,
			// validator 应已报 *_OUTSIDE_* 但跑到此说明 graph 跑了未校验路径或 runtime path bug.
			// emit container:node-validation 让前端高亮.
			if _, wrapped := r.checkSentinelLeak(node, err); wrapped != err {
				return wrapped
			}
			return err
		}
		queue = append(queue, out...)
	}
	// 主 dispatch 结束：若容器含 listener-driven 节点, 它们仍在后台跑直到 ctx 取消。
	// 等外部 ctx 超时/取消后再返回, 让 listener 完成当前触发周期再退。
	if hasListeners {
		<-ctx.Done()
	}
	return nil
}

func (r *ContainerRunner) findStart() *container.GraphNode {
	for i := range r.rt.Container.Graph.Nodes {
		n := &r.rt.Container.Graph.Nodes[i]
		if n.Kind == "Start" {
			return n
		}
	}
	return nil
}

// errStopRun Stop 节点用：让主循环优雅退。
var errStopRun = errors.New("stop")

// All exec nodes read inputs via r.pullDataPin (data edge or inline literal).
// Expr nodes use expr.InputEnv (bare-identifier only — no $-namespace).
// Variables / sys / params are routed via GetVar / GetSys / GetParam nodes wired through
// data edges, NOT through env paths in expressions.

// configString 读 pin 字面量字符串 (literal 优先 + 顶层 config fallback, 镜像 newInputs)。
// key 必须是规范 PascalCase Spec.Input 名。
func configString(node *container.GraphNode, key string) string {
	return container.PinString(node, key)
}

func configStringList(node *container.GraphNode, key string) []string {
	return container.PinStringList(node, key)
}

// ----------------------------------------------------------------------------
// runtime bootstrap: setupRuntime / teardownRuntime / WindowTarget 解析.
// ----------------------------------------------------------------------------

// setupRuntime 按容器级配置建 input/capture backend → populate rt.
// 不在启动期解析窗口 — 窗口由 WindowTarget.Run 运行时解析 (SetActive).
// 幂等: 测试预设过 (fixture 注入 window + input) 就跳过.
func (r *ContainerRunner) setupRuntime() error {
	// 测试预设过 (fixture 注入 window + input) 就跳过.
	if r.rt.WindowHandle().HWND != 0 && r.rt.Input != nil {
		return nil
	}
	if !containerNeedsWindow(r.rt.Container) {
		return nil
	}

	// 后端按容器级配置建 (hwnd 按调用传参, 无窗口也能先建). 窗口由 WindowTarget.Run 运行时解析.
	inputName := r.rt.Container.InputBackend
	if inputName == "" {
		inputName = "postmessage"
	}
	rawInput, err := pkginput.NewBackend(inputName)
	if err != nil {
		return fmt.Errorf("input backend %q: %w", inputName, err)
	}
	if r.rt.Input == nil {
		r.rt.Input = NewSafeInputBackend(rawInput, r.rt)
	}

	if r.rt.InputBackend == nil {
		r.rt.InputBackend = backends.NewHybridBackend(func() uintptr {
			h, _ := r.rt.ActiveHWND()
			return h
		})
	}

	captureName := r.rt.Container.CaptureBackend
	if captureName == "" {
		captureName = "auto"
	}
	rawCapture, captureWarning, err := pkgcapture.NewIBackend(captureName)
	if err != nil {
		return fmt.Errorf("capture backend %q: %w", captureName, err)
	}
	if captureWarning != "" && r.rt.Emit != nil {
		r.rt.Emit("container:warning", map[string]any{"message": captureWarning})
	}
	if r.rt.Capture == nil {
		r.rt.Capture = NewSafeCaptureBackend(rawCapture, r.rt)
	}
	return nil
}

// teardownRuntime LIFO 关闭. 按 ReleaseAll → Input.Close → Capture.Close 序退出.
// (defer 内的 defer 也是 LIFO 弹出, 写顺序反着即可)
func (r *ContainerRunner) teardownRuntime() {
	// 写序: Capture.Close → Input.Close → ReleaseAll
	// LIFO 执行序: ReleaseAll → Input.Close → Capture.Close (符合契约)
	if r.rt.Capture != nil {
		defer r.rt.Capture.Close()
	}
	if r.rt.Input != nil {
		defer r.rt.Input.Close()
		defer r.rt.Input.ReleaseAll()
	}
}

// containerNeedsWindow 容器是否含任一需要目标窗口的节点 (Spec.NeedsWindow) — 主图或任一子图.
// 跟 validator.containerNeedsWindow 同判定: 决定 runtime 是否解析 WindowTarget. 纯窗口无关
// 容器跳过解析, 窗口类节点 (ClickAt/Detect/Capture/PlayClip...) 才要求.
func containerNeedsWindow(c *container.Container) bool {
	if graphHasWindowNode(c.Graph.Nodes) {
		return true
	}
	for i := range c.Subgraphs {
		if graphHasWindowNode(c.Subgraphs[i].Graph.Nodes) {
			return true
		}
	}
	return false
}

func graphHasWindowNode(nodes []container.GraphNode) bool {
	for i := range nodes {
		if rn, ok := node.Get(nodes[i].Kind); ok && rn.Spec.NeedsWindow {
			return true
		}
	}
	return false
}

// isListenerDriven — kind 是否走后台 listener goroutine (无 exec-in, 不进主 dispatch)。
// 新增同类事件节点在这里加 kind 即可, 无需动 spawn 循环。
func isListenerDriven(kind string) bool {
	return kind == "OnEvent" || kind == "EventTick"
}

