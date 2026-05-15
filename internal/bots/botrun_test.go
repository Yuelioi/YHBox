package bots

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"yhbox/internal/services"
)

func TestBotRun_AfterExitOnce(t *testing.T) {
	app := &services.App{}
	r := newBotRun("fish", app)
	lease := services.NewBotLease(app, "fish")
	r.lease = lease
	r.done = make(chan struct{})

	// 调 afterExit 3 次：必须幂等不 panic
	r.afterExit()
	r.afterExit()
	r.afterExit()

	if !lease.IsReleased() {
		t.Error("lease should be released")
	}
}

func TestBotRun_WorkerPanicRecover(t *testing.T) {
	app := &services.App{}
	r := newBotRun("fish", app)
	r.done = make(chan struct{})
	lease := services.NewBotLease(app, "fish")
	r.lease = lease
	r.ctx, r.cancel = context.WithCancel(context.Background())

	logger := zerolog.Nop()

	// worker 故意 panic
	go r.runWorker(logger, func() {
		panic("intentional test panic")
	})

	// 等 worker 退出
	select {
	case <-r.done:
		// 正常退出
	case <-time.After(time.Second):
		t.Fatal("worker did not exit after panic")
	}

	if !lease.IsReleased() {
		t.Error("lease should be released after panic recovery")
	}
}

func TestBotRun_StopSyncWaitsForDone(t *testing.T) {
	app := &services.App{}
	r := newBotRun("fish", app)
	r.done = make(chan struct{})
	r.lease = services.NewBotLease(app, "fish")
	r.ctrl = NewBotControl()
	r.ctx, r.cancel = context.WithCancel(context.Background())

	logger := zerolog.Nop()

	var workerDone atomic.Bool
	go r.runWorker(logger, func() {
		<-r.ctx.Done()
		time.Sleep(50 * time.Millisecond)
		workerDone.Store(true)
	})

	r.stopSync()

	if !workerDone.Load() {
		t.Error("stopSync should wait for worker to actually finish")
	}
}

func TestBotRun_OpMuSerializesOps(t *testing.T) {
	app := &services.App{}
	r := newBotRun("fish", app)

	var counter atomic.Int32
	var max atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.opMu.Lock()
			cur := counter.Add(1)
			for {
				m := max.Load()
				if cur <= m || max.CompareAndSwap(m, cur) {
					break
				}
			}
			time.Sleep(time.Millisecond)
			counter.Add(-1)
			r.opMu.Unlock()
		}()
	}
	wg.Wait()
	if max.Load() != 1 {
		t.Errorf("opMu should serialize; observed max concurrent = %d", max.Load())
	}
}
