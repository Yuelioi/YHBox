package runtime

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"yhbox/internal/services/container"
	"yhbox/internal/services/expr"
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
		out = append(out, ExecToken{NodeID: parts[0], InPin: parts[1], LoopStack: copyLoops(currentLoops)})
	}
	return out
}

func copyLoops(src []*LoopFrame) []*LoopFrame {
	cp := make([]*LoopFrame, len(src))
	copy(cp, src)
	return cp
}

// ContainerRunner 跑一个 container 节点图（token dispatch loop）。
//
// FUTURE-WORK (spec §11): 当前 dispatch 是 preemptive — 单 goroutine + ctx.Done 检查间隔
// 决定响应性. 节点内长任务 (如 WaitTemplate 默认 30s timeout) 无法中途响应高优先级
// 中断 (热键 Stop 要等节点 return). 长期应换 cooperative scheduler: 节点要按 budget 主动
// yield ((rt.Tick(); if budget exceeded { return CONTINUE })), runner 在 yield 点检查
// 中断 / 优先级队列 / cancellation. 收益: 1) Stop 真·瞬停 (<5ms); 2) 多容器并发跑能公平
// 调度而不是抢 CPU; 3) 资源限制 (单容器 CPU/帧抓取配额). 改的时机: 用户开始抱怨"停止
// 慢"或"两个容器一起跑卡". 注意改造范围大, 14+ 节点 exec 函数都要加 yield 点.
type ContainerRunner struct {
	rt        *RuntimeContext
	nodesByID map[string]*container.GraphNode
	edges     *edgeIndex
	state     *ExecState
}

func NewContainerRunner(rt *RuntimeContext) *ContainerRunner {
	r := &ContainerRunner{
		rt:        rt,
		nodesByID: make(map[string]*container.GraphNode),
		edges:     buildEdgeIndex(rt.Container.Graph),
	}
	for i := range rt.Container.Graph.Nodes {
		n := &rt.Container.Graph.Nodes[i]
		r.nodesByID[n.ID] = n
	}
	r.state = NewExecState(rt.Container.ID, snapshotMainCalibCounts(rt.Container))
	return r
}

// snapshotMainCalibCounts 从主图找 MouseCalibration 节点 config.counts360 当启动 snapshot。
// 没节点 / counts360=0 → 返 0（runtime 不缩放）。v2 spec §2.7。
func snapshotMainCalibCounts(c *container.Container) int {
	for _, n := range c.Graph.Nodes {
		if n.Kind == "MouseCalibration" {
			if v, ok := n.Config["counts360"]; ok {
				switch x := v.(type) {
				case int:
					return x
				case int64:
					return int(x)
				case float64:
					return int(x)
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

		out, err := r.execNode(ctx, node, tok)
		if err != nil {
			if errors.Is(err, errStopRun) {
				return nil
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

// configExpr 拿节点 config 字段 string，parse + eval。缺字段返 (zero, nil)。
func (r *ContainerRunner) configExpr(node *container.GraphNode, key string) (expr.Value, error) {
	if node.Config == nil {
		return nil, nil
	}
	raw, ok := node.Config[key]
	if !ok {
		return nil, nil
	}
	s, ok := raw.(string)
	if !ok {
		// 已经是非 string 字面量（前端 v1 不该走到这）：直接返
		return raw, nil
	}
	if s == "" {
		return nil, nil
	}
	ast, err := expr.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("node %s.%s: parse %q: %w", node.ID, key, s, err)
	}
	return expr.Eval(ast, r.env())
}

// env 把 ExecState 的 LocalVars 链叠加到 rt.Env() 上层（Gemini 雷点三）。
// $vars.X 解析顺序：当前 frame.LocalVars → parent.LocalVars → ... → rt.vars。
// 其它命名空间（$params / $sys）原样走 rt.Env()。
func (r *ContainerRunner) env() expr.Env {
	return &runnerEnv{rt: r.rt, state: r.state}
}

type runnerEnv struct {
	rt    *RuntimeContext
	state *ExecState
}

func (e *runnerEnv) Get(path string) (expr.Value, error) {
	// 仅 $vars.X 走 frame-local 优先；$params / $sys 直接转给 rt.Env。
	if len(path) > 6 && path[:6] == "$vars." {
		name := path[6:]
		if v, ok := e.state.ResolveVar(name); ok {
			// LocalVars 命中：转换为 expr.Value 类型（Set 时存什么类型就拿什么类型）
			return v, nil
		}
		// 兜底走 rt.vars
	}
	return e.rt.Env().Get(path)
}

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

// configFloat helper：返 float64 或 0。
func (r *ContainerRunner) configFloat(node *container.GraphNode, key string, fallback float64) float64 {
	v, err := r.configExpr(node, key)
	if err != nil || v == nil {
		return fallback
	}
	if f, ok := expr.AsNumber(v); ok {
		return f
	}
	return fallback
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
	inputName := runtimeSpec["inputBackend"]
	if inputName == "" {
		inputName = "postmessage"
	}
	rawInput, err := pkginput.NewBackend(inputName)
	if err != nil {
		return fmt.Errorf("input backend %q: %w", inputName, err)
	}
	r.rt.Input = NewSafeInputBackend(rawInput, r.rt)

	// 3) capture backend (auto + fallback)
	captureName := runtimeSpec["captureBackend"]
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

// readWindowTargetMatchSpec 解析 WindowTarget.config.match (匹配条件 — JSON 来料 map).
func readWindowTargetMatchSpec(n *container.GraphNode) winutil.MatchSpec {
	if n.Config == nil {
		return winutil.MatchSpec{}
	}
	matchAny, _ := n.Config["match"].(map[string]any)
	getStr := func(k string) string {
		v, _ := matchAny[k].(string)
		return v
	}
	return winutil.MatchSpec{
		Title:       getStr("title"),
		Class:       getStr("class"),
		ProcessName: getStr("processName"),
		TitleMatch:  getStr("titleMatch"),
	}
}

// readWindowTargetRuntimeSpec 解析 WindowTarget.config.runtime (inputBackend / captureBackend).
// 返 string map — 当前两个 key 都是 string 枚举 ("postmessage" / "auto" 等).
func readWindowTargetRuntimeSpec(n *container.GraphNode) map[string]string {
	if n.Config == nil {
		return map[string]string{}
	}
	rtAny, _ := n.Config["runtime"].(map[string]any)
	out := map[string]string{}
	for k, v := range rtAny {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	return out
}
