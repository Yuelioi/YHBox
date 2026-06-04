package runtime

import (
	"context"
	"image"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"

	"yotta/internal/services/container"
)

// EventListener 单 OnEvent 节点配套的监听 goroutine。
//
// 支持两种 kind: template_appeared (OnEvent, 周期性 Detect 命中即触发) 与 tick
// (EventTick, 每 IntervalMs 无条件触发, Out 带 DeltaMs)。命中 → spawn 子 runner 跑 Out
// pin 后裔子图。maxConcurrent / retriggerPolicy 决定 spawn 节奏。
// cooldownMs 仅 OnEvent 适用 (tick 路径不读、恒为 0; IntervalMs 本身即频率控制)。
type EventListener struct {
	runner *ContainerRunner
	node   *container.GraphNode

	// homeEdges / homeNodesByID: 主图视图快照，listener 自家持有避免跟主 dispatch 抢
	// runner.edges / runner.nodesByID（主 dispatch 进 subgraph 时会改写那两字段，
	// 此处使用 runner 的字段会产生 data race + 行为错乱）。
	// listener 永远跑主图视图——OnEvent 节点只允许在主图，subgraph dispatch 由主流程处理。
	homeEdges     *edgeIndex
	homeNodesByID map[string]*container.GraphNode

	// 配置（启动时一次性 eval；运行期不重算）
	kind            string
	templates       []string
	matchMode       string // any | all
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
	// r.compiled.Main 是 immutable 预编译产物 (CompileContainer 后从不写), 直接复用.
	// runner.edges/nodesByID 是 swap 目标不能直接读, 但 r.compiled.Main 安全.
	l := &EventListener{
		runner:        r,
		node:          n,
		homeEdges:     r.compiled.Main.Edges,
		homeNodesByID: r.compiled.Main.NodesByID,
		queueSig:      make(chan struct{}, 16),
	}
	switch n.Kind {
	case "EventTick":
		l.kind = "tick"
		l.pollIntervalMs = int(r.pullNumber(context.Background(), n, "IntervalMs", 100))
		l.maxConcurrent = int(r.pullNumber(context.Background(), n, "MaxConcurrent", 1))
		l.retriggerPolicy = configString(n, "RetriggerPolicy")
		if l.pollIntervalMs < 1 {
			l.pollIntervalMs = 1 // 下限保护; 实际分辨率看 OS
		}
		// DeltaMs 基准: lastFired 必须初始化为启动时刻, 否则首次后 delta 从零值算出天文数字。
		l.lastFired.Store(time.Now().UnixNano())
	default: // OnEvent —— 原有字段读取
		l.kind = configString(n, "Kind")
		if l.kind == "" {
			l.kind = "template_appeared"
		}
		l.templates = configStringList(n, "Templates")
		l.matchMode = configString(n, "MatchMode")
		// thresholds via data-in pin. listener init 不在 dispatch tick scope, 传
		// context.Background() — config 走 literal/常量 不依赖 frozen Vars, 无 tick 行为一致.
		l.threshold = r.pullNumber(context.Background(), n, "Threshold", 0.85)
		l.pollIntervalMs = int(r.pullNumber(context.Background(), n, "PollIntervalMs", 100))
		l.maxConcurrent = int(r.pullNumber(context.Background(), n, "MaxConcurrent", 1))
		l.retriggerPolicy = configString(n, "RetriggerPolicy")
		l.cooldownMs = int(r.pullNumber(context.Background(), n, "CooldownMs", 0))
		if l.pollIntervalMs <= 0 {
			l.pollIntervalMs = 100
		}
	}
	if l.maxConcurrent <= 0 {
		l.maxConcurrent = 1
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
	switch l.kind {
	case "template_appeared", "tick":
	default:
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
		if !l.shouldFire(ctx) {
			continue
		}
		l.handleFire(ctx)
	}
}

// shouldFire — 触发源「要不要 fire」判定 (FireSource seam 第一步): tick 永真; template_appeared 走 detect。
// 注意: 这只是触发判定; 「fire 时注入什么数据」(tick 的 DeltaMs) 在 spawn() 里另有一处 kind 分流。
// 未来加 EventTimer/EventCron 时, shouldFire 与 spawn 两处都要顾到, 别只改这里。
func (l *EventListener) shouldFire(ctx context.Context) bool {
	if l.kind == "tick" {
		return true
	}
	return l.detectFired(ctx)
}

// detectFired 单帧多模板判定. matchMode="all": 全部命中才 fire; 否则 "any": 任一命中即 fire.
// Detect error 视作该 key miss (跟旧单值 `err != nil || !found` 行为一致).
func (l *EventListener) detectFired(ctx context.Context) bool {
	if len(l.templates) == 0 {
		return false
	}
	rt := l.runner.rt
	var frame *image.RGBA
	if rt.Capture != nil {
		if h, err := rt.ActiveHWND(); err == nil {
			if f, err := rt.CaptureFrameCached(h); err == nil {
				frame = f
			}
		}
	}
	if l.matchMode == "all" {
		for _, key := range l.templates {
			found, _, _, _, err := rt.Matcher.Detect(ctx, rt.Container.ID, frame, key, l.threshold, nil)
			if err != nil || !found {
				return false
			}
		}
		return true
	}
	for _, key := range l.templates {
		found, _, _, _, err := rt.Matcher.Detect(ctx, rt.Container.ID, frame, key, l.threshold, nil)
		if err == nil && found {
			return true
		}
	}
	return false
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

// makeSubRunner 给一次 spawn 用的独立 dispatch 实例。
//
// 独立: edges / nodesByID / state / bundle — 避免跟主 dispatch 抢 swap 字段 + 不污染
// frame 栈; 独立 bundle 的 stateGetter 闭 sub.state, 让 Vars (scope=local/auto) / Params
// 落到子流程自己的 frame 栈, 不 stomp 主 runner.state.
// 共享: rt / compiled / dataEdges / stopwatches — compiled/dataEdges 是 immutable 预编译
// 产物; stopwatches 是 *stopwatchTable 全 container 共享; rt.Vars (容器级 global 变量,
// 有锁) 仍跨 flow 共享 — global scope 走 rt 不走 frame.
func (l *EventListener) makeSubRunner() *ContainerRunner {
	sub := &ContainerRunner{
		rt:          l.runner.rt,
		compiled:    l.runner.compiled,
		nodesByID:   l.homeNodesByID,
		edges:       l.homeEdges,
		dataEdges:   l.runner.dataEdges,
		state:       NewExecState(l.runner.rt.Container.ID, l.runner.state.CalibCounts),
		stopwatches: l.runner.stopwatches,
	}
	sub.bundle = NewServiceBundleFor(
		l.runner.rt,
		l.runner.stopwatches,
		zerolog.Nop(),
		func() *ExecState { return sub.state },
	)
	// 主 bundle 启动期被 SetLogger 注入真 logger; 子流程沿用同一 Log adapter.
	sub.bundle.Log = l.runner.bundle.Log
	return sub
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
	now := time.Now()
	var deltaMs float64
	if l.kind == "tick" {
		deltaMs = float64(now.Sub(time.Unix(0, l.lastFired.Load())).Milliseconds())
	}
	l.lastFired.Store(now.UnixNano())

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
		// Listener 子流程跑一个独立的 ContainerRunner 实例（共享 rt，独立 edges/nodesByID/
		// state/bundle）避免跟主 dispatch 的 r.edges 抢（主 dispatch 进 subgraph 时会改写那
		// 两字段产生数据竞争和行为错乱）。独立 ExecState + 独立 bundle 让 listener 子流程入
		// 嵌套子图不污染主 dispatch 的 frame 栈, 且 Vars(local/auto)/Params 隔离到 sub.state。
		// rt.Vars（容器级 global 变量, 有锁）跨 flow 共享 OK — global scope 仍走 rt。
		sub := l.makeSubRunner()
		// pin 名必须等同 OnEvent/EventTick Spec 的 OutputSpec.Name ("Out") —— 前端 Handle id
		// 用 pin 名原样, 边存 "<id>.Out"; 小写 .out 历史 bug 会 seed 不到下游。
		var seeds []ExecToken
		if l.kind == "tick" {
			seeds = sub.edges.nextWithData(l.node.ID+".Out", nil,
				map[string]any{"DeltaMs": deltaMs}) // 注入侧显式 float64
		} else {
			seeds = sub.edges.next(l.node.ID+".Out", nil)
		}
		_ = sub.runSubFlow(subCtx, seeds)
	}()
}
