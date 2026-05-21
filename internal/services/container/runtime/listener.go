package runtime

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"yhbox/internal/services/container"
)

// EventListener 单 OnEvent 节点配套的监听 goroutine。
//
// v1 只支持 kind=template_appeared：周期性 Detect，命中 → spawn 子 runner 跑 out
// pin 后裔子图。maxConcurrent / retriggerPolicy / cooldownMs 决定 spawn 节奏。
type EventListener struct {
	runner *ContainerRunner
	node   *container.GraphNode

	// homeEdges / homeNodesByID: 主图视图快照，listener 自家持有避免跟主 dispatch 抢
	// runner.edges / runner.nodesByID（主 dispatch 进 subgraph 时会改写那两字段，
	// 此处使用 runner 的字段会产生 data race + 行为错乱，Gemini review B-2 修复）。
	// listener 永远跑主图视图——OnEvent 节点只允许在主图，subgraph dispatch 由主流程处理。
	homeEdges     *edgeIndex
	homeNodesByID map[string]*container.GraphNode

	// 配置（启动时一次性 eval；运行期不重算）
	kind            string
	template        string
	threshold       float64
	pollIntervalMs  int
	maxConcurrent   int
	retriggerPolicy string // drop | queue | restart
	cooldownMs      int

	activeSubs atomic.Int32
	lastFired  atomic.Int64 // unix nano

	// queue 模式：用 buffered channel 当 FIFO 队列（cap=16）。
	// listener.run 写、spawn 完成时的 defer 读，避免 defer 内递归调 spawn 把栈炸了。
	queueSig chan struct{}

	// restart policy 用：当前正在跑的子 ctx 取消。
	// 跟 spawn 的 activeSubs.Add(1) 一起持 mu 锁，避免并发 spawn stomp 上一个 cancel。
	mu               sync.Mutex
	currentSubCancel context.CancelFunc
}

func newEventListener(r *ContainerRunner, n *container.GraphNode) *EventListener {
	// 注册时一次性快照主图视图。runner.edges / runner.nodesByID 后续会被
	// 主 dispatch 在进入 subgraph 时改写——若 listener 直接读那两字段会产生数据竞争。
	homeNodes := make(map[string]*container.GraphNode, len(r.rt.Container.Graph.Nodes))
	for i := range r.rt.Container.Graph.Nodes {
		nn := &r.rt.Container.Graph.Nodes[i]
		homeNodes[nn.ID] = nn
	}
	l := &EventListener{
		runner:          r,
		node:            n,
		homeEdges:       buildEdgeIndex(r.rt.Container.Graph),
		homeNodesByID:   homeNodes,
		kind:            configString(n, "kind"),
		template:        configString(n, "template"),
		// v4: thresholds via data-in pin.
		threshold:       r.pullNumber(n, "threshold", 0.85),
		pollIntervalMs:  int(r.pullNumber(n, "pollIntervalMs", 100)),
		maxConcurrent:   int(r.pullNumber(n, "maxConcurrent", 1)),
		retriggerPolicy: configString(n, "retriggerPolicy"),
		cooldownMs:      int(r.pullNumber(n, "cooldownMs", 0)),
		queueSig:        make(chan struct{}, 16),
	}
	if l.kind == "" {
		l.kind = "template_appeared"
	}
	if l.maxConcurrent <= 0 {
		l.maxConcurrent = 1
	}
	if l.pollIntervalMs <= 0 {
		l.pollIntervalMs = 100
	}
	if l.retriggerPolicy == "" {
		l.retriggerPolicy = "drop"
	}
	// restart 语义本身要求最多 1 个 sub 在跑
	if l.retriggerPolicy == "restart" {
		l.maxConcurrent = 1
	}
	return l
}

