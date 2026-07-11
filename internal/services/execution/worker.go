package execution

import (
	"context"
	"errors"
	"sync"
	"time"
)

// RunFunc 实际跑一个 target 的回调。Worker 自己不知道 container 怎么解析，
// 把回调注入进来由 main.go 绑到 container/runtime.Run(container.ID, ctx)。
type RunFunc func(ctx context.Context, target TargetRef) error

// Worker 单 goroutine 串行消费 ExecutionQueue。键鼠独占的保证根。
type Worker struct {
	queue     *ExecutionQueue
	run       RunFunc
	lifetime  context.Context
	cancelAll context.CancelFunc
	stopCh    chan struct{}
	done      chan struct{}
	wg        sync.WaitGroup
	emitRun   func(WorkerState)
	classify  ClassifyFunc

	lifecycleMu sync.Mutex
	started     bool
	stopped     bool

	mu     sync.Mutex
	active *QueuedRun // 当前在跑的 run，nil = 闲
	cancel context.CancelFunc
}

// RunValErr 是 ValidationError 的搬运 DTO。json 字段名必须与 container.ValidationError 完全一致,
// 让通道A(cause.Errors[]) 与通道B(errors[]) 的元素结构逐字段相同, FE 一套解包通吃。
type RunValErr struct {
	Severity  string         `json:"severity,omitempty"`
	Code      string         `json:"code"`
	GraphPath []string       `json:"graphPath,omitempty"`
	NodeID    string         `json:"nodeId,omitempty"`
	Params    map[string]any `json:"params,omitempty"`
}

// RunError 是 run 失败的结构化事件信封 (替代拍平的 string)。
type RunError struct {
	Message string         `json:"message,omitempty"`
	Errors  []RunValErr    `json:"errors,omitempty"`
	Code    string         `json:"code,omitempty"`
	Params  map[string]any `json:"params,omitempty"`
}

// ClassifyFunc 把 run error 分类成信封。注入 (worker 包不知道 container/apperr 类型)。
type ClassifyFunc func(error) *RunError

// WorkerState 给 UI status bar 推的事件 payload。
// Error 非空 → 最后一次 target run 出错，前端可 toast。
type WorkerState struct {
	Running   bool
	RunID     int64
	Source    TriggerSource
	Targets   []TargetRef
	TargetIdx int       // 当前在跑 targets[idx]
	Error     *RunError `json:"Error,omitempty"`
}

func NewWorker(q *ExecutionQueue, run RunFunc, emit func(WorkerState), classify ClassifyFunc) *Worker {
	lifetime, cancelAll := context.WithCancel(context.Background())
	return &Worker{
		queue:     q,
		run:       run,
		lifetime:  lifetime,
		cancelAll: cancelAll,
		stopCh:    make(chan struct{}),
		done:      make(chan struct{}),
		emitRun:   emit,
		classify:  classify,
	}
}

// Start 起后台 goroutine 不断 Pop 跑。重复或 Stop 后调用均为 no-op。
func (w *Worker) Start() {
	w.lifecycleMu.Lock()
	defer w.lifecycleMu.Unlock()
	if w.started || w.stopped {
		return
	}
	w.started = true
	w.wg.Add(1)
	go w.loop()
}

// Stop 关 stopCh + cancel 当前 run + 等 goroutine 退出。并发、重复调用幂等。
func (w *Worker) Stop() {
	_ = w.StopContext(context.Background())
}

// StopContext initiates shutdown once and waits for the worker or context.
func (w *Worker) StopContext(ctx context.Context) error {
	w.lifecycleMu.Lock()
	if !w.stopped {
		w.stopped = true
		w.cancelAll()
		close(w.stopCh)
		w.queue.Close()
		w.mu.Lock()
		if w.cancel != nil {
			w.cancel()
		}
		w.mu.Unlock()
		if !w.started {
			close(w.done)
		}
	}
	w.lifecycleMu.Unlock()
	select {
	case <-w.done:
		w.wg.Wait()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// CancelCurrent 强停当前正在跑的 run（不清队列）。
func (w *Worker) CancelCurrent() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.cancel != nil {
		w.cancel()
	}
}

func (w *Worker) loop() {
	defer w.wg.Done()
	defer close(w.done)
	for {
		select {
		case <-w.stopCh:
			return
		default:
		}
		run, ok := w.queue.Pop()
		if !ok {
			return // queue closed
		}
		// 再次检查 stopCh：Pop 拿到 run 后、executeRun 开始前，Stop 可能调用过
		// （把 cancel 设到 nil）。这时直接退出避免空跑。
		select {
		case <-w.stopCh:
			return
		default:
		}
		w.executeRun(run)
	}
}

func (w *Worker) executeRun(run QueuedRun) {
	parent := w.lifetime
	var ctx context.Context
	var cancel context.CancelFunc
	if run.Deadline != nil {
		ctx, cancel = context.WithDeadline(parent, *run.Deadline)
	} else {
		ctx, cancel = context.WithCancel(parent)
	}
	w.mu.Lock()
	w.active = &run
	w.cancel = cancel
	w.mu.Unlock()

	// 总会发一次 Running:true（即使 Targets 为空），让前端"试运行 → 立刻跑完"
	// 也能看到 status:true → status:false 一对对称事件。
	if w.emitRun != nil {
		w.emitRun(WorkerState{Running: true, RunID: run.ID, Source: run.Source, Targets: run.Targets})
	}

	var lastErr error
	for i, target := range run.Targets {
		if ctx.Err() != nil {
			break
		}
		if w.emitRun != nil && i > 0 {
			// targets[0] 已在循环外 emit 过 Running:true（带 TargetIdx=0 隐式）；
			// targets[1+] 才需要逐个推进 TargetIdx 让 status bar 更新。
			w.emitRun(WorkerState{Running: true, RunID: run.ID, Source: run.Source, Targets: run.Targets, TargetIdx: i})
		}
		err := w.run(ctx, target)
		if err != nil {
			lastErr = err
			if run.OnError == OnErrorStop {
				break
			}
		}
	}

	cancel()
	w.mu.Lock()
	w.active = nil
	w.cancel = nil
	w.mu.Unlock()
	if w.emitRun != nil {
		state := WorkerState{Running: false, RunID: run.ID, Source: run.Source}
		if lastErr != nil && w.classify != nil {
			state.Error = w.classify(lastErr)
		}
		w.emitRun(state)
	}
}

// IsRunning true 当 worker 正在执行某 run。
func (w *Worker) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.active != nil
}

// ActiveRun 当前 run 快照（nil = 闲）。
func (w *Worker) ActiveRun() *QueuedRun {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.active == nil {
		return nil
	}
	cp := *w.active
	return &cp
}

// ErrWorkerStopped Worker 停了之后再 Pop。
var ErrWorkerStopped = errors.New("execution: worker stopped")

// Sleep 给节点用的可取消 sleep helper。
func Sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
