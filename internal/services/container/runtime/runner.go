package runtime

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"

	"github.com/rs/zerolog"

	"github.com/yottaapp/yotta/internal/node"
	"github.com/yottaapp/yotta/internal/nodes/control"
	"github.com/yottaapp/yotta/internal/services/container"
	"github.com/yottaapp/yotta/internal/services/inputclip/backends"
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

// has 报 from "<nodeId>.<pin>" 出口是否有出边 (失败路由判定用).
func (idx *edgeIndex) has(from string) bool { return len(idx.out[from]) > 0 }

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
	rt          *RuntimeContext
	registry    node.RegistrySnapshot
	execNodes   map[string]*node.RegisteredNode // private immutable execution table
	compiled    *CompiledContainer              // 主图 + 所有 subgraphs 的 CompiledGraph 一次性产物.
	currentSG   *CompiledGraph                  // subgraph swap 时设, runRegionBody 用来识 entry/output marker.
	nodesByID   map[string]*container.GraphNode
	edges       *edgeIndex
	dataEdges   *dataEdgeIndex
	state       *ExecState
	stopwatches *stopwatchTable

	// execOutputs per-run held output 缓存: exec 节点 fire 某出口时, 该出口 OutputData 每个字段
	// 写进 execOutputs["<nodeID>.<field>"]; 下游数据线经 pullDataPin 任意距离直连读 (免 GetVar、
	// 免紧邻). 键全局唯一 (node ID 含随机后缀), 主图/子图/listener 共用一张. per-run 生命周期 =
	// runner 实例 (NewContainerRunner 起一张新的, 一次 Run 用完即随实例释放).
	execOutputs map[string]any

	// bundle 是 node.ServiceBundle (LogService / VarStore / VisionService 等 8 个 adapter),
	// execNode 走 node.RunNode dispatch 时消费. 默认 Log 是 zerolog.Nop, main.go 启动后
	// SetLogger 注入真 logger.
	//
	// per-tick snapshot 不是 instance 字段, 而是 ctx (tickCtxKey) — dispatchInRegion 入口
	// withTickSnapshot 写, bundle.Snapshot 闭包从 ctx 读, per-goroutine 独立.
	bundle node.ServiceBundle

	queue          []ExecToken
	runtimeStarted bool

	debugCaptureNodeID string
	debugLastResult    DebugStepResult
}

// DebugStepResult summarizes one outer runtime token dispatch.
type DebugStepResult struct {
	NodeID     string
	NodeKind   string
	InPin      string
	Exit       string
	Output     map[string]any
	Downstream []ExecToken
	Finished   bool
}

func NewContainerRunner(rt *RuntimeContext) *ContainerRunner {
	return NewContainerRunnerWithRegistry(rt, node.DefaultRegistrySnapshot())
}

