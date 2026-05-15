package execution

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestQueueFIFO(t *testing.T) {
	q := NewExecutionQueue()
	q.Enqueue(QueuedRun{Targets: []TargetRef{{Kind: "container", ID: "A"}}})
	q.Enqueue(QueuedRun{Targets: []TargetRef{{Kind: "container", ID: "B"}}})
	r1, _ := q.Pop()
	r2, _ := q.Pop()
	if r1.Targets[0].ID != "A" || r2.Targets[0].ID != "B" {
		t.Errorf("FIFO 顺序错: %q %q", r1.Targets[0].ID, r2.Targets[0].ID)
	}
}

func TestQueueCancelAll(t *testing.T) {
	q := NewExecutionQueue()
	q.Enqueue(QueuedRun{Targets: []TargetRef{{ID: "A"}}})
	q.Enqueue(QueuedRun{Targets: []TargetRef{{ID: "B"}}})
	q.CancelAll()
	if got := q.Snapshot(); len(got) != 0 {
		t.Errorf("CancelAll 后还剩 %d 个", len(got))
	}
}

func TestWorkerExecutesSequentially(t *testing.T) {
	q := NewExecutionQueue()
	var counter atomic.Int32
	w := NewWorker(q, func(ctx context.Context, t TargetRef) error {
		counter.Add(1)
		return nil
	}, nil)
	w.Start()
	defer w.Stop()
	q.Enqueue(QueuedRun{Targets: []TargetRef{{ID: "A"}, {ID: "B"}, {ID: "C"}}})

	for i := 0; i < 50 && counter.Load() < 3; i++ {
		time.Sleep(10 * time.Millisecond)
	}
	if got := counter.Load(); got != 3 {
		t.Errorf("expected 3 targets executed, got %d", got)
	}
}

func TestWorkerOnErrorStop(t *testing.T) {
	q := NewExecutionQueue()
	var counter atomic.Int32
	w := NewWorker(q, func(ctx context.Context, t TargetRef) error {
		counter.Add(1)
		if t.ID == "B" {
			return context.Canceled // 故意报错
		}
		return nil
	}, nil)
	w.Start()
	defer w.Stop()
	q.Enqueue(QueuedRun{Targets: []TargetRef{{ID: "A"}, {ID: "B"}, {ID: "C"}}, OnError: OnErrorStop})

	for i := 0; i < 50 && counter.Load() < 2; i++ {
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond)
	if got := counter.Load(); got != 2 {
		t.Errorf("OnErrorStop 后应跑 2，实际跑 %d", got)
	}
}

func TestWorkerDeadlineCancel(t *testing.T) {
	q := NewExecutionQueue()
	var finished atomic.Bool
	w := NewWorker(q, func(ctx context.Context, t TargetRef) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
			finished.Store(true)
			return nil
		}
	}, nil)
	w.Start()
	defer w.Stop()

	dl := time.Now().Add(50 * time.Millisecond)
	q.Enqueue(QueuedRun{Targets: []TargetRef{{ID: "long"}}, Deadline: &dl})

	time.Sleep(200 * time.Millisecond)
	if finished.Load() {
		t.Error("deadline 应该把 run 切断，target 不应该自然完成")
	}
}
