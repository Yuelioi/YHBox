package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"yhbox/internal/node"
	"yhbox/internal/services/container"
	pkgcapture "yhbox/pkg/capture"
	pkginput "yhbox/pkg/input"
	"yhbox/pkg/winutil"
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
	// 跟 exit pin 绑定; 多下游 fanout 共享同一份 map). 老 v4 runtime "exec-data" 行为.
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
//
// Backlog: 改 cooperative scheduler 让 Stop 真·瞬停 + 多容器公平调度 — 见 backend-backlog.md B6.
type ContainerRunner struct {
	rt          *RuntimeContext
	compiled    *CompiledContainer // B3: 主图 + 所有 subgraphs 的 CompiledGraph 一次性产物.
	nodesByID   map[string]*container.GraphNode
	edges       *edgeIndex
	dataEdges   *dataEdgeIndex
	state       *ExecState
	stopwatches *stopwatchTable

	// currentTick is the per-exec-tick snapshot of rt.vars + rt.sys.
	// Captured before each execNode dispatch and cleared after; consumed by GetVar / GetSys
	// pure-data nodes during the same tick to guarantee data consistency.
	currentTick *TickSnapshot

	// bundle 是 Phase 5.4 wire 的 node.ServiceBundle (LogService / VarStore / VisionService 等
	// 8 个 adapter). Phase 5.5 ContainerRunner.execNode 改 node.RunNode dispatch 时消费.
	// 默认 Log 是 zerolog.Nop, main.go 启动后 SetLogger 注入真 logger.
	bundle node.ServiceBundle
}

func NewContainerRunner(rt *RuntimeContext) *ContainerRunner {
	cc := CompileContainer(rt.Container)
	r := &ContainerRunner{
		rt:          rt,
		compiled:    cc,
		nodesByID:   cc.Main.NodesByID,
		edges:       cc.Main.Edges,
		dataEdges:   cc.Main.DataEdges,
		stopwatches: newStopwatchTable(),
	}
	r.state = NewExecState(rt.Container.ID, cc.MainCalibCounts)
	// Phase 5.4: 默认 LogService 是 zerolog.Nop (沉默). main.go SetLogger 注入真 logger.
	// stateGetter — closure 让 VarStoreAdapter scope=local/auto 拿到 frame.LocalVars 栈.
	// tickGetter — closure 让 PureData Evaluator (EvaluatePureData wrap) 拿到 per-tick
	// frozen Vars/Sys view (r.currentTick 在 dispatch_v5.execNode 入口 capture).
	r.bundle = NewServiceBundleFor(
		rt,
		r.stopwatches,
		zerolog.Nop(),
		func() *ExecState { return r.state },
		func() *TickSnapshot { return r.currentTick },
	)
	return r
}

// SetLogger 替换 bundle 里的 LogService 为真 zerolog logger.
// main.go runFunc 在 NewContainerRunner 后调一次.
func (r *ContainerRunner) SetLogger(log zerolog.Logger) {
	r.bundle.Log = NewLogAdapter(log)
}

// Bundle 返当前 ContainerRunner 持有的 ServiceBundle (Phase 5.5 dispatch 用).
func (r *ContainerRunner) Bundle() node.ServiceBundle { return r.bundle }

// snapshotMainCalibCounts 从主图找 MouseCalibration 节点 config.counts360 当启动 snapshot.
// 没节点 / counts360=0 → 返 0 (runtime 不缩放).
func snapshotMainCalibCounts(c *container.Container) int {
	for _, n := range c.Graph.Nodes {
		if n.Kind == "MouseCalibration" {
			if v, ok := n.Config["Counts360"]; ok {
				switch x := v.(type) {
				case int:
					return x
				case int64:
					return int(x)
				case float64:
					return int(x)
				case json.Number:
					if i, err := x.Int64(); err == nil {
						return int(i)
					}
				}
			}
		}
	}
	return 0
}