// run 长跑 goroutine：ctx cancel 即退。除了周期性 Detect，还要消费 queue 信号
// （queue 模式下 spawn 完成后 push 信号让我们再 spawn 下一个）。
func (l *EventListener) run(ctx context.Context) {
	if l.kind != "template_appeared" {
		// v1 不支持别的 kind
		return
	}
	ticker := time.NewTicker(time.Duration(l.pollIntervalMs) * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-l.queueSig:
			// queue 模式：上一个 sub 完成，看看 activeSubs 是否还有余位。
			if l.activeSubs.Load() < int32(l.maxConcurrent) {
				l.spawn(ctx)
			}
			continue
		case <-ticker.C:
		}
		// cooldown
		if l.cooldownMs > 0 {
			last := time.Unix(0, l.lastFired.Load())
			if time.Since(last) < time.Duration(l.cooldownMs)*time.Millisecond {
				continue
			}
		}
		// Detect
		found, _, _, err := l.runner.rt.Matcher.Detect(ctx, l.runner.rt.Container.ID, l.runner.rt.Window.HWND, l.template, l.threshold, nil)
		if err != nil || !found {
			continue
		}
		l.handleFire(ctx)
	}
}

func (l *EventListener) handleFire(ctx context.Context) {
	switch l.retriggerPolicy {
	case "drop":
		if l.activeSubs.Load() >= int32(l.maxConcurrent) {
			return
		}
		l.spawn(ctx)
	case "queue":
		if l.activeSubs.Load() >= int32(l.maxConcurrent) {
			// 排队：buffered channel 满了就丢（cap=16）
			select {
			case l.queueSig <- struct{}{}:
			default:
			}
			return
		}
		l.spawn(ctx)
	case "restart":
		// 持锁取出旧 cancel + 写新 cancel，避免并发 stomp
		l.mu.Lock()
		old := l.currentSubCancel
		l.mu.Unlock()
		if old != nil {
			old()
		}
		l.spawn(ctx)
	default:
		// 同 drop
		if l.activeSubs.Load() >= int32(l.maxConcurrent) {
			return
		}
		l.spawn(ctx)
	}
}

// makeSubRunner 给一次 spawn 用的独立 dispatch 实例。共享 rt（含 Input/Cap/Vars 等），
// 但 edges/nodesByID/state 独立——避免跟主 dispatch 的 r.edges 抢以及 ExecState frame 栈污染。
func (l *EventListener) makeSubRunner() *ContainerRunner {
	return &ContainerRunner{
		rt:        l.runner.rt,
		nodesByID: l.homeNodesByID,
		edges:     l.homeEdges,
		state:     NewExecState(l.runner.rt.Container.ID, l.runner.state.CalibCounts),
	}
}

func (l *EventListener) spawn(parentCtx context.Context) {
	// 父 ctx 已 cancel → 不浪费 goroutine 启动
	if parentCtx.Err() != nil {
		return
	}
	subCtx, cancel := context.WithCancel(parentCtx)

	l.mu.Lock()
	if l.retriggerPolicy == "restart" {
		l.currentSubCancel = cancel
	}
	l.mu.Unlock()

	l.activeSubs.Add(1)
	l.lastFired.Store(time.Now().UnixNano())

	go func() {
		defer cancel()
		defer func() {
			l.activeSubs.Add(-1)
			// queue 模式：sub 跑完，通过 channel 通知 run() 看看队列里还有没有
			if l.retriggerPolicy == "queue" {
				// 非阻塞：满了说明 run() 还没消费完之前的信号，无所谓
				select {
				case l.queueSig <- struct{}{}:
				default:
				}
			}
		}()
		// Listener 子流程跑一个独立的 ContainerRunner 实例（共享 rt，独立 edges/nodesByID/state）
		// 避免跟主 dispatch 的 r.edges 抢（主 dispatch 进 subgraph 时会改写那两字段产生数据
		// 竞争和行为错乱，Gemini review B-2）。独立 ExecState 让 listener 子流程也能入嵌套子图
		// 而不污染主 dispatch 的 frame 栈。注意：rt.vars（容器级变量）是 rt 上的字段且有锁，跨
		// flow 共享 OK；ExecState.GlobalVars 不共享是当前节点执行器尚未接入 LocalVars/GlobalVars
		// 的现状，B-10 修完后再处理跨流程共享。
		sub := l.makeSubRunner()
		seeds := sub.edges.next(l.node.ID+".out", nil)
		_ = sub.runSubFlow(subCtx, seeds)
	}()
}