// NewContainerRunnerWithRegistry assembles a runner against a fixed node
// registry generation. Later default registrations cannot change a live run.
func NewContainerRunnerWithRegistry(rt *RuntimeContext, registryReader node.RegistryReader) *ContainerRunner {
	registry := node.SnapshotRegistry(registryReader)
	execNodes := make(map[string]*node.RegisteredNode)
	for _, registered := range registry.All() {
		execNodes[registered.Spec.Kind] = registered
	}
	// 防御性 normalize — 兜底 in-memory 构造 (test fixture / 工具脚本) 没走 Store 路径的
	// container/子图, 保证 sg.Entry / OutputPins[*].NodeID 不空. Store-loaded 已 normalize 过, 幂等.
	rt.Container.Normalize()
	for i := range rt.Subgraphs {
		container.NormalizeSubgraph(&rt.Subgraphs[i])
	}
	cc := CompileContainerWithRegistry(registry, rt.Container, rt.Subgraphs)
	// PlayClip 回放 target (rt.MouseCounts360) 用主图 MouseCalibration 节点值 — 容器自包含,
	// 跟 MouseMoveRel 的 state.CalibCounts 同源 (都来自 snapshotMainCalibCounts). 无节点 (=0)
	// 时保留构造期传入的 settings 本机值兜底.
	if cc.MainCalibCounts > 0 {
		rt.MouseCounts360 = cc.MainCalibCounts
	}
	r := &ContainerRunner{
		rt:          rt,
		registry:    registry,
		execNodes:   execNodes,
		compiled:    cc,
		nodesByID:   cc.Main.NodesByID,
		edges:       cc.Main.Edges,
		dataEdges:   cc.Main.DataEdges,
		stopwatches: newStopwatchTable(),
		execOutputs: map[string]any{},
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
	r.bundle.Registry = registry
	// 脚本调子图 (node.SubgraphCaller) — runner 自身实现, 闭住本实例的 swap 字段/frame 栈.
	r.bundle.Subgraphs = r
	return r
}

func (r *ContainerRunner) registeredNode(kind string) (*node.RegisteredNode, bool) {
	registered, ok := r.execNodes[kind]
	return registered, ok
}

// SetLogger 替换 bundle 里的 LogService 为真 zerolog logger.
// main.go runFunc 在 NewContainerRunner 后调一次.
func (r *ContainerRunner) SetLogger(log zerolog.Logger) {
	r.bundle.Log = NewLogAdapter(log)
}

// SetAIProvider 注入 AI 节点用的按连接缓存 Provider 服务(进程级单例, 从 settings 解析)。
// main.go runFunc 在 NewContainerRunner 后调一次; 不注入则 ctx.Services().AI 为 nil(无 AI 节点的图不受影响)。
func (r *ContainerRunner) SetAIProvider(p node.AIProviderService) {
	r.bundle.AI = p
}

// Bundle 返当前 ContainerRunner 持有的 ServiceBundle (dispatch 用).
func (r *ContainerRunner) Bundle() node.ServiceBundle { return r.bundle }

// ExecOutputs 返 held-output 缓存的浅拷贝 (键 "<nodeID>.<field>").
// MCP run_node 跑完单节点后据此收割节点输出 (见 docs/held-exec-outputs).
func (r *ContainerRunner) ExecOutputs() map[string]any {
	out := make(map[string]any, len(r.execOutputs))
	for k, v := range r.execOutputs {
		out[k] = v
	}
	return out
}

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

// StartRuntime initializes runtime backends for either normal run or debug.
func (r *ContainerRunner) StartRuntime(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if r.runtimeStarted {
		return nil
	}
	if err := r.setupRuntime(); err != nil {
		return err
	}
	r.runtimeStarted = true
	return nil
}

// StopRuntime releases runtime resources initialized by StartRuntime.
func (r *ContainerRunner) StopRuntime() {
	if !r.runtimeStarted {
		return
	}
	r.teardownRuntime()
	r.runtimeStarted = false
	if r.rt.Emit != nil {
		r.rt.Emit("container:node-dump-flush", map[string]any{"containerId": r.rt.Container.ID})
	}
}

// SeedFromEntry queues tokens produced by Start.Done.
func (r *ContainerRunner) SeedFromEntry() error {
	startNode := r.findStart()
	if startNode == nil {
		return errors.New("container: no Start node")
	}
	r.queue = append(r.queue[:0], r.edges.next(startNode.ID+".Done", nil)...)
	return r.skipDisabledQueueHeads()
}

// SeedFromNode queues the selected node's first exec input.
func (r *ContainerRunner) SeedFromNode(nodeID string) error {
	n, ok := r.nodesByID[nodeID]
	if !ok {
		return fmt.Errorf("debug_invalid_start_node: node %q not found", nodeID)
	}
	rn, ok := r.registeredNode(n.Kind)
	if !ok {
		return fmt.Errorf("debug_invalid_start_node: kind %q not registered", n.Kind)
	}
	for _, in := range rn.Spec.Inputs {
		if in.Type == "Exec" {
			r.queue = append(r.queue[:0], ExecToken{NodeID: n.ID, InPin: in.Name})
			return r.skipDisabledQueueHeads()
		}
	}
	return fmt.Errorf("debug_start_node_not_executable: node %q has no exec input", nodeID)
}

func (r *ContainerRunner) skipDisabledQueueHeads() error {
	const maxDisabledHops = 10000
	for hops := 0; len(r.queue) > 0; hops++ {
		if hops >= maxDisabledHops {
			return fmt.Errorf("debug_disabled_passthrough_cycle: exceeded %d disabled hops", maxDisabledHops)
		}
		tok := r.queue[0]
		n, ok := r.nodesByID[tok.NodeID]
		if !ok || n == nil || !n.Disabled {
			return nil
		}
		r.queue = r.queue[1:]
		out, err := r.passthroughDisabled(n, tok)
		if err != nil {
			return err
		}
		r.queue = append(r.queue, out...)
	}
	return nil
}

// QueueSnapshot returns a shallow copy of the queued outer runtime tokens.
func (r *ContainerRunner) QueueSnapshot() []ExecToken {
	out := make([]ExecToken, len(r.queue))
	copy(out, r.queue)
	for i := range out {
		out[i].LoopStack = copyLoops(out[i].LoopStack)
	}
	return out
}

// VarSnapshot returns a shallow copy of runtime variables.
func (r *ContainerRunner) VarSnapshot() map[string]any {
	vars := r.rt.Vars()
	out := make(map[string]any, len(vars))
	for k, v := range vars {
		out[k] = v
	}
	return out
}

func (r *ContainerRunner) NodeKind(nodeID string) string {
	if n, ok := r.nodesByID[nodeID]; ok && n != nil {
		return n.Kind
	}
	return ""
}

// StepOnce executes exactly one queued outer runtime token.
func (r *ContainerRunner) StepOnce(ctx context.Context) (DebugStepResult, error) {
	if err := ctx.Err(); err != nil {
		return DebugStepResult{Finished: len(r.queue) == 0}, err
	}
	if err := r.skipDisabledQueueHeads(); err != nil {
		return DebugStepResult{Finished: len(r.queue) == 0}, err
	}
	if len(r.queue) == 0 {
		return DebugStepResult{Finished: true}, nil
	}
	tok := r.queue[0]
	r.queue = r.queue[1:]

	n, ok := r.nodesByID[tok.NodeID]
	if !ok {
		return DebugStepResult{NodeID: tok.NodeID, InPin: tok.InPin, Finished: len(r.queue) == 0},
			fmt.Errorf("container: token references unknown node %q", tok.NodeID)
	}

	if r.rt.Emit != nil {
		r.rt.Emit("container:node-enter", map[string]any{
			"containerId": r.rt.Container.ID,
			"nodeId":      n.ID,
			"nodeKind":    n.Kind,
		})
	}

	r.debugCaptureNodeID = tok.NodeID
	r.debugLastResult = DebugStepResult{NodeID: n.ID, NodeKind: n.Kind, InPin: tok.InPin}
	if n.Kind == "Loop" && !n.Disabled {
		res, err := r.debugEnterLoop(n, tok)
		r.debugCaptureNodeID = ""
		r.debugLastResult = DebugStepResult{}
		return res, err
	}
	out, err := r.execNode(ctx, n, tok)
	res := r.debugLastResult
	r.debugCaptureNodeID = ""
	r.debugLastResult = DebugStepResult{}

	if err != nil {
		if len(tok.LoopStack) > 0 && (control.IsBreakRequested(err) || control.IsContinueRequested(err)) {
			if control.IsBreakRequested(err) {
				err = r.debugBreakLoop(tok.LoopStack)
			} else {
				err = r.debugContinueLoop(tok.LoopStack)
			}
			res.Finished = len(r.queue) == 0
			return res, err
		}
		if errors.Is(err, errStopRun) {
			res.Finished = true
			return res, nil
		}
		if _, wrapped := r.checkSentinelLeak(n, err); wrapped != err {
			return res, wrapped
		}
		return res, err
	}
	r.queue = append(r.queue, out...)
	if err := r.skipDisabledQueueHeads(); err != nil {
		return res, err
	}
	if err := r.debugAdvanceCompletedLoops(tok.LoopStack); err != nil {
		return res, err
	}
	res.Downstream = copyTokens(out)
	res.Finished = len(r.queue) == 0
	return res, nil
}

func (r *ContainerRunner) debugEnterLoop(node *container.GraphNode, tok ExecToken) (DebugStepResult, error) {
	mode := container.PinString(node, "Mode")
	if mode == "" {
		mode = "count"
	}
	if mode != "count" && mode != "forever" {
		return DebugStepResult{NodeID: node.ID, NodeKind: node.Kind, InPin: tok.InPin, Finished: len(r.queue) == 0},
			fmt.Errorf("loop: unknown mode %q", mode)
	}
	if mode == "count" {
		count, ok := container.PinInt(node, "Count")
		if !ok {
			count = 10
		}
		if count <= 0 {
			out := r.edges.next(node.ID+".Done", tok.LoopStack)
			r.queue = append(out, r.queue...)
			if err := r.skipDisabledQueueHeads(); err != nil {
				return DebugStepResult{}, err
			}
			return DebugStepResult{
				NodeID:     node.ID,
				NodeKind:   node.Kind,
				InPin:      tok.InPin,
				Exit:       "Done",
				Downstream: copyTokens(out),
				Finished:   len(r.queue) == 0,
			}, nil
		}
	}
	res, frameStack, err := r.debugStartLoopIteration(node, tok.LoopStack, 0, tok.InPin)
	if err != nil {
		return res, err
	}
	if err := r.debugAdvanceCompletedLoops(frameStack); err != nil {
		return res, err
	}
	res.Finished = len(r.queue) == 0
	return res, nil
}

func (r *ContainerRunner) debugStartLoopIteration(node *container.GraphNode, parent []*LoopFrame, iter int64, inPin string) (DebugStepResult, []*LoopFrame, error) {
	frame := &LoopFrame{LoopNodeID: node.ID, Iter: iter}
	stack := append(copyLoops(parent), frame)
	data := map[string]any{"Index": float64(iter)}
	r.applyCaptures(node, data)
	r.captureExecOutputs(node, data)
	seeds := r.edges.nextWithData(node.ID+".Body", parent, data)
	for i := range seeds {
		seeds[i].LoopStack = copyLoops(stack)
	}
	r.queue = append(seeds, r.queue...)
	if len(seeds) == 0 && container.PinString(node, "Mode") == "forever" {
		return DebugStepResult{}, stack, fmt.Errorf("debug_loop_empty_forever_body: loop %q has no Body downstream", node.ID)
	}
	if err := r.skipDisabledQueueHeads(); err != nil {
		return DebugStepResult{}, stack, err
	}
	return DebugStepResult{
		NodeID:     node.ID,
		NodeKind:   node.Kind,
		InPin:      inPin,
		Exit:       "Body",
		Output:     copyAnyMap(data),
		Downstream: copyTokens(seeds),
		Finished:   len(r.queue) == 0,
	}, stack, nil
}

func (r *ContainerRunner) debugAdvanceCompletedLoops(stack []*LoopFrame) error {
	for len(stack) > 0 {
		if r.queueHasLoopPrefix(stack) {
			return nil
		}
		frame := stack[len(stack)-1]
		parent := stack[:len(stack)-1]
		node := r.nodesByID[frame.LoopNodeID]
		if node == nil {
			return fmt.Errorf("debug_loop_frame_missing_node: %s", frame.LoopNodeID)
		}
		nextIter := frame.Iter + 1
		if r.debugLoopHasIteration(node, nextIter) {
			_, nextStack, err := r.debugStartLoopIteration(node, parent, nextIter, "In")
			if err != nil {
				return err
			}
			stack = nextStack
			continue
		}
		done := r.edges.next(node.ID+".Done", parent)
		r.queue = append(done, r.queue...)
		if err := r.skipDisabledQueueHeads(); err != nil {
			return err
		}
		stack = parent
	}
	return nil
}

func (r *ContainerRunner) debugBreakLoop(stack []*LoopFrame) error {
	frame := stack[len(stack)-1]
	parent := stack[:len(stack)-1]
	r.removeQueuedLoopPrefix(stack)
	node := r.nodesByID[frame.LoopNodeID]
	if node == nil {
		return fmt.Errorf("debug_loop_frame_missing_node: %s", frame.LoopNodeID)
	}
	done := r.edges.next(node.ID+".Done", parent)
	r.queue = append(done, r.queue...)
	if err := r.skipDisabledQueueHeads(); err != nil {
		return err
	}
	return r.debugAdvanceCompletedLoops(parent)
}

func (r *ContainerRunner) debugContinueLoop(stack []*LoopFrame) error {
	frame := stack[len(stack)-1]
	parent := stack[:len(stack)-1]
	r.removeQueuedLoopPrefix(stack)
	node := r.nodesByID[frame.LoopNodeID]
	if node == nil {
		return fmt.Errorf("debug_loop_frame_missing_node: %s", frame.LoopNodeID)
	}
	nextIter := frame.Iter + 1
	if r.debugLoopHasIteration(node, nextIter) {
		_, nextStack, err := r.debugStartLoopIteration(node, parent, nextIter, "In")
		if err != nil {
			return err
		}
		return r.debugAdvanceCompletedLoops(nextStack)
	}
	done := r.edges.next(node.ID+".Done", parent)
	r.queue = append(done, r.queue...)
	if err := r.skipDisabledQueueHeads(); err != nil {
		return err
	}
	return r.debugAdvanceCompletedLoops(parent)
}

func (r *ContainerRunner) debugLoopHasIteration(node *container.GraphNode, iter int64) bool {
	mode := container.PinString(node, "Mode")
	if mode == "" {
		mode = "count"
	}
	if mode == "forever" {
		return true
	}
	count, ok := container.PinInt(node, "Count")
	if !ok {
		count = 10
	}
	return iter < int64(count)
}

func (r *ContainerRunner) queueHasLoopPrefix(prefix []*LoopFrame) bool {
	for _, tok := range r.queue {
		if loopStackHasPrefix(tok.LoopStack, prefix) {
			return true
		}
	}
	return false
}

func (r *ContainerRunner) removeQueuedLoopPrefix(prefix []*LoopFrame) {
	out := r.queue[:0]
	for _, tok := range r.queue {
		if loopStackHasPrefix(tok.LoopStack, prefix) {
			continue
		}
		out = append(out, tok)
	}
	r.queue = out
}

func loopStackHasPrefix(stack, prefix []*LoopFrame) bool {
	if len(prefix) == 0 {
		return true
	}
	if len(stack) < len(prefix) {
		return false
	}
	for i := range prefix {
		if stack[i] == nil || prefix[i] == nil {
			return false
		}
		if stack[i].LoopNodeID != prefix[i].LoopNodeID || stack[i].Iter != prefix[i].Iter {
			return false
		}
	}
	return true
}

func copyTokens(src []ExecToken) []ExecToken {
	out := make([]ExecToken, len(src))
	copy(out, src)
	for i := range out {
		out[i].LoopStack = copyLoops(out[i].LoopStack)
	}
	return out
}

func (r *ContainerRunner) recordDebugRoute(node *container.GraphNode, exit string, output map[string]any) {
	if r.debugCaptureNodeID == "" || r.debugCaptureNodeID != node.ID {
		return
	}
	r.debugLastResult.NodeID = node.ID
	r.debugLastResult.NodeKind = node.Kind
	r.debugLastResult.Exit = exit
	r.debugLastResult.Output = copyAnyMap(output)
}

func copyAnyMap(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

// Run 启动 token dispatch：找 Start → 入队 → 主循环。
// 同时为每个 listener-driven 节点 (EventTick) 起后台 listener goroutine。
// 返回时机:
//   - 无 listener: 主流程 (queue 清空) 即返回。
//   - 有 listener: 等外层 ctx 取消/超时 → defer cancelChild → listenerWG.Wait → 返回
//     (后台 listener 要活到容器被外部停掉, 不能因主流程跑完就被秒杀)。
func (r *ContainerRunner) Run(ctx context.Context) error {
	if err := r.StartRuntime(ctx); err != nil {
		return err
	}
	defer r.StopRuntime() // LIFO 内部顺序保证 ReleaseAll → Input.Close → Capture.Close

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

	// Start 输出 → 入第一批 token
	if err := r.SeedFromEntry(); err != nil {
		return err
	}

	dispatchCount := 0
	for len(r.queue) > 0 {
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
		if _, err := r.StepOnce(ctx); err != nil {
			return err
		}
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
// Variables / params are routed via GetVar / GetParam nodes wired through
// data edges, NOT through env paths in expressions.

// configString 读 pin 字面量字符串 (literal 优先 + 顶层 config fallback, 镜像 newInputs)。
// key 必须是规范 PascalCase Spec.Input 名。
func configString(node *container.GraphNode, key string) string {
	return container.PinString(node, key)
}

// ----------------------------------------------------------------------------
// runtime bootstrap: setupRuntime / teardownRuntime / Win32 backend bootstrap.
// ----------------------------------------------------------------------------

// setupRuntime 按容器级配置建 Win32 controller provider → populate rt.
// 不在启动期解析窗口 — 窗口由 Win32WindowTarget.Run 运行时解析 (SetActive).
// Android/Browser-only target graph 不需要 Win32 backend, controller factory 会按 active target 构造。
// 幂等: 测试预设过 controller provider 就跳过.
func (r *ContainerRunner) setupRuntime() error {
	if r.rt.win32Provider != nil || r.rt.win32Factory != nil {
		return nil
	}
	if !r.containerNeedsWin32Backends(r.rt.Container, r.rt.Subgraphs) {
		return nil
	}

	if r.rt.InputBackend == nil {
		r.rt.InputBackend = backends.NewHybridBackend(func() uintptr {
			h, _ := r.rt.ActiveHWND()
			return h
		})
	}

	provider, err := newWin32ControllerProvider(r.rt)
	if err != nil {
		return err
	}
	r.rt.win32Provider = provider
	return nil
}

// teardownRuntime 关闭 Win32 controller provider，由 provider 维护后端释放顺序。
func (r *ContainerRunner) teardownRuntime() {
	if r.rt.win32Provider != nil {
		_ = r.rt.win32Provider.Close()
		r.rt.win32Provider = nil
	}
}

// containerNeedsWin32Backends 判断本容器是否需要初始化 Win32 input/capture backend。
// Direct NeedsWindow 节点一定需要。NeedsTarget 节点只有在图使用 Win32WindowTarget 或没有显式
// target selection 时才按 Windows 默认路径初始化；Android/Browser-only 图不初始化 Win32 backend。
func (r *ContainerRunner) containerNeedsWin32Backends(c *container.Container, sgs []container.Subgraph) bool {
	if r.graphHasWindowNode(c.Graph.Nodes) {
		return true
	}
	for i := range sgs {
		if r.graphHasWindowNode(sgs[i].Graph.Nodes) {
			return true
		}
	}
	if !containerHasTargetNode(c, sgs) {
		return r.containerHasTargetAwareNode(c, sgs)
	}
	return containerHasTargetKind(c, sgs, "Win32WindowTarget")
}

func (r *ContainerRunner) graphHasWindowNode(nodes []container.GraphNode) bool {
	for i := range nodes {
		if rn, ok := r.registeredNode(nodes[i].Kind); ok && rn.Spec.NeedsWindow {
			return true
		}
	}
	return false
}

func (r *ContainerRunner) containerHasTargetAwareNode(c *container.Container, sgs []container.Subgraph) bool {
	if r.graphHasTargetAwareNode(c.Graph.Nodes) {
		return true
	}
	for i := range sgs {
		if r.graphHasTargetAwareNode(sgs[i].Graph.Nodes) {
			return true
		}
	}
	return false
}

func (r *ContainerRunner) graphHasTargetAwareNode(nodes []container.GraphNode) bool {
	for i := range nodes {
		if rn, ok := r.registeredNode(nodes[i].Kind); ok && rn.Spec.NeedsTarget {
			return true
		}
	}
	return false
}

func containerHasTargetNode(c *container.Container, sgs []container.Subgraph) bool {
	if graphHasTargetNode(c.Graph.Nodes) {
		return true
	}
	for i := range sgs {
		if graphHasTargetNode(sgs[i].Graph.Nodes) {
			return true
		}
	}
	return false
}

func graphHasTargetNode(nodes []container.GraphNode) bool {
	for i := range nodes {
		if isTargetNodeKind(nodes[i].Kind) {
			return true
		}
	}
	return false
}

func containerHasTargetKind(c *container.Container, sgs []container.Subgraph, kind string) bool {
	if graphHasTargetKind(c.Graph.Nodes, kind) {
		return true
	}
	for i := range sgs {
		if graphHasTargetKind(sgs[i].Graph.Nodes, kind) {
			return true
		}
	}
	return false
}

func graphHasTargetKind(nodes []container.GraphNode, kind string) bool {
	for i := range nodes {
		if nodes[i].Kind == kind {
			return true
		}
	}
	return false
}

func isTargetNodeKind(kind string) bool {
	switch kind {
	case "Win32WindowTarget", "AndroidTarget":
		return true
	default:
		return false
	}
}

// isListenerDriven — kind 是否走后台 listener goroutine (无 exec-in, 不进主 dispatch)。
// 新增同类事件节点在这里加 kind 即可, 无需动 spawn 循环。
func isListenerDriven(kind string) bool {
	return kind == "EventTick"
}
