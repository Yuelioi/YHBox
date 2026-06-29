package execution

import (
	"context"
	"testing"
)

func TestWorker_IsRunning_FalseWhenIdle(t *testing.T) {
	q := NewExecutionQueue()
	w := NewWorker(q, func(ctx context.Context, _ TargetRef) error { return nil }, nil, nil)
	if w.IsRunning() {
		t.Fatal("空闲 Worker 不该 IsRunning")
	}
}
