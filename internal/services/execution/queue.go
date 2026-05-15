package execution

import (
	"sync"
	"sync/atomic"
	"time"
)

// TriggerSource 标记 run 怎么进队。
type TriggerSource string

const (
	SourceHotkey   TriggerSource = "hotkey"
	SourceSchedule TriggerSource = "schedule"
	SourceManual   TriggerSource = "manual"
)

// OnErrorMode：run 含多 target 时，某 target 报错怎么处理。
type OnErrorMode string

const (
	OnErrorStop     OnErrorMode = "stop"
	OnErrorContinue OnErrorMode = "continue"
)

// TargetRef 一次 run 的目标。v1 仅 container。
type TargetRef struct {
	Kind string // "container"
	ID   string
}

// QueuedRun ExecutionQueue 的元素。一条入队一次 dispatch。
type QueuedRun struct {
	ID         int64
	Targets    []TargetRef
	OnError    OnErrorMode
	Deadline   *time.Time // 由 Schedule.timeoutMinutes 算出；nil = 无限
	Source     TriggerSource
	EnqueuedAt time.Time
}

// ExecutionQueue 单生产者多消费者 FIFO。生产侧（hotkey/schedule/UI）非阻塞调
// Enqueue；消费侧（Worker）block 在 Pop 上直到队首到达或 ctx cancel。
type ExecutionQueue struct {
	mu      sync.Mutex
	items   []QueuedRun
	cond    *sync.Cond
	closed  bool
	runIDSq atomic.Int64
}

func NewExecutionQueue() *ExecutionQueue {
	q := &ExecutionQueue{}
	q.cond = sync.NewCond(&q.mu)
	return q
}

// Enqueue 非阻塞入队。closed 后入队失败返 false。
func (q *ExecutionQueue) Enqueue(r QueuedRun) (int64, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return 0, false
	}
	if r.ID == 0 {
		r.ID = q.runIDSq.Add(1)
	}
	if r.EnqueuedAt.IsZero() {
		r.EnqueuedAt = time.Now()
	}
	q.items = append(q.items, r)
	q.cond.Signal()
	return r.ID, true
}

// Pop 阻塞取队首；closed 后返 (zero, false)。
func (q *ExecutionQueue) Pop() (QueuedRun, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for !q.closed && len(q.items) == 0 {
		q.cond.Wait()
	}
	if q.closed && len(q.items) == 0 {
		return QueuedRun{}, false
	}
	r := q.items[0]
	q.items = q.items[1:]
	return r, true
}

// Snapshot 当前队列只读视图（给 UI 显示用）。
func (q *ExecutionQueue) Snapshot() []QueuedRun {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := make([]QueuedRun, len(q.items))
	copy(out, q.items)
	return out
}

// CancelAll 清空整队（不影响已经 pop 走的 run）。
func (q *ExecutionQueue) CancelAll() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.items = nil
}

// Close 关闭队列。Pop 返 false，后续 Enqueue 失败。
func (q *ExecutionQueue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.closed = true
	q.cond.Broadcast()
}