// Run 启动 token dispatch：找 Start → 入队 → 主循环。
// 同时为每个 OnEvent 节点起 listener goroutine（§5.4）。
// Run 返回 = 主流程结束 + 所有 listener 退出。
func (r *ContainerRunner) Run(ctx context.Context) error {
	// Phase B: resolve WindowTarget → Window/Input/Capture (per-run state).
	// 必须最先做 — 后续 startNode/listener 都假设 rt.Window/Input/Capture 已 populate.
	if err := r.setupRuntime(); err != nil {
		return err
	}
	defer r.teardownRuntime() // LIFO 内部顺序保证 ReleaseAll → Input.Close → Capture.Close

	startNode := r.findStart()
	if startNode == nil {
		return errors.New("container: no Start node")
	}

	// 子 ctx：主流程结束时 cancel listener；同时 listener 自己跑 sub-runner 用这个 ctx 派生。
	// Defer 顺序敏感（LIFO）：Wait 必须晚于 cancel 注册，cancel 先触发 → listener 退出 → Wait 解锁。
	// 顺序反了会死锁：Wait 阻塞等 listener，但 listener 等 ctx cancel 才退。
	childCtx, cancelChild := context.WithCancel(ctx)
	var listenerWG sync.WaitGroup
	defer listenerWG.Wait() // LIFO 后注册 → 后执行（在 cancelChild 之后跑）
	defer cancelChild()     // LIFO 后注册 → 先执行（先 cancel，让 listener 退出）

	for i := range r.rt.Container.Graph.Nodes {
		n := &r.rt.Container.Graph.Nodes[i]
		if n.Kind != "OnEvent" {
			continue
		}
		l := newEventListener(r, n)
		listenerWG.Add(1)
		go func() {
			defer listenerWG.Done()
			l.run(childCtx)
		}()
	}

	queue := []ExecToken{}
	// Start 输出 → 入第一批 token
	queue = append(queue, r.edges.next(startNode.ID+".out", nil)...)

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

		// per-exec-tick snapshot 由 dispatchInRegion 入口统一抓 (P1.6 单一抓点), 这里不再抓.
		// passthroughDisabled / IsVisualOnly / IsPureData reject 路径都不需要 snapshot
		// (consumer GetVar/GetSys.Evaluate 经 framework snapshot wrap 在 data pull 阶段读).
		out, err := r.execNode(ctx, node, tok)
		if err != nil {
			if errors.Is(err, errStopRun) {
				return nil
			}
			// P1.2: Break/Continue/Throw sentinel 漏到顶层 dispatch — Loop/Try 没截获,
			// validator 应已报 *_OUTSIDE_* 但跑到此说明 graph 跑了未校验路径或 runtime path bug.
			// emit container:node-validation 让前端高亮.
			if _, wrapped := r.checkSentinelLeak(node, err); wrapped != err {
				return wrapped
			}
			return err
		}
		queue = append(queue, out...)
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

// v4 single-path: configExpr / configFloat / runnerEnv / r.env() removed.
// All exec nodes now read inputs via r.pullDataPin (data edge or inline literal).
// Expr nodes use expr.InputEnv (bare-identifier only — no $-namespace).
// Variables / sys / params are routed via GetVar / GetSys / GetParam nodes wired through
// data edges, NOT through env paths in expressions.

// configString 拿 config[key] 当字面量字符串（不当表达式 parse）。
func configString(node *container.GraphNode, key string) string {
	if node.Config == nil {
		return ""
	}
	if v, ok := node.Config[key].(string); ok {
		return v
	}
	return ""
}

// ----------------------------------------------------------------------------
// v3 Phase B runtime bootstrap: setupRuntime / teardownRuntime / WindowTarget 解析.
// ----------------------------------------------------------------------------

// setupRuntime 找 WindowTarget 节点 → resolve hwnd → 建 input/capture backend → populate rt.
// 幂等: 如果 rt.Window.HWND != 0 且 rt.Input != nil (测试预设过) 就跳过 — 测试 fixture 可以
// 注入 fake backend + stub hwnd 不走 Win32 resolve.
func (r *ContainerRunner) setupRuntime() error {
	if r.rt.Window.HWND != 0 && r.rt.Input != nil {
		// 测试预设过了, 跳过 resolve. capture 也跳过 (测试通常用不到 capture).
		return nil
	}

	wtNode := findMainGraphNode(r.rt.Container, "WindowTarget")
	if wtNode == nil {
		return errors.New("MISSING_WINDOW_TARGET — container 缺 WindowTarget 节点")
	}
	matchSpec := readWindowTargetMatchSpec(wtNode)
	runtimeSpec := readWindowTargetRuntimeSpec(wtNode)

	// 1) hwnd + metadata
	wh, err := winutil.ResolveWindow(matchSpec, 3*time.Second, 500*time.Millisecond)
	if err != nil {
		return fmt.Errorf("WindowTarget resolve: %w", err)
	}
	r.rt.Window = wh

	// 2) input backend (默认 postmessage)
	inputName := runtimeSpec["InputBackend"]
	if inputName == "" {
		inputName = "postmessage"
	}
	rawInput, err := pkginput.NewBackend(inputName)
	if err != nil {
		return fmt.Errorf("input backend %q: %w", inputName, err)
	}
	r.rt.Input = NewSafeInputBackend(rawInput, r.rt)

	// 3) capture backend (auto + fallback)
	captureName := runtimeSpec["CaptureBackend"]
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
	r.rt.Capture = NewSafeCaptureBackend(rawCapture, r.rt)

	// ROI 分辨率检查: 遍历主图 + 所有子图，找带 _capturedAtResolution 的节点。
	// 窗口 clientW/clientH 已由上面的 ResolveWindow 填好，逐节点对比后发出 warning（不阻断）。
	if r.rt.Emit != nil {
		clientW := r.rt.Window.ClientW
		clientH := r.rt.Window.ClientH
		checkROINode := func(n *container.GraphNode) {
			rawCap, ok := n.Config["_capturedAtResolution"].([]any)
			if !ok || len(rawCap) != 2 {
				return
			}
			cw, _ := rawCap[0].(float64)
			ch, _ := rawCap[1].(float64)
			if int(cw) != clientW || int(ch) != clientH {
				r.rt.Emit("container:warning", map[string]any{
					"nodeId":  n.ID,
					"code":    "ROI_RESOLUTION_MISMATCH",
					"message": fmt.Sprintf("node %q ROI captured at %vx%v but window is %dx%d — accuracy may degrade", n.ID, cw, ch, clientW, clientH),
				})
			}
		}
		for i := range r.rt.Container.Graph.Nodes {
			checkROINode(&r.rt.Container.Graph.Nodes[i])
		}
		for i := range r.rt.Container.Subgraphs {
			for j := range r.rt.Container.Subgraphs[i].Graph.Nodes {
				checkROINode(&r.rt.Container.Subgraphs[i].Graph.Nodes[j])
			}
		}
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

// findMainGraphNode 主图找指定 kind 的第一个节点. v3 WindowTarget / MouseCalibration 等
// 声明式节点都是 single-instance per container — 找到即停.
func findMainGraphNode(c *container.Container, kind string) *container.GraphNode {
	for i := range c.Graph.Nodes {
		if c.Graph.Nodes[i].Kind == kind {
			return &c.Graph.Nodes[i]
		}
	}
	return nil
}

// readWindowTargetMatchSpec 解析 WindowTarget.config 顶级匹配字段.
// P2.1: 之前 nested 在 config.match — 已扁平化跟 Spec.Inputs 对齐.
func readWindowTargetMatchSpec(n *container.GraphNode) winutil.MatchSpec {
	if n.Config == nil {
		return winutil.MatchSpec{}
	}
	getStr := func(k string) string {
		v, _ := n.Config[k].(string)
		return v
	}
	return winutil.MatchSpec{
		Title:       getStr("Title"),
		Class:       getStr("Class"),
		ProcessName: getStr("ProcessName"),
		TitleMatch:  getStr("TitleMatch"),
	}
}

// readWindowTargetRuntimeSpec 解析 WindowTarget.config 顶级 runtime 字段 (InputBackend /
// CaptureBackend). 之前 nested 在 config.runtime — 已扁平化.
// 返 map[string]string 兼容老 caller (key 是 PascalCase: "InputBackend"/"CaptureBackend").
func readWindowTargetRuntimeSpec(n *container.GraphNode) map[string]string {
	if n.Config == nil {
		return map[string]string{}
	}
	out := map[string]string{}
	for _, k := range []string{"InputBackend", "CaptureBackend"} {
		if s, ok := n.Config[k].(string); ok {
			out[k] = s
		}
	}
	return out
}
