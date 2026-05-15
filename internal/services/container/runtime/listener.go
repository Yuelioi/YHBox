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
	l := &EventListener{
		runner:          r,
		node:            n,
		kind:            configString(n, "kind"),
		template:        configString(n, "template"),
		threshold:       r.configFloat(n, "threshold", 0.85),
		pollIntervalMs:  int(r.configFloat(n, "pollIntervalMs", 100)),
		maxConcurrent:   int(r.configFloat(n, "maxConcurrent", 1)),
		retriggerPolicy: configString(n, "retriggerPolicy"),
		cooldownMs:      int(r.configFloat(n, "cooldownMs", 0)),
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
		found, _, _, err := l.runner.rt.Matcher.Detect(ctx, l.template, l.threshold, nil)
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
		// seed token = OnEvent.out 后裔
		seeds := l.runner.edges.next(l.node.ID+".out", nil)
		_ = l.runner.runSubFlow(subCtx, seeds)
	}()
}
